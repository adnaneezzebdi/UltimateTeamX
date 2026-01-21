-- 002_create_user_cards.sql
-- User-owned cards
-- locked = true means the card is not transferable

CREATE TABLE user_cards (
    id UUID PRIMARY KEY,
    club_id UUID NOT NULL,
    CONSTRAINT fk_user_cards_club
    FOREIGN KEY (club_id) REFERENCES clubs(id)
    ON DELETE RESTRICT,
    player_id UUID NOT NULL,
    CONSTRAINT uq_user_cards_club_player
    UNIQUE (club_id, player_id),
    locked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);





