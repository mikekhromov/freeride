.PHONY: deploy deploy-bot deploy-landing fmt-go logs status restart

REPO_DIR := /root/vpn
BOT_SRC  := $(REPO_DIR)/vpn-bot
BOT_BIN  := /usr/local/bin/vpn-bot
LAND_DIR := $(REPO_DIR)/arengate-landing

fmt-go:
	cd vpn-bot && gofmt -s -w .

# Deploy everything
deploy:
	git pull origin main
	$(MAKE) deploy-bot
	$(MAKE) deploy-landing

# Deploy bot only
deploy-bot:
	go build -o $(BOT_BIN) ./vpn-bot/
	systemctl restart vpn-bot
	@echo "Bot deployed and restarted"

# Deploy landing only
deploy-landing:
	cd $(LAND_DIR) && npm run build
	@echo "Landing deployed"

# Logs
logs:
	journalctl -u vpn-bot -f

status:
	systemctl status vpn-bot --no-pager -n 20

restart:
	systemctl restart vpn-bot
