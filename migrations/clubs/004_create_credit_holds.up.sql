
CREATE TABLE credit_holds (
    id         UUID PRIMARY KEY,
    club_id    UUID NOT NULL,
    amount     BIGINT NOT NULL CHECK (amount > 0),
    reason     TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at TIMESTAMPTZ,

    CONSTRAINT fk_hold_club
        FOREIGN KEY (club_id)
        REFERENCES clubs(id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX one_active_hold_per_club
    ON credit_holds (club_id)
    WHERE released_at IS NULL;
