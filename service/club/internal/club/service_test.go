package club

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// fakeRepo simula il repository per testare la logica di dominio.
type fakeRepo struct {
	club     Club
	cards    []UserCard
	clubErr  error
	cardsErr error
}

func (f *fakeRepo) GetClubByUserID(_ context.Context, _ uuid.UUID) (Club, error) {
	if f.clubErr != nil {
		return Club{}, f.clubErr
	}
	return f.club, nil
}

func (f *fakeRepo) ListUserCardsByClubID(_ context.Context, _ uuid.UUID) ([]UserCard, error) {
	if f.cardsErr != nil {
		return nil, f.cardsErr
	}
	return f.cards, nil
}

// Caso: club esistente con carte.
func TestServiceGetMyClubOK(t *testing.T) {
	clubID := uuid.New()
	repo := &fakeRepo{
		club: Club{
			ID:      clubID,
			Credits: 1200,
		},
		cards: []UserCard{
			{ID: uuid.New(), PlayerID: uuid.New(), Locked: false},
			{ID: uuid.New(), PlayerID: uuid.New(), Locked: true},
		},
	}
	service := NewService(repo)

	result, err := service.GetMyClub(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Credits != 1200 {
		t.Fatalf("expected credits 1200, got %d", result.Credits)
	}
	if len(result.Cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(result.Cards))
	}
}

// Caso: club esistente senza carte.
func TestServiceGetMyClubNoCards(t *testing.T) {
	clubID := uuid.New()
	repo := &fakeRepo{
		club: Club{
			ID:      clubID,
			Credits: 500,
		},
		cards: nil,
	}
	service := NewService(repo)

	result, err := service.GetMyClub(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Credits != 500 {
		t.Fatalf("expected credits 500, got %d", result.Credits)
	}
	if len(result.Cards) != 0 {
		t.Fatalf("expected 0 cards, got %d", len(result.Cards))
	}
}

// Caso: club inesistente.
func TestServiceGetMyClubNotFound(t *testing.T) {
	repo := &fakeRepo{clubErr: sql.ErrNoRows}
	service := NewService(repo)

	_, err := service.GetMyClub(context.Background(), uuid.New())
	if !errors.Is(err, ErrClubNotFound) {
		t.Fatalf("expected ErrClubNotFound, got %v", err)
	}
}

// Caso: errore generico dal repository.
func TestServiceGetMyClubDBError(t *testing.T) {
	repo := &fakeRepo{clubErr: errors.New("db down")}
	service := NewService(repo)

	_, err := service.GetMyClub(context.Background(), uuid.New())
	if err == nil || errors.Is(err, ErrClubNotFound) {
		t.Fatalf("expected generic error, got %v", err)
	}
}
