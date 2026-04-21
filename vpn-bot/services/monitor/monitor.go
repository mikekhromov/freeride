package monitor

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"freeride/vpn-bot/config"
	"freeride/vpn-bot/services/revoke"
	"freeride/vpn-bot/store"

	tb "gopkg.in/telebot.v3"
)

var ipRE = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)

type Runner struct {
	Bot    *tb.Bot
	Cfg    config.Config
	Store  *store.Store
	Revoke *revoke.Service
}

func (r *Runner) Start(ctx context.Context) {
	go func() { r.tick(context.Background()) }()
	t := time.NewTicker(r.Cfg.MonitorInterval)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				r.tick(context.Background())
			}
		}
	}()
}

func (r *Runner) tick(ctx context.Context) {
	r.expireWarnings(ctx)
	r.checkLimits(ctx)
}

func (r *Runner) expireWarnings(ctx context.Context) {
	ids, err := r.Store.ListExpiredPendingWarnings(ctx)
	if err != nil {
		return
	}
	for _, wid := range ids {
		w, err := r.Store.GetWarningByID(ctx, wid)
		if err != nil {
			continue
		}
		res, _ := r.Revoke.RevokeBySecretID(ctx, w.SecretID, false)
		_ = r.Store.UpdateWarningStatus(ctx, wid, "expired", "auto")
		if res != nil && res.SecretRevoked {
			msg := "🔴 Ваш секрет отозван.\nПревышение лимита не устранено в течение отведённого времени.\nДля восстановления обратитесь к администратору."
			_, _ = r.Bot.Send(&tb.User{ID: res.TelegramID}, msg)
			r.notifyAdmins("🔴 Секрет @" + res.Username + " автоматически отозван (истёк срок предупреждения).")
		}
	}
}

func (r *Runner) checkLimits(ctx context.Context) {
	logPath := strings.TrimSpace(r.Cfg.MtproxyLog)
	if logPath == "" {
		return
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		return
	}
	content := string(data)

	rows, err := r.Store.ListActiveSecrets(ctx)
	if err != nil {
		return
	}

	for _, row := range rows {
		if row.MTProxySecret == "" {
			continue
		}
		n := uniqueIPsForSecret(content, row.MTProxySecret)
		if n <= r.Cfg.DeviceLimit {
			continue
		}
		pending, err := r.Store.CountPendingWarningForSecret(ctx, row.SecretID)
		if err != nil || pending > 0 {
			continue
		}
		exp := time.Now().UTC().Add(r.Cfg.WarningTTL)
		wid, err := r.Store.InsertWarning(ctx, row.SecretID, exp)
		if err != nil {
			continue
		}

		uTxt := "@" + row.TelegramUser
		if row.TelegramUser == "" {
			uTxt = "id:" + strconv.FormatInt(row.TelegramID, 10)
		}
		adm := "⚠️ Превышен лимит!\n\nПользователь: " + uTxt +
			"\nУникальных IP (по логу, строки с этим секретом): " + strconv.Itoa(n) +
			"\nЛимит: " + strconv.Itoa(r.Cfg.DeviceLimit)

		kb := &tb.ReplyMarkup{}
		kb.InlineKeyboard = [][]tb.InlineButton{
			{
				{Text: "🔄 Перевыпустить", Data: "W:" + strconv.FormatInt(wid, 10) + ":R"},
				{Text: "🔴 Отозвать", Data: "W:" + strconv.FormatInt(wid, 10) + ":V"},
				{Text: "✅ Игнорировать", Data: "W:" + strconv.FormatInt(wid, 10) + ":I"},
			},
		}
		r.notifyAdminsWithMarkup(adm, kb)

		userMsg := "⚠️ Внимание!\n\nПо логам зафиксировано больше уникальных адресов, чем лимит (" +
			strconv.Itoa(r.Cfg.DeviceLimit) + ").\nЕсли это не вы — сообщите администратору.\nПри отсутствии действий секрет будет отозван автоматически."
		_, _ = r.Bot.Send(&tb.User{ID: row.TelegramID}, userMsg)
	}
}

func uniqueIPsForSecret(logContent, secret string) int {
	sec := strings.ToLower(secret)
	seen := make(map[string]struct{})
	for _, line := range strings.Split(logContent, "\n") {
		if !strings.Contains(strings.ToLower(line), sec) {
			continue
		}
		for _, ip := range ipRE.FindAllString(line, -1) {
			seen[ip] = struct{}{}
		}
	}
	return len(seen)
}

func (r *Runner) notifyAdmins(text string) {
	for id := range r.Cfg.AdminIDs {
		_, _ = r.Bot.Send(&tb.User{ID: id}, text)
	}
}

func (r *Runner) notifyAdminsWithMarkup(text string, kb *tb.ReplyMarkup) {
	opts := &tb.SendOptions{ReplyMarkup: kb}
	for id := range r.Cfg.AdminIDs {
		_, _ = r.Bot.Send(&tb.User{ID: id}, text, opts)
	}
}
