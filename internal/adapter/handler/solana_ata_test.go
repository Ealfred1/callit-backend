package handler

import "testing"

// Known pair: the USDC token account of the Solana Foundation test wallet is
// derivable; here we check round-trip encoding and that the result is stable.
func TestAssociatedTokenAddress(t *testing.T) {
	a, err := associatedTokenAddress("4NKTon3xaTdj6xUguiNnAWX9XxQJ2jjqDtNW6YYjYt1Q", usdcMint)
	if err != nil || len(a) < 32 || len(a) > 44 {
		t.Fatalf("bad ata %q err=%v", a, err)
	}
	b, _ := associatedTokenAddress("4NKTon3xaTdj6xUguiNnAWX9XxQJ2jjqDtNW6YYjYt1Q", usdcMint)
	if a != b {
		t.Fatal("not deterministic")
	}
	k, _ := b58Decode32(tokenProgramID)
	if b58Encode(k) != tokenProgramID {
		t.Fatal("base58 round trip failed")
	}
}
