package handlers

import tb "gopkg.in/telebot.v3"

// RegisterAll подключает все хендлеры (vpn-setup-guide).
func RegisterAll(bot *tb.Bot, d Deps) {
	registerStart(bot)
	registerCallbacks(bot, d)
	registerAdmin(bot, d)
	registerStatus(bot, d)
	RegisterRevoke(bot, d)
}
