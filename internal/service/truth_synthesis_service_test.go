package service

import (
	"context"
	"testing"

	"github.com/eric/callit-backend/internal/adapter/panta"
)

func TestTruthSynthesisService_Synthesize(t *testing.T) {
	mockPanta := panta.NewMockClient()
	svc := NewTruthSynthesisService(mockPanta)

	ctx := context.Background()
	// Use mock market fixture ID
	consensus, err := svc.SynthesizeTruthConsensus(ctx, "7y45RD53QHg7tfcbeZQt76NRPkhXK6CVN3e3K9d9oCS6")
	if err != nil {
		t.Fatalf("SynthesizeTruthConsensus failed: %v", err)
	}

	if consensus.MarketID == "" {
		t.Errorf("expected marketID, got empty")
	}
	if consensus.ConsensusVerdict == "" {
		t.Errorf("expected consensus verdict")
	}
	if consensus.EncyclopediaEntry == "" {
		t.Errorf("expected encyclopedia markdown article")
	}
	if consensus.ProbabilityYes <= 0 || consensus.ProbabilityYes > 1.0 {
		t.Errorf("expected valid probability [0, 1], got %f", consensus.ProbabilityYes)
	}
}
