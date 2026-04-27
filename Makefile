.PHONY: deploy deploy-bot deploy-landing install-hooks fmt-go logs status restart

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
	cd $(BOT_SRC) && go build -o $(BOT_BIN) .
	systemctl restart vpn-bot
	@echo "Bot deployed and restarted"

# Deploy landing only
deploy-landing:
	cd $(LAND_DIR) && npm run build
	@echo "Landing deployed"

# Install go.arengate.tech HAProxy routing + systemd watcher
install-hooks:
	cp $(REPO_DIR)/scripts/go-landing.cfg /opt/hiddify-manager/haproxy/go-landing.cfg
	cp $(REPO_DIR)/scripts/go-landing-watch.path /etc/systemd/system/
	cp $(REPO_DIR)/scripts/go-landing-watch.service /etc/systemd/system/
	systemctl daemon-reload
	systemctl enable --now go-landing-watch.path
	bash $(REPO_DIR)/scripts/restore-go-landing.sh
	@echo "go.arengate.tech routing installed and watching"

# Logs
logs:
	journalctl -u vpn-bot -f

status:
	systemctl status vpn-bot --no-pager -n 20

restart:
	systemctl restart vpn-bot
