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
	"freeride/vpn-bot/services/monitor"
	"freeride/vpn-bot/services/mtproxy"
	"freeride/vpn-bot/services/reissue"
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
	h := hiddify.New(cfg.HiddifyURL, cfg.HiddifyKey)
	mt := &mtproxy.Manager{ConfigPath: cfg.MtproxyData}

	approveSvc := &approve.Service{Store: st, Hiddify: h, MT: mt, Cfg: cfg}
	reissueSvc := &reissue.Service{Store: st, Hiddify: h, MT: mt, Cfg: cfg}
	revokeSvc := &revoke.Service{DB: sqlDB, Hiddify: h, MTProxy: mt}

	bot, err := tb.NewBot(tb.Settings{
		Token:  cfg.BotToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
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
		Approve: approveSvc,
		Reissue: reissueSvc,
		Revoke:  revokeSvc,
		Bot:     bot,
	}
	handlers.RegisterAll(bot, deps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		cancel()
		bot.Stop()
	}()

	runner := &monitor.Runner{Bot: bot, Cfg: cfg, Store: st, Revoke: revokeSvc}
	runner.Start(ctx)

	log.Println("vpn-bot listening…")
	bot.Start()
}
