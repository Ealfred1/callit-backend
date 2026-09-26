# Handoff — Callit backend

Written 2026-09-26 for whoever (human or agent) picks this up next. Read
this first; `docs/NOTES.md` has the full chronological build log with every
bug found and why each decision was made, if you need more depth than
what's here.

## Do this first, before anything else

1. **Check whether the public URL is still alive** — it isn't guaranteed to
   be (see "The one fragile thing" below):
   ```
   ssh eric 'pm2 logs callit-tunnel --lines 60 --nostream | grep trycloudflare.com | tail -3'
   ```
   Compare the URL in that output to `PUBLIC_URL` in `~/callit-backend/.env`
   on the VPS. If they don't match, the tunnel restarted since and every
   share-card link + the Flutter app's connection is currently broken.
   Fix: `sed -i "s#^PUBLIC_URL=.*#PUBLIC_URL=<current-url>#" .env && pm2 restart callit-api && pm2 save` (no `--update-env`, see the pm2 gotcha below).
2. **Run the smoke test** to confirm the fix / current state:
   ```
   ssh eric 'cd ~/callit-backend && BASE_URL=http://127.0.0.1:8095 PSQL_EXEC="docker exec -i callit-postgres psql -U callit -d callit_db" bash scripts/smoke.sh'
   ```
   All 24 checks should pass. If they don't, something regressed — don't
   assume it's fine.
3. **Fix the Flutter app's `ApiConfig.baseUrl`** (not done yet — flagging,
   not fixing, since it's a teammate's file in a different repo):
   `/Users/eric/callit/callitApp/lib/core/api/api_config.dart` is hardcoded
   to `http://quirky-comet.62-171-156-198.sslip.io:8095/v1`, which is
   **not reachable** — that port is firewalled externally (see below). It
   needs to point at whatever URL step 1 confirmed is live (the
   `https://*.trycloudflare.com` one), or the app can't reach the backend
   at all. This was true as of this handoff and is exactly the kind of
   thing that silently breaks between sessions — verify it's still wrong
   before assuming it needs fixing again.

## What this is

Callit: paste a headline → AI compiles it into a Panta (Solana prediction
market) draft → creator reviews/signs → live market with a share card.
Backend proxies every Panta call and holds the Panta/Anthropic API keys
server-side (the Flutter app never talks to Panta directly). Built for the
Panta API Side Track at Colosseum's Crypto World's Fair, submission Oct 12,
2026 — this is a hackathon project on a real deadline.

- **Full PRD**: `/Users/eric/Downloads/Callit — Product Requirements Document (1).pdf`
- **This repo**: `/Users/eric/callit/callit-backend`, pushed to
  `github.com/Ealfred1/callit-backend` (private)
- **Frontend**: `/Users/eric/callit/callitApp` — Flutter, was a fully
  mocked UI when this backend work started; a teammate (Gideon Jones,
  git author `gideonjones63@gmail.com`) has since wired a real API client
  (`lib/core/api/`) that matches this backend's contract closely — check
  their latest commits (`cd ../callitApp && git log --oneline -10`) for
  what's changed since this doc was written.
- **The approved build plan** (context on *why* things are shaped this
  way): `/Users/eric/.claude/plans/quiet-mapping-haven.md`

## Stack & where things live

Go + `echo` + `pgx` (raw SQL, no ORM), hexagonal layout
(`internal/core/{domain,ports}`, `internal/service`, `internal/adapter/*`),
mirroring the user's other projects `~/mollah` (mollah-backend) and
`~/flowvent-tracker` — same conventions (godotenv config, JWT+bcrypt auth,
log/slog, pm2/systemd process supervision). If a pattern here looks
unusual, check whether one of those two repos already solves it the same
way before inventing something new.

- **Deploy target**: `eric` VPS (SSH alias, `62.171.156.198`) — a **shared
  box**, not dedicated to this project. 4 user accounts, ~50 containers
  from other live client projects (immiro, quickleen, savestack, edey,
  xyzbot, etc). Disk was at 94% (6.4GB free) as of this handoff and has
  been *trending up* since setup (92% → 93% → 94% over 5 days) — check
  `ssh eric 'df -h /; free -h'` before adding anything new, and don't add
  Docker images casually.
- **No sudo**: `eric` has no passwordless sudo, and system nginx (which
  owns ports 80/443) is configured by a different user (`ojogu`). Confirmed
  by testing (not assumption): **only 80/443 pass this VPS's firewall for
  every service on the box**, not just ours — every other port, ours
  included, times out from outside.
- **Process supervision**: pm2, not systemd (systemd restart would need
  sudo). Two pm2 processes: `callit-api` (the Go binary) and
  `callit-tunnel` (`cloudflared`, see below). `ssh eric 'pm2 list'` to check.
- **Postgres**: Docker container `callit-postgres`
  (`postgres:16-alpine`), bound to `127.0.0.1:5434` only — never exposed
  publicly. Named volume `callit_pgdata`.
- **Public access**: a `cloudflared` "quick tunnel" (no Cloudflare account),
  since the firewall blocks everything except 80/443 and we don't have
  root to add an nginx vhost there. This is the one fragile piece — see
  next section.

## The one fragile thing: the public URL is not permanent

Quick tunnels get a **new random `*.trycloudflare.com` hostname every time
`cloudflared` restarts** — crash, `pm2 restart`, VPS reboot, all of it.
When that happens: `.env`'s `PUBLIC_URL` goes stale, every previously
shared card link breaks, and the Flutter app (whichever URL it's pointed
at) can't reach the server. This **already happened once** between the
initial setup and this handoff (2 days of uptime before it cycled) — it
will keep happening. Step 1 at the top of this doc is how to check/fix it
each time you pick this project back up.

Two ways to make this permanent instead of babysitting it, in order of
effort:
1. **Free Cloudflare account** + `cloudflared tunnel login` → a *named*
   tunnel gets a permanent hostname, no root needed on the VPS at all.
   Probably the fastest real fix.
2. **Get root** (from `ojogu` or the user's own sudo password) to add one
   nginx vhost for `quirky-comet.62-171-156-198.sslip.io` →
   `127.0.0.1:8095`, then `certbot --nginx -d quirky-comet.62-171-156-198.sslip.io`.
   This is the "intended" long-term address (the user specifically asked
   for a sslip.io hostname) and needs no port ever opened publicly. Exact
   commands are in `docs/NOTES.md`.

## What's built and verified working

All PRD must-haves are implemented and passing an end-to-end smoke test
(`scripts/smoke.sh` — register → OTP verify → login → AI compile →
community create → market create quote/build/register → share-card render
→ buy quote/build/submit/verify → trade report → positions → win-claim
build → creator-fee-claim build → creator dashboard → community
leaderboard, plus a 401-envelope regression check). Endpoint map,
request/response examples: `docs/openapi.yaml`, served at `/docs` (Scalar,
grouped into tagged folders).

**Everything is currently running against mocks**, not real external
services — check `~/callit-backend/.env` on the VPS:
- `PANTA_MOCK=true` — no real `PANTA_API_KEY` yet. The mock Panta client
  (`internal/adapter/panta/mock.go`) is a full in-memory fixture of the
  entire create/buy/claim pipeline, not just canned responses.
- `AI_MOCK=true` — no real `ANTHROPIC_API_KEY` yet. Mock compiler
  (`internal/adapter/ai/mock.go`) is structurally valid but not
  intelligent (doesn't actually judge ambiguity beyond a length check).
- `RESEND_API_KEY` unset — signup OTP codes are logged to stdout
  (`pm2 logs callit-api`), not emailed.

Flip any of these to real once the corresponding key exists (`.env.example`
has exactly how to obtain each one), then `pm2 restart callit-api` (not
`--update-env`, see below) on the VPS.

## Gotchas already hit and fixed (don't re-break these)

- **`godotenv.Load()` → `Overload()`** in `internal/config/config.go`.
  pm2 caches a process's OS environment from whenever it was first
  started, and reuses it across `pm2 restart` regardless of
  `--update-env` unless the *invoking shell* has the new value exported.
  `Load()` never overrides an already-set OS var, so editing `.env` and
  restarting silently did nothing for days. `Overload()` makes the file
  always win. If you ever see an env change on the server not taking
  effect, this is why — try `pm2 delete callit-api && pm2 start bin/callit-api --name callit-api`
  (a real fresh start, not `restart`) to purge pm2's cached copy entirely.
- **Mock Panta client IDs seeded from `time.Now().UnixNano()`**, not 0
  (`internal/adapter/panta/mock.go`). `create_sessions`/`market_meta` in
  Postgres persist across restarts; the mock's ID sequence didn't, so a
  fresh process's `cr_mock000001` collided with the same primary key still
  in the DB from before.
- **Unauthorized requests must go through `respond.FromError`**, never
  return a bare `*AppError` from a handler/middleware — it doesn't write a
  response, so it fell through to Echo's default error handler and came
  back as a generic 500 instead of the unified 401 envelope. Also why
  `e.HTTPErrorHandler = respond.GlobalErrorHandler` is set in
  `cmd/api/main.go`, as a blanket safety net.
- Both `/claims/build/` and `/claims/creator-fees/` Panta paths were
  **inferred** from the docs' URL slugs, not seen as a literal example —
  verify against a real `PANTA_API_KEY` the first time either is called
  for real.

## How to deploy a change

```
cd /Users/eric/callit/callit-backend
# make your change, commit, push
git push
./deploy.sh   # ssh's in, git pulls, rebuilds both binaries, runs migrations, pm2 restarts
```
First-time-only server setup (already done, only needed if starting fresh
on a new box) is `deploy/setup-server.sh`.

**Do not** run Postgres or any Docker container on the local Mac for this
project — the user was explicit about that. Test against the deployed VPS
instance instead (`ssh eric` + the smoke test), same as this whole project
has been built and verified.

## Commit convention

No `Co-Authored-By: Claude` or any AI-attribution line in commits, ever —
explicit standing user preference (saved in memory:
`feedback_no_claude_attribution.md`). Commit "bit by bit" — small, focused
commits with a clear why, pushed as you go, not one giant dump at the end.

## Open items (not blocking, need the user's input/access)

- Real `PANTA_API_KEY` (`pk_test_`), `ANTHROPIC_API_KEY`, `RESEND_API_KEY`
  — `.env.example` has exact steps to obtain each.
  - Confirm the two inferred Panta claim-endpoint paths against a real call.
  - Confirm `primaryFeeBps` (currently a configurable guess, `PANTA_PRIMARY_FEE_BPS=200`) against the live `MarketConfig` once real trades exist.
- Permanent public URL — pick one of the two options above.
- `pm2 startup` (one-time, interactive, needs sudo) so both pm2 processes
  survive a full VPS reboot — not done; right now a reboot loses everything
  until someone manually `pm2 resurrect`s or restarts them.
- Should-have PRD features already have basic support (breaking markets =
  the same create endpoint with `marketType:"breaking"`, community
  leaderboard, creator dashboard) but haven't been exercised with real
  Panta data yet, only mocks.
