CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE incomes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    description     VARCHAR(255) NOT NULL,
    source          VARCHAR(100) NOT NULL,
    income_type     VARCHAR(50) NOT NULL CHECK (income_type IN ('salary', 'freelance', 'investment', 'business', 'refund', 'other')),
    received_date   DATE NOT NULL,
    received_amount BIGINT NOT NULL CHECK (received_amount > 0),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_incomes_user_date ON incomes (user_id, received_date);
CREATE INDEX idx_incomes_user_status ON incomes (user_id, status);
CREATE INDEX idx_incomes_deleted_at ON incomes (deleted_at) WHERE deleted_at IS NULL;
