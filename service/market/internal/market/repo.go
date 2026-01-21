package market

import (
	"context"
	"database/sql"
	"log/slog"
)

type Repo struct {
	db *sql.DB
}

// Listing mappa la tabella listings per insert/letture.
type Listing struct {
	ID               string
	SellerClubID     string
	UserCardID       string
	StartPrice       int64
	BuyNowPrice      *int64
	Status           string
	ExpiresAtUnix    int64
	BestBid          *int64
	BestBidderClubID *string
}

// NewRepo collega il repository a una connessione SQL.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// ActiveListingByCard ritorna l'ID del listing attivo per la carta, o vuoto se non c'è.
func (r *Repo) ActiveListingByCard(ctx context.Context, userCardID string) (string, error) {
	const query = `
SELECT id
FROM listings
WHERE user_card_id = $1 AND status = 'ACTIVE'
LIMIT 1`

	var id string
	err := r.db.QueryRowContext(ctx, query, userCardID).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		slog.Error("errore query listing attivo", "error", err, "user_card_id", userCardID)
		return "", err
	}
	return id, nil
}

// CreateListing inserisce un nuovo listing in stato ACTIVE.
func (r *Repo) CreateListing(ctx context.Context, listing Listing) error {
	const query = `
INSERT INTO listings (
  id,
  seller_club_id,
  user_card_id,
  start_price,
  buy_now_price,
  best_bid,
  best_bidder_club_id,
  status,
  expires_at,
  created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,to_timestamp($9),now())`

	_, err := r.db.ExecContext(
		ctx,
		query,
		listing.ID,
		listing.SellerClubID,
		listing.UserCardID,
		listing.StartPrice,
		nullInt64(listing.BuyNowPrice),
		nullInt64(listing.BestBid),
		nullString(listing.BestBidderClubID),
		listing.Status,
		listing.ExpiresAtUnix,
	)
	if err != nil {
		slog.Error("errore insert listing", "error", err, "listing_id", listing.ID)
	}
	return err
}

// nullInt64 prepara valori numerici opzionali per SQL.
func nullInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

// nullString prepara stringhe opzionali per SQL.
func nullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}
