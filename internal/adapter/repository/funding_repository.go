package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fundingRepository struct {
	db *pgxpool.Pool
}

func NewFundingRepository(db *pgxpool.Pool) ports.FundingRepository {
	return &fundingRepository{db: db}
}

func (r *fundingRepository) Create(ctx context.Context, tx *domain.FundingTransaction) error {
	query := `
		INSERT INTO funding_transactions (
			id, user_id, wallet_address, reference, checkout_id,
			amount_ngn, amount_payable_ngn, estimated_usdc, status,
			bank_name, account_name, account_number, expires_at,
			completed_at, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17
		)
	`
	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}
	now := time.Now()
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = now
	}
	tx.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		tx.ID, tx.UserID, tx.WalletAddress, tx.Reference, tx.CheckoutID,
		tx.AmountNGN, tx.AmountPayableNGN, tx.EstimatedUSDC, tx.Status,
		tx.BankName, tx.AccountName, tx.AccountNumber, tx.ExpiresAt,
		tx.CompletedAt, tx.Metadata, tx.CreatedAt, tx.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create funding transaction: %w", err)
	}
	return nil
}

func (r *fundingRepository) scanTx(row interface{ Scan(dest ...any) error }) (*domain.FundingTransaction, error) {
	tx := &domain.FundingTransaction{}
	var statusStr string
	err := row.Scan(
		&tx.ID, &tx.UserID, &tx.WalletAddress, &tx.Reference, &tx.CheckoutID,
		&tx.AmountNGN, &tx.AmountPayableNGN, &tx.EstimatedUSDC, &statusStr,
		&tx.BankName, &tx.AccountName, &tx.AccountNumber, &tx.ExpiresAt,
		&tx.CompletedAt, &tx.Metadata, &tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	tx.Status = domain.FundingStatus(statusStr)
	return tx, nil
}

func (r *fundingRepository) GetByReference(ctx context.Context, reference string) (*domain.FundingTransaction, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, wallet_address, reference, checkout_id,
		       amount_ngn, amount_payable_ngn, estimated_usdc, status,
		       bank_name, account_name, account_number, expires_at,
		       completed_at, metadata, created_at, updated_at
		FROM funding_transactions
		WHERE reference = $1
	`, reference)
	return r.scanTx(row)
}

func (r *fundingRepository) GetByCheckoutID(ctx context.Context, checkoutID string) (*domain.FundingTransaction, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, wallet_address, reference, checkout_id,
		       amount_ngn, amount_payable_ngn, estimated_usdc, status,
		       bank_name, account_name, account_number, expires_at,
		       completed_at, metadata, created_at, updated_at
		FROM funding_transactions
		WHERE checkout_id = $1
	`, checkoutID)
	return r.scanTx(row)
}

func (r *fundingRepository) ListByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.FundingTransaction, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, wallet_address, reference, checkout_id,
		       amount_ngn, amount_payable_ngn, estimated_usdc, status,
		       bank_name, account_name, account_number, expires_at,
		       completed_at, metadata, created_at, updated_at
		FROM funding_transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.FundingTransaction
	for rows.Next() {
		tx, err := r.scanTx(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, tx)
	}
	return list, nil
}

func (r *fundingRepository) UpdateStatus(ctx context.Context, reference string, status domain.FundingStatus, completedAt *time.Time) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE funding_transactions
		SET status = $1, completed_at = $2, updated_at = $3
		WHERE reference = $4
	`, string(status), completedAt, now, reference)
	return err
}

// ListPending returns checkouts still waiting for a bank transfer, newest first.
func (r *fundingRepository) ListPending(ctx context.Context, newerThan time.Time, limit int) ([]*domain.FundingTransaction, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, wallet_address, reference, checkout_id,
		       amount_ngn, amount_payable_ngn, estimated_usdc, status,
		       bank_name, account_name, account_number, expires_at,
		       completed_at, metadata, created_at, updated_at
		FROM funding_transactions
		WHERE status = 'pending' AND created_at > $1
		ORDER BY created_at DESC
		LIMIT $2
	`, newerThan, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*domain.FundingTransaction
	for rows.Next() {
		tx, err := r.scanTx(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, tx)
	}
	return list, nil
}
