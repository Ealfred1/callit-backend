#!/bin/bash
# Repeatable deploy: pull latest, build, migrate, restart via pm2.
#
# Not systemd: `eric` has no passwordless sudo on this shared VPS (confirmed
# during setup — see docs/NOTES.md), so a non-interactive `sudo systemctl
# restart` would just hang waiting for a password. pm2 (already installed,
# already used on this box) runs and supervises the binary entirely under
# the `eric` user, no sudo required. First-time server provisioning
# (Postgres container + pm2 process registration) is deploy/setup-server.sh,
# run once before this.
set -e

VPS="eric"
REMOTE_DIR="~/callit-backend"

echo "→ Deploying to $VPS..."
ssh "$VPS" "
  set -e
  # non-interactive SSH doesn't source ~/.zshrc, so pm2 (installed via
  # pnpm) isn't on PATH by default — add it explicitly.
  export PATH=\"\$HOME/.local/share/pnpm:\$PATH\"
  if [ ! -d $REMOTE_DIR/.git ]; then
    git clone git@github.com:Ealfred1/callit-backend.git $REMOTE_DIR
  fi
  cd $REMOTE_DIR
  git pull

  echo '→ Building...'
  go build -o bin/callit-api ./cmd/api
  go build -o bin/migrate ./cmd/migrate

  echo '→ Running migrations...'
  set -a; source .env; set +a
  ./bin/migrate

  echo '→ Restarting via pm2...'
  pm2 restart callit-api --update-env || pm2 start bin/callit-api --name callit-api
  pm2 save
"

echo "✓ Deployed callit-backend"
