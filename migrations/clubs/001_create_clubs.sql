-- 001_create_clubs.sql
-- Core entity: club
-- credits is an operational snapshot, NOT the source of truth
CREATE TABLE clubs (
  id         UUID PRIMARY KEY,
  user_id    UUID NOT NULL UNIQUE,
  credits    BIGINT NOT NULL CHECK (credits >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
