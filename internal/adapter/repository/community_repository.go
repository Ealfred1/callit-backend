package repository

import (
	"context"
	"fmt"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

type communityRepository struct {
	db *pgxpool.Pool
}

func NewCommunityRepository(db *pgxpool.Pool) ports.CommunityRepository {
	return &communityRepository{db: db}
}

func (r *communityRepository) Create(ctx context.Context, c *domain.Community) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO communities (name, slug, attribution_id, creator_user_id, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id, created_at
	`, c.Name, c.Slug, c.AttributionID, c.CreatorUserID).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return fmt.Errorf("create community: %w", err)
	}
	return nil
}

func (r *communityRepository) GetByID(ctx context.Context, id string) (*domain.Community, error) {
	c := &domain.Community{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, slug, attribution_id, creator_user_id, created_at
		FROM communities WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Slug, &c.AttributionID, &c.CreatorUserID, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *communityRepository) List(ctx context.Context) ([]domain.Community, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, slug, attribution_id, creator_user_id, created_at
		FROM communities ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Community
	for rows.Next() {
		var c domain.Community
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.AttributionID, &c.CreatorUserID, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
