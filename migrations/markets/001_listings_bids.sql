-- Market listings/bids schema (state + audit only).
-- Concurrency handled via Redis locks; no cross-service foreign keys.
-- Listings are never deleted; status changes only.

CREATE TABLE listings (
  id                 UUID PRIMARY KEY,
  seller_club_id     UUID NOT NULL,
  user_card_id       UUID NOT NULL,
  start_price        BIGINT NOT NULL,
  buy_now_price      BIGINT,
  best_bid           BIGINT,
  best_bidder_club_id UUID,
  status             TEXT NOT NULL,
  expires_at         TIMESTAMPTZ NOT NULL,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT listings_status_check
    CHECK (status IN ('ACTIVE', 'SOLD', 'EXPIRED'))
);

CREATE INDEX listings_status_idx ON listings (status);
CREATE INDEX listings_expires_at_idx ON listings (expires_at);
CREATE INDEX listings_seller_club_id_idx ON listings (seller_club_id);

CREATE TABLE bids (
  id               UUID PRIMARY KEY,
  listing_id       UUID NOT NULL,
  bidder_club_id   UUID NOT NULL,
  amount           BIGINT NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX bids_listing_id_idx ON bids (listing_id);
CREATE INDEX bids_bidder_club_id_idx ON bids (bidder_club_id);
