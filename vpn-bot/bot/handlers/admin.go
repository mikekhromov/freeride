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
			return c.Send(fmt.Sprintf("@%s\nСтатус: %s\nАктивных доступов: %d\nTelegram ID: %d", arg, status, n, tgID))
		}

		m, err := d.Store.StatsByStatus(ctx)
		if err != nil {
			return c.Send("Ошибка: " + err.Error())
		}
		hUsers, hErr := d.Hiddify.AdminUsersCount(ctx)

		var b strings.Builder
		b.WriteString("Статистика пользователей:\n")
		for k, v := range m {
			b.WriteString(fmt.Sprintf("• %s: %d\n", k, v))
		}
		if hErr != nil {
			b.WriteString("• hiddify_total: недоступно (" + hErr.Error() + ")\n")
		} else {
			b.WriteString(fmt.Sprintf("• hiddify_total: %d\n", hUsers))
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
			formatApprovedUserMessage(h, mt),
		)

		mark := tgUser
		if mark != "" {
			mark = "@" + mark
		} else {
			mark = strconv.FormatInt(tgID, 10)
		}
		return c.Send("✅ Одобрено: " + mark)
	})

}
