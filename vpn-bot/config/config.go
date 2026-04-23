package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken          string
	AdminIDs          map[int64]struct{}
	DBPath            string
	HiddifyDomain     string
	HiddifyAdminPath  string
	HiddifyClientPath string
	HiddifyKey        string
	UserPackageDays   int
	UserUsageLimitGB  int
	WebhookURL        string // e.g. https://bot.arengate.tech
	WebhookListen     string // e.g. :8080
}

func Load() Config {
	token := strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	dbPath := strings.TrimSpace(os.Getenv("DB_PATH"))
	if dbPath == "" {
		dbPath = "./data/database.db"
	}
	packageDays := 90
	if s := strings.TrimSpace(os.Getenv("USER_PACKAGE_DAYS")); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			packageDays = v
		}
	}
	usageLimitGB := 1000
	if s := strings.TrimSpace(os.Getenv("USER_USAGE_LIMIT_GB")); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			usageLimitGB = v
		}
	}
	webhookListen := strings.TrimSpace(os.Getenv("WEBHOOK_LISTEN"))
	if webhookListen == "" {
		webhookListen = ":8080"
	}
	return Config{
		BotToken:          token,
		AdminIDs:          parseAdminIDs(os.Getenv("ADMIN_IDS")),
		DBPath:            dbPath,
		HiddifyDomain:     strings.TrimSpace(os.Getenv("HIDDIFY_DOMAIN")),
		HiddifyAdminPath:  strings.TrimSpace(os.Getenv("HIDDIFY_ADMIN_PATH")),
		HiddifyClientPath: strings.TrimSpace(os.Getenv("HIDDIFY_CLIENT_PATH")),
		HiddifyKey:        strings.TrimSpace(os.Getenv("HIDDIFY_API_KEY")),
		UserPackageDays:   packageDays,
		UserUsageLimitGB:  usageLimitGB,
		WebhookURL:        strings.TrimSpace(os.Getenv("WEBHOOK_URL")),
		WebhookListen:     webhookListen,
	}
}

func parseAdminIDs(raw string) map[int64]struct{} {
	out := make(map[int64]struct{})
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			continue
		}
		out[id] = struct{}{}
	}
	return out
}

func (c Config) IsAdmin(telegramID int64) bool {
	_, ok := c.AdminIDs[telegramID]
	return ok
}
