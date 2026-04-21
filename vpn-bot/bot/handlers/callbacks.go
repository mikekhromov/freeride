package handlers

import (
	"context"
	"fmt"
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
		case strings.HasPrefix(data, "W:"):
			return handleWarningAction(ctx, c, d, data)
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
			if sec, e := d.Store.GetActiveSecretByUserID(ctx, u.ID); e == nil && sec != nil {
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

	userText := formatApprovedUserMessage(hLink, mtLink, d.Cfg.DeviceLimit)
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
		"❌ Ваш запрос доступа отклонён администратором.",
	)
	return c.Send("Запрос пользователя отклонён.")
}

func handleWarningAction(ctx context.Context, c tb.Context, d Deps, data string) error {
	if c.Sender() == nil || !d.Cfg.IsAdmin(c.Sender().ID) {
		_ = c.Respond(&tb.CallbackResponse{Text: "Нет доступа"})
		return nil
	}
	parts := strings.Split(data, ":")
	if len(parts) != 3 || parts[0] != "W" {
		_ = c.Respond()
		return nil
	}
	wid, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		_ = c.Respond()
		return nil
	}

	act := parts[2]
	w, err := d.Store.GetWarningByID(ctx, wid)
	if err != nil || w.Status != "pending" {
		_ = c.Respond(&tb.CallbackResponse{Text: "Не найдено"})
		return nil
	}
	sec, err := d.Store.GetSecretByID(ctx, w.SecretID)
	if err != nil {
		_ = c.Respond()
		return nil
	}

	switch act {
	case "I":
		if err := d.Store.UpdateWarningStatus(ctx, wid, "ignored", "admin"); err != nil {
			_ = c.Respond(&tb.CallbackResponse{Text: "Ошибка"})
			return err
		}
		_ = c.Respond(&tb.CallbackResponse{Text: "Ок"})
		return c.Send("Предупреждение помечено как игнорируемое.")

	case "V":
		res, err := d.Revoke.RevokeBySecretID(ctx, w.SecretID, true)
		if err != nil {
			_ = c.Respond(&tb.CallbackResponse{Text: err.Error()})
			return nil
		}
		_ = d.Store.UpdateWarningStatus(ctx, wid, "revoked", "admin")
		_ = c.Respond(&tb.CallbackResponse{Text: "Отозвано"})
		if res != nil && res.SecretRevoked {
			_, _ = d.Bot.Send(
				&tb.User{ID: res.TelegramID},
				"🔴 Секрет отозван администратором после предупреждения.",
			)
		}
		return c.Send("Секрет отозван.")

	case "R":
		tgID, uname, h, mt, err := d.Reissue.ReissueForUserID(ctx, sec.UserID, &wid)
		if err != nil {
			_ = c.Respond(&tb.CallbackResponse{Text: err.Error()})
			return nil
		}
		_ = c.Respond(&tb.CallbackResponse{Text: "Перевыпущено"})

		txt := fmt.Sprintf(
			"🔄 Секрет перевыпущен!\n\n"+
				"🔐 Новый VPN (Hiddify):\n%s\n\n"+
				"📱 Новый прокси Telegram:\n%s\n\n"+
				"Старые ссылки больше не работают.",
			h, mt,
		)
		_, _ = d.Bot.Send(&tb.User{ID: tgID}, txt)
		uMark := uname
		if uMark != "" {
			uMark = "@" + uMark
		} else {
			uMark = strconv.FormatInt(tgID, 10)
		}
		return c.Send("✅ Секрет " + uMark + " перевыпущен.")

	default:
		_ = c.Respond()
		return nil
	}
}

func formatApprovedUserMessage(hLink, mtLink string, limit int) string {
	return fmt.Sprintf(
		"✅ Доступ одобрен!\n\n"+
			"🔐 VPN (Hiddify):\n%s\nЛимит: %d устройств\n\n"+
			"📱 Прокси Telegram:\n%s\n\n"+
			"⚠️ Ссылки только для личного использования.\n"+
			"При превышении лимита секрет будет отозван.",
		hLink, limit, mtLink,
	)
}
