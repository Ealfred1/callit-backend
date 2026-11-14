package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/google/uuid"
)

type fakeRepo struct {
	status, payout string
	wallet         string
	est            float64
	sig            string
	claims         int
}

func (f *fakeRepo) Create(context.Context, *domain.FundingTransaction) error { return nil }
func (f *fakeRepo) GetByReference(context.Context, string) (*domain.FundingTransaction, error) {
	return nil, nil
}
func (f *fakeRepo) GetByCheckoutID(context.Context, string) (*domain.FundingTransaction, error) {
	return nil, nil
}
func (f *fakeRepo) ListByUserID(context.Context, uuid.UUID, int) ([]*domain.FundingTransaction, error) {
	return nil, nil
}
func (f *fakeRepo) ListPending(context.Context, time.Time, int) ([]*domain.FundingTransaction, error) {
	return nil, nil
}
func (f *fakeRepo) UpdateStatus(context.Context, string, domain.FundingStatus, *time.Time) error {
	return nil
}
func (f *fakeRepo) ClaimPayout(_ context.Context, ref string) (*ports.PayoutJob, error) {
	if f.status == "completed" && (f.payout == "none" || f.payout == "retry" || f.payout == "waiting_treasury") {
		f.payout = "sending"
		f.claims++
		return &ports.PayoutJob{Reference: ref, Wallet: f.wallet, EstimatedUSDC: f.est}, nil
	}
	return nil, nil
}
func (f *fakeRepo) SetPayoutSending(_ context.Context, _ string, sig string, _ float64) error {
	f.sig = sig
	return nil
}
func (f *fakeRepo) SetPayoutResult(_ context.Context, _ string, st, _ string) error {
	f.payout = st
	return nil
}
func (f *fakeRepo) ListPayable(context.Context, int) ([]string, error) { return nil, nil }
func (f *fakeRepo) ListStuckSending(context.Context, time.Duration) ([]ports.PayoutJob, error) {
	return nil, nil
}
func (f *fakeRepo) GetPayoutInfo(context.Context, string) (*domain.PayoutInfo, error) {
	return nil, nil
}

type fakePayer struct {
	sol, usdc float64
	sent      int
	state     string
}

func (p *fakePayer) Address() string                                    { return "TREASURY" }
func (p *fakePayer) Balances(context.Context) (float64, float64, error) { return p.sol, p.usdc, nil }
func (p *fakePayer) Check(context.Context, string) (string, error)      { return "confirmed", nil }
func (p *fakePayer) Pay(_ context.Context, _ string, _ float64, on func(string) error) (string, string, error) {
	_ = on("SIG123")
	p.sent++
	return "SIG123", p.state, nil
}

func newSvc(r *fakeRepo, p *fakePayer) *FundingService {
	s := &FundingService{fundingRepo: r, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	s.SetPayer(p, 100)
	return s
}

func TestPayoutPaysOnceAndRecordsSignature(t *testing.T) {
	r := &fakeRepo{status: "completed", payout: "none", wallet: "W", est: 3.23}
	p := &fakePayer{sol: 0.1, usdc: 50, state: "confirmed"}
	s := newSvc(r, p)
	s.TryPayout(context.Background(), "REF")
	s.TryPayout(context.Background(), "REF") // second call must be a no-op
	if p.sent != 1 || r.payout != "paid" || r.sig != "SIG123" {
		t.Fatalf("sent=%d payout=%s sig=%s", p.sent, r.payout, r.sig)
	}
}

func TestPayoutWaitsWhenTreasuryEmpty(t *testing.T) {
	r := &fakeRepo{status: "completed", payout: "none", wallet: "W", est: 3.23}
	p := &fakePayer{sol: 0.1, usdc: 1, state: "confirmed"}
	newSvc(r, p).TryPayout(context.Background(), "REF")
	if p.sent != 0 || r.payout != "waiting_treasury" {
		t.Fatalf("sent=%d payout=%s", p.sent, r.payout)
	}
}

func TestPayoutHoldsOverCap(t *testing.T) {
	r := &fakeRepo{status: "completed", payout: "none", wallet: "W", est: 500}
	p := &fakePayer{sol: 1, usdc: 1000, state: "confirmed"}
	newSvc(r, p).TryPayout(context.Background(), "REF")
	if p.sent != 0 || r.payout != "failed" {
		t.Fatalf("sent=%d payout=%s", p.sent, r.payout)
	}
}

func TestPayoutIgnoresUnpaidCheckout(t *testing.T) {
	r := &fakeRepo{status: "pending", payout: "none", wallet: "W", est: 3}
	p := &fakePayer{sol: 1, usdc: 100, state: "confirmed"}
	newSvc(r, p).TryPayout(context.Background(), "REF")
	if p.sent != 0 {
		t.Fatal("paid an unconfirmed checkout")
	}
}
