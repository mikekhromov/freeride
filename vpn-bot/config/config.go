package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BotToken          string
	AdminIDs          map[int64]struct{}
	DBPath            string
	HiddifyURL        string
	HiddifyKey        string
	MtproxyData       string
	MtproxyLog        string
	MtproxyPublicHost string
	MtproxyPublicPort int
	MonitorInterval   time.Duration
	WarningTTL        time.Duration
	DeviceLimit       int
}

func Load() Config {
	token := strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	dbPath := strings.TrimSpace(os.Getenv("DB_PATH"))
	if dbPath == "" {
		dbPath = "./data/database.db"
	}
	port := 8443
	if p := strings.TrimSpace(os.Getenv("MTPROXY_PUBLIC_PORT")); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}
	mon := 10 * time.Minute
	if s := strings.TrimSpace(os.Getenv("MONITOR_INTERVAL")); s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			mon = d
		}
	}
	warn := 24 * time.Hour
	if s := strings.TrimSpace(os.Getenv("WARNING_TTL")); s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			warn = d
		}
	}
	dev := 5
	if s := strings.TrimSpace(os.Getenv("DEVICE_LIMIT")); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			dev = v
		}
	}
	return Config{
		BotToken:          token,
		AdminIDs:          parseAdminIDs(os.Getenv("ADMIN_IDS")),
		DBPath:            dbPath,
		HiddifyURL:        strings.TrimSpace(os.Getenv("HIDDIFY_URL")),
		HiddifyKey:        strings.TrimSpace(os.Getenv("HIDDIFY_API_KEY")),
		MtproxyData:       strings.TrimSpace(os.Getenv("MTPROXY_CONFIG")),
		MtproxyLog:        strings.TrimSpace(os.Getenv("MTPROXY_LOG")),
		MtproxyPublicHost: strings.TrimSpace(os.Getenv("MTPROXY_PUBLIC_HOST")),
		MtproxyPublicPort: port,
		MonitorInterval:   mon,
		WarningTTL:        warn,
		DeviceLimit:       dev,
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
