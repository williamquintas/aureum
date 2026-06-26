CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Investments table
CREATE TABLE investments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    name            VARCHAR(255) NOT NULL,
    ticker          VARCHAR(20) NOT NULL,
    asset_type      VARCHAR(50) NOT NULL CHECK (asset_type IN (
                        'stock', 'etf', 'real_estate_fund', 'treasury',
                        'cdb', 'lci', 'lca', 'crypto', 'pension',
                        'fund', 'dollar', 'gold', 'other'
                    )),
    quantity        BIGINT NOT NULL CHECK (quantity > 0),
    average_price   BIGINT NOT NULL CHECK (average_price >= 0),  -- cents per unit
    total_invested  BIGINT NOT NULL CHECK (total_invested >= 0), -- cents
    status          VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'sold', 'cancelled')),
    broker          VARCHAR(100) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_investments_user_status ON investments (user_id, status);
CREATE INDEX idx_investments_user_asset ON investments (user_id, asset_type);
CREATE INDEX idx_investments_deleted_at ON investments (deleted_at) WHERE deleted_at IS NULL;

-- Investment transactions table
CREATE TABLE investment_transactions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id     UUID NOT NULL REFERENCES investments(id) ON DELETE CASCADE,
    user_id           UUID NOT NULL,
    transaction_type  VARCHAR(20) NOT NULL CHECK (transaction_type IN ('buy', 'sell', 'dividend', 'jcp', 'amortization')),
    quantity          BIGINT NOT NULL CHECK (quantity > 0),
    unit_price        BIGINT NOT NULL CHECK (unit_price >= 0),    -- cents per unit
    total_amount      BIGINT NOT NULL CHECK (total_amount >= 0),  -- cents
    transaction_date  DATE NOT NULL,
    notes             TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_investment ON investment_transactions (investment_id);
CREATE INDEX idx_transactions_user_date ON investment_transactions (user_id, transaction_date);

-- Outbox events table
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    aggregate_type VARCHAR(255) NOT NULL,
    aggregate_id VARCHAR(255) NOT NULL DEFAULT '',
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_events_published_at ON outbox_events(published_at);
CREATE INDEX idx_outbox_events_event_type ON outbox_events(event_type);

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_investments_updated_at
    BEFORE UPDATE ON investments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
