package handler

import (
	"crypto/sha256"
	"errors"
	"math/big"

	"filippo.io/edwards25519"
)

const (
	tokenProgramID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	ataProgramID   = "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL"
	b58Alphabet    = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
)

func b58Decode32(s string) ([]byte, error) {
	n := new(big.Int)
	base := big.NewInt(58)
	for _, c := range []byte(s) {
		i := -1
		for j := 0; j < len(b58Alphabet); j++ {
			if b58Alphabet[j] == c {
				i = j
				break
			}
		}
		if i < 0 {
			return nil, errors.New("invalid base58")
		}
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(i)))
	}
	b := n.Bytes()
	if len(b) > 32 {
		return nil, errors.New("not a 32 byte key")
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out, nil
}

func b58Encode(b []byte) string {
	n := new(big.Int).SetBytes(b)
	base := big.NewInt(58)
	mod := new(big.Int)
	var out []byte
	for n.Sign() > 0 {
		n.DivMod(n, base, mod)
		out = append(out, b58Alphabet[mod.Int64()])
	}
	for _, v := range b {
		if v != 0 {
			break
		}
		out = append(out, b58Alphabet[0])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

// associatedTokenAddress derives the standard token account for owner+mint.
// It is a program derived address, so it must not be a valid ed25519 point.
func associatedTokenAddress(owner, mint string) (string, error) {
	o, err := b58Decode32(owner)
	if err != nil {
		return "", err
	}
	m, err := b58Decode32(mint)
	if err != nil {
		return "", err
	}
	tp, _ := b58Decode32(tokenProgramID)
	ap, _ := b58Decode32(ataProgramID)
	for bump := 255; bump >= 0; bump-- {
		h := sha256.New()
		h.Write(o)
		h.Write(tp)
		h.Write(m)
		h.Write([]byte{byte(bump)})
		h.Write(ap)
		h.Write([]byte("ProgramDerivedAddress"))
		sum := h.Sum(nil)
		if _, err := new(edwards25519.Point).SetBytes(sum); err != nil {
			return b58Encode(sum), nil // off the curve: this is the PDA
		}
	}
	return "", errors.New("no valid bump")
}
