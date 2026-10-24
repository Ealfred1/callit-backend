package domain

import (
	"crypto/ed25519"
	"crypto/sha256"
	"math/big"
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusPendingVerification UserStatus = "PENDING_VERIFICATION"
	UserStatusActive              UserStatus = "ACTIVE"
)

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func EncodeBase58(b []byte) string {
	x := new(big.Int).SetBytes(b)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)

	var res []byte
	for x.Cmp(zero) > 0 {
		x.DivMod(x, base, mod)
		res = append(res, base58Alphabet[mod.Int64()])
	}

	for _, v := range b {
		if v != 0 {
			break
		}
		res = append(res, base58Alphabet[0])
	}

	for i, j := 0, len(res)-1; i < j; i, j = i+1, j-1 {
		res[i], res[j] = res[j], res[i]
	}
	return string(res)
}

// DeriveSolanaAddress generates a deterministic Ed25519 Solana public key for a user ID.
func DeriveSolanaAddress(userID string) string {
	seed := sha256.Sum256([]byte("callit:solana:seed:" + userID))
	privKey := ed25519.NewKeyFromSeed(seed[:])
	pubKey := privKey.Public().(ed25519.PublicKey)
	return EncodeBase58(pubKey)
}

// Unlike mollah-backend's 3-step register/verify/onboard (which needs
// DOB/country for KYC), Callit's sign-up screen already collects
// name+email+password in one step (see callitApp/lib/screens/auth/sign_up_screen.dart),
// so there's no separate onboarding stage here.
type User struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	PasswordHash  string     `json:"-"`
	Name          string     `json:"name"`
	Status        UserStatus `json:"status"`
	WalletAddress string     `json:"walletAddress"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type VerificationCode struct {
	Email     string    `json:"email"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}
