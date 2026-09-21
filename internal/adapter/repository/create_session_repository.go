package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

type createSessionRepository struct {
	db *pgxpool.Pool
}

func NewCreateSessionRepository(db *pgxpool.Pool) ports.CreateSessionRepository {
	return &createSessionRepository{db: db}
}

func (r *createSessionRepository) Create(ctx context.Context, s *domain.CreateSession) error {
	draftJSON, err := json.Marshal(s.Draft)
	if err != nil {
		return fmt.Errorf("marshal draft: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO create_sessions (create_id, creator_user_id, community_id, draft, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, s.CreateID, s.CreatorUserID, s.CommunityID, json.RawMessage(draftJSON), s.Status)
	if err != nil {
		return fmt.Errorf("create create_session: %w", err)
	}
	return nil
}

func (r *createSessionRepository) GetByCreateID(ctx context.Context, createID string) (*domain.CreateSession, error) {
	var s domain.CreateSession
	var draftJSON json.RawMessage
	err := r.db.QueryRow(ctx, `
		SELECT create_id, creator_user_id, community_id, draft, status, market_id, created_at, updated_at
		FROM create_sessions WHERE create_id = $1
	`, createID).Scan(&s.CreateID, &s.CreatorUserID, &s.CommunityID, &draftJSON, &s.Status, &s.MarketID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(draftJSON, &s.Draft); err != nil {
		return nil, fmt.Errorf("unmarshal draft: %w", err)
	}
	return &s, nil
}

func (r *createSessionRepository) UpdateStatus(ctx context.Context, createID string, status domain.CreateSessionStatus, marketID *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE create_sessions SET status = $1, market_id = COALESCE($2, market_id), updated_at = NOW()
		WHERE create_id = $3
	`, status, marketID, createID)
	return err
}
