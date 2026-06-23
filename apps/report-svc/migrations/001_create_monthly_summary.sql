CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE monthly_summary (
    user_id         UUID NOT NULL,
    year            INT NOT NULL,
    month           INT NOT NULL CHECK (month >= 1 AND month <= 12),
    total_income    BIGINT NOT NULL DEFAULT 0,
    total_expenses  BIGINT NOT NULL DEFAULT 0,
    net_savings     BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, year, month)
);

CREATE INDEX idx_monthly_summary_user ON monthly_summary (user_id, year, month);
