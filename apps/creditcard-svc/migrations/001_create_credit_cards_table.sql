CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE credit_cards (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL,
    name             VARCHAR(255) NOT NULL,
    brand            VARCHAR(50) NOT NULL CHECK (brand IN ('visa', 'mastercard', 'amex', 'elo', 'hipercard', 'diners', 'other')),
    card_type        VARCHAR(50) NOT NULL CHECK (card_type IN ('credit', 'debit', 'multiple')),
    last_four_digits VARCHAR(4) NOT NULL,
    closing_day      INT NOT NULL CHECK (closing_day BETWEEN 1 AND 31),
    due_day          INT NOT NULL CHECK (due_day BETWEEN 1 AND 31),
    credit_limit     BIGINT NOT NULL CHECK (credit_limit >= 0),
    available_credit BIGINT NOT NULL CHECK (available_credit >= 0),
    active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX idx_credit_cards_user_id ON credit_cards (user_id);
CREATE INDEX idx_credit_cards_user_active ON credit_cards (user_id, active);
CREATE INDEX idx_credit_cards_deleted_at ON credit_cards (deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE invoices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credit_card_id  UUID NOT NULL REFERENCES credit_cards(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL,
    reference_month VARCHAR(7) NOT NULL,
    total_amount    BIGINT NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    paid_amount     BIGINT NOT NULL DEFAULT 0 CHECK (paid_amount >= 0),
    status          VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed', 'paid', 'overdue')),
    closing_date    DATE NOT NULL,
    due_date        DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_invoices_credit_card_id ON invoices (credit_card_id);
CREATE INDEX idx_invoices_user_id ON invoices (user_id);
CREATE INDEX idx_invoices_reference_month ON invoices (credit_card_id, reference_month);
CREATE INDEX idx_invoices_status ON invoices (status);
CREATE INDEX idx_invoices_deleted_at ON invoices (deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE invoice_transactions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id       UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL,
    description      VARCHAR(500) NOT NULL,
    amount           BIGINT NOT NULL CHECK (amount != 0),
    category         VARCHAR(100) NOT NULL DEFAULT 'other',
    transaction_date DATE NOT NULL,
    installments     INT NOT NULL DEFAULT 1 CHECK (installments >= 1),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoice_transactions_invoice_id ON invoice_transactions (invoice_id);
CREATE INDEX idx_invoice_transactions_category ON invoice_transactions (category);
CREATE INDEX idx_invoice_transactions_date ON invoice_transactions (transaction_date);
