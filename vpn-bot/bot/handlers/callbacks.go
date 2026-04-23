package handlers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tb "gopkg.in/telebot.v3"
)

const callbackTimeout = 45 * time.Second

func registerCallbacks(bot *tb.Bot, d Deps) {
	bot.Handle(tb.OnCallback, func(c tb.Context) error {
		data := c.Callback().Data

		ctx, cancel := context.WithTimeout(context.Background(), callbackTimeout)
		defer cancel()

		switch {
		case data == "req":
			return handleRequestAccess(ctx, c, d)
		case strings.HasPrefix(data, "a:"):
			return handleApprove(ctx, c, d, strings.TrimPrefix(data, "a:"))
		case strings.HasPrefix(data, "x:"):
			return handleReject(ctx, c, d, strings.TrimPrefix(data, "x:"))
		default:
			_ = c.Respond()
			return nil
		}
	})
}

func handleRequestAccess(ctx context.Context, c tb.Context, d Deps) error {
	sender := c.Sender()
	if sender == nil {
		return c.Respond()
	}

	uname := sender.Username
	if u, err := d.Store.GetUserByTelegramID(ctx, sender.ID); err == nil {
		if u.Status == "banned" {
			_ = c.Respond(&tb.CallbackResponse{Text: "Недоступно"})
			return c.Send("Доступ заблокирован. Обратитесь к администратору.")
		}
		if u.Status == "active" {
			if u.HiddifyUUID != "" {
				_ = c.Respond(&tb.CallbackResponse{Text: "У вас уже есть доступ"})
				return c.Send("У вас уже активирован доступ. Используйте /status.")
			}
		}
	}

	if err := d.Store.SetPendingRequest(ctx, sender.ID, uname); err != nil {
		_ = c.Respond(&tb.CallbackResponse{Text: "Ошибка БД"})
		return err
	}

	u, err := d.Store.GetUserByTelegramID(ctx, sender.ID)
	if err != nil {
		_ = c.Respond()
		return err
	}

	uTxt := "@" + uname
	if uname == "" {
		uTxt = fmt.Sprintf("id:%d", sender.ID)
	}
	kb := &tb.ReplyMarkup{}
	kb.InlineKeyboard = [][]tb.InlineButton{
		{
			{Text: "✅ Одобрить", Data: "a:" + strconv.FormatInt(u.ID, 10)},
			{Text: "❌ Отклонить", Data: "x:" + strconv.FormatInt(u.ID, 10)},
		},
	}

	msg := fmt.Sprintf(
		"👤 Новый запрос доступа\nПользователь: %s\nID: %d",
		uTxt, sender.ID,
	)
	for aid := range d.Cfg.AdminIDs {
		_, _ = d.Bot.Send(
			&tb.User{ID: aid},
			msg,
			&tb.SendOptions{ReplyMarkup: kb},
		)
	}
	_ = c.Respond(&tb.CallbackResponse{Text: "Запрос отправлен"})
	return c.Send("Запрос отправлен администратору. Ожидайте решения.")
}

func handleApprove(ctx context.Context, c tb.Context, d Deps, idStr string) error {
	if c.Sender() == nil || !d.Cfg.IsAdmin(c.Sender().ID) {
		_ = c.Respond(&tb.CallbackResponse{Text: "Нет доступа"})
		return nil
	}

	uid, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = c.Respond()
		return nil
	}

	tgUser, tgID, hLink, mtLink, already, err := d.Approve.ApproveUser(ctx, uid, c.Sender().ID)
	if err != nil {
		log.Printf("approve uid=%d: %v", uid, err)
		_ = c.Respond(&tb.CallbackResponse{Text: "Ошибка: " + err.Error()})
		return nil
	}
	if already {
		_ = c.Respond(&tb.CallbackResponse{Text: "Уже был доступ"})
		_, _ = d.Bot.Send(
			&tb.User{ID: tgID},
			"У вас уже есть активный доступ. Используйте /status.",
		)
		return nil
	}

	userText := formatApprovedUserMessage(hLink, mtLink)
	_, _ = d.Bot.Send(&tb.User{ID: tgID}, userText)
	_ = c.Respond(&tb.CallbackResponse{Text: "Одобрено"})

	un := tgUser
	if un != "" {
		un = "@" + un
	} else {
		un = strconv.FormatInt(tgID, 10)
	}
	return c.Send("✅ Пользователь " + un + " одобрен.")
}

func handleReject(ctx context.Context, c tb.Context, d Deps, idStr string) error {
	if c.Sender() == nil || !d.Cfg.IsAdmin(c.Sender().ID) {
		_ = c.Respond(&tb.CallbackResponse{Text: "Нет доступа"})
		return nil
	}

	uid, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = c.Respond()
		return nil
	}
	u, err := d.Store.GetUserByID(ctx, uid)
	if err != nil {
		_ = c.Respond()
		return nil
	}

	ok, err := d.Store.RejectPendingUser(ctx, uid)
	if err != nil {
		_ = c.Respond(&tb.CallbackResponse{Text: "Ошибка БД"})
		return err
	}
	if !ok {
		_ = c.Respond(&tb.CallbackResponse{Text: "Уже обработано"})
		return nil
	}
	_ = c.Respond(&tb.CallbackResponse{Text: "Отклонено"})
	_, _ = d.Bot.Send(
		&tb.User{ID: u.TelegramID},
		"❌ Ваш запрос доступа отклонён администратором.\nВаш статус изменён на banned.",
	)
	return c.Send("Запрос пользователя отклонён. Пользователь переведен в статус banned.")
}

func formatApprovedUserMessage(hLink, mtLink string) string {
	return fmt.Sprintf(
		"✅ Доступ одобрен!\n\n"+
			"🔐 VPN (Hiddify):\n%s\n\n"+
			"📱 Прокси Telegram:\n%s\n\n"+
			"⚠️ Ссылки только для личного использования.",
		hLink, mtLink,
	)
}
