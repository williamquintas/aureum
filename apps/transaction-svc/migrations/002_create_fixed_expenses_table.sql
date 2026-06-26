CREATE TABLE fixed_expenses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    description     VARCHAR(255) NOT NULL,
    category        VARCHAR(100) NOT NULL,
    day_of_month    INT NOT NULL CHECK (day_of_month BETWEEN 1 AND 31),
    payment_method  VARCHAR(50) NOT NULL CHECK (payment_method IN ('credit_card', 'debit_card', 'cash', 'bank_transfer', 'pix', 'other')),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_fixed_expenses_user_day ON fixed_expenses (user_id, day_of_month);
CREATE INDEX idx_fixed_expenses_user_status ON fixed_expenses (user_id, status);
CREATE INDEX idx_fixed_expenses_deleted_at ON fixed_expenses (deleted_at) WHERE deleted_at IS NULL;
