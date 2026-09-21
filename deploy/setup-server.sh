#!/bin/bash
# One-time (idempotent) provisioning on the `eric` VPS: the Postgres
# container and the pm2 process entry. Run this once before the first
# deploy.sh; safe to re-run.
#
# Deliberately minimal footprint — see docs/NOTES.md for why: this is a
# shared box (4 users, ~50 containers from other live projects) at 92% disk
# and ~1.3GB free RAM when this was written. reuses the postgres:16-alpine
# image already cached on the host (quickleen_db uses the same tag) instead
# of pulling a fresh one, and binds Postgres to 127.0.0.1 only — never the
# public interface.
set -e
export PATH="$HOME/.local/share/pnpm:$PATH" # non-interactive shells don't source ~/.zshrc

CONTAINER_NAME=callit-postgres
VOLUME_NAME=callit_pgdata
PG_PORT=5434

# A deliberately random, non-obvious sslip.io hostname (not
# "62-171-156-198.sslip.io", which is guessable/scannable, and not styled
# like this box's other project names) resolving to this VPS's IPv4 — so we
# have a real hostname for TLS later without needing our own DNS. sslip.io
# resolves ANY label in front of a dash-encoded IP to that IP; the label
# itself is arbitrary. See docs/NOTES.md for why it's still plain HTTP today
# (port 80/443 are already owned by system nginx, which we can't configure
# without sudo).
PUBLIC_HOSTNAME=quirky-comet.62-171-156-198.sslip.io

if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  echo "→ Starting Postgres container on 127.0.0.1:${PG_PORT}..."
  docker run -d \
    --name "$CONTAINER_NAME" \
    --restart unless-stopped \
    -e POSTGRES_USER=callit \
    -e POSTGRES_PASSWORD=secret \
    -e POSTGRES_DB=callit_db \
    -p 127.0.0.1:${PG_PORT}:5432 \
    -v ${VOLUME_NAME}:/var/lib/postgresql/data \
    postgres:16-alpine
else
  echo "✓ Postgres container already running"
fi

if [ ! -f ~/callit-backend/.env ]; then
  echo "→ .env missing — copying .env.example. Fill in real secrets before the first real deploy (PANTA_MOCK/AI_MOCK can stay true until then)."
  cp ~/callit-backend/.env.example ~/callit-backend/.env
  sed -i "s#^DATABASE_URL=.*#DATABASE_URL=postgres://callit:secret@127.0.0.1:${PG_PORT}/callit_db?sslmode=disable#" ~/callit-backend/.env
  sed -i "s#^PUBLIC_URL=.*#PUBLIC_URL=http://${PUBLIC_HOSTNAME}:8095#" ~/callit-backend/.env
fi

echo "→ pm2 process..."
cd ~/callit-backend
if pm2 describe callit-api > /dev/null 2>&1; then
  echo "✓ pm2 process already registered — deploy.sh will restart it"
else
  mkdir -p bin
  echo "  (no binary yet — deploy.sh's first run will build + pm2 start it)"
fi

echo "✓ Server provisioned. Run ./deploy.sh next."
echo "  Note: run 'pm2 startup' once (needs sudo, interactive) + 'pm2 save' so callit-api survives a VPS reboot."
