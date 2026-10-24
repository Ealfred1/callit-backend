package repository

import (
	"context"
	"fmt"

	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) ports.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (email, password_hash, name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.db.QueryRow(ctx, query,
		user.Email, user.PasswordHash, user.Name, user.Status, user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	user.WalletAddress = domain.DeriveSolanaAddress(user.ID.String())
	return nil
}

func (r *userRepository) scanUser(row interface {
	Scan(dest ...any) error
}) (*domain.User, error) {
	user := &domain.User{}
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	user.WalletAddress = domain.DeriveSolanaAddress(user.ID.String())
	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, name, status, created_at, updated_at
		FROM users WHERE email = $1
	`, email)
	return r.scanUser(row)
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, name, status, created_at, updated_at
		FROM users WHERE id = $1
	`, id)
	return r.scanUser(row)
}

func (r *userRepository) SetStatus(ctx context.Context, userID string, status domain.UserStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2`, status, userID)
	return err
}

func (r *userRepository) UpsertVerificationCode(ctx context.Context, cv *domain.VerificationCode) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO verification_codes (email, code, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE SET code = EXCLUDED.code, expires_at = EXCLUDED.expires_at
	`, cv.Email, cv.Code, cv.ExpiresAt)
	if err != nil {
		return fmt.Errorf("upsert verification code: %w", err)
	}
	return nil
}

func (r *userRepository) GetVerificationCode(ctx context.Context, email string) (*domain.VerificationCode, error) {
	cv := &domain.VerificationCode{}
	err := r.db.QueryRow(ctx, `SELECT email, code, expires_at FROM verification_codes WHERE email = $1`, email).
		Scan(&cv.Email, &cv.Code, &cv.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return cv, nil
}

func (r *userRepository) DeleteVerificationCode(ctx context.Context, email string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM verification_codes WHERE email = $1`, email)
	return err
}
