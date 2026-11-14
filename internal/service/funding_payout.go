package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/eric/callit-backend/internal/core/ports"
)

const minPayoutUSDC = 0.01

// SetPayer turns on automatic USDC payouts. maxUSDC caps a single payout;
// anything larger is held for a human.
func (s *FundingService) SetPayer(p ports.Payer, maxUSDC float64) {
	s.payer = p
	if maxUSDC <= 0 {
		maxUSDC = 100
	}
	s.maxPayout = maxUSDC
}

// TryPayout sends the USDC for one confirmed naira payment. Safe to call any
// number of times: the claim in the database lets exactly one caller proceed,
// and the signature is stored before broadcast so a crash can be reconciled.
func (s *FundingService) TryPayout(ctx context.Context, reference string) {
	if s.payer == nil {
		return
	}
	job, err := s.fundingRepo.ClaimPayout(ctx, reference)
	if err != nil {
		s.logger.Error("payout: claim failed", "reference", reference, "err", err)
		return
	}
	if job == nil {
		return // not completed, or already handled
	}
	usdc := math.Floor(job.EstimatedUSDC*1e6) / 1e6
	switch {
	case usdc < minPayoutUSDC:
		_ = s.fundingRepo.SetPayoutResult(ctx, job.Reference, "failed", "amount too small to send")
		return
	case usdc > s.maxPayout:
		_ = s.fundingRepo.SetPayoutResult(ctx, job.Reference, "failed", fmt.Sprintf("%.2f USDC is over the %.0f USDC automatic limit; needs manual review", usdc, s.maxPayout))
		s.logger.Warn("payout: over cap, held for review", "reference", job.Reference, "usdc", usdc)
		return
	}

	sol, bal, err := s.payer.Balances(ctx)
	if err != nil {
		_ = s.fundingRepo.SetPayoutResult(ctx, job.Reference, "retry", "could not read treasury balance: "+err.Error())
		return
	}
	if sol < 0.004 || bal < usdc {
		_ = s.fundingRepo.SetPayoutResult(ctx, job.Reference, "waiting_treasury", fmt.Sprintf("treasury %s needs %.2f USDC and a little SOL (has %.2f USDC, %.4f SOL)", s.payer.Address(), usdc, bal, sol))
		s.logger.Warn("payout: treasury underfunded", "reference", job.Reference, "need", usdc, "usdc", bal, "sol", sol, "address", s.payer.Address())
		return
	}

	sig, state, err := s.payer.Pay(ctx, job.Wallet, usdc, func(sig string) error {
		return s.fundingRepo.SetPayoutSending(ctx, job.Reference, sig, usdc)
	})
	switch {
	case err != nil && sig == "":
		_ = s.fundingRepo.SetPayoutResult(ctx, job.Reference, "retry", err.Error())
		s.logger.Error("payout: not sent", "reference", job.Reference, "err", err)
	case state == "confirmed":
		_ = s.fundingRepo.SetPayoutResult(ctx, job.Reference, "paid", "")
		s.logger.Info("payout: sent", "reference", job.Reference, "usdc", usdc, "signature", sig)
	case state == "failed":
		_ = s.fundingRepo.SetPayoutResult(ctx, job.Reference, "retry", fmt.Sprintf("transfer failed (%v)", err))
		s.logger.Error("payout: transfer failed", "reference", job.Reference, "signature", sig, "err", err)
	default:
		// signed and broadcast but not confirmed yet: left as 'sending'; the sweep checks the chain
		s.logger.Warn("payout: awaiting confirmation", "reference", job.Reference, "signature", sig)
	}
}

// SweepPayouts pays anything completed-but-unpaid and resolves payouts that
// were signed but never confirmed.
func (s *FundingService) SweepPayouts(ctx context.Context) {
	if s.payer == nil {
		return
	}
	stuck, err := s.fundingRepo.ListStuckSending(ctx, 3*time.Minute)
	if err == nil {
		for _, j := range stuck {
			if j.Signature == "" {
				_ = s.fundingRepo.SetPayoutResult(ctx, j.Reference, "retry", "interrupted before signing")
				continue
			}
			st, cerr := s.payer.Check(ctx, j.Signature)
			if cerr != nil {
				continue
			}
			switch st {
			case "confirmed":
				_ = s.fundingRepo.SetPayoutResult(ctx, j.Reference, "paid", "")
				s.logger.Info("payout: confirmed on chain after restart", "reference", j.Reference, "signature", j.Signature)
			default:
				// a signed transaction older than ~3 minutes has an expired blockhash and can never land
				_ = s.fundingRepo.SetPayoutResult(ctx, j.Reference, "retry", "previous attempt did not land")
			}
		}
	}
	refs, err := s.fundingRepo.ListPayable(ctx, 20)
	if err != nil {
		s.logger.Error("payout: list payable failed", "err", err)
		return
	}
	for _, ref := range refs {
		s.TryPayout(ctx, ref)
	}
}
