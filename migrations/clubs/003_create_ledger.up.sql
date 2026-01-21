
CREATE TABLE ledger (
    id         UUID PRIMARY KEY,
    club_id    UUID NOT NULL,
    amount     BIGINT NOT NULL,
    reason     TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_ledger_club
        FOREIGN KEY (club_id)
        REFERENCES clubs(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_ledger_club_created_at
    ON ledger (club_id, created_at);
