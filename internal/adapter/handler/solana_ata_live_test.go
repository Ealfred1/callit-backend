package handler

import (
	"encoding/json"
	"os"
	"testing"
)

// Run with ATA_PAIRS=/path/pairs.json: real owner -> list of USDC token accounts.
func TestAssociatedTokenAddressMatchesChain(t *testing.T) {
	path := os.Getenv("ATA_PAIRS")
	if path == "" {
		t.Skip("set ATA_PAIRS to run")
	}
	raw, _ := os.ReadFile(path)
	var owners map[string][]string
	_ = json.Unmarshal(raw, &owners)
	match := 0
	for owner, accounts := range owners {
		got, _ := associatedTokenAddress(owner, usdcMint)
		for _, a := range accounts {
			if a == got {
				match++
			}
		}
	}
	t.Logf("%d of %d owners: derived USDC token account found on chain", match, len(owners))
	if match == 0 {
		t.Fatal("derivation does not match any real token account")
	}
}
