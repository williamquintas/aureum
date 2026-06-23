CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Budget periods enum as CHECK constraint
DO $$ BEGIN
    CREATE TYPE budget_period AS ENUM (
        'monthly', 'bimonthly', 'quarterly', 'semestral', 'yearly', 'custom'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE budget_status AS ENUM (
        'active', 'paused', 'completed', 'cancelled'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE budgets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    period          VARCHAR(20) NOT NULL CHECK (period IN ('monthly', 'bimonthly', 'quarterly', 'semestral', 'yearly', 'custom')),
    total_limit     BIGINT NOT NULL CHECK (total_limit > 0),
    spent_amount    BIGINT NOT NULL DEFAULT 0 CHECK (spent_amount >= 0),
    status          VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed', 'cancelled')),
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT chk_date_range CHECK (end_date >= start_date)
);

CREATE TABLE budget_categories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_id       UUID NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    limit_amount    BIGINT NOT NULL CHECK (limit_amount > 0),
    spent_amount    BIGINT NOT NULL DEFAULT 0 CHECK (spent_amount >= 0),
    category        VARCHAR(100) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_budgets_user_start_date ON budgets (user_id, start_date);
CREATE INDEX idx_budgets_user_status ON budgets (user_id, status);
CREATE INDEX idx_budgets_user_end_date ON budgets (user_id, end_date);
CREATE INDEX idx_budgets_deleted_at ON budgets (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_budget_categories_budget_id ON budget_categories (budget_id);
CREATE INDEX idx_budget_categories_deleted_at ON budget_categories (deleted_at) WHERE deleted_at IS NULL;

-- Trigger function for updating updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_budgets_updated_at
    BEFORE UPDATE ON budgets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_budget_categories_updated_at
    BEFORE UPDATE ON budget_categories
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
