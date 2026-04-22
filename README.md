# Arengate VPN Bot

Telegram bot for managing user access in Hiddify Manager v2 API.

## Architecture

- Hiddify Manager runs directly on VPS and provides VPN + MTProxy links.
- `vpn-bot` is a Go binary managed by `systemd`.
- SQLite stores only Telegram users and `hiddify_uuid` mapping.

## Environment

Copy `.env.example` to `.env.local` and fill values:

- `BOT_TOKEN`
- `ADMIN_IDS` (comma-separated Telegram IDs)
- `HIDDIFY_DOMAIN`
- `HIDDIFY_ADMIN_PATH`
- `HIDDIFY_CLIENT_PATH`
- `HIDDIFY_API_KEY`
- `USER_PACKAGE_DAYS`
- `USER_USAGE_LIMIT_GB`
- `DB_PATH`

## Bot Commands

User commands:

- `/start`
- `/status`

Admin commands:

- `/users`
- `/approve @username`
- `/revoke @username`
- `/stats`

## Local Build

```bash
cd vpn-bot
go build -o vpn-bot .
```

## Deploy (systemd)

1. Build binary:

```bash
cd /home/vpnbot/vpn-bot
go build -o vpn-bot .
sudo cp vpn-bot /usr/local/bin/vpn-bot
```

2. Install unit file:

```bash
sudo cp vpn-bot.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable vpn-bot
sudo systemctl restart vpn-bot
```

3. Check status:

```bash
systemctl status vpn-bot
journalctl -u vpn-bot -f
```
