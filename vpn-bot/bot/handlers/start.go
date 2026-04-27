package handlers

import (
	"strings"

	tb "gopkg.in/telebot.v3"
)

func registerStart(bot *tb.Bot, d Deps) {
	bot.Handle("/start", func(c tb.Context) error {
		menu := &tb.ReplyMarkup{}
		menu.InlineKeyboard = [][]tb.InlineButton{
			{{Text: "Запросить доступ", Data: "req"}},
		}
		title := greetingTitle(c)
		return sendGeneratedCardOrText(d, c.Recipient(), title, "", &tb.SendOptions{ReplyMarkup: menu})
	})
}

func greetingTitle(c tb.Context) string {
	title := "Привет"
	if c.Sender() == nil {
		return title
	}
	if u := strings.TrimSpace(c.Sender().Username); u != "" {
		return "Привет, @" + u
	}
	if fn := strings.TrimSpace(c.Sender().FirstName); fn != "" {
		return "Привет, " + fn
	}
	return title
}
