package market

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"UltimateTeamX/pkg/grpcx"
	clubv1 "UltimateTeamX/proto/club/v1"
	marketv1 "UltimateTeamX/proto/market/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Test suite per i flussi CreateListing e PlaceBid.

type fakeRepo struct {
	activeListingID string
	activeErr       error
	createErr       error
	createdListing  Listing
	createCalls     int
	listing         Listing
	getListingErr   error
	insertErr       error
	insertBidID     string
	lastInsert      struct {
		listingID    string
		bidderClubID string
		holdID       string
		amount       int64
	}
	holdIDForBid string
	holdIDErr    error
}

func (r *fakeRepo) ActiveListingByCard(_ context.Context, _ string) (string, error) {
	return r.activeListingID, r.activeErr
}

func (r *fakeRepo) CreateListing(_ context.Context, listing Listing) error {
	r.createCalls++
	r.createdListing = listing
	return r.createErr
}

func (r *fakeRepo) GetListing(_ context.Context, _ string) (Listing, error) {
	if r.getListingErr != nil {
		return Listing{}, r.getListingErr
	}
	return r.listing, nil
}

func (r *fakeRepo) InsertBidAndUpdateListing(_ context.Context, listingID, bidderClubID, holdID string, amount int64) (string, error) {
	r.lastInsert.listingID = listingID
	r.lastInsert.bidderClubID = bidderClubID
	r.lastInsert.holdID = holdID
	r.lastInsert.amount = amount
	if r.insertErr != nil {
		return "", r.insertErr
	}
	if r.insertBidID != "" {
		return r.insertBidID, nil
	}
	return "bid-1", nil
}

func (r *fakeRepo) GetHoldIDForBid(_ context.Context, _, _ string, _ int64) (string, error) {
	if r.holdIDErr != nil {
		return "", r.holdIDErr
	}
	return r.holdIDForBid, nil
}

// fakeClub simula il client gRPC di club-svc.
type fakeClub struct {
	getMyClubResp     *clubv1.GetMyClubResponse
	getMyClubErr      error
	getMyClubCalls    int
	getMyClubUserID   string
	lockResp          *clubv1.LockCardResponse
	lockErr           error
	releaseCalls      int
	releaseLastLockID string
	holdResp          *clubv1.CreateCreditHoldResponse
	holdErr           error
	releaseHoldCalls  int
	releaseHoldID     string
}

func (c *fakeClub) LockCard(_ context.Context, _ *clubv1.LockCardRequest, _ ...grpc.CallOption) (*clubv1.LockCardResponse, error) {
	if c.lockErr != nil {
		return nil, c.lockErr
	}
	if c.lockResp != nil {
		return c.lockResp, nil
	}
	return &clubv1.LockCardResponse{LockId: "lock-1"}, nil
}

func (c *fakeClub) ReleaseCardLock(_ context.Context, req *clubv1.ReleaseCardLockRequest, _ ...grpc.CallOption) (*clubv1.ReleaseCardLockResponse, error) {
	c.releaseCalls++
	c.releaseLastLockID = req.LockId
	return &clubv1.ReleaseCardLockResponse{Released: true}, nil
}

func (c *fakeClub) GetMyClub(ctx context.Context, _ *clubv1.GetMyClubRequest, _ ...grpc.CallOption) (*clubv1.GetMyClubResponse, error) {
	c.getMyClubCalls++
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(grpcx.UserIDMetadataKey); len(values) > 0 {
			c.getMyClubUserID = values[0]
		}
	}
	if c.getMyClubUserID == "" {
		if md, ok := metadata.FromOutgoingContext(ctx); ok {
			if values := md.Get(grpcx.UserIDMetadataKey); len(values) > 0 {
				c.getMyClubUserID = values[0]
			}
		}
	}
	if c.getMyClubErr != nil {
		return nil, c.getMyClubErr
	}
	if c.getMyClubResp != nil {
		return c.getMyClubResp, nil
	}
	return &clubv1.GetMyClubResponse{ClubId: "club-1"}, nil
}

func (c *fakeClub) GetClub(_ context.Context, _ *clubv1.GetClubRequest, _ ...grpc.CallOption) (*clubv1.GetClubResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *fakeClub) CreateCreditHold(_ context.Context, _ *clubv1.CreateCreditHoldRequest, _ ...grpc.CallOption) (*clubv1.CreateCreditHoldResponse, error) {
	if c.holdErr != nil {
		return nil, c.holdErr
	}
	if c.holdResp != nil {
		return c.holdResp, nil
	}
	return &clubv1.CreateCreditHoldResponse{HoldId: "hold-1"}, nil
}

func (c *fakeClub) ReleaseCreditHold(_ context.Context, req *clubv1.ReleaseCreditHoldRequest, _ ...grpc.CallOption) (*clubv1.ReleaseCreditHoldResponse, error) {
	c.releaseHoldCalls++
	c.releaseHoldID = req.HoldId
	return &clubv1.ReleaseCreditHoldResponse{Released: true}, nil
}

func (c *fakeClub) SettleTrade(_ context.Context, _ *clubv1.SettleTradeRequest, _ ...grpc.CallOption) (*clubv1.SettleTradeResponse, error) {
	return nil, errors.New("not implemented")
}

// fakeLock simula un lock Redis.
type fakeLock struct {
	token string
	ok    bool
	err   error
}

func (l *fakeLock) Acquire(_ context.Context, _ string) (string, bool, error) {
	return l.token, l.ok, l.err
}

func (l *fakeLock) Release(_ context.Context, _, _ string) error {
	return nil
}

// contendedLock simula lock conteso per test concorrenti.
type contendedLock struct {
	token string
	taken chan struct{}
}

func newContendedLock() *contendedLock {
	return &contendedLock{
		token: "token",
		taken: make(chan struct{}, 1),
	}
}

func (l *contendedLock) Acquire(_ context.Context, _ string) (string, bool, error) {
	select {
	case l.taken <- struct{}{}:
		return l.token, true, nil
	default:
		return "", false, nil
	}
}

func (l *contendedLock) Release(_ context.Context, _, _ string) error {
	select {
	case <-l.taken:
	default:
	}
	return nil
}

// oneShotLock permette un solo acquire riuscito.
type oneShotLock struct {
	used bool
}

func (l *oneShotLock) Acquire(_ context.Context, _ string) (string, bool, error) {
	if l.used {
		return "", false, nil
	}
	l.used = true
	return "token", true, nil
}

func (l *oneShotLock) Release(_ context.Context, _, _ string) error {
	return nil
}

func TestCreateListingSuccess(t *testing.T) {
	repo := &fakeRepo{}
	club := &fakeClub{getMyClubResp: &clubv1.GetMyClubResponse{ClubId: "club-seller"}}
	server := NewServer(slog.Default(), repo, club, nil)

	req := &marketv1.CreateListingRequest{
		SellerUserId:  "11111111-1111-1111-1111-111111111111",
		UserCardId:    "22222222-2222-2222-2222-222222222222",
		StartPrice:    1000,
		BuyNowPrice:   2000,
		ExpiresAtUnix: time.Now().Add(time.Hour).Unix(),
	}

	resp, err := server.CreateListing(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.ListingId == "" {
		t.Fatalf("expected listing_id to be set")
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected CreateListing to be called once, got %d", repo.createCalls)
	}
	if repo.createdListing.Status != listingStatusActive {
		t.Fatalf("expected status ACTIVE, got %s", repo.createdListing.Status)
	}
	if repo.createdListing.SellerClubID != "club-seller" {
		t.Fatalf("unexpected seller_club_id")
	}
	if repo.createdListing.UserCardID != req.UserCardId {
		t.Fatalf("unexpected user_card_id")
	}
	if repo.createdListing.StartPrice != req.StartPrice {
		t.Fatalf("unexpected start_price")
	}
	if repo.createdListing.BuyNowPrice == nil || *repo.createdListing.BuyNowPrice != req.BuyNowPrice {
		t.Fatalf("unexpected buy_now_price")
	}
	if club.getMyClubUserID != req.SellerUserId {
		t.Fatalf("expected GetMyClub to use seller user_id")
	}
}

func TestCreateListingAlreadyExists(t *testing.T) {
	repo := &fakeRepo{activeListingID: "listing-1"}
	club := &fakeClub{}
	server := NewServer(slog.Default(), repo, club, nil)

	req := &marketv1.CreateListingRequest{
		SellerUserId:  "11111111-1111-1111-1111-111111111111",
		UserCardId:    "22222222-2222-2222-2222-222222222222",
		StartPrice:    1000,
		ExpiresAtUnix: time.Now().Add(time.Hour).Unix(),
	}

	_, err := server.CreateListing(context.Background(), req)
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists, got %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatalf("did not expect CreateListing to be called")
	}
}

func TestCreateListingLockErrorPropagates(t *testing.T) {
	lockErr := status.Error(codes.FailedPrecondition, "card locked")
	repo := &fakeRepo{}
	club := &fakeClub{lockErr: lockErr}
	server := NewServer(slog.Default(), repo, club, nil)

	req := &marketv1.CreateListingRequest{
		SellerUserId:  "11111111-1111-1111-1111-111111111111",
		UserCardId:    "22222222-2222-2222-2222-222222222222",
		StartPrice:    1000,
		ExpiresAtUnix: time.Now().Add(time.Hour).Unix(),
	}

	_, err := server.CreateListing(context.Background(), req)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatalf("did not expect CreateListing to be called")
	}
}

func TestCreateListingCreateErrorReleasesLock(t *testing.T) {
	repo := &fakeRepo{createErr: errors.New("db down")}
	club := &fakeClub{lockResp: &clubv1.LockCardResponse{LockId: "lock-2"}}
	server := NewServer(slog.Default(), repo, club, nil)

	req := &marketv1.CreateListingRequest{
		SellerUserId:  "11111111-1111-1111-1111-111111111111",
		UserCardId:    "22222222-2222-2222-2222-222222222222",
		StartPrice:    1000,
		ExpiresAtUnix: time.Now().Add(time.Hour).Unix(),
	}

	_, err := server.CreateListing(context.Background(), req)
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", err)
	}
	if club.releaseCalls != 1 || club.releaseLastLockID != "lock-2" {
		t.Fatalf("expected ReleaseCardLock to be called with lock-2")
	}
}

func TestCreateListingValidation(t *testing.T) {
	repo := &fakeRepo{}
	club := &fakeClub{}
	server := NewServer(slog.Default(), repo, club, nil)

	_, err := server.CreateListing(context.Background(), &marketv1.CreateListingRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestPlaceBidSuccess(t *testing.T) {
	repo := &fakeRepo{
		listing: Listing{
			ID:            "listing-1",
			Status:        listingStatusActive,
			StartPrice:    1000,
			ExpiresAtUnix: time.Now().Add(time.Hour).Unix(),
		},
	}
	club := &fakeClub{
		getMyClubResp: &clubv1.GetMyClubResponse{ClubId: "club-bidder"},
		holdResp:      &clubv1.CreateCreditHoldResponse{HoldId: "hold-1"},
	}
	locker := &fakeLock{token: "token", ok: true}
	server := NewServer(slog.Default(), repo, club, locker)

	req := &marketv1.PlaceBidRequest{
		ListingId:    "11111111-1111-1111-1111-111111111111",
		BidderUserId: "11111111-1111-1111-1111-111111111111",
		BidAmount:    1500,
	}

	resp, err := server.PlaceBid(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.BestBid != req.BidAmount {
		t.Fatalf("unexpected best_bid")
	}
	if repo.lastInsert.bidderClubID != "club-bidder" {
		t.Fatalf("expected bidder_club_id to be used")
	}
	if repo.lastInsert.holdID != "hold-1" {
		t.Fatalf("expected hold_id to be used")
	}
}

func TestPlaceBidLockUnavailable(t *testing.T) {
	repo := &fakeRepo{}
	club := &fakeClub{}
	locker := &fakeLock{token: "", ok: false}
	server := NewServer(slog.Default(), repo, club, locker)

	req := &marketv1.PlaceBidRequest{
		ListingId:    "11111111-1111-1111-1111-111111111111",
		BidderUserId: "11111111-1111-1111-1111-111111111111",
		BidAmount:    1500,
	}

	_, err := server.PlaceBid(context.Background(), req)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
}

func TestPlaceBidReleasesPreviousHold(t *testing.T) {
	prevBid := int64(1200)
	prevBidder := "prev-bidder"
	repo := &fakeRepo{
		listing: Listing{
			ID:               "listing-1",
			Status:           listingStatusActive,
			StartPrice:       1000,
			ExpiresAtUnix:    time.Now().Add(time.Hour).Unix(),
			BestBid:          &prevBid,
			BestBidderClubID: &prevBidder,
		},
		holdIDForBid: "hold-prev",
	}
	club := &fakeClub{
		getMyClubResp: &clubv1.GetMyClubResponse{ClubId: "club-bidder"},
		holdResp:      &clubv1.CreateCreditHoldResponse{HoldId: "hold-new"},
	}
	locker := &fakeLock{token: "token", ok: true}
	server := NewServer(slog.Default(), repo, club, locker)

	req := &marketv1.PlaceBidRequest{
		ListingId:    "11111111-1111-1111-1111-111111111111",
		BidderUserId: "22222222-2222-2222-2222-222222222222",
		BidAmount:    1500,
	}

	_, err := server.PlaceBid(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if club.releaseHoldCalls != 1 || club.releaseHoldID != "hold-prev" {
		t.Fatalf("expected release of previous hold")
	}
}

func TestPlaceBidConcurrentLock(t *testing.T) {
	repo := &fakeRepo{
		listing: Listing{
			ID:            "listing-1",
			Status:        listingStatusActive,
			StartPrice:    1000,
			ExpiresAtUnix: time.Now().Add(time.Hour).Unix(),
		},
	}
	club := &fakeClub{holdResp: &clubv1.CreateCreditHoldResponse{HoldId: "hold-1"}}
	locker := &oneShotLock{}
	server := NewServer(slog.Default(), repo, club, locker)

	req := &marketv1.PlaceBidRequest{
		ListingId:    "11111111-1111-1111-1111-111111111111",
		BidderUserId: "22222222-2222-2222-2222-222222222222",
		BidAmount:    1500,
	}

	errCh := make(chan error, 2)
	go func() {
		_, err := server.PlaceBid(context.Background(), req)
		errCh <- err
	}()
	go func() {
		_, err := server.PlaceBid(context.Background(), req)
		errCh <- err
	}()

	err1 := <-errCh
	err2 := <-errCh

	okCount := 0
	failCount := 0
	for _, err := range []error{err1, err2} {
		if err == nil {
			okCount++
			continue
		}
		if status.Code(err) == codes.FailedPrecondition {
			failCount++
		}
	}

	if okCount != 1 || failCount != 1 {
		t.Fatalf("expected 1 success and 1 FailedPrecondition, got ok=%d fail=%d", okCount, failCount)
	}
}
