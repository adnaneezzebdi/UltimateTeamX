package club

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Test d'integrazione: inserisce dati reali e li rilegge dal DB.
func TestRepoGetClubAndCards(t *testing.T) {
	dsn := os.Getenv("CLUB_TEST_DSN")
	if dsn == "" {
		t.Skip("CLUB_TEST_DSN not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := NewRepo(db)

	clubID := uuid.New()
	userID := uuid.New()
	cardID := uuid.New()
	playerID := uuid.New()

	if _, err := db.ExecContext(ctx, `INSERT INTO clubs (id, user_id, credits) VALUES ($1,$2,$3)`, clubID, userID, 900); err != nil {
		t.Fatalf("insert club: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM user_cards WHERE club_id = $1`, clubID)
		_, _ = db.ExecContext(ctx, `DELETE FROM clubs WHERE id = $1`, clubID)
	})

	if _, err := db.ExecContext(ctx, `INSERT INTO user_cards (id, club_id, player_id, locked) VALUES ($1,$2,$3,$4)`, cardID, clubID, playerID, false); err != nil {
		t.Fatalf("insert user_cards: %v", err)
	}

	club, err := repo.GetClubByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetClubByUserID: %v", err)
	}
	if club.ID != clubID || club.Credits != 900 {
		t.Fatalf("unexpected club data: %+v", club)
	}

	cards, err := repo.ListUserCardsByClubID(ctx, clubID)
	if err != nil {
		t.Fatalf("ListUserCardsByClubID: %v", err)
	}
	if len(cards) != 1 || cards[0].ID != cardID || cards[0].PlayerID != playerID {
		t.Fatalf("unexpected cards: %+v", cards)
	}
}

// Test d'integrazione: club inesistente.
func TestRepoGetClubNotFound(t *testing.T) {
	dsn := os.Getenv("CLUB_TEST_DSN")
	if dsn == "" {
		t.Skip("CLUB_TEST_DSN not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := NewRepo(db)
	_, err = repo.GetClubByUserID(context.Background(), uuid.New())
	if !errors.Is(err, ErrClubNotFound) {
		t.Fatalf("expected ErrClubNotFound, got %v", err)
	}
}
