package repository

import (
	"context"
	"fmt"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

type marketMetaRepository struct {
	db *pgxpool.Pool
}

func NewMarketMetaRepository(db *pgxpool.Pool) ports.MarketMetaRepository {
	return &marketMetaRepository{db: db}
}

func (r *marketMetaRepository) Upsert(ctx context.Context, m *domain.MarketMeta) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO market_meta (market_id, create_id, creator_user_id, community_id, category, title, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (market_id) DO UPDATE SET
			category = EXCLUDED.category,
			title = EXCLUDED.title
	`, m.MarketID, m.CreateID, m.CreatorUserID, m.CommunityID, m.Category, m.Title)
	if err != nil {
		return fmt.Errorf("upsert market_meta: %w", err)
	}
	return nil
}

func (r *marketMetaRepository) scan(rows interface {
	Scan(dest ...any) error
}) (domain.MarketMeta, error) {
	var m domain.MarketMeta
	err := rows.Scan(&m.MarketID, &m.CreateID, &m.CreatorUserID, &m.CommunityID, &m.Category, &m.Title, &m.ShareImagePath, &m.CreatedAt)
	return m, err
}

const marketMetaColumns = `market_id, create_id, creator_user_id, community_id, category, title, share_image_path, created_at`

func (r *marketMetaRepository) GetByMarketID(ctx context.Context, marketID string) (*domain.MarketMeta, error) {
	row := r.db.QueryRow(ctx, `SELECT `+marketMetaColumns+` FROM market_meta WHERE market_id = $1`, marketID)
	m, err := r.scan(row)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *marketMetaRepository) ListByCommunity(ctx context.Context, communityID string) ([]domain.MarketMeta, error) {
	rows, err := r.db.Query(ctx, `SELECT `+marketMetaColumns+` FROM market_meta WHERE community_id = $1 ORDER BY created_at DESC`, communityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MarketMeta
	for rows.Next() {
		m, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *marketMetaRepository) ListByCreator(ctx context.Context, creatorUserID string) ([]domain.MarketMeta, error) {
	rows, err := r.db.Query(ctx, `SELECT `+marketMetaColumns+` FROM market_meta WHERE creator_user_id = $1 ORDER BY created_at DESC`, creatorUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MarketMeta
	for rows.Next() {
		m, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *marketMetaRepository) SetShareImagePath(ctx context.Context, marketID, path string) error {
	_, err := r.db.Exec(ctx, `UPDATE market_meta SET share_image_path = $1 WHERE market_id = $2`, path, marketID)
	return err
}
