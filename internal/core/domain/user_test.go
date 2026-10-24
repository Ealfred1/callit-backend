package domain

import (
	"testing"
)

func TestDeriveSolanaAddress(t *testing.T) {
	addr1 := DeriveSolanaAddress("e22610cb-e342-4d31-89ef-dc0fd9613459")
	if len(addr1) < 40 || len(addr1) > 45 {
		t.Fatalf("expected valid base58 address length, got %d (%s)", len(addr1), addr1)
	}

	// Deterministic
	addr2 := DeriveSolanaAddress("e22610cb-e342-4d31-89ef-dc0fd9613459")
	if addr1 != addr2 {
		t.Fatalf("expected deterministic address, got %s and %s", addr1, addr2)
	}

	// Different for different users
	addr3 := DeriveSolanaAddress("44fad27f-1afc-41fa-98b9-263e70fd2655")
	if addr1 == addr3 {
		t.Fatalf("expected distinct addresses for different users, got %s", addr1)
	}
}
