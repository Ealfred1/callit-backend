DROP INDEX IF EXISTS idx_funding_payout;
ALTER TABLE funding_transactions
    DROP COLUMN IF EXISTS payout_status,
    DROP COLUMN IF EXISTS payout_usdc,
    DROP COLUMN IF EXISTS payout_signature,
    DROP COLUMN IF EXISTS payout_error,
    DROP COLUMN IF EXISTS payout_at;
