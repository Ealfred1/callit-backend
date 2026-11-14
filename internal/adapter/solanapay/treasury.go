// Package solanapay sends USDC from the Callit treasury wallet to users.
package solanapay

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
)

const (
	usdcMint     = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	usdcDecimals = 6
	// rent for a new token account (~0.00204 SOL) plus fees, with headroom
	minSOLForPayout = 0.004
)

// Treasury holds the signing key for the payout wallet.
type Treasury struct {
	key  solana.PrivateKey
	rpc  *rpc.Client
	mint solana.PublicKey
}

// ParseSecret accepts a base58 secret key or a JSON array of 64 bytes.
func ParseSecret(s string) (solana.PrivateKey, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") {
		var arr []byte
		if err := json.Unmarshal([]byte(s), &arr); err != nil {
			return nil, err
		}
		if len(arr) != 64 {
			return nil, errors.New("secret key array must have 64 bytes")
		}
		return solana.PrivateKey(arr), nil
	}
	return solana.PrivateKeyFromBase58(s)
}

func NewTreasury(secret, rpcURL string) (*Treasury, error) {
	k, err := ParseSecret(secret)
	if err != nil {
		return nil, fmt.Errorf("treasury key: %w", err)
	}
	if rpcURL == "" {
		rpcURL = "https://solana-rpc.publicnode.com"
	}
	return &Treasury{key: k, rpc: rpc.New(rpcURL), mint: solana.MustPublicKeyFromBase58(usdcMint)}, nil
}

func (t *Treasury) Address() string { return t.key.PublicKey().String() }

// Balances returns the treasury's SOL and USDC.
func (t *Treasury) Balances(ctx context.Context) (sol, usdc float64, err error) {
	b, err := t.rpc.GetBalance(ctx, t.key.PublicKey(), rpc.CommitmentConfirmed)
	if err != nil {
		return 0, 0, err
	}
	sol = float64(b.Value) / 1e9
	ata, _, err := solana.FindAssociatedTokenAddress(t.key.PublicKey(), t.mint)
	if err != nil {
		return sol, 0, err
	}
	acc, err := t.rpc.GetAccountInfo(ctx, ata)
	if err != nil {
		if errors.Is(err, rpc.ErrNotFound) {
			return sol, 0, nil
		}
		return sol, 0, err
	}
	raw := acc.Value.Data.GetBinary()
	if len(raw) < 72 {
		return sol, 0, nil
	}
	return sol, float64(binary.LittleEndian.Uint64(raw[64:72])) / math.Pow10(usdcDecimals), nil
}

// ToBaseUnits converts a USDC amount to 6-decimal base units, rounding down.
func ToBaseUnits(usdc float64) uint64 { return uint64(math.Floor(usdc*1e6 + 1e-9)) }

// Prepared is a signed transfer, ready to broadcast. Its signature is known
// before it is sent, so it can be stored first and checked later.
type Prepared struct {
	Tx        *solana.Transaction
	Signature solana.Signature
}

// Prepare builds and signs: create the recipient's USDC account if missing
// (idempotent, treasury pays rent), then transfer.
func (t *Treasury) Prepare(ctx context.Context, toWallet string, baseUnits uint64) (*Prepared, error) {
	to, err := solana.PublicKeyFromBase58(toWallet)
	if err != nil {
		return nil, fmt.Errorf("bad recipient wallet: %w", err)
	}
	from := t.key.PublicKey()
	srcATA, _, err := solana.FindAssociatedTokenAddress(from, t.mint)
	if err != nil {
		return nil, err
	}
	dstATA, _, err := solana.FindAssociatedTokenAddress(to, t.mint)
	if err != nil {
		return nil, err
	}
	createIdem := solana.NewInstruction(
		solana.SPLAssociatedTokenAccountProgramID,
		solana.AccountMetaSlice{
			solana.Meta(from).WRITE().SIGNER(),
			solana.Meta(dstATA).WRITE(),
			solana.Meta(to),
			solana.Meta(t.mint),
			solana.Meta(solana.SystemProgramID),
			solana.Meta(solana.TokenProgramID),
		},
		[]byte{1}, // CreateIdempotent
	)
	transfer := token.NewTransferCheckedInstruction(baseUnits, usdcDecimals, srcATA, t.mint, dstATA, from, nil).Build()

	bh, err := t.rpc.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("blockhash: %w", err)
	}
	tx, err := solana.NewTransaction([]solana.Instruction{createIdem, transfer}, bh.Value.Blockhash, solana.TransactionPayer(from))
	if err != nil {
		return nil, err
	}
	if _, err := tx.Sign(func(k solana.PublicKey) *solana.PrivateKey {
		if k.Equals(from) {
			return &t.key
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &Prepared{Tx: tx, Signature: tx.Signatures[0]}, nil
}

// Send broadcasts a prepared transaction.
func (t *Treasury) Send(ctx context.Context, p *Prepared) error {
	_, err := t.rpc.SendTransactionWithOpts(ctx, p.Tx, rpc.TransactionOpts{SkipPreflight: false, PreflightCommitment: rpc.CommitmentConfirmed})
	return err
}

// Status says whether a signature has landed: confirmed, failed, or unknown.
func (t *Treasury) Status(ctx context.Context, sig solana.Signature) (state string, err error) {
	res, err := t.rpc.GetSignatureStatuses(ctx, true, sig)
	if err != nil {
		return "", err
	}
	if len(res.Value) == 0 || res.Value[0] == nil {
		return "unknown", nil
	}
	st := res.Value[0]
	if st.Err != nil {
		return "failed", nil
	}
	if st.ConfirmationStatus == rpc.ConfirmationStatusConfirmed || st.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
		return "confirmed", nil
	}
	return "unknown", nil
}

// WaitConfirmed polls until the signature confirms, fails or the timeout hits.
func (t *Treasury) WaitConfirmed(ctx context.Context, sig solana.Signature, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		st, err := t.Status(ctx, sig)
		if err == nil && (st == "confirmed" || st == "failed") {
			return st, nil
		}
		select {
		case <-ctx.Done():
			return "unknown", ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return "unknown", nil
}

// Enough reports whether the treasury can cover a payout of usdc.
func Enough(sol, usdcBal, need float64) bool { return sol >= minSOLForPayout && usdcBal >= need }

// Pay implements ports.Payer.
func (t *Treasury) Pay(ctx context.Context, toWallet string, usdc float64, onSigned func(string) error) (string, string, error) {
	p, err := t.Prepare(ctx, toWallet, ToBaseUnits(usdc))
	if err != nil {
		return "", "", err
	}
	sig := p.Signature.String()
	if onSigned != nil {
		if err := onSigned(sig); err != nil {
			return sig, "", fmt.Errorf("could not record signature, not sending: %w", err)
		}
	}
	if err := t.Send(ctx, p); err != nil {
		return sig, "failed", err
	}
	state, err := t.WaitConfirmed(ctx, p.Signature, 45*time.Second)
	return sig, state, err
}

// Check implements ports.Payer.
func (t *Treasury) Check(ctx context.Context, signature string) (string, error) {
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return "", err
	}
	return t.Status(ctx, sig)
}
