package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	clubv1 "UltimateTeamX/proto/club/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type mockClubServer struct {
	clubv1.UnimplementedClubServiceServer
	logger *slog.Logger
}

// Mock minimale per simulare il club-svc in locale.
func (s *mockClubServer) GetMyClub(_ context.Context, _ *clubv1.GetMyClubRequest) (*clubv1.GetMyClubResponse, error) {
	s.logger.Info("mock get my club")
	return &clubv1.GetMyClubResponse{ClubId: "00000000-0000-0000-0000-000000000001", Credits: 0, Cards: nil}, nil
}

// LockCard simula un lock carta e genera un lock_id fittizio.
func (s *mockClubServer) LockCard(_ context.Context, req *clubv1.LockCardRequest) (*clubv1.LockCardResponse, error) {
	lockID := uuid.NewString()
	s.logger.Info("mock lock card", "user_id", req.UserId, "user_card_id", req.UserCardId, "lock_id", lockID)
	return &clubv1.LockCardResponse{LockId: lockID}, nil
}

// ReleaseCardLock simula lo sblocco carta e conferma l'operazione.
func (s *mockClubServer) ReleaseCardLock(_ context.Context, req *clubv1.ReleaseCardLockRequest) (*clubv1.ReleaseCardLockResponse, error) {
	s.logger.Info("mock release card lock", "lock_id", req.LockId)
	return &clubv1.ReleaseCardLockResponse{Released: true}, nil
}

// CreateCreditHold simula un hold crediti e ritorna un hold_id fittizio.
func (s *mockClubServer) CreateCreditHold(_ context.Context, req *clubv1.CreateCreditHoldRequest) (*clubv1.CreateCreditHoldResponse, error) {
	holdID := uuid.NewString()
	s.logger.Info("mock credit hold", "user_id", req.UserId, "amount", req.Amount, "hold_id", holdID)
	return &clubv1.CreateCreditHoldResponse{HoldId: holdID}, nil
}

// ReleaseCreditHold simula il rilascio di un hold crediti.
func (s *mockClubServer) ReleaseCreditHold(_ context.Context, req *clubv1.ReleaseCreditHoldRequest) (*clubv1.ReleaseCreditHoldResponse, error) {
	s.logger.Info("mock release credit hold", "hold_id", req.HoldId)
	return &clubv1.ReleaseCreditHoldResponse{Released: true}, nil
}

func main() {
	// Avvio server gRPC mock su GRPC_ADDR (default :50052).
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	addr := os.Getenv("GRPC_ADDR")
	if addr == "" {
		// Default porta mock compatibile con club-svc.
		addr = ":50052"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("grpc listen failed", "error", err)
		os.Exit(1)
	}

	server := grpc.NewServer()
	clubv1.RegisterClubServiceServer(server, &mockClubServer{logger: logger})

	logger.Info("mock club grpc listening", "addr", addr)
	if err := server.Serve(lis); err != nil {
		logger.Error("grpc serve failed", "error", err)
		os.Exit(1)
	}
}
