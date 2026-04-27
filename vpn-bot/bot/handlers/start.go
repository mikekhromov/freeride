package handlers

import (
	"bytes"
	"log"
	"strings"

	"freeride/vpn-bot/services/media"

	tb "gopkg.in/telebot.v3"
)

func registerStart(bot *tb.Bot, d Deps) {
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
		log.Printf("/start from id=%d title=%q", func() int64 {
			if c.Sender() != nil {
				return c.Sender().ID
			}
			return 0
		}(), title)

		card, err := media.RenderTitleCard(title)
		if err == nil {
			photo := &tb.Photo{
				File:    tb.FromReader(bytes.NewReader(card)),
				Caption: "",
			}
			if sendErr := c.Send(photo, menu); sendErr == nil {
				return nil
			} else {
				log.Printf("/start photo send error: %v", sendErr)
			}
		} else {
			log.Printf("/start render error: %v", err)
		}
		return c.Send(title, menu)
	})
}
