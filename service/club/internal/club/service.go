package club

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

// Service applica la logica di dominio usando il repository.
// Qui si mappano errori del DB in errori di dominio.
type Service struct {
	repo ClubRepository
}

// NewService crea il servizio di dominio per il club.
func NewService(repo ClubRepository) *Service {
	return &Service{repo: repo}
}

// GetMyClub carica il club dell'utente con le carte associate.
func (s *Service) GetMyClub(ctx context.Context, userID uuid.UUID) (*MyClub, error) {
	club, err := s.repo.GetClubByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrClubNotFound) {
			return nil, ErrClubNotFound
		}
		return nil, err
	}

	cards, err := s.repo.ListUserCardsByClubID(ctx, club.ID)
	if err != nil {
		return nil, err
	}

	return &MyClub{
		ClubID:  club.ID,
		Credits: club.Credits,
		Cards:   cards,
	}, nil
}
