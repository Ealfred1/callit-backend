package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusPendingVerification UserStatus = "PENDING_VERIFICATION"
	UserStatusActive              UserStatus = "ACTIVE"
)

// Unlike mollah-backend's 3-step register/verify/onboard (which needs
// DOB/country for KYC), Callit's sign-up screen already collects
// name+email+password in one step (see callitApp/lib/screens/auth/sign_up_screen.dart),
// so there's no separate onboarding stage here.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Name         string     `json:"name"`
	Status       UserStatus `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type VerificationCode struct {
	Email     string    `json:"email"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}
