package market

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"UltimateTeamX/pkg/grpcx"
	clubv1 "UltimateTeamX/proto/club/v1"
	marketv1 "UltimateTeamX/proto/market/v1"
	"UltimateTeamX/service/market/internal/lock"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Stato usato per le listing attive nel market DB.
const listingStatusActive = "ACTIVE"

// Server implementa l'interfaccia gRPC MarketService.
// Integra il club-svc per risolvere club_id e gestire lock/hold.
type Server struct {
	marketv1.UnimplementedMarketServiceServer
	logger *slog.Logger
	repo   ListingRepo
	club   clubv1.ClubServiceClient
	locker lock.Manager
}

// ListingRepo is the minimal persistence interface used by the server.
type ListingRepo interface {
	ActiveListingByCard(ctx context.Context, userCardID string) (string, error)
	CreateListing(ctx context.Context, listing Listing) error
	GetListing(ctx context.Context, listingID string) (Listing, error)
	InsertBidAndUpdateListing(ctx context.Context, listingID, bidderClubID, holdID string, amount int64) (string, error)
	GetHoldIDForBid(ctx context.Context, listingID, bidderClubID string, amount int64) (string, error)
}

// NewServer collega logger, repo e client del club-svc.
func NewServer(logger *slog.Logger, repo ListingRepo, club clubv1.ClubServiceClient, locker lock.Manager) *Server {
	return &Server{logger: logger, repo: repo, club: club, locker: locker}
}

// CreateListing valida la richiesta, blocca la carta in club-svc e inserisce il listing.
func (s *Server) CreateListing(ctx context.Context, req *marketv1.CreateListingRequest) (*marketv1.CreateListingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if err := validateCreateListing(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// 1) Evita piu' listing attivi per la stessa carta.
	existingID, err := s.repo.ActiveListingByCard(ctx, req.UserCardId)
	if err != nil {
		s.logger.Error("errore verifica listing attivo", "error", err)
		return nil, status.Error(codes.Internal, "failed to check existing listing")
	}
	if existingID != "" {
		return nil, status.Error(codes.AlreadyExists, "active listing already exists for card")
	}

	// 2) Risolve seller_club_id via club-svc.
	sellerClubID, err := s.clubIDForUser(ctx, req.SellerUserId)
	if err != nil {
		return nil, err
	}

	// Il lock in club-svc vale come verifica di ownership/disponibilità.
	// 3) Lock carta in club-svc (ownership/disponibilita').
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

	// 4) Inserisce il listing nel DB market.
	listingID := uuid.NewString()
	expiresAt := time.Unix(req.ExpiresAtUnix, 0)
	listing := Listing{
		ID:           listingID,
		SellerClubID: sellerClubID,
		UserCardID:   req.UserCardId,
		StartPrice:   req.StartPrice,
		BuyNowPrice:  optionalPrice(req.BuyNowPrice),
		Status:       listingStatusActive,
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

func (s *Server) PlaceBid(ctx context.Context, req *marketv1.PlaceBidRequest) (*marketv1.PlaceBidResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if strings.TrimSpace(req.ListingId) == "" {
		return nil, status.Error(codes.InvalidArgument, "listing_id is required")
	}
	if strings.TrimSpace(req.BidderUserId) == "" {
		return nil, status.Error(codes.InvalidArgument, "bidder_user_id is required")
	}
	if !isUUID(req.ListingId) {
		return nil, status.Error(codes.InvalidArgument, "listing_id must be a valid UUID")
	}
	if !isUUID(req.BidderUserId) {
		return nil, status.Error(codes.InvalidArgument, "bidder_user_id must be a valid UUID")
	}
	if req.BidAmount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "bid_amount must be positive")
	}
	if s.locker == nil {
		return nil, status.Error(codes.Internal, "redis lock not configured")
	}

	// 1) Risolve bidder_club_id via club-svc.
	bidderClubID, err := s.clubIDForUser(ctx, req.BidderUserId)
	if err != nil {
		return nil, err
	}

	// 2) Acquisisce lock Redis per serializzare i bid.
	lockKey := "lock:listing:" + req.ListingId
	token, ok, err := s.locker.Acquire(ctx, lockKey)
	if err != nil {
		s.logger.Error("errore acquisizione lock redis", "error", err, "listing_id", req.ListingId)
		return nil, status.Error(codes.Internal, "failed to acquire listing lock")
	}
	if !ok {
		return nil, status.Error(codes.FailedPrecondition, "listing is locked")
	}
	defer func() {
		if err := s.locker.Release(context.Background(), lockKey, token); err != nil {
			s.logger.Warn("errore rilascio lock redis", "error", err, "listing_id", req.ListingId)
		}
	}()

	// 3) Carica listing e valida lo stato.
	listing, err := s.repo.GetListing(ctx, req.ListingId)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Error(codes.NotFound, "listing not found")
		}
		s.logger.Error("errore lettura listing", "error", err, "listing_id", req.ListingId)
		return nil, status.Error(codes.Internal, "failed to load listing")
	}
	if listing.Status != listingStatusActive {
		return nil, status.Error(codes.FailedPrecondition, "listing not active")
	}
	if listing.ExpiresAtUnix <= time.Now().Unix() {
		return nil, status.Error(codes.FailedPrecondition, "listing expired")
	}

	if listing.BestBid != nil {
		if req.BidAmount <= *listing.BestBid {
			return nil, status.Error(codes.FailedPrecondition, "bid must be higher than best_bid")
		}
	} else if req.BidAmount < listing.StartPrice {
		return nil, status.Error(codes.FailedPrecondition, "bid must be >= start_price")
	}

	// 4) Crea hold crediti nel club-svc.
	holdResp, err := s.club.CreateCreditHold(ctx, &clubv1.CreateCreditHoldRequest{
		UserId: req.BidderUserId,
		Amount: req.BidAmount,
		Reason: "market_bid",
	})
	if err != nil {
		if grpcStatus, ok := status.FromError(err); ok {
			s.logger.Warn("hold crediti rifiutato da club-svc", "code", grpcStatus.Code(), "error", grpcStatus.Message())
			return nil, grpcStatus.Err()
		}
		s.logger.Error("errore creazione hold crediti", "error", err)
		return nil, status.Error(codes.Internal, "failed to create credit hold")
	}

	// 5) Inserisce bid e aggiorna best_bid in DB.
	bidID, err := s.repo.InsertBidAndUpdateListing(ctx, listing.ID, bidderClubID, holdResp.HoldId, req.BidAmount)
	if err != nil {
		s.logger.Error("errore inserimento bid", "error", err, "listing_id", req.ListingId)
		_, _ = s.club.ReleaseCreditHold(ctx, &clubv1.ReleaseCreditHoldRequest{HoldId: holdResp.HoldId})
		return nil, status.Error(codes.Internal, "failed to place bid")
	}

	// 6) Rilascia l'hold precedente (se presente).
	if listing.BestBid != nil && listing.BestBidderClubID != nil {
		holdID, err := s.repo.GetHoldIDForBid(ctx, listing.ID, *listing.BestBidderClubID, *listing.BestBid)
		if err != nil {
			s.logger.Warn("errore lettura hold precedente", "error", err, "listing_id", listing.ID)
		} else if holdID != "" {
			if _, err := s.club.ReleaseCreditHold(ctx, &clubv1.ReleaseCreditHoldRequest{HoldId: holdID}); err != nil {
				s.logger.Warn("errore rilascio hold precedente", "error", err, "hold_id", holdID)
			}
		}
	}

	s.logger.Info("bid inserito", "listing_id", listing.ID, "bid_id", bidID, "amount", req.BidAmount)
	return &marketv1.PlaceBidResponse{
		BestBid:          req.BidAmount,
		BestBidderUserId: req.BidderUserId,
	}, nil
}

// clubIDForUser chiama GetMyClub e legge club_id passando user_id via metadata gRPC.
func (s *Server) clubIDForUser(ctx context.Context, userID string) (string, error) {
	if s.club == nil {
		return "", status.Error(codes.Internal, "club client not configured")
	}
	ctxWithUser := metadata.AppendToOutgoingContext(ctx, grpcx.UserIDMetadataKey, userID)
	resp, err := s.club.GetMyClub(ctxWithUser, &clubv1.GetMyClubRequest{})
	if err != nil {
		if grpcStatus, ok := status.FromError(err); ok {
			switch grpcStatus.Code() {
			case codes.Unauthenticated, codes.NotFound:
				return "", grpcStatus.Err()
			default:
				return "", status.Error(codes.Internal, "failed to resolve club")
			}
		}
		return "", status.Error(codes.Internal, "failed to resolve club")
	}
	if strings.TrimSpace(resp.ClubId) == "" {
		return "", status.Error(codes.Internal, "club_id missing")
	}
	return resp.ClubId, nil
}

// validateCreateListing applica le invarianti di base della request.
func validateCreateListing(req *marketv1.CreateListingRequest) error {
	if strings.TrimSpace(req.SellerUserId) == "" {
		return errors.New("seller_user_id is required")
	}
	if strings.TrimSpace(req.UserCardId) == "" {
		return errors.New("user_card_id is required")
	}
	if !isUUID(req.SellerUserId) {
		return errors.New("seller_user_id must be a valid UUID")
	}
	if !isUUID(req.UserCardId) {
		return errors.New("user_card_id must be a valid UUID")
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

func isUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

