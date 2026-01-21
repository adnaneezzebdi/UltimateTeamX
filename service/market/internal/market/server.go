package market

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	clubv1 "UltimateTeamX/proto/club/v1"
	marketv1 "UltimateTeamX/proto/market/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const listingStatusActive = "ACTIVE"

// Server implementa l'interfaccia gRPC MarketService.
type Server struct {
	marketv1.UnimplementedMarketServiceServer
	logger *slog.Logger
	repo   ListingRepo
	club   clubv1.ClubServiceClient
}

// ListingRepo is the minimal persistence interface used by the server.
type ListingRepo interface {
	ActiveListingByCard(ctx context.Context, userCardID string) (string, error)
	CreateListing(ctx context.Context, listing Listing) error
}

// NewServer collega logger, repo e client del club-svc.
func NewServer(logger *slog.Logger, repo ListingRepo, club clubv1.ClubServiceClient) *Server {
	return &Server{logger: logger, repo: repo, club: club}
}

// CreateListing valida la richiesta, blocca la carta in club-svc e inserisce il listing.
func (s *Server) CreateListing(ctx context.Context, req *marketv1.CreateListingRequest) (*marketv1.CreateListingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if err := validateCreateListing(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Evita più listing attivi per la stessa carta.
	existingID, err := s.repo.ActiveListingByCard(ctx, req.UserCardId)
	if err != nil {
		s.logger.Error("errore verifica listing attivo", "error", err)
		return nil, status.Error(codes.Internal, "failed to check existing listing")
	}
	if existingID != "" {
		return nil, status.Error(codes.AlreadyExists, "active listing already exists for card")
	}

	// Il lock in club-svc vale come verifica di ownership/disponibilità.
	lockResp, err := s.club.LockCard(ctx, &clubv1.LockCardRequest{
		UserId:     req.SellerUserId,
		UserCardId: req.UserCardId,
		Reason:     "market_listing",
	})
	if err != nil {
		if grpcStatus, ok := status.FromError(err); ok {
			s.logger.Warn("lock carta rifiutato da club-svc", "code", grpcStatus.Code(), "error", grpcStatus.Message())
			return nil, grpcStatus.Err()
		}
		s.logger.Error("errore lock carta in club-svc", "error", err)
		return nil, status.Error(codes.Internal, "failed to lock card")
	}

	listingID := uuid.NewString()
	expiresAt := time.Unix(req.ExpiresAtUnix, 0)
	listing := Listing{
		ID:            listingID,
	// TODO: risolvere seller_club_id via club-svc (user_id -> club_id).
	SellerClubID:  req.SellerUserId,
		UserCardID:    req.UserCardId,
		StartPrice:    req.StartPrice,
		BuyNowPrice:   optionalPrice(req.BuyNowPrice),
		Status:        listingStatusActive,
		ExpiresAtUnix: expiresAt.Unix(),
	}

	if err := s.repo.CreateListing(ctx, listing); err != nil {
		// TODO: decidere come gestire il retry/idempotenza quando il lock e' preso ma l'insert fallisce.
		// Sblocco best-effort per evitare di lasciare la carta bloccata.
		s.logger.Error("errore creazione listing nel db", "error", err, "listing_id", listingID)
		_, _ = s.club.ReleaseCardLock(ctx, &clubv1.ReleaseCardLockRequest{LockId: lockResp.LockId})
		return nil, status.Error(codes.Internal, "failed to create listing")
	}

	s.logger.Info("listing creato", "listing_id", listingID, "user_card_id", req.UserCardId)
	return &marketv1.CreateListingResponse{ListingId: listingID}, nil
}

// validateCreateListing applica le invarianti di base della request.
func validateCreateListing(req *marketv1.CreateListingRequest) error {
	if strings.TrimSpace(req.SellerUserId) == "" {
		return errors.New("seller_user_id is required")
	}
	if strings.TrimSpace(req.UserCardId) == "" {
		return errors.New("user_card_id is required")
	}
	if req.StartPrice <= 0 {
		return errors.New("start_price must be positive")
	}
	if req.BuyNowPrice < 0 {
		return errors.New("buy_now_price cannot be negative")
	}
	if req.BuyNowPrice > 0 && req.BuyNowPrice < req.StartPrice {
		return errors.New("buy_now_price must be >= start_price")
	}
	if req.ExpiresAtUnix <= time.Now().Unix() {
		return errors.New("expires_at must be in the future")
	}
	return nil
}

// optionalPrice converte un prezzo non positivo in nil per SQL NULL.
func optionalPrice(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}
