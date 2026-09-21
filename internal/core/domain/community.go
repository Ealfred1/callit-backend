package domain

import (
	"time"

	"github.com/google/uuid"
)

// AttributionID is sent to Panta as the buy-flow `userId` (PRD "Attribution:
// send an X-User-Id per community on quote and build so volume is credited
// per community") so every community's trading volume can be rolled up
// straight from Panta's own data — Callit never stores volume itself.
type Community struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	AttributionID string    `json:"attributionId"`
	CreatorUserID uuid.UUID `json:"creatorUserId"`
	CreatedAt     time.Time `json:"createdAt"`
}
