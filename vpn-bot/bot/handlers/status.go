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

		if u.Status != "active" || u.HiddifyUUID == "" {
			return c.Send(fmt.Sprintf("Статус: %s\nАктивного секрета нет.", u.Status))
		}

		profileURL, err := d.Hiddify.ProfileURLByUUID(ctx, u.HiddifyUUID)
		if err != nil {
			return c.Send("Не удалось получить VPN-ссылку. Обратитесь к администратору.")
		}
		mtproxyURL, err := d.Hiddify.MTProxyLinkByUUID(ctx, u.HiddifyUUID)
		if err != nil {
			return c.Send("Не удалось получить MTProxy-ссылку. Обратитесь к администратору.")
		}
		links := buildVPNLinks(profileURL)
		mtproxyURL = normalizeMTProxyURL(mtproxyURL, d.Cfg.UsersProxyHost, d.Cfg.HiddifyDomain)
		return sendConnectionPack(d, c.Recipient(), links, mtproxyURL)
	})
}
