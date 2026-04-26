package handlers

import (
	"context"
	"strconv"
	"strings"
	"time"

	tb "gopkg.in/telebot.v3"
)

func RegisterRevoke(bot *tb.Bot, d Deps) {
	handler := func(c tb.Context) error {
		if c.Sender() == nil {
			return nil
		}
		arg := strings.TrimSpace(c.Message().Payload)
		arg = strings.TrimPrefix(arg, "@")

		if d.Cfg.IsAdmin(c.Sender().ID) && arg != "" {
			ctx, cancel := context.WithTimeout(
				context.Background(),
				30*time.Second,
			)
			defer cancel()

			res, err := d.Revoke.RevokeByUsername(ctx, arg)
			if err != nil {
				return c.Send("Ошибка: " + err.Error())
			}

			uMark := res.Username
			if uMark != "" {
				uMark = "@" + uMark
			} else {
				uMark = "id:" + strconv.FormatInt(res.TelegramID, 10)
			}
			msg := "Доступ отозван. Пользователь " + uMark + " переведён в статус banned."
			if !res.Revoked {
				msg = "Пользователь " + uMark + " переведён в статус banned (активного UUID не было)."
			}
			if err := c.Send(msg); err != nil {
				return err
			}

			_, _ = d.Bot.Send(
				&tb.User{ID: res.TelegramID},
				"🔴 Ваш доступ отозван.\nОбратитесь к администратору для восстановления.",
			)
			return nil
		}

		ctx, cancel := context.WithTimeout(
			context.Background(),
			45*time.Second,
		)
		defer cancel()
		_, tgID, hLink, mtLink, err := d.Approve.ReissueForTelegramUser(ctx, c.Sender().ID)
		if err != nil {
			if d.Cfg.IsAdmin(c.Sender().ID) && arg == "" {
				return c.Send("Использование (админ): /revoke @username\nИли без аргумента для self-перевыпуска.")
			}
			return c.Send("Не удалось перевыпустить конфиг: " + err.Error())
		}
		links := buildVPNLinks(hLink)
		mtLink = normalizeMTProxyURL(mtLink, d.Cfg.UsersProxyHost)
		recipient := &tb.User{ID: tgID}
		return sendConnectionPack(d, recipient, links, mtLink)
	}
	bot.Handle("/revoke", handler)
	bot.Handle("/revok", handler)
}
