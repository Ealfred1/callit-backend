package domain

import (
	"time"

	"github.com/google/uuid"
)

type CreateSessionStatus string

const (
	CreateSessionStatusQuoted     CreateSessionStatus = "quoted"
	CreateSessionStatusBuilt      CreateSessionStatus = "built"
	CreateSessionStatusRegistered CreateSessionStatus = "registered"
)

// CreateSession bridges the AI-compiled draft through Panta's
// quote -> build -> register pipeline (PRD "Creator flow"). It's looked up
// by CreateID (Panta's id) when /markets/register/ comes back, so we know
// which community/creator to attach the finished market to.
type CreateSession struct {
	CreateID      string              `json:"createId"`
	CreatorUserID uuid.UUID           `json:"creatorUserId"`
	CommunityID   *uuid.UUID          `json:"communityId,omitempty"`
	Draft         CreateQuoteRequest  `json:"draft"`
	Status        CreateSessionStatus `json:"status"`
	MarketID      *string             `json:"marketId,omitempty"`
	CreatedAt     time.Time           `json:"createdAt"`
	UpdatedAt     time.Time           `json:"updatedAt"`
}

// MarketMeta is what Callit stores about a market that Panta doesn't already
// serve — Panta stays the source of truth for prices/volume/phase (PRD "Key
// decisions"). One row per market that came through Callit's create flow.
type MarketMeta struct {
	MarketID       string     `json:"marketId"`
	CreateID       *string    `json:"createId,omitempty"`
	CreatorUserID  uuid.UUID  `json:"creatorUserId"`
	CommunityID    *uuid.UUID `json:"communityId,omitempty"`
	Category       string     `json:"category"`
	Title          string     `json:"title"`
	ShareImagePath *string    `json:"shareImagePath,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}
