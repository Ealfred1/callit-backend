package ports

import (
	"context"

	"github.com/eric/callit-backend/internal/core/domain"
)

type CommunityRepository interface {
	Create(ctx context.Context, c *domain.Community) error
	GetByID(ctx context.Context, id string) (*domain.Community, error)
	List(ctx context.Context) ([]domain.Community, error)
}

type CreateSessionRepository interface {
	Create(ctx context.Context, s *domain.CreateSession) error
	GetByCreateID(ctx context.Context, createID string) (*domain.CreateSession, error)
	UpdateStatus(ctx context.Context, createID string, status domain.CreateSessionStatus, marketID *string) error
}

type MarketMetaRepository interface {
	Upsert(ctx context.Context, m *domain.MarketMeta) error
	GetByMarketID(ctx context.Context, marketID string) (*domain.MarketMeta, error)
	ListByCommunity(ctx context.Context, communityID string) ([]domain.MarketMeta, error)
	ListByCreator(ctx context.Context, creatorUserID string) ([]domain.MarketMeta, error)
	SetShareImagePath(ctx context.Context, marketID, path string) error
}
