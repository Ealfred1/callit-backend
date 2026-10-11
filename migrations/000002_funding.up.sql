CREATE TABLE IF NOT EXISTS funding_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_address VARCHAR(128) NOT NULL,
    reference VARCHAR(128) NOT NULL UNIQUE,
    checkout_id VARCHAR(128),
    amount_ngn NUMERIC(14, 2) NOT NULL,
    amount_payable_ngn NUMERIC(14, 2) NOT NULL,
    estimated_usdc NUMERIC(14, 4) NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    bank_name VARCHAR(128),
    account_name VARCHAR(255),
    account_number VARCHAR(64),
    expires_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_funding_transactions_user ON funding_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_funding_transactions_ref ON funding_transactions(reference);
CREATE INDEX IF NOT EXISTS idx_funding_transactions_wallet ON funding_transactions(wallet_address);
