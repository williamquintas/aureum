CREATE TABLE variable_expenses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    description     VARCHAR(255) NOT NULL,
    destination     VARCHAR(100) NOT NULL,
    category        VARCHAR(100) NOT NULL,
    expense_type    VARCHAR(50) NOT NULL CHECK (expense_type IN ('essential', 'discretionary', 'occasional', 'emergency', 'other')),
    payment_method  VARCHAR(50) NOT NULL CHECK (payment_method IN ('credit_card', 'debit_card', 'cash', 'bank_transfer', 'pix', 'other')),
    payment_date    DATE NOT NULL,
    paid_amount     BIGINT NOT NULL CHECK (paid_amount > 0),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_variable_expenses_user_date ON variable_expenses (user_id, payment_date);
CREATE INDEX idx_variable_expenses_user_status ON variable_expenses (user_id, status);
CREATE INDEX idx_variable_expenses_user_category ON variable_expenses (user_id, category);
CREATE INDEX idx_variable_expenses_deleted_at ON variable_expenses (deleted_at) WHERE deleted_at IS NULL;
