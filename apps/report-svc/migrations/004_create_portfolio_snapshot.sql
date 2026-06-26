CREATE TABLE portfolio_snapshot (
    user_id          UUID NOT NULL,
    date             DATE NOT NULL,
    total_invested   BIGINT NOT NULL DEFAULT 0,
    current_value    BIGINT NOT NULL DEFAULT 0,
    total_return     BIGINT NOT NULL DEFAULT 0,
    return_pct       DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, date)
);

CREATE INDEX idx_portfolio_snapshot_user ON portfolio_snapshot (user_id, date);
