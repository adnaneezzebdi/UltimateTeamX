package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"github.com/google/uuid"
	clubv1 "UltimateTeamX/proto/club/v1"
	"google.golang.org/grpc"
)

type mockClubServer struct {
	clubv1.UnimplementedClubServiceServer
	logger *slog.Logger
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

func main() {
	// Mock gRPC server per test/dev: sostituire con il vero club-svc in ambienti reali.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	addr := os.Getenv("GRPC_ADDR")
	if addr == "" {
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
