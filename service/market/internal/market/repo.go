package market

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/google/uuid"
)

// ErrNotFound indica che la risorsa non esiste.
var ErrNotFound = sql.ErrNoRows

// Repo gestisce le query SQL per il market.
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

// GetListing carica il listing completo dal DB.
func (r *Repo) GetListing(ctx context.Context, listingID string) (Listing, error) {
	const query = `
SELECT
  id,
  seller_club_id,
  user_card_id,
  start_price,
  buy_now_price,
  best_bid,
  best_bidder_club_id,
  status,
  EXTRACT(EPOCH FROM expires_at)::bigint
FROM listings
WHERE id = $1`

	var listing Listing
	var buyNow sql.NullInt64
	var bestBid sql.NullInt64
	var bestBidder sql.NullString

	err := r.db.QueryRowContext(ctx, query, listingID).Scan(
		&listing.ID,
		&listing.SellerClubID,
		&listing.UserCardID,
		&listing.StartPrice,
		&buyNow,
		&bestBid,
		&bestBidder,
		&listing.Status,
		&listing.ExpiresAtUnix,
	)
	if err == sql.ErrNoRows {
		return Listing{}, ErrNotFound
	}
	if err != nil {
		slog.Error("errore lettura listing", "error", err, "listing_id", listingID)
		return Listing{}, err
	}

	listing.BuyNowPrice = nullInt64Ptr(buyNow)
	listing.BestBid = nullInt64Ptr(bestBid)
	listing.BestBidderClubID = nullStringPtr(bestBidder)

	return listing, nil
}

// InsertBidAndUpdateListing inserisce il bid e aggiorna il best_bid in transazione.
func (r *Repo) InsertBidAndUpdateListing(ctx context.Context, listingID, bidderClubID, holdID string, amount int64) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	bidID := uuid.NewString()
	const insertBid = `
INSERT INTO bids (
  id,
  listing_id,
  bidder_club_id,
  amount,
  hold_id,
  created_at
) VALUES ($1,$2,$3,$4,$5,now())`

	if _, err := tx.ExecContext(ctx, insertBid, bidID, listingID, bidderClubID, amount, holdID); err != nil {
		return "", err
	}

	const updateListing = `
UPDATE listings
SET best_bid = $1,
    best_bidder_club_id = $2
WHERE id = $3`
	if _, err := tx.ExecContext(ctx, updateListing, amount, bidderClubID, listingID); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return bidID, nil
}

// GetHoldIDForBid ritorna l'hold_id del bid specifico (se presente).
func (r *Repo) GetHoldIDForBid(ctx context.Context, listingID, bidderClubID string, amount int64) (string, error) {
	const query = `
SELECT hold_id
FROM bids
WHERE listing_id = $1 AND bidder_club_id = $2 AND amount = $3
ORDER BY created_at DESC
LIMIT 1`

	var holdID sql.NullString
	err := r.db.QueryRowContext(ctx, query, listingID, bidderClubID, amount).Scan(&holdID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !holdID.Valid {
		return "", nil
	}
	return holdID.String, nil
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

func nullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}
