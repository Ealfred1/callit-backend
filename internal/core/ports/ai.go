package ports

import "context"

// CompiledDraft is the AI compiler's output — a Panta-shaped market draft
// plus the PRD's "ambiguity check" fields, so the creator can review/edit
// before anything is quoted (and before any USDC is spent).
type CompiledDraft struct {
	Question              string   `json:"question"`
	ResolutionRule        string   `json:"resolutionRule"`
	SourcesOfTruth        []string `json:"sourcesOfTruth"`
	Category              string   `json:"category"`
	Title                 string   `json:"title"`
	Description           string   `json:"description"`
	DurationHours         float64  `json:"durationHours"`         // start -> trading close
	ResolutionBufferHours float64  `json:"resolutionBufferHours"` // trading close -> resolution deadline
	MarketType            string   `json:"marketType"`            // "standard" | "breaking"
	EventInProgress       bool     `json:"eventInProgress"`
	Ambiguous             bool     `json:"ambiguous"`
	AmbiguityReason       string   `json:"ambiguityReason,omitempty"`
	SuggestedRewrite      string   `json:"suggestedRewrite,omitempty"`
}

// MarketCompiler is implemented by internal/adapter/ai/anthropic.go (real)
// and internal/adapter/ai/mock.go (deterministic, AI_MOCK=true).
type MarketCompiler interface {
	Compile(ctx context.Context, pastedText string) (*CompiledDraft, error)
}
