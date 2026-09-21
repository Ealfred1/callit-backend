package service

import (
	"context"
	"crypto/rand"
	"math/big"
	"time"

	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo ports.UserRepository
	email    ports.EmailService
}

func NewAuthService(userRepo ports.UserRepository, email ports.EmailService) *AuthService {
	return &AuthService{userRepo: userRepo, email: email}
}

// Register: Callit's sign-up screen collects name+email+password together
// (unlike mollah-backend's 3-step KYC-style flow), so this both creates the
// user (PENDING_VERIFICATION) and sends the code in one call.
func (s *AuthService) Register(ctx context.Context, email, password, name string) error {
	if existing, err := s.userRepo.GetByEmail(ctx, email); err == nil && existing != nil && existing.Status == domain.UserStatusActive {
		return respond.BadRequest("EMAIL_TAKEN", "an account with this email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &domain.User{
		Email: email, PasswordHash: string(hash), Name: name,
		Status: domain.UserStatusPendingVerification, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}

	code, err := generateRandomCode(6)
	if err != nil {
		return err
	}
	if err := s.userRepo.UpsertVerificationCode(ctx, &domain.VerificationCode{
		Email: email, Code: code, ExpiresAt: time.Now().Add(10 * time.Minute),
	}); err != nil {
		return err
	}

	return s.email.SendVerificationEmail(email, code)
}

func (s *AuthService) Verify(ctx context.Context, email, code string) (*domain.User, error) {
	stored, err := s.userRepo.GetVerificationCode(ctx, email)
	if err != nil {
		return nil, respond.Unauthorized("invalid verification code")
	}
	if stored.Code != code {
		return nil, respond.Unauthorized("incorrect verification code")
	}
	if time.Now().After(stored.ExpiresAt) {
		return nil, respond.Unauthorized("code expired")
	}
	_ = s.userRepo.DeleteVerificationCode(ctx, email)

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, respond.NotFound("USER_NOT_FOUND", "no pending signup for this email")
	}
	if err := s.userRepo.SetStatus(ctx, user.ID.String(), domain.UserStatusActive); err != nil {
		return nil, err
	}
	user.Status = domain.UserStatusActive
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, respond.Unauthorized("invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, respond.Unauthorized("invalid credentials")
	}
	if user.Status != domain.UserStatusActive {
		return nil, respond.Unauthorized("account not verified")
	}
	return user, nil
}

func (s *AuthService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func generateRandomCode(length int) (string, error) {
	const digits = "0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		result[i] = digits[n.Int64()]
	}
	return string(result), nil
}
