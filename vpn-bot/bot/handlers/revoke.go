package handlers

import (
	"context"
	"strconv"
	"strings"
	"time"

	tb "gopkg.in/telebot.v3"
)

func RegisterRevoke(bot *tb.Bot, d Deps) {
	bot.Handle("/revoke", func(c tb.Context) error {
		if c.Sender() == nil || !d.Cfg.IsAdmin(c.Sender().ID) {
			return c.Send("Нет доступа к этой команде.")
		}
		arg := strings.TrimSpace(c.Message().Payload)
		arg = strings.TrimPrefix(arg, "@")
		if arg == "" {
			return c.Send("Использование: /revoke @username")
		}

		ctx, cancel := context.WithTimeout(
			context.Background(),
			30*time.Second,
		)
		defer cancel()

		res, err := d.Revoke.RevokeByUsername(ctx, arg)
		if err != nil {
			return c.Send("Ошибка: " + err.Error())
		}
		if !res.SecretRevoked {
			return c.Send("У пользователя нет активного секрета (ничего не отозвано).")
		}

		uMark := res.Username
		if uMark != "" {
			uMark = "@" + uMark
		} else {
			uMark = "id:" + strconv.FormatInt(res.TelegramID, 10)
		}
		msg := "Секрет отозван. Пользователь " + uMark + " переведён в статус banned."
		if err := c.Send(msg); err != nil {
			return err
		}

		_, err = d.Bot.Send(
			&tb.User{ID: res.TelegramID},
			"🔴 Ваш доступ отозван администратором. Старые ссылки больше не действуют.",
		)
		return err
	})
}
