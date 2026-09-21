# Process notes

Running log of what was built, why, and what a future agent/dev should
re-verify before trusting it. Update this as work continues — it's the "how
we got here" that `git log` alone doesn't reconstruct.

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
- App listens on `127.0.0.1:8095` (checked free via `ss -tlnp` at setup
  time) — no domain, no TLS yet. **The Flutter app needs an iOS ATS
  exception / Android cleartext-traffic allowance** pointed at
  `http://62.171.156.198:8095` to call it during dev. Real fix (domain +
  certbot) needs `ojogu` or the user's own sudo password later.
- Before adding anything else to this box, re-check headroom:
  `ssh eric 'df -h /; free -h'`.

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
