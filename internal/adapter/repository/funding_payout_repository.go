package repository

import (
	"context"
	"time"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/jackc/pgx/v5"
)

// ClaimPayout atomically takes a completed, unpaid checkout so exactly one
// worker pays it. Returns nil if nothing was claimable.
func (r *fundingRepository) ClaimPayout(ctx context.Context, reference string) (*ports.PayoutJob, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE funding_transactions
		SET payout_status = 'sending', payout_error = NULL, updated_at = now()
		WHERE reference = $1 AND status = 'completed' AND payout_status IN ('none','retry','waiting_treasury')
		RETURNING reference, wallet_address, amount_ngn::float8, estimated_usdc::float8
	`, reference)
	j := &ports.PayoutJob{}
	if err := row.Scan(&j.Reference, &j.Wallet, &j.AmountNGN, &j.EstimatedUSDC); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return j, nil
}

// SetPayoutSending stores the signature BEFORE broadcast, so a crash can be reconciled on chain.
func (r *fundingRepository) SetPayoutSending(ctx context.Context, reference, signature string, usdc float64) error {
	_, err := r.db.Exec(ctx, `UPDATE funding_transactions SET payout_signature=$2, payout_usdc=$3, updated_at=now() WHERE reference=$1`, reference, signature, usdc)
	return err
}

func (r *fundingRepository) SetPayoutResult(ctx context.Context, reference, status, errMsg string) error {
	var paidAt *time.Time
	if status == "paid" {
		now := time.Now()
		paidAt = &now
	}
	_, err := r.db.Exec(ctx, `
		UPDATE funding_transactions
		SET payout_status=$2, payout_error=NULLIF($3,''), payout_at=COALESCE($4, payout_at), updated_at=now()
		WHERE reference=$1
	`, reference, status, errMsg, paidAt)
	return err
}

// ListPayable returns completed checkouts that still need a payout.
func (r *fundingRepository) ListPayable(ctx context.Context, limit int) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT reference FROM funding_transactions
		WHERE status='completed' AND payout_status IN ('none','retry','waiting_treasury')
		ORDER BY created_at ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var refs []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		refs = append(refs, s)
	}
	return refs, nil
}

// ListStuckSending returns payouts that were signed but never confirmed.
func (r *fundingRepository) ListStuckSending(ctx context.Context, olderThan time.Duration) ([]ports.PayoutJob, error) {
	rows, err := r.db.Query(ctx, `
		SELECT reference, wallet_address, amount_ngn::float8, estimated_usdc::float8, COALESCE(payout_signature,'')
		FROM funding_transactions
		WHERE payout_status='sending' AND updated_at < now() - make_interval(secs => $1)`, olderThan.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ports.PayoutJob
	for rows.Next() {
		var j ports.PayoutJob
		if err := rows.Scan(&j.Reference, &j.Wallet, &j.AmountNGN, &j.EstimatedUSDC, &j.Signature); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, nil
}

// GetPayoutInfo is shown to the app so it can say "sent to your wallet".
func (r *fundingRepository) GetPayoutInfo(ctx context.Context, reference string) (*domain.PayoutInfo, error) {
	row := r.db.QueryRow(ctx, `SELECT payout_status, COALESCE(payout_signature,''), COALESCE(payout_usdc,0)::float8, COALESCE(payout_error,'') FROM funding_transactions WHERE reference=$1`, reference)
	p := &domain.PayoutInfo{}
	if err := row.Scan(&p.Status, &p.Signature, &p.USDC, &p.Error); err != nil {
		return nil, err
	}
	return p, nil
}
