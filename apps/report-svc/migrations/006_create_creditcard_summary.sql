CREATE TABLE creditcard_summary (
    user_id          UUID NOT NULL,
    card_name        VARCHAR(100) NOT NULL,
    statement_date   DATE,
    total_balance    BIGINT NOT NULL DEFAULT 0,
    total_limit      BIGINT NOT NULL DEFAULT 0,
    util_pct         DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, card_name)
);

CREATE INDEX idx_creditcard_summary_user ON creditcard_summary (user_id);
