CREATE TABLE category_summary (
    user_id          UUID NOT NULL,
    year             INT NOT NULL,
    month            INT NOT NULL CHECK (month >= 1 AND month <= 12),
    category_type    VARCHAR(20) NOT NULL CHECK (category_type IN ('income', 'expense')),
    category_name    VARCHAR(100) NOT NULL,
    total_amount     BIGINT NOT NULL DEFAULT 0,
    transaction_count INT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, year, month, category_type, category_name)
);

CREATE INDEX idx_category_summary_user ON category_summary (user_id, year, month);
