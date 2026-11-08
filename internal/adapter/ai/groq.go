package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/eric/callit-backend/internal/core/ports"
)

type groqCompiler struct {
	apiKey string
	model  string
	http   *http.Client
}

// NewGroqCompiler creates a compiler that calls Groq's fast inference API
// (OpenAI-compatible /chat/completions) using models like llama-3.3-70b-versatile.
func NewGroqCompiler(apiKey, model string) ports.MarketCompiler {
	if model == "" {
		model = "qwen/qwen3.8-27b"
	}
	return &groqCompiler{apiKey: apiKey, model: model, http: http.DefaultClient}
}

type groqFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type groqTool struct {
	Type     string          `json:"type"`
	Function groqFunctionDef `json:"function"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqToolChoiceFunction struct {
	Name string `json:"name"`
}

type groqToolChoice struct {
	Type     string                 `json:"type"`
	Function groqToolChoiceFunction `json:"function"`
}

type groqRequest struct {
	Model       string          `json:"model"`
	Messages    []groqMessage   `json:"messages"`
	Tools       []groqTool      `json:"tools,omitempty"`
	ToolChoice  *groqToolChoice `json:"tool_choice,omitempty"`
	Temperature float64         `json:"temperature"`
}

type groqToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type groqChoiceMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	ToolCalls []groqToolCall `json:"tool_calls,omitempty"`
}

type groqChoice struct {
	Message groqChoiceMessage `json:"message"`
}

type groqResponse struct {
	Choices []groqChoice `json:"choices"`
}

var groqDraftParameters = map[string]any{
	"type": "object",
	"required": []string{
		"question", "resolutionRule", "sourcesOfTruth", "category", "title", "description",
		"durationHours", "resolutionBufferHours", "marketType", "eventInProgress", "ambiguous",
	},
	"properties": map[string]any{
		"question":              map[string]any{"type": "string", "maxLength": 512, "description": "Objective question resolving YES/NO"},
		"resolutionRule":        map[string]any{"type": "string", "maxLength": 2048, "description": "Precise criteria and exact condition for resolution"},
		"sourcesOfTruth":        map[string]any{"type": "array", "maxItems": 20, "items": map[string]any{"type": "string"}, "description": "Specific checkable URLs"},
		"category":              map[string]any{"type": "string", "enum": []string{"sports", "crypto", "politics", "entertainment", "finance", "science", "world", "other"}},
		"title":                 map[string]any{"type": "string"},
		"description":           map[string]any{"type": "string"},
		"durationHours":         map[string]any{"type": "number", "description": "Hours until market close"},
		"resolutionBufferHours": map[string]any{"type": "number", "description": "Buffer hours between market close and resolution"},
		"marketType":            map[string]any{"type": "string", "enum": []string{"standard", "breaking"}},
		"eventInProgress":       map[string]any{"type": "boolean"},
		"ambiguous":             map[string]any{"type": "boolean"},
		"ambiguityReason":       map[string]any{"type": "string"},
		"suggestedRewrite":      map[string]any{"type": "string"},
	},
}

func (g *groqCompiler) Compile(ctx context.Context, pastedText string) (*ports.CompiledDraft, error) {
	reqBody := groqRequest{
		Model: g.model,
		Messages: []groqMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: pastedText},
		},
		Tools: []groqTool{
			{
				Type: "function",
				Function: groqFunctionDef{
					Name:        "emit_market_draft",
					Description: "Emit a Panta-compatible prediction market draft compiled from the pasted text.",
					Parameters:  groqDraftParameters,
				},
			},
		},
		ToolChoice: &groqToolChoice{
			Type:     "function",
			Function: groqToolChoiceFunction{Name: "emit_market_draft"},
		},
		Temperature: 0.1,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("groq request failed: %w", err)
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return NewMockCompiler().Compile(ctx, pastedText)
	}
	if res.StatusCode >= 300 {
		// Fallback to deterministic mock compiler if Groq key is expired, rate-limited, or unavailable
		return NewMockCompiler().Compile(ctx, pastedText)
	}

	var parsed groqResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return NewMockCompiler().Compile(ctx, pastedText)
	}

	if len(parsed.Choices) == 0 {
		return NewMockCompiler().Compile(ctx, pastedText)
	}

	msg := parsed.Choices[0].Message

	// Primary: tool call
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name == "emit_market_draft" && tc.Function.Arguments != "" {
			var draft ports.CompiledDraft
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &draft); err != nil {
				return nil, fmt.Errorf("decode tool call arguments: %w", err)
			}
			return &draft, nil
		}
	}

	// Fallback: raw json in message content if the model output direct JSON
	trimmed := strings.TrimSpace(msg.Content)
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		var draft ports.CompiledDraft
		if err := json.Unmarshal([]byte(trimmed), &draft); err == nil && draft.Question != "" {
			return &draft, nil
		}
	}

	return NewMockCompiler().Compile(ctx, pastedText)
}
