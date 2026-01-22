package club

import (
	"context"

	"github.com/google/uuid"
)

// Contratti e modelli del dominio "club".
// Espongono cosa serve al resto dell'app senza dettagli di DB/gRPC.
type MyClubReader interface {
	GetMyClub(ctx context.Context, userID uuid.UUID) (*MyClub, error)
}

// MyClub rappresenta il club con i dati necessari al dominio.
type MyClub struct {
	ClubID  uuid.UUID
	Credits int64
	Cards   []UserCard
}

// UserCard rappresenta una carta posseduta dal club.
type UserCard struct {
	ID       uuid.UUID
	PlayerID uuid.UUID
	Locked   bool
}
