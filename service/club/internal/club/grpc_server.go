package club

import (
	"context"
	"errors"
	"strings"

	clubv1 "UltimateTeamX/proto/club/v1"
	"UltimateTeamX/pkg/grpcx"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCServer espone il handler gRPC per il club-svc.
// Qui si leggono le metadata gRPC e si mappano gli errori in codici gRPC.
type GRPCServer struct {
	clubv1.UnimplementedClubServiceServer
	reader MyClubReader
}

// NewGRPCServer crea il server gRPC con il dominio.
func NewGRPCServer(reader MyClubReader) *GRPCServer {
	return &GRPCServer{reader: reader}
}

// GetMyClub ritorna il club associato all'user_id dal context/metadata gRPC.
func (s *GRPCServer) GetMyClub(ctx context.Context, req *clubv1.GetMyClubRequest) (*clubv1.GetMyClubResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	result, err := s.reader.GetMyClub(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnauthenticated):
			return nil, status.Error(codes.Unauthenticated, "unauthenticated")
		case errors.Is(err, ErrClubNotFound):
			return nil, status.Error(codes.NotFound, "club not found")
		default:
			return nil, status.Error(codes.Internal, "failed to load club")
		}
	}

	cards := make([]*clubv1.Card, 0, len(result.Cards))
	for _, card := range result.Cards {
		cards = append(cards, &clubv1.Card{
			Id:       card.ID.String(),
			PlayerId: card.PlayerID.String(),
			Locked:   card.Locked,
		})
	}

	return &clubv1.GetMyClubResponse{
		ClubId:  result.ClubID.String(),
		Credits: result.Credits,
		Cards:   cards,
	}, nil
}

// userIDFromContext prova prima dalle metadata gRPC, poi dal context locale.
func userIDFromContext(ctx context.Context) (uuid.UUID, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(grpcx.UserIDMetadataKey); len(values) > 0 {
			userID := strings.TrimSpace(values[0])
			parsed, err := uuid.Parse(userID)
			if err == nil {
				return parsed, nil
			}
		}
	}

	value := ctx.Value(grpcx.ContextUserIDKey)
	userID, ok := value.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return uuid.Nil, ErrUnauthenticated
	}
	parsed, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, ErrUnauthenticated
	}
	return parsed, nil
}
