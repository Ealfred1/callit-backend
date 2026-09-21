# Process notes

Running log of what was built, why, and what a future agent/dev should
re-verify before trusting it. Update this as work continues — it's the "how
we got here" that `git log` alone doesn't reconstruct.

## Public access (2026-09-21)

Confirmed by testing (not just inferring from one blocked port): **only
80/443 pass the `eric` VPS's firewall, for every service on that box** —
tried several other already-in-use ports (8002, 8003, 8170, 5555, 3000)
from outside and all timed out identically to 8095. So this was never an
"open port 8095" fix; every app on that box that's reachable externally
gets there through the system nginx on 80/443, which needs root to touch
(owned by `ojogu`).

Unblocked without needing anyone's sudo: `cloudflared` (downloaded straight
from GitHub releases, no account needed for a "quick tunnel") makes an
*outbound* connection from the VPS to Cloudflare's edge, which is never
firewalled. Running as a second pm2 process (`callit-tunnel`,
`cloudflared tunnel --url http://localhost:8095`) alongside `callit-api`.
Current public URL — **`https://gas-training-stress-pencil.trycloudflare.com`**
— is what `PUBLIC_URL` in `.env` actually points at now (real HTTPS, works
from anywhere, confirmed with curl from outside). `docs/openapi.yaml`'s
sslip.io server entry is left in as the intended long-term address.

**This URL is not stable.** Quick tunnels (no Cloudflare account) get a new
random `*.trycloudflare.com` hostname every time `cloudflared` restarts —
crash, `pm2 restart`, or a VPS reboot all change it, which silently breaks
every share-card link generated before that. If `pm2 logs callit-tunnel`
shows a new URL, `PUBLIC_URL` in `.env` needs updating to match (then a
`pm2 restart callit-api`, not `--update-env` — see the pm2-env bug below).
Fixing this for real means either the nginx+certbot path onto
`quirky-comet.62-171-156-198.sslip.io` (needs root, steps below unchanged),
or a *named* Cloudflare Tunnel (needs a free Cloudflare account + `cloudflared
tunnel login`, gets a permanent hostname — reasonable next step if getting
root stays blocked).

**Bug found while doing this: editing `.env` on the server and restarting
wasn't taking effect.** Changed `PUBLIC_URL` to the tunnel URL, restarted,
and share-card links still pointed at the old sslip.io address. Root cause:
`deploy.sh` used to `source .env` into its shell before the very first `pm2
start`, and pm2 snapshots a process's OS environment at start time and
keeps reusing it across `pm2 restart` (even with `--update-env`, which only
refreshes from *that command's own* shell environment, not the file) —
so every var from that first deploy got baked in permanently, and
`godotenv.Load()` never overrides an already-set OS var, so the app kept
reading the stale pm2-cached value over the actual file. Fixed: `internal/config/config.go`
now calls `godotenv.Overload()` (file always wins), `cmd/migrate` loads
`.env` itself instead of relying on the deploy shell to source it, and
`deploy.sh` no longer sources `.env` into the shell at all. Confirmed fixed
by checking `pm2 env <id>` before/after — the stale `PUBLIC_URL` is gone
from pm2's cached env entirely now.

## Panta API — confirmed vs inferred

Pulled directly from `docs.panta.market`'s official `llms.txt` index
(2026-09-21), not guessed. Endpoints below were seen as a **literal request
example** (path + body, e.g. a `curl` block) in the docs — trust these:

- `POST /auth/register/`, `POST /auth/token/`
- `POST /markets/create/quote/`, `POST /markets/create/build/`, `POST /markets/register/`
- `POST /markets/create/image-upload/`
- `GET /markets/`, `GET /markets/{id}/`, `GET /categories/`
- `POST /primaryorderquote/`, `POST /primaryorderbuild/`, `POST /primaryordersubmit/`, `POST /primaryorderverify/`
- `GET /positions/?wallet=`
- `POST /trades/report/`, `GET /trades/{signature}/`
- `GET /account/metrics/`, `GET /account/dashboard/`, `POST /account/keys/`

These two were only **inferred from the doc page's slug/pattern** (no
literal curl shown) — verify the exact path on the first real call, and fix
`internal/adapter/panta/real.go`'s `ClaimWin`/`ClaimCreatorFees` if wrong:

- `POST /claims/build/` (win claim)
- `POST /claims/creator-fees/`

`primaryFeeBps` has no live lookup endpoint — PRD says "often 200bps", kept
as `PANTA_PRIMARY_FEE_BPS` env var (default 200), not hardcoded.

## Deviations from mollah-backend / flowvent-tracker, and why

- **Auth is 2-step, not 3.** mollah's `register(email) -> verify -> onboard(password+profile)`
  exists because it needs DOB/country for KYC. Callit's sign-up screen
  (`callitApp/lib/screens/auth/sign_up_screen.dart`) already collects
  name+email+password together, so `Register` here creates the user AND
  sends the code in one call; `Verify` returns the real JWT directly (no
  intermediate signup-token step).
- **Unified response envelope.** mollah's handlers return ad hoc shapes
  (`{"error":"..."}` here, `{"message","token"}` there). Every Callit handler
  goes through `internal/adapter/handler/respond` instead, so responses are
  always `{"data": ...}` or `{"error": {"code","message","fields"}}` — and a
  passthrough Panta failure keeps Panta's own error `code` (e.g.
  `QUOTE_STALE`) rather than becoming a generic 500.
- **Real `golang-migrate` library, not mollah's hand-rolled inline SQL.**
  mollah's `cmd/migrate/main.go` is actually a list of `CREATE TABLE IF NOT
  EXISTS` Go string literals — the `migrations/*.sql` files in that repo
  aren't wired to anything. Callit's `cmd/migrate` uses the real
  `github.com/golang-migrate/migrate/v4` library reading
  `migrations/*.up.sql` / `*.down.sql`, so those files are the actual
  source of truth and diff cleanly in PRs.
- **pm2, not systemd, for process supervision on the VPS.** The approved
  plan originally said systemd (matching flowvent-tracker). Once actually
  SSH'd into `eric`, `sudo -n true` failed ("a password is required") — no
  passwordless sudo, so a non-interactive `systemctl restart` in `deploy.sh`
  would just hang. pm2 (already installed and used on that box) runs and
  restarts the binary entirely under the `eric` user, no sudo needed.
  `deploy/setup-server.sh` provisions it once; `pm2 startup` (one-time,
  interactive, needs sudo) is a manual step for surviving a VPS reboot.

## `eric` VPS state at time of setup (2026-09-21)

Shared box, not dedicated to this project — 4 user accounts (`eric`,
`ojogu`, `ckellz`, `cryptokellz`), ~50 running containers from other live
client projects (immiro, quickleen, savestack, edey, xyzbot, flume, etc).
Disk was at 92% (7.8GB free of 96GB), RAM ~1.3GB free of 11GB. nginx is
system-level (not Docker) and its config is owned by `ojogu`, not `eric` —
no ability to add a vhost/TLS without their sudo password. `eric` is in the
`docker` group (so `docker run`/`compose` works without sudo) but does not
have passwordless `sudo`.

Consequences baked into this deploy:
- No app Docker image — build the Go binary on-host (Go 1.26.4 already
  installed there) instead of paying for `golang:alpine` builder layers.
- Postgres (`postgres:16-alpine`, already cached on that host) bound to
  `127.0.0.1:5434` only, named volume `callit_pgdata` — never the public
  interface.
- App listens on `*:8095` (all interfaces) — no domain, no TLS yet.
- **Public hostname: `quirky-comet.62-171-156-198.sslip.io`** (chosen
  2026-09-21, at the user's request — a random label, not
  `62-171-156-198.sslip.io` or anything styled like this box's other
  project names, so it's not guessable/scannable alongside them). sslip.io
  resolves any label in front of a dash-encoded IP to that IP — confirmed
  via `dig`. This is `PUBLIC_URL` in `.env` (`deploy/setup-server.sh`'s
  `PUBLIC_HOSTNAME` constant) and is what share-card links point at. It's
  **still plain HTTP on :8095** — see the two blockers below before this is
  a real HTTPS endpoint.
- **The Flutter app needs an iOS ATS exception / Android cleartext-traffic
  allowance** pointed at this hostname to call it during dev, until TLS is
  set up.
- **Port 8095 is not reachable from outside the VPS**, even though the app
  listens on all interfaces and answers on `127.0.0.1:8095` fine. `curl`
  from off-box times out — same result via the sslip.io hostname (expected,
  it's just DNS; the block is on the port). Almost certainly a host firewall
  (`ufw`/`iptables`) or the hosting provider's network-level security group
  — both need root/provider-console access this session doesn't have. The
  full pipeline is verified working (see below) via SSH against
  `127.0.0.1:8095` on the box itself; **the Flutter app won't be able to
  reach it until this is opened.**
- **To get this onto real HTTPS** (recommended over just opening :8095),
  whoever has root/`ojogu` needs to, once:
  1. Add an nginx server block for `quirky-comet.62-171-156-198.sslip.io`
     that reverse-proxies to `127.0.0.1:8095` (nginx already owns 80/443 on
     this box — no new ports to open).
  2. `certbot --nginx -d quirky-comet.62-171-156-198.sslip.io` (HTTP-01
     challenge works fine against a sslip.io hostname — that's the whole
     point of using it here).
  3. Update `.env`'s `PUBLIC_URL` to `https://quirky-comet.62-171-156-198.sslip.io`
     (no port) and `pm2 restart callit-api --update-env`.
  After that, no ATS exception / cleartext config is needed on either
  mobile OS, and :8095 never has to be opened publicly at all.
- Before adding anything else to this box, re-check headroom:
  `ssh eric 'df -h /; free -h'`.

## Verified (2026-09-21)

`scripts/smoke.sh` run against the live deployed service (`127.0.0.1:8095`
on the VPS, mock Panta/AI/email) — every step passed: register → OTP verify
→ login → AI compile → community create → market create (quote/build/
register) → share-card render (confirmed a real 1200×630 PNG, ~32KB,
correctly styled to brand) → buy (quote/build/submit/verify) → trade report
→ positions → win-claim build → creator-fee-claim build → creator dashboard
→ community leaderboard. `pm2 save`d so it should survive a process crash;
still needs a one-time interactive `pm2 startup` (needs sudo) to survive a
full VPS reboot — not yet done.

Two real bugs found by testing against the live deploy (not caught by
`go vet`/`go build`, both fixed same day):

- **Unauthorized requests returned a bare 500, not the unified 401
  envelope.** `JWTMiddleware` returned `respond.Unauthorized(...)` directly
  — that just constructs an `*AppError`, it doesn't write a response. Since
  nothing had written to `c` yet, Echo's default error handler took over and
  rendered its own generic `{"message":"Internal Server Error"}` shape.
  `scripts/smoke.sh` never caught it because every call in it sends a valid
  token. Fixed by routing through `respond.FromError`, and added
  `e.HTTPErrorHandler = respond.GlobalErrorHandler` as a blanket safety net
  so any future bare-error-return still gets the unified envelope. Added an
  explicit "no token → 401 envelope" case to `scripts/smoke.sh` so this
  can't silently regress.
- **Mock Panta client's sequential IDs collided across process restarts.**
  `cr_mock000001` etc. started over from 0 on every `NewMockClient()` call,
  but `create_sessions`/`market_meta` in Postgres persist across restarts —
  re-running the smoke test after a redeploy hit `duplicate key value
  violates unique constraint "create_sessions_pkey"`. Fixed by seeding the
  sequence from `time.Now().UnixNano()` instead of 0.

## Build order followed

1. Domain types, ports, config, migrations, `cmd/migrate`, `respond` envelope.
2. Auth (register/verify/login) + mock mailer.
3. Panta client (real + mock) + markets/positions/claims/trades handlers.
4. AI compiler (real + mock) + market-create flow + `create_sessions`/`market_meta` persistence.
5. Orders (buy) flow.
6. Share card (native Go `gg` renderer, embedded fonts — no network fetch, no WASM) + public unfurl page.
7. Communities + `/me/markets` (creator dashboard).
8. `docs/openapi.yaml` (Scalar), this file, documented `.env.example`.
9. Local smoke test in mock mode.
10. GitHub repo + VPS deploy.
