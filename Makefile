.PHONY: deploy update logs status restart build down fmt-go

# Форматирование Go (встроенный gofmt; импорты — см. goimports в ответе README)
fmt-go:
	cd vpn-bot && gofmt -s -w .

# First-time setup on VPS — run once after git clone
setup:
	cp .env.example .env.local
	@echo "Fill in .env.local, then run: make up"

# Start all services
up:
	mkdir -p hiddify mtproxy/data mtproxy/logs vpn-bot/data
	docker compose up -d

# Rebuild and restart bot only (after code change)
deploy:
	git pull origin main
	docker compose up -d --build vpn-bot

# Full rebuild of all services
build:
	docker compose build

# Stop all services
down:
	docker compose down

# Status
status:
	docker compose ps

# Logs
logs:
	docker compose logs -f

logs-bot:
	docker compose logs -f vpn-bot

logs-hiddify:
	docker compose logs -f hiddify

logs-mtproxy:
	docker compose logs -f mtproxy

# Restart individual services
restart-bot:
	docker compose restart vpn-bot

restart-all:
	docker compose restart
