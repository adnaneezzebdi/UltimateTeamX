package club

import (
	"context"
	"errors"
	"testing"

	"UltimateTeamX/pkg/grpcx"
	clubv1 "UltimateTeamX/proto/club/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeMyClubReader simula il layer dominio per testare il handler gRPC.
type fakeMyClubReader struct {
	result *MyClub
	err    error
}

func (f *fakeMyClubReader) GetMyClub(_ context.Context, _ uuid.UUID) (*MyClub, error) {
	return f.result, f.err
}

// Verifica mapping OK e conversione a risposta gRPC.
func TestGetMyClubOK(t *testing.T) {
	reader := &fakeMyClubReader{
		result: &MyClub{
			ClubID:  uuid.New(),
			Credits: 123,
			Cards: []UserCard{
				{ID: uuid.New(), PlayerID: uuid.New(), Locked: false},
				{ID: uuid.New(), PlayerID: uuid.New(), Locked: true},
			},
		},
	}
	server := NewGRPCServer(reader)

	ctx := context.WithValue(context.Background(), grpcx.ContextUserIDKey, uuid.NewString())
	resp, err := server.GetMyClub(ctx, &clubv1.GetMyClubRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ClubId == "" {
		t.Fatalf("expected club_id to be set")
	}
	if resp.Credits != 123 {
		t.Fatalf("expected credits 123, got %d", resp.Credits)
	}
	if len(resp.Cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(resp.Cards))
	}
}

// Verifica errore quando manca user_id.
func TestGetMyClubUnauthenticated(t *testing.T) {
	server := NewGRPCServer(&fakeMyClubReader{})

	_, err := server.GetMyClub(context.Background(), &clubv1.GetMyClubRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

// Verifica NotFound quando il dominio ritorna ErrClubNotFound.
func TestGetMyClubNotFound(t *testing.T) {
	server := NewGRPCServer(&fakeMyClubReader{err: ErrClubNotFound})

	ctx := context.WithValue(context.Background(), grpcx.ContextUserIDKey, uuid.NewString())
	_, err := server.GetMyClub(ctx, &clubv1.GetMyClubRequest{})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

// Verifica Internal su errori generici.
func TestGetMyClubInternal(t *testing.T) {
	server := NewGRPCServer(&fakeMyClubReader{err: errors.New("db down")})

	ctx := context.WithValue(context.Background(), grpcx.ContextUserIDKey, uuid.NewString())
	_, err := server.GetMyClub(ctx, &clubv1.GetMyClubRequest{})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
}
