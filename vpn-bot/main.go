package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"freeride/vpn-bot/bot/handlers"
	"freeride/vpn-bot/config"
	"freeride/vpn-bot/db"
	"freeride/vpn-bot/services/approve"
	"freeride/vpn-bot/services/hiddify"
	"freeride/vpn-bot/services/revoke"
	"freeride/vpn-bot/store"

	tb "gopkg.in/telebot.v3"
)

func main() {
	cfg := config.Load()
	if cfg.BotToken == "" {
		log.Fatal("BOT_TOKEN is required")
	}
	if len(cfg.AdminIDs) == 0 {
		log.Fatal("ADMIN_IDS is required (comma-separated Telegram user ids)")
	}

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		log.Fatal(err)
	}
	sqlDB, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	st := &store.Store{DB: sqlDB}
	h := hiddify.New(cfg.HiddifyDomain, cfg.HiddifyAdminPath, cfg.HiddifyClientPath, cfg.HiddifyKey)

	approveSvc := &approve.Service{Store: st, Hiddify: h, Cfg: cfg}
	revokeSvc := &revoke.Service{DB: sqlDB, Hiddify: h}

	var poller tb.Poller
	if cfg.WebhookURL != "" {
		poller = &tb.Webhook{
			Listen: cfg.WebhookListen,
			Endpoint: &tb.WebhookEndpoint{
				PublicURL: cfg.WebhookURL,
			},
		}
		log.Printf("webhook mode: %s → %s", cfg.WebhookListen, cfg.WebhookURL)
	} else {
		poller = &tb.LongPoller{Timeout: 10 * time.Second}
		log.Println("long-polling mode")
	}

	bot, err := tb.NewBot(tb.Settings{
		Token:  cfg.BotToken,
		Poller: poller,
		OnError: func(err error, c tb.Context) {
			log.Println("bot:", err)
			if c != nil {
				_ = c.Send("Произошла ошибка. Попробуйте позже.")
			}
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	deps := handlers.Deps{
		Cfg:     cfg,
		Store:   st,
		Hiddify: h,
		Approve: approveSvc,
		Revoke:  revokeSvc,
		Bot:     bot,
	}
	handlers.RegisterAll(bot, deps)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		cancel()
		bot.Stop()
	}()

	log.Println("vpn-bot listening…")
	bot.Start()
}
