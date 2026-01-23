package club

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/google/uuid"
)

// Accesso dati del club su Postgres (persistence layer).
// Qui restano le query SQL e la traduzione in tipi di dominio.
type Club struct {
	ID      uuid.UUID
	Credits int64
}

// ClubRepository espone le letture necessarie al dominio.
type ClubRepository interface {
	GetClubByUserID(ctx context.Context, userID uuid.UUID) (Club, error)
	ListUserCardsByClubID(ctx context.Context, clubID uuid.UUID) ([]UserCard, error)
}

// Repo implementa l'accesso al DB per il club.
type Repo struct {
	db *sql.DB
}

// NewRepo collega il repository a una connessione SQL.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// GetClubByUserID carica club_id e credits dal user_id.
func (r *Repo) GetClubByUserID(ctx context.Context, userID uuid.UUID) (Club, error) {
	const query = `
SELECT id, credits
FROM clubs
WHERE user_id = $1`

	var club Club
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&club.ID, &club.Credits)
	if err == sql.ErrNoRows {
		return Club{}, ErrClubNotFound
	}
	if err != nil {
		slog.Error("errore lettura club", "error", err, "user_id", userID)
		return Club{}, err
	}
	return club, nil
}

// ListUserCardsByClubID ritorna tutte le carte del club.
func (r *Repo) ListUserCardsByClubID(ctx context.Context, clubID uuid.UUID) ([]UserCard, error) {
	const query = `
SELECT id, player_id, locked
FROM user_cards
WHERE club_id = $1`

	rows, err := r.db.QueryContext(ctx, query, clubID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []UserCard
	for rows.Next() {
		var card UserCard
		if err := rows.Scan(&card.ID, &card.PlayerID, &card.Locked); err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cards, nil
}
