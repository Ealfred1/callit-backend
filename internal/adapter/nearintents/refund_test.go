package nearintents

import "testing"

func TestValidateRefundAddress(t *testing.T) {
	cases := []struct {
		asset, addr string
		ok          bool
	}{
		{AssetIdNearWNear, "you.near", true},
		{AssetIdNearWNear, "", false},
		{AssetIdNearWNear, "alice.near", true},
		{AssetIdNearWNear, "BuviaRJWHby737pShLbyd53UXqYmtWCWAsvJ7pDEsxEa", false},
		{AssetIdEthETH, "0x1111111111111111111111111111111111111111", true},
		{AssetIdEthETH, "BuviaRJWHby737pShLbyd53UXqYmtWCWAsvJ7pDEsxEa", false},
		{AssetIdBaseETH, "0xabc", false},
		{AssetIdSolSOL, "BuviaRJWHby737pShLbyd53UXqYmtWCWAsvJ7pDEsxEa", true},
		{AssetIdSolSOL, "you.near", false},
	}
	for _, c := range cases {
		err := ValidateRefundAddress(c.asset, c.addr)
		if (err == nil) != c.ok {
			t.Errorf("asset=%s addr=%q: ok=%v err=%v", c.asset, c.addr, c.ok, err)
		}
	}
}
