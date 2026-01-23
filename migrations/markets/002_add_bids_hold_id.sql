-- Aggiunge hold_id ai bid per poter rilasciare l'hold precedente.
-- Necessario per gestire il rilascio dell'hold del bidder precedente.

ALTER TABLE bids
ADD COLUMN hold_id TEXT;

CREATE INDEX bids_listing_bidder_amount_idx
ON bids (listing_id, bidder_club_id, amount);
