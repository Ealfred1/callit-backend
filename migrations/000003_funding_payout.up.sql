-- Tracks the USDC payout that follows a confirmed naira payment.
-- payout_status: none -> sending -> paid | retry | waiting_treasury | failed
ALTER TABLE funding_transactions
    ADD COLUMN IF NOT EXISTS payout_status    text NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS payout_usdc      numeric(18,6),
    ADD COLUMN IF NOT EXISTS payout_signature text,
    ADD COLUMN IF NOT EXISTS payout_error     text,
    ADD COLUMN IF NOT EXISTS payout_at        timestamptz;

CREATE INDEX IF NOT EXISTS idx_funding_payout ON funding_transactions (status, payout_status);
