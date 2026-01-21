package market

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	clubv1 "UltimateTeamX/proto/club/v1"
	marketv1 "UltimateTeamX/proto/market/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeRepo struct {
	activeListingID string
	activeErr       error
	createErr       error
	createdListing  Listing
	createCalls     int
}

func (r *fakeRepo) ActiveListingByCard(_ context.Context, _ string) (string, error) {
	return r.activeListingID, r.activeErr
}

func (r *fakeRepo) CreateListing(_ context.Context, listing Listing) error {
	r.createCalls++
	r.createdListing = listing
	return r.createErr
}

type fakeClub struct {
	lockResp          *clubv1.LockCardResponse
	lockErr           error
	releaseCalls      int
	releaseLastLockID string
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

func (c *fakeClub) GetClub(_ context.Context, _ *clubv1.GetClubRequest, _ ...grpc.CallOption) (*clubv1.GetClubResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *fakeClub) CreateCreditHold(_ context.Context, _ *clubv1.CreateCreditHoldRequest, _ ...grpc.CallOption) (*clubv1.CreateCreditHoldResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *fakeClub) ReleaseCreditHold(_ context.Context, _ *clubv1.ReleaseCreditHoldRequest, _ ...grpc.CallOption) (*clubv1.ReleaseCreditHoldResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *fakeClub) SettleTrade(_ context.Context, _ *clubv1.SettleTradeRequest, _ ...grpc.CallOption) (*clubv1.SettleTradeResponse, error) {
	return nil, errors.New("not implemented")
}

func TestCreateListingSuccess(t *testing.T) {
	repo := &fakeRepo{}
	club := &fakeClub{}
	server := NewServer(slog.Default(), repo, club)

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
	if repo.createdListing.UserCardID != req.UserCardId {
		t.Fatalf("unexpected user_card_id")
	}
	if repo.createdListing.StartPrice != req.StartPrice {
		t.Fatalf("unexpected start_price")
	}
	if repo.createdListing.BuyNowPrice == nil || *repo.createdListing.BuyNowPrice != req.BuyNowPrice {
		t.Fatalf("unexpected buy_now_price")
	}
}

func TestCreateListingAlreadyExists(t *testing.T) {
	repo := &fakeRepo{activeListingID: "listing-1"}
	club := &fakeClub{}
	server := NewServer(slog.Default(), repo, club)

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
	server := NewServer(slog.Default(), repo, club)

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
	server := NewServer(slog.Default(), repo, club)

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
	server := NewServer(slog.Default(), repo, club)

	_, err := server.CreateListing(context.Background(), &marketv1.CreateListingRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}
