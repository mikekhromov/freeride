#!/bin/bash
set -euo pipefail

# First-time VPS setup for vpn-system
# Run as root on Ubuntu 22.04

echo "=== [1/5] Update system ==="
apt update && apt upgrade -y

echo "=== [2/5] Install Docker ==="
if ! command -v docker &>/dev/null; then
  curl -fsSL https://get.docker.com | sh
else
  echo "Docker already installed, skipping."
fi

echo "=== [3/5] Clone repo ==="
REPO="${REPO:-https://github.com/YOUR_USER/YOUR_REPO.git}"
DEST="${DEST:-$HOME/vpn}"

if [ -d "$DEST/.git" ]; then
  echo "Repo already cloned at $DEST, pulling latest..."
  git -C "$DEST" pull origin main
else
  git clone "$REPO" "$DEST"
fi

cd "$DEST"

echo "=== [4/5] Create .env.local ==="
if [ ! -f .env.local ]; then
  cp .env.example .env.local
  echo ""
  echo ">>> Fill in .env.local before continuing:"
  echo "    nano $DEST/.env.local"
  echo ""
  echo "Then run: cd $DEST && make up"
  exit 0
else
  echo ".env.local already exists, skipping."
fi

echo "=== [5/5] Create data directories and start ==="
mkdir -p hiddify mtproxy/data mtproxy/logs vpn-bot/data
docker compose up -d

echo ""
echo "=== Done! ==="
echo "Check status: docker compose ps"
echo "Logs:         docker compose logs -f"
