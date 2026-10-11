package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/google/uuid"
)

type FundingService struct {
	fundingRepo   ports.FundingRepository
	userRepo      ports.UserRepository
	fossapay      ports.FossapayClient
	webhookSecret string
	logger        *slog.Logger
}

func NewFundingService(
	fundingRepo ports.FundingRepository,
	userRepo ports.UserRepository,
	fossapay ports.FossapayClient,
	webhookSecret string,
	logger *slog.Logger,
) *FundingService {
	return &FundingService{
		fundingRepo:   fundingRepo,
		userRepo:      userRepo,
		fossapay:      fossapay,
		webhookSecret: webhookSecret,
		logger:        logger,
	}
}

// Fixed or indicative exchange rate: 1 USDC ≈ 1,550 NGN
const ngnPerUsdc = 1550.0

func (s *FundingService) CreateCheckout(ctx context.Context, userIDStr string, req domain.CreateFundingRequest) (*domain.FundingCheckoutResponse, error) {
	if req.AmountNGN < 100 {
		return nil, fmt.Errorf("minimum funding amount is 100 NGN")
	}
	if req.WalletAddress == "" {
		return nil, fmt.Errorf("walletAddress is required")
	}

	user, err := s.userRepo.GetByID(ctx, userIDStr)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	userUUID := user.ID

	reference := fmt.Sprintf("CALLIT-%d-%s", time.Now().Unix(), uuid.New().String()[:8])
	amountWhole := int64(req.AmountNGN)
	estimatedUSDC := float64(amountWhole) / ngnPerUsdc

	phone := normalizePhoneNumber(req.PhoneNumber)
	fpReq := domain.FossapayCreateCheckoutReq{
		Amount:    amountWhole,
		Currency:  "NGN",
		Reference: reference,
		FeeBearer: "customer",
		Customer: domain.FossapayCustomer{
			Name:        user.Name,
			Email:       user.Email,
			PhoneNumber: phone,
		},
		Metadata: map[string]any{
			"userId":        user.ID.String(),
			"walletAddress": req.WalletAddress,
			"reference":     reference,
		},
		Description: fmt.Sprintf("Callit wallet funding (%s)", req.WalletAddress),
	}

	idempotencyKey := "checkout-" + reference
	resp, err := s.fossapay.CreateCheckout(ctx, idempotencyKey, fpReq)
	if err != nil {
		s.logger.Error("FossaPay CreateCheckout failed", "err", err, "reference", reference)
		return nil, fmt.Errorf("failed to create payment checkout: %w", err)
	}

	amountPayable, _ := strconv.ParseFloat(resp.Data.AmountPayable, 64)
	if amountPayable == 0 {
		amountPayable = float64(amountWhole)
	}

	tx := &domain.FundingTransaction{
		UserID:           userUUID,
		WalletAddress:    req.WalletAddress,
		Reference:        reference,
		CheckoutID:       resp.Data.ID,
		AmountNGN:        float64(amountWhole),
		AmountPayableNGN: amountPayable,
		EstimatedUSDC:    estimatedUSDC,
		Status:           domain.FundingStatusPending,
		BankName:         resp.Data.Account.BankName,
		AccountName:      resp.Data.Account.AccountName,
		AccountNumber:    resp.Data.Account.AccountNumber,
		ExpiresAt:        resp.Data.ExpiresAt,
		Metadata:         fpReq.Metadata,
	}

	if err := s.fundingRepo.Create(ctx, tx); err != nil {
		s.logger.Error("failed to record funding transaction in db", "err", err, "reference", reference)
		return nil, fmt.Errorf("record funding transaction: %w", err)
	}

	return &domain.FundingCheckoutResponse{
		Reference:        reference,
		CheckoutID:       resp.Data.ID,
		AmountNGN:        float64(amountWhole),
		AmountPayableNGN: amountPayable,
		EstimatedUSDC:    estimatedUSDC,
		Currency:         "NGN",
		Status:           resp.Data.Status,
		BankName:         resp.Data.Account.BankName,
		AccountName:      resp.Data.Account.AccountName,
		AccountNumber:    resp.Data.Account.AccountNumber,
		ExpiresAt:        resp.Data.ExpiresAt,
	}, nil
}

func (s *FundingService) GetCheckoutStatus(ctx context.Context, reference string) (*domain.FundingTransaction, error) {
	tx, err := s.fundingRepo.GetByReference(ctx, reference)
	if err != nil {
		return nil, err
	}

	// If pending and expired, check with FossaPay upstream for reconciliation
	if tx.Status == domain.FundingStatusPending {
		fpResp, err := s.fossapay.GetCheckoutByReference(ctx, reference)
		if err == nil && fpResp != nil && fpResp.Data.Status != "" {
			newStatus := domain.FundingStatus(fpResp.Data.Status)
			if newStatus != tx.Status {
				tx.Status = newStatus
				tx.CompletedAt = fpResp.Data.CompletedAt
				_ = s.fundingRepo.UpdateStatus(ctx, reference, newStatus, tx.CompletedAt)
			}
		}
	}

	return tx, nil
}

func (s *FundingService) HandleWebhook(ctx context.Context, rawBody []byte, signature string) error {
	if s.webhookSecret != "" {
		if !s.fossapay.VerifyWebhookSignature(rawBody, signature, s.webhookSecret) {
			s.logger.Warn("invalid fossapay webhook signature", "signature", signature)
			return fmt.Errorf("invalid webhook signature")
		}
	}

	var payload domain.FossapayWebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return fmt.Errorf("unmarshal webhook payload: %w", err)
	}

	ref, _ := payload.Data["reference"].(string)
	if ref == "" {
		s.logger.Warn("fossapay webhook missing reference", "eventId", payload.EventID, "type", payload.EventType)
		return nil
	}

	s.logger.Info("processing fossapay webhook", "eventType", payload.EventType, "reference", ref)

	switch payload.EventType {
	case "checkout.completed":
		now := time.Now()
		if err := s.fundingRepo.UpdateStatus(ctx, ref, domain.FundingStatusCompleted, &now); err != nil {
			s.logger.Error("failed to mark funding completed", "reference", ref, "err", err)
			return err
		}
		s.logger.Info("funding successfully completed", "reference", ref)

	case "checkout.failed":
		if err := s.fundingRepo.UpdateStatus(ctx, ref, domain.FundingStatusFailed, nil); err != nil {
			s.logger.Error("failed to mark funding failed", "reference", ref, "err", err)
			return err
		}

	case "checkout.expired":
		if err := s.fundingRepo.UpdateStatus(ctx, ref, domain.FundingStatusExpired, nil); err != nil {
			s.logger.Error("failed to mark funding expired", "reference", ref, "err", err)
			return err
		}
	}

	return nil
}

func normalizePhoneNumber(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = strings.ReplaceAll(p, " ", "")
	p = strings.ReplaceAll(p, "-", "")
	p = strings.ReplaceAll(p, "(", "")
	p = strings.ReplaceAll(p, ")", "")
	if strings.HasPrefix(p, "+") {
		return p
	}
	if strings.HasPrefix(p, "0") && len(p) == 11 {
		return "+234" + p[1:]
	}
	if strings.HasPrefix(p, "234") {
		return "+" + p
	}
	return ""
}

