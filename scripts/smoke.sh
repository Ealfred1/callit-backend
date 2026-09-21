#!/bin/bash
# End-to-end smoke test against a locally running `make dev` server in mock
# mode (PANTA_MOCK=true, AI_MOCK=true, RESEND_API_KEY unset). Exercises the
# full register -> verify -> login -> ai-compile -> create -> buy -> verify
# -> claim -> share-card -> communities chain and asserts each response
# shape, so a broken wire-up fails loudly here instead of in the Flutter app.
set -euo pipefail

ROOT="${BASE_URL:-http://localhost:8095}"
BASE="$ROOT/v1"
EMAIL="smoke-$(date +%s)@example.com"
PASSWORD="a-strong-password"
FAIL=0

step() { echo; echo "→ $1"; }
ok() { echo "  ✓ $1"; }
fail() { echo "  ✗ $1"; FAIL=1; }

req() { curl -sS -H "Content-Type: application/json" "$@"; }
authed() { curl -sS -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" "$@"; }

step "Health"
res=$(req "$ROOT/health")
[ "$(echo "$res" | jq -r .status)" = "ok" ] && ok "health" || fail "health: $res"

step "Register"
res=$(req -X POST "$BASE/auth/register" -d "{\"name\":\"Smoke Test\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
echo "$res" | jq -e .data.message > /dev/null && ok "registered $EMAIL" || fail "register: $res"

step "Fetch OTP code from DB (mock mailer logs it, doesn't send it)"
CODE=$(docker compose exec -T db psql -U callit -d callit_db -t -A -c "select code from verification_codes where email='$EMAIL'")
[ -n "$CODE" ] && ok "code: $CODE" || { fail "could not read code"; exit 1; }

step "Verify"
res=$(req -X POST "$BASE/auth/verify" -d "{\"email\":\"$EMAIL\",\"code\":\"$CODE\"}")
TOKEN=$(echo "$res" | jq -r .data.token)
[ "$TOKEN" != "null" ] && [ -n "$TOKEN" ] && ok "got JWT" || fail "verify: $res"

step "Login (confirms password + verified status)"
res=$(req -X POST "$BASE/auth/login" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
echo "$res" | jq -e .data.token > /dev/null && ok "login" || fail "login: $res"

step "AI compile"
res=$(authed -X POST "$BASE/ai/compile" -d '{"text":"Will it rain in London this Friday?"}')
QUESTION=$(echo "$res" | jq -r .data.draft.question)
START_TIME=$(echo "$res" | jq -r .data.startTime)
END_TIME=$(echo "$res" | jq -r .data.endTime)
RES_TIME=$(echo "$res" | jq -r .data.resolutionTime)
[ "$QUESTION" != "null" ] && ok "draft: $QUESTION" || fail "ai/compile: $res"

step "Create community"
res=$(authed -X POST "$BASE/communities" -d '{"name":"Smoke Test Community"}')
COMMUNITY_ID=$(echo "$res" | jq -r .data.id)
[ "$COMMUNITY_ID" != "null" ] && ok "community: $COMMUNITY_ID" || fail "communities: $res"

step "Create quote"
WALLET="SmokeWallet11111111111111111111111111111"
res=$(authed -X POST "$BASE/markets/create/quote" -d "{
  \"wallet\":\"$WALLET\",\"question\":\"$QUESTION\",
  \"resolutionRule\":\"Resolves per the Met Office forecast page.\",
  \"sourcesOfTruth\":[\"https://example.com/forecast\"],\"category\":\"world\",
  \"startTime\":$START_TIME,\"endTime\":$END_TIME,\"resolutionTime\":$RES_TIME,
  \"imageUrl\":\"https://example.com/image.png\",\"communityId\":\"$COMMUNITY_ID\"
}")
CREATE_ID=$(echo "$res" | jq -r .data.createId)
[ "$CREATE_ID" != "null" ] && ok "createId: $CREATE_ID" || fail "create/quote: $res"

step "Create build"
res=$(authed -X POST "$BASE/markets/create/build" -d "{\"createId\":\"$CREATE_ID\"}")
echo "$res" | jq -e .data.transaction > /dev/null && ok "build" || fail "create/build: $res"

step "Register market"
res=$(authed -X POST "$BASE/markets/create/register" -d "{\"createId\":\"$CREATE_ID\",\"signature\":\"MockSig1111\"}")
MARKET_ID=$(echo "$res" | jq -r .data.marketId)
[ "$MARKET_ID" != "null" ] && ok "marketId: $MARKET_ID" || fail "create/register: $res"

step "Share card"
res=$(authed -X POST "$BASE/share-card" -d "{\"marketId\":\"$MARKET_ID\"}")
echo "$res" | jq -e .data.imageUrl > /dev/null && ok "share card rendered" || fail "share-card: $res"

step "Share unfurl page (public)"
html=$(curl -sS "$BASE/share/$MARKET_ID")
echo "$html" | grep -q "og:image" && ok "unfurl HTML has og:image" || fail "unfurl page missing og:image"

step "Buy: order quote"
res=$(authed -X POST "$BASE/orders/quote" -d "{\"wallet\":\"$WALLET\",\"marketId\":\"$MARKET_ID\",\"side\":\"yes\",\"amountUsdc\":\"20.00\",\"communityId\":\"$COMMUNITY_ID\"}")
QUOTE_ID=$(echo "$res" | jq -r .data.quoteId)
[ "$QUOTE_ID" != "null" ] && ok "quoteId: $QUOTE_ID" || fail "orders/quote: $res"

step "Buy: order build"
res=$(authed -X POST "$BASE/orders/build" -d "{\"quoteId\":\"$QUOTE_ID\",\"wallet\":\"$WALLET\"}")
ORDER_ID=$(echo "$res" | jq -r .data.orderId)
[ "$ORDER_ID" != "null" ] && ok "orderId: $ORDER_ID" || fail "orders/build: $res"

step "Buy: order submit"
res=$(authed -X POST "$BASE/orders/submit" -d "{\"orderId\":\"$ORDER_ID\",\"signature\":\"MockBuySig1\"}")
echo "$res" | jq -e '.data.status == "submitted"' > /dev/null && ok "submitted" || fail "orders/submit: $res"

step "Buy: order verify"
res=$(authed -X POST "$BASE/orders/verify" -d "{\"orderId\":\"$ORDER_ID\"}")
echo "$res" | jq -e '.data.status == "confirmed"' > /dev/null && ok "confirmed" || fail "orders/verify: $res"

step "Report trade"
res=$(authed -X POST "$BASE/trades/report" -d "{\"signature\":\"MockBuySig1\",\"wallet\":\"$WALLET\",\"marketId\":\"$MARKET_ID\",\"communityId\":\"$COMMUNITY_ID\"}")
echo "$res" | jq -e '.data.status == "processed"' > /dev/null && ok "trade reported" || fail "trades/report: $res"

step "Positions"
res=$(authed "$BASE/positions?wallet=$WALLET")
echo "$res" | jq -e '.data.positions != null' > /dev/null && ok "positions" || fail "positions: $res"

step "Claim win (build only — mock market never resolves, just checks the endpoint wires up)"
res=$(authed -X POST "$BASE/claims/win" -d "{\"wallet\":\"$WALLET\",\"marketId\":\"$MARKET_ID\"}")
echo "$res" | jq -e .data.instructions > /dev/null && ok "claim build" || fail "claims/win: $res"

step "Creator fees claim"
res=$(authed -X POST "$BASE/claims/creator-fees" -d "{\"wallet\":\"$WALLET\",\"marketId\":\"$MARKET_ID\"}")
echo "$res" | jq -e .data.claimableFeesUsdc > /dev/null && ok "creator fees" || fail "claims/creator-fees: $res"

step "Creator dashboard (/me/markets)"
res=$(authed "$BASE/me/markets")
echo "$res" | jq -e '.data.items | length > 0' > /dev/null && ok "creator dashboard has the new market" || fail "me/markets: $res"

step "Community leaderboard"
res=$(req "$BASE/communities")
echo "$res" | jq -e '.data.items | length > 0' > /dev/null && ok "leaderboard" || fail "communities: $res"

echo
if [ "$FAIL" -eq 0 ]; then
  echo "✓ All smoke checks passed."
else
  echo "✗ Some smoke checks failed — see above."
  exit 1
fi
