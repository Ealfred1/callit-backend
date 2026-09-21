package ai

import (
	"context"
	"strings"

	"github.com/eric/callit-backend/internal/core/ports"
)

// Deterministic stand-in so the create flow can be built/tested without an
// Anthropic key (AI_MOCK=true). Not a quality bar — just structurally valid.
type mockCompiler struct{}

func NewMockCompiler() ports.MarketCompiler {
	return &mockCompiler{}
}

func (m *mockCompiler) Compile(ctx context.Context, pastedText string) (*ports.CompiledDraft, error) {
	trimmed := strings.TrimSpace(pastedText)
	question := trimmed + "?"
	if len(trimmed) > 500 {
		question = trimmed[:497] + "...?"
	}
	preview := trimmed
	if len(preview) > 80 {
		preview = preview[:80]
	}
	ambiguous := len(trimmed) < 12

	draft := &ports.CompiledDraft{
		Question:              question,
		ResolutionRule:         `Resolves YES if, by the resolution deadline, the outcome described ("` + preview + `") is confirmed true by the listed sources; NO otherwise.`,
		SourcesOfTruth:          []string{"https://example.com/source-1"},
		Category:                 "other",
		Title:                     truncate(trimmed, 60),
		Description:                truncate(trimmed, 200),
		DurationHours:                24,
		ResolutionBufferHours:          3,
		MarketType:                       "standard",
		EventInProgress:                    false,
		Ambiguous:                            ambiguous,
	}
	if ambiguous {
		draft.AmbiguityReason = "Too short to identify a checkable claim."
		draft.SuggestedRewrite = `Will "` + trimmed + `" happen by [specific date]?`
	}
	return draft, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
