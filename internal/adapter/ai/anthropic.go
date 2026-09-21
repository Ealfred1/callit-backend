package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/eric/callit-backend/internal/core/ports"
)

type anthropicCompiler struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewAnthropicCompiler(apiKey, model string) ports.MarketCompiler {
	return &anthropicCompiler{apiKey: apiKey, model: model, http: http.DefaultClient}
}

const systemPrompt = `You compile a pasted headline, tweet or take into a Panta prediction-market draft for Callit ("Turn any take into a market").

Rules:
- The question must resolve objectively from public sources by the resolution deadline — no matter of opinion, no "who did better".
- resolutionRule must name the exact source(s) and condition, precisely enough that two people who disagree today would both compute the same answer later.
- sourcesOfTruth must be real, specific, checkable URLs (official results pages, exchange price pages, verified accounts) — not a bare homepage.
- category must be the single best fit from the fixed 8-value enum.
- If the input is too vague, subjective, or has no checkable source, set ambiguous=true, explain why in ambiguityReason, and propose a resolvable rewrite in suggestedRewrite — still fill the other fields with your best attempt so the creator has something to edit.
- Use eventInProgress + marketType="breaking" only when the text describes something already happening live right now; otherwise marketType="standard".
- durationHours and resolutionBufferHours should fit the news cycle of the topic (e.g. a same-day sports result needs hours, not weeks).
- title/description are user-facing copy: short sentences, plain verbs, no "bonding curve" / "parimutuel" jargon.

Always call emit_market_draft exactly once with your best structured draft.`

var draftTool = map[string]any{
	"name":        "emit_market_draft",
	"description": "Emit a Panta-compatible prediction market draft compiled from the pasted text.",
	"input_schema": map[string]any{
		"type": "object",
		"required": []string{
			"question", "resolutionRule", "sourcesOfTruth", "category", "title", "description",
			"durationHours", "resolutionBufferHours", "marketType", "eventInProgress", "ambiguous",
		},
		"properties": map[string]any{
			"question":              map[string]any{"type": "string", "maxLength": 512},
			"resolutionRule":        map[string]any{"type": "string", "maxLength": 2048},
			"sourcesOfTruth":        map[string]any{"type": "array", "maxItems": 20, "items": map[string]any{"type": "string"}},
			"category":              map[string]any{"type": "string", "enum": []string{"sports", "crypto", "politics", "entertainment", "finance", "science", "world", "other"}},
			"title":                 map[string]any{"type": "string"},
			"description":           map[string]any{"type": "string"},
			"durationHours":         map[string]any{"type": "number"},
			"resolutionBufferHours": map[string]any{"type": "number"},
			"marketType":            map[string]any{"type": "string", "enum": []string{"standard", "breaking"}},
			"eventInProgress":       map[string]any{"type": "boolean"},
			"ambiguous":             map[string]any{"type": "boolean"},
			"ambiguityReason":       map[string]any{"type": "string"},
			"suggestedRewrite":      map[string]any{"type": "string"},
		},
	},
}

type anthropicRequest struct {
	Model      string           `json:"model"`
	MaxTokens  int              `json:"max_tokens"`
	System     string           `json:"system"`
	Tools      []map[string]any `json:"tools"`
	ToolChoice map[string]any   `json:"tool_choice"`
	Messages   []map[string]any `json:"messages"`
}

type anthropicContentBlock struct {
	Type  string          `json:"type"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
}

func (a *anthropicCompiler) Compile(ctx context.Context, pastedText string) (*ports.CompiledDraft, error) {
	reqBody := anthropicRequest{
		Model:      a.model,
		MaxTokens:  1024,
		System:     systemPrompt,
		Tools:      []map[string]any{draftTool},
		ToolChoice: map[string]any{"type": "tool", "name": "emit_market_draft"},
		Messages:   []map[string]any{{"role": "user", "content": pastedText}},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	res, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("anthropic API error %d: %s", res.StatusCode, string(respBytes))
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}

	for _, block := range parsed.Content {
		if block.Type == "tool_use" && block.Name == "emit_market_draft" {
			var draft ports.CompiledDraft
			if err := json.Unmarshal(block.Input, &draft); err != nil {
				return nil, fmt.Errorf("decode tool_use input: %w", err)
			}
			return &draft, nil
		}
	}
	return nil, fmt.Errorf("model did not return a structured draft")
}
