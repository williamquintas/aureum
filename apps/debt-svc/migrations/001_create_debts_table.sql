CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE debts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL,
    name             VARCHAR(255) NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    debt_type        VARCHAR(50) NOT NULL CHECK (debt_type IN ('personal_loan', 'student_loan', 'mortgage', 'car_loan', 'credit_card_debt', 'medical_debt', 'other')),
    total_amount     BIGINT NOT NULL CHECK (total_amount > 0),
    remaining_amount BIGINT NOT NULL CHECK (remaining_amount >= 0),
    interest_rate    BIGINT NOT NULL DEFAULT 0,
    start_date       DATE NOT NULL,
    expected_end_date DATE,
    status           VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'paid_off', 'defaulted', 'settled')),
    creditor         VARCHAR(255) NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX idx_debts_user_status ON debts (user_id, status);
CREATE INDEX idx_debts_user_type ON debts (user_id, debt_type);
CREATE INDEX idx_debts_user_id ON debts (user_id);
CREATE INDEX idx_debts_deleted_at ON debts (deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE payments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    debt_id       UUID NOT NULL REFERENCES debts(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL,
    amount        BIGINT NOT NULL CHECK (amount > 0),
    payment_date  DATE NOT NULL,
    notes         TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_payments_debt_id ON payments (debt_id);
CREATE INDEX idx_payments_user_id ON payments (user_id);
CREATE INDEX idx_payments_debt_date ON payments (debt_id, payment_date);
CREATE INDEX idx_payments_deleted_at ON payments (deleted_at) WHERE deleted_at IS NULL;

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

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_debts_updated_at
    BEFORE UPDATE ON debts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
