package service

import (
	"context"
	"strings"

	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/google/uuid"
)

type MarketCreateService struct {
	panta          ports.PantaClient
	compiler       ports.MarketCompiler
	createSessions ports.CreateSessionRepository
	marketMeta     ports.MarketMetaRepository
	communities    ports.CommunityRepository
}

func NewMarketCreateService(
	panta ports.PantaClient, compiler ports.MarketCompiler,
	createSessions ports.CreateSessionRepository, marketMeta ports.MarketMetaRepository, communities ports.CommunityRepository,
) *MarketCreateService {
	return &MarketCreateService{panta: panta, compiler: compiler, createSessions: createSessions, marketMeta: marketMeta, communities: communities}
}

type CompileResult struct {
	Draft          *ports.CompiledDraft `json:"draft"`
	StartTime      int64                `json:"startTime"`
	EndTime        int64                `json:"endTime"`
	ResolutionTime int64                `json:"resolutionTime"`
}

// Compile is step 1 of the create flow (PRD "Creator flow" 1-2). It does NOT
// call Panta — the Quote call is what the PRD calls "the validator", and
// doubles as the ambiguity/USDC-fee check before anything is spent.
func (s *MarketCreateService) Compile(ctx context.Context, text string) (*CompileResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, respond.BadRequest("INVALID_MARKET_PARAMS", "text is required")
	}
	if len(text) > 4000 {
		return nil, respond.BadRequest("INVALID_MARKET_PARAMS", "text is too long to compile (max 4000 chars)")
	}

	draft, err := s.compiler.Compile(ctx, text)
	if err != nil {
		return nil, err
	}
	ts := ComputeMarketTimestamps(draft.EventInProgress, draft.DurationHours, draft.ResolutionBufferHours)

	return &CompileResult{Draft: draft, StartTime: ts.StartTime, EndTime: ts.EndTime, ResolutionTime: ts.ResolutionTime}, nil
}

func (s *MarketCreateService) Quote(ctx context.Context, creatorUserID string, req domain.CreateQuoteRequest, communityID string) (*domain.CreateQuoteResponse, error) {
	if req.Wallet == "" {
		return nil, respond.BadRequest("INVALID_MARKET_PARAMS", "wallet is required")
	}

	var communityUUID *uuid.UUID
	if communityID != "" {
		if _, err := s.communities.GetByID(ctx, communityID); err != nil {
			return nil, respond.NotFound("COMMUNITY_NOT_FOUND", "unknown communityId")
		}
		id, err := uuid.Parse(communityID)
		if err != nil {
			return nil, respond.BadRequest("INVALID_MARKET_PARAMS", "communityId is not a valid id")
		}
		communityUUID = &id
	}

	quote, err := s.panta.CreateQuote(ctx, req)
	if err != nil {
		return nil, err
	}

	creatorUUID, err := uuid.Parse(creatorUserID)
	if err != nil {
		return nil, respond.Unauthorized("invalid user")
	}

	session := &domain.CreateSession{
		CreateID: quote.CreateID, CreatorUserID: creatorUUID, CommunityID: communityUUID,
		Draft: req, Status: domain.CreateSessionStatusQuoted,
	}
	if err := s.createSessions.Create(ctx, session); err != nil {
		return nil, err
	}

	return quote, nil
}

func (s *MarketCreateService) Build(ctx context.Context, req domain.CreateBuildRequest) (*domain.CreateBuildResponse, error) {
	if req.CreateID == "" {
		return nil, respond.BadRequest("INVALID_MARKET_PARAMS", "createId is required")
	}
	build, err := s.panta.CreateBuild(ctx, req)
	if err != nil {
		return nil, err
	}
	_ = s.createSessions.UpdateStatus(ctx, req.CreateID, domain.CreateSessionStatusBuilt, nil)
	return build, nil
}

func (s *MarketCreateService) Register(ctx context.Context, fallbackCreatorUserID string, req domain.RegisterMarketRequest) (*domain.RegisterMarketResponse, error) {
	if req.CreateID == "" || req.Signature == "" {
		return nil, respond.BadRequest("INVALID_MARKET_PARAMS", "createId and signature are required")
	}

	result, err := s.panta.RegisterMarket(ctx, req)
	if err != nil {
		return nil, err
	}

	session, sessErr := s.createSessions.GetByCreateID(ctx, req.CreateID)

	creatorUserID := fallbackCreatorUserID
	var communityID *uuid.UUID
	if sessErr == nil && session != nil {
		creatorUserID = session.CreatorUserID.String()
		communityID = session.CommunityID
	}

	_ = s.createSessions.UpdateStatus(ctx, req.CreateID, domain.CreateSessionStatusRegistered, &result.MarketID)

	creatorUUID, err := uuid.Parse(creatorUserID)
	if err != nil {
		return result, nil // registration already succeeded upstream; metadata persistence is best-effort
	}
	createID := req.CreateID
	_ = s.marketMeta.Upsert(ctx, &domain.MarketMeta{
		MarketID: result.MarketID, CreateID: &createID, CreatorUserID: creatorUUID, CommunityID: communityID,
		Category: result.Category, Title: result.Title,
	})

	return result, nil
}
