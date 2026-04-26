package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	tb "gopkg.in/telebot.v3"
)

func handleUserStats(c tb.Context, d Deps) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	u, err := d.Store.GetUserByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Send("Нет записи. Нажмите /start и запросите доступ.")
		}
		return c.Send("Ошибка: " + err.Error())
	}
	if u.Status != "active" || u.HiddifyUUID == "" {
		return c.Send(fmt.Sprintf("Статус: %s\nТрафик недоступен: нет активного доступа.", u.Status))
	}

	usedGB, limitGB, err := d.Hiddify.UsageByUUID(ctx, u.HiddifyUUID)
	if err != nil {
		return c.Send("Не удалось получить статистику трафика. Попробуйте позже.")
	}
	remaining := limitGB - usedGB
	if remaining < 0 {
		remaining = 0
	}
	return c.Send(fmt.Sprintf(
		"Ваша статистика:\n• Использовано: %.2f GB\n• Лимит: %.2f GB\n• Осталось: %.2f GB",
		usedGB, limitGB, remaining,
	))
}
