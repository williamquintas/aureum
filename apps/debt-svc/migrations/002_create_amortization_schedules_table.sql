CREATE TABLE amortization_schedules (
    debt_id           UUID PRIMARY KEY REFERENCES debts(id) ON DELETE CASCADE,
    total_amount      BIGINT NOT NULL,
    monthly_payment   BIGINT NOT NULL,
    interest_rate     BIGINT NOT NULL,
    remaining_months  INT NOT NULL,
    total_interest    BIGINT NOT NULL,
    total_paid        BIGINT NOT NULL,
    entries           JSONB NOT NULL DEFAULT '[]',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_amortization_schedules_updated_at
    BEFORE UPDATE ON amortization_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
