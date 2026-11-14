package ports

import "context"

// Payer sends USDC from the Callit treasury to a user's Solana wallet.
type Payer interface {
	Address() string
	Balances(ctx context.Context) (sol, usdc float64, err error)
	// Pay signs, calls onSigned(signature) BEFORE broadcasting, then sends and
	// waits. state is "confirmed", "failed" or "unknown".
	Pay(ctx context.Context, toWallet string, usdc float64, onSigned func(signature string) error) (signature, state string, err error)
	// Check looks up a previously signed payout: "confirmed", "failed" or "unknown".
	Check(ctx context.Context, signature string) (string, error)
}
