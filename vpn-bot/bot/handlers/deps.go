package handlers

import (
	"freeride/vpn-bot/config"
	"freeride/vpn-bot/services/approve"
	"freeride/vpn-bot/services/hiddify"
	"freeride/vpn-bot/services/revoke"
	"freeride/vpn-bot/store"

	tb "gopkg.in/telebot.v3"
)

type Deps struct {
	Cfg     config.Config
	Store   *store.Store
	Hiddify *hiddify.Client
	Approve *approve.Service
	Revoke  *revoke.Service
	Bot     *tb.Bot
}
