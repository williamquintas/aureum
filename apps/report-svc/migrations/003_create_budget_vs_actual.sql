CREATE TABLE budget_vs_actual (
    user_id          UUID NOT NULL,
    budget_id        UUID NOT NULL,
    year             INT NOT NULL,
    month            INT NOT NULL CHECK (month >= 1 AND month <= 12),
    category         VARCHAR(100) NOT NULL,
    budgeted         BIGINT NOT NULL DEFAULT 0,
    actual           BIGINT NOT NULL DEFAULT 0,
    variance         BIGINT NOT NULL DEFAULT 0,
    variance_pct     DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, budget_id, year, month, category)
);

CREATE INDEX idx_budget_vs_actual_user ON budget_vs_actual (user_id, budget_id);
CREATE INDEX idx_budget_vs_actual_period ON budget_vs_actual (user_id, year, month);
