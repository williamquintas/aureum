CREATE TABLE debt_summary (
    user_id          UUID PRIMARY KEY,
    date             DATE NOT NULL,
    total_debt       BIGINT NOT NULL DEFAULT 0,
    total_limit      BIGINT NOT NULL DEFAULT 0,
    credit_util_pct  DOUBLE PRECISION NOT NULL DEFAULT 0
);

CREATE INDEX idx_debt_summary_user ON debt_summary (user_id);
