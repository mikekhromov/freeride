package handlers

import (
	"strings"

	tb "gopkg.in/telebot.v3"
)

func registerStart(bot *tb.Bot) {
	bot.Handle("/start", func(c tb.Context) error {
		menu := &tb.ReplyMarkup{}
		menu.InlineKeyboard = [][]tb.InlineButton{
			{{Text: "Запросить доступ", Data: "req"}},
		}
		title := "Привет"
		if c.Sender() != nil {
			username := strings.TrimSpace(c.Sender().Username)
			if username != "" {
				title = "Привет, @" + username
			} else if fn := strings.TrimSpace(c.Sender().FirstName); fn != "" {
				title = "Привет, " + fn
			}
		}
		return sendGeneratedCardOrText(Deps{Bot: bot}, c.Recipient(), title, "", &tb.SendOptions{ReplyMarkup: menu})
	})
}
