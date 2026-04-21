package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	tb "gopkg.in/telebot.v3"
)

func registerStatus(bot *tb.Bot, d Deps) {
	bot.Handle("/status", func(c tb.Context) error {
		if c.Sender() == nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		u, err := d.Store.GetUserByTelegramID(ctx, c.Sender().ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.Send("Нет записи. Нажмите /start и запросите доступ.")
			}
			return c.Send("Ошибка: " + err.Error())
		}

		sec, err := d.Store.GetActiveSecretByUserID(ctx, u.ID)
		if err != nil || sec == nil {
			return c.Send(fmt.Sprintf("Статус: %s\nАктивного секрета нет.", u.Status))
		}
		txt := fmt.Sprintf(
			"Статус: %s\n\n🔐 VPN (Hiddify):\n%s\n\n📱 MTProxy:\n%s",
			u.Status, sec.HiddifyLink, sec.MTProxyLink,
		)
		return c.Send(txt)
	})
}
