package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
)

type TruthSynthesisService struct {
	panta ports.PantaClient
}

func NewTruthSynthesisService(panta ports.PantaClient) *TruthSynthesisService {
	return &TruthSynthesisService{panta: panta}
}

func (s *TruthSynthesisService) SynthesizeTruthConsensus(ctx context.Context, marketID string) (*domain.TruthConsensus, error) {
	market, err := s.panta.GetMarket(ctx, marketID)
	if err != nil {
		return nil, fmt.Errorf("fetch market %s: %w", marketID, err)
	}

	var yesPrice, noPrice float64
	if market.YesPrice != nil {
		yesPrice, _ = strconv.ParseFloat(*market.YesPrice, 64)
	}
	if market.NoPrice != nil {
		noPrice, _ = strconv.ParseFloat(*market.NoPrice, 64)
	}
	volume, _ := strconv.ParseFloat(market.VolumeUsdc, 64)

	// In prediction markets, price represents direct probability [0, 1]
	probYes := yesPrice
	probNo := noPrice
	if probYes+probNo == 0 {
		probYes = 0.50
		probNo = 0.50
	} else if probYes+probNo != 1.0 {
		// Normalize
		total := probYes + probNo
		probYes = probYes / total
		probNo = probNo / total
	}

	pctYes := probYes * 100.0
	pctNo := probNo * 100.0

	var verdict string
	var confidence string

	switch {
	case pctYes >= 80:
		verdict = fmt.Sprintf("HIGH CONSENSUS TRUE (%.1f%% Probability)", pctYes)
		confidence = "HIGH"
	case pctYes >= 60:
		verdict = fmt.Sprintf("MODERATELY LIKELY (%.1f%% Probability)", pctYes)
		confidence = "MODERATE"
	case pctYes <= 20:
		verdict = fmt.Sprintf("HIGH CONSENSUS FALSE (%.1f%% Probability)", pctNo)
		confidence = "HIGH"
	case pctYes <= 40:
		verdict = fmt.Sprintf("MODERATELY UNLIKELY (%.1f%% Probability)", pctYes)
		confidence = "MODERATE"
	default:
		verdict = fmt.Sprintf("HIGHLY CONTESTED / SPLIT OPINION (%.1f%% YES / %.1f%% NO)", pctYes, pctNo)
		confidence = "SPECULATIVE"
	}

	title := market.Title
	if title == "" {
		title = fmt.Sprintf("Market #%s", market.MarketID)
	}

	desc := ""
	if market.Description != nil {
		desc = *market.Description
	}

	entryBuilder := strings.Builder{}
	entryBuilder.WriteString(fmt.Sprintf("## Truth Consensus: %s\n\n", title))
	entryBuilder.WriteString(fmt.Sprintf("**Current Market Verdict:** %s  \n", verdict))
	entryBuilder.WriteString(fmt.Sprintf("**Confidence Level:** %s | **Active Volume:** $%.2f USDC  \n\n", confidence, volume))
	entryBuilder.WriteString("### 1. Market-Derived Reality Assessment\n")
	entryBuilder.WriteString(fmt.Sprintf("Rather than relying on unverified editorial opinion or static encyclopedia curation, this entry is perpetually truth-derived from economic positions staked by participants on Callit.\n\nAt present, real capital allocation assesses the likelihood of this event at **%.1f%%** YES versus **%.1f%%** NO.\n\n", pctYes, pctNo))

	if desc != "" {
		entryBuilder.WriteString("### 2. Context & Verified Details\n")
		entryBuilder.WriteString(fmt.Sprintf("%s\n\n", desc))
	}

	endTimeStr := time.Unix(market.EndTime, 0).UTC().Format("January 2, 2006 at 15:04 UTC")
	entryBuilder.WriteString("### 3. Resolution Ground Rules & Horizon\n")
	entryBuilder.WriteString(fmt.Sprintf("Governed by on-chain consensus settling by **%s**. Settles strictly based on verifiable public evidence.\n\n", endTimeStr))

	entryBuilder.WriteString("### 4. Economic Consensus Dynamics\n")
	entryBuilder.WriteString(fmt.Sprintf("This fact sheet continuously updates with every transaction on Solana and cross-chain NEAR Intents settlements. As new evidence or information enters the public domain, pricing updates reflect the real-time truth equilibrium."))

	keyFactors := []string{
		fmt.Sprintf("Market probability currently prices YES at %.1f%% and NO at %.1f%%", pctYes, pctNo),
		fmt.Sprintf("Total market liquidity commitment: $%.2f USDC", volume),
		fmt.Sprintf("Settlement horizon concludes on %s", endTimeStr),
	}

	sources := []string{
		"https://live-api.panta.market/api/v1/markets/" + market.MarketID,
		"https://solscan.io/account/" + market.MarketID,
	}

	return &domain.TruthConsensus{
		MarketID:          market.MarketID,
		Title:             title,
		Category:          string(market.Category),
		ProbabilityYes:    probYes,
		ProbabilityNo:     probNo,
		ConsensusVerdict:  verdict,
		ConfidenceLevel:   confidence,
		VolumeUSD:         volume,
		EncyclopediaEntry: entryBuilder.String(),
		KeyFactors:        keyFactors,
		SourcesOfTruth:    sources,
		ResolutionRule:    desc,
		LastSynthesizedAt: time.Now().UTC(),
		Attribution:       "Synthesized via Callit Truth-Seeking Engine (Augur/NEAR Thesis)",
	}, nil
}
