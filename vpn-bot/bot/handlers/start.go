package handlers

import (
	tb "gopkg.in/telebot.v3"
)

func registerStart(bot *tb.Bot) {
	bot.Handle("/start", func(c tb.Context) error {
		menu := &tb.ReplyMarkup{}
		menu.InlineKeyboard = [][]tb.InlineButton{
			{{Text: "Запросить доступ", Data: "req"}},
		}
		return c.Send("Привет! Нажми кнопку для запроса доступа.", menu)
	})
}
