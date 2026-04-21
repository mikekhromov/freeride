package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	tb "gopkg.in/telebot.v3"
)

const (
	timeoutStats   = 15 * time.Second
	timeoutApprove = 45 * time.Second
)

func registerAdmin(bot *tb.Bot, d Deps) {
	bot.Handle("/stats", func(c tb.Context) error {
		if c.Sender() == nil || !d.Cfg.IsAdmin(c.Sender().ID) {
			return c.Send("Нет доступа.")
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeoutStats)
		defer cancel()

		arg := strings.TrimSpace(c.Message().Payload)
		arg = strings.TrimPrefix(arg, "@")
		if arg != "" {
			tgID, status, n, err := d.Store.UserStatsByUsername(ctx, arg)
			if err != nil {
				if err == sql.ErrNoRows {
					return c.Send("Пользователь не найден.")
				}
				return c.Send("Ошибка: " + err.Error())
			}
			return c.Send(fmt.Sprintf(
				"@%s\nСтатус: %s\nАктивных секретов: %d\nTelegram ID: %d",
				arg, status, n, tgID,
			))
		}

		m, err := d.Store.StatsByStatus(ctx)
		if err != nil {
			return c.Send("Ошибка: " + err.Error())
		}

		var b strings.Builder
		b.WriteString("Статистика пользователей по статусам:\n")
		for k, v := range m {
			b.WriteString(fmt.Sprintf("• %s: %d\n", k, v))
		}
		return c.Send(strings.TrimSpace(b.String()))
	})

	bot.Handle("/users", func(c tb.Context) error {
		if c.Sender() == nil || !d.Cfg.IsAdmin(c.Sender().ID) {
			return c.Send("Нет доступа.")
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeoutStats)
		defer cancel()

		list, err := d.Store.ListUsersRecent(ctx, 40)
		if err != nil {
			return c.Send("Ошибка: " + err.Error())
		}
		if len(list) == 0 {
			return c.Send("Пользователей пока нет.")
		}

		var b strings.Builder
		b.WriteString("Последние пользователи:\n")
		for _, u := range list {
			un := u.TelegramUsername
			if un != "" {
				un = "@" + un
			} else {
				un = "—"
			}
			b.WriteString(fmt.Sprintf("• %s id=%d %s\n", un, u.TelegramID, u.Status))
		}
		return c.Send(b.String())
	})

	bot.Handle("/warnings", func(c tb.Context) error {
		if c.Sender() == nil || !d.Cfg.IsAdmin(c.Sender().ID) {
			return c.Send("Нет доступа.")
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeoutStats)
		defer cancel()

		ws, err := d.Store.ListPendingWarnings(ctx)
		if err != nil {
			return c.Send("Ошибка: " + err.Error())
		}
		if len(ws) == 0 {
			return c.Send("Активных предупреждений нет.")
		}

		var b strings.Builder
		b.WriteString("Активные предупреждения:\n")
		for _, w := range ws {
			un := w.Username
			if un != "" {
				un = "@" + un
			}
			b.WriteString(fmt.Sprintf("• #%d user %s до %s\n", w.ID, un, w.ExpiresAt))
		}
		return c.Send(b.String())
	})

	bot.Handle("/approve", func(c tb.Context) error {
		if c.Sender() == nil || !d.Cfg.IsAdmin(c.Sender().ID) {
			return c.Send("Нет доступа.")
		}

		arg := strings.TrimSpace(strings.TrimPrefix(c.Message().Payload, "@"))
		if arg == "" {
			return c.Send("Использование: /approve @username")
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeoutApprove)
		defer cancel()

		u, err := d.Store.GetUserByUsername(ctx, arg)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.Send("Пользователь не найден.")
			}
			return c.Send("Ошибка: " + err.Error())
		}

		tgUser, tgID, h, mt, already, err := d.Approve.ApproveUser(ctx, u.ID, c.Sender().ID)
		if err != nil {
			return c.Send("Ошибка: " + err.Error())
		}
		if already {
			_, _ = d.Bot.Send(
				&tb.User{ID: tgID},
				"У вас уже есть активный доступ. /status",
			)
			return c.Send("У пользователя уже был активный доступ (ссылки переотправлены).")
		}

		_, _ = d.Bot.Send(
			&tb.User{ID: tgID},
			formatApprovedUserMessage(h, mt, d.Cfg.DeviceLimit),
		)

		mark := tgUser
		if mark != "" {
			mark = "@" + mark
		} else {
			mark = strconv.FormatInt(tgID, 10)
		}
		return c.Send("✅ Одобрено: " + mark)
	})

	bot.Handle("/reissue", func(c tb.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), timeoutApprove)
		defer cancel()

		arg := strings.TrimSpace(c.Message().Payload)
		arg = strings.TrimPrefix(arg, "@")
		if c.Sender() == nil {
			return nil
		}

		if d.Cfg.IsAdmin(c.Sender().ID) && arg != "" {
			u, err := d.Store.GetUserByUsername(ctx, arg)
			if err != nil {
				if err == sql.ErrNoRows {
					return c.Send("Пользователь не найден.")
				}
				return c.Send("Ошибка: " + err.Error())
			}

			tgID, uname, h, mt, err := d.Reissue.ReissueForUserID(ctx, u.ID, nil)
			if err != nil {
				return c.Send("Ошибка: " + err.Error())
			}

			txt := fmt.Sprintf(
				"🔄 Секрет перевыпущен!\n\n🔐 Новый VPN:\n%s\n\n📱 Новый MTProxy:\n%s",
				h, mt,
			)
			_, _ = d.Bot.Send(&tb.User{ID: tgID}, txt)

			mark := uname
			if mark != "" {
				mark = "@" + mark
			} else {
				mark = strconv.FormatInt(tgID, 10)
			}
			return c.Send("✅ Перевыпущено для " + mark)
		}

		u, err := d.Store.GetUserByTelegramID(ctx, c.Sender().ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.Send("Сначала запросите доступ через /start.")
			}
			return c.Send("Ошибка: " + err.Error())
		}
		if u.Status != "active" {
			return c.Send("Перевыпуск доступен только для активных пользователей.")
		}

		_, _, h, mt, err := d.Reissue.ReissueForUserID(ctx, u.ID, nil)
		if err != nil {
			return c.Send("Ошибка: " + err.Error())
		}

		txt := fmt.Sprintf(
			"🔄 Секрет перевыпущен!\n\n🔐 Новый VPN:\n%s\n\n📱 Новый MTProxy:\n%s",
			h, mt,
		)
		return c.Send(txt)
	})
}
