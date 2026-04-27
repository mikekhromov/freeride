package handlers

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"freeride/vpn-bot/services/media"

	tb "gopkg.in/telebot.v3"
)

type vpnLinks struct {
	WireGuard string
	Xray      string
	All       string
}

func buildVPNLinks(profileURL string) vpnLinks {
	return vpnLinks{
		WireGuard: withQuery(profileURL, "client", "wireguard"),
		Xray:      withQuery(profileURL, "client", "xray"),
		All:       profileURL,
	}
}

func withQuery(raw, k, v string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	q.Set(k, v)
	u.RawQuery = q.Encode()
	return u.String()
}

func normalizeMTProxyURL(rawURL, usersHost string) string {
	if strings.TrimSpace(rawURL) == "" {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if usersHost == "" {
		return rawURL
	}
	port := u.Port()
	u.Host = usersHost
	if port != "" {
		u.Host = net.JoinHostPort(usersHost, port)
	}
	return u.String()
}

func sendConnectionPack(d Deps, recipient tb.Recipient, links vpnLinks, proxyURL string) error {
	links = normalizeVPNLinks(links, d.Cfg.UsersProxyHost)
	proxyURL = normalizeMTProxyURL(proxyURL, d.Cfg.UsersProxyHost)

	kb := &tb.ReplyMarkup{}
	kb.InlineKeyboard = [][]tb.InlineButton{
		{
			{Text: "Скачать WireGuard", Data: "dl_wg"},
			{Text: "Скачать Full Xray", Data: "dl_xr"},
		},
		{
			{Text: "Скопировать для приложения", Data: "cp_all"},
		},
	}

	vpnBody := fmt.Sprintf("Варианты подключения VPN:\n\n• <a href=\"%s\">WireGuard</a>\n\n• <a href=\"%s\">Full Xray</a>\n\n• <a href=\"%s\">Все конфиги</a>",
		html.EscapeString(links.WireGuard),
		html.EscapeString(links.Xray),
		html.EscapeString(links.All),
	)
	proxyBody := "Telegram Proxy:\n\nИспользуйте кнопки ниже."
	proxyKB := &tb.ReplyMarkup{}
	proxyKB.InlineKeyboard = [][]tb.InlineButton{
		{
			{Text: "Скопировать", Data: "cp_tg"},
			{Text: "Открыть", URL: proxyURL},
		},
	}

	if err := sendCardOrText(d, recipient, "VPN", vpnBody, &tb.SendOptions{ReplyMarkup: kb, ParseMode: tb.ModeHTML}); err != nil {
		return err
	}
	if err := sendCardOrText(d, recipient, "Telegram Proxy", proxyBody, &tb.SendOptions{ReplyMarkup: proxyKB, ParseMode: tb.ModeHTML}); err != nil {
		return err
	}
	if _, err := d.Bot.Send(recipient, buildClientAppsMessage()); err != nil {
		return err
	}

	tag := strings.TrimSpace(d.Cfg.SupportTag)
	if tag == "" {
		tag = "@support"
	}
	_, err := d.Bot.Send(recipient, fmt.Sprintf("Если возникнут проблемы, напишите %s.", tag))
	return err
}

func buildClientAppsMessage() string {
	return strings.Join([]string{
		"Рекомендуемые VPN-клиенты:",
		"• Happ Plus: https://happ.su",
		"• Hiddify Next: https://github.com/hiddify/hiddify-next/releases",
	}, "\n")
}

func sendCardOrText(d Deps, recipient tb.Recipient, title, body string, opts *tb.SendOptions) error {
	card, err := media.RenderTitleCard(title)
	if err == nil {
		photo := &tb.Photo{
			File:    tb.FromReader(bytes.NewReader(card)),
			Caption: body,
		}
		_, err = d.Bot.Send(recipient, photo, opts)
		if err == nil {
			return nil
		}
	}
	_, err = d.Bot.Send(recipient, body, opts)
	return err
}

func normalizeVPNLinks(links vpnLinks, usersHost string) vpnLinks {
	return vpnLinks{
		WireGuard: normalizeURLHost(links.WireGuard, usersHost),
		Xray:      normalizeURLHost(links.Xray, usersHost),
		All:       normalizeURLHost(links.All, usersHost),
	}
}

func normalizeURLHost(rawURL, usersHost string) string {
	if strings.TrimSpace(rawURL) == "" || strings.TrimSpace(usersHost) == "" {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	port := u.Port()
	u.Host = usersHost
	if port != "" {
		u.Host = net.JoinHostPort(usersHost, port)
	}
	return u.String()
}

func sendConfigFile(ctx context.Context, c tb.Context, d Deps, protocol string) error {
	if c.Sender() == nil {
		return c.Respond()
	}
	user, err := d.Store.GetUserByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		_ = c.Respond(&tb.CallbackResponse{Text: "Пользователь не найден"})
		return nil
	}
	if user.Status != "active" || user.HiddifyUUID == "" {
		_ = c.Respond(&tb.CallbackResponse{Text: "Нет активного доступа"})
		return nil
	}

	profileURL, err := d.Hiddify.ProfileURLByUUID(ctx, user.HiddifyUUID)
	if err != nil {
		_ = c.Respond(&tb.CallbackResponse{Text: "Ошибка ссылки"})
		return nil
	}
	links := buildVPNLinks(profileURL)
	links = normalizeVPNLinks(links, d.Cfg.UsersProxyHost)
	src := links.WireGuard
	if protocol == "xray" {
		src = links.Xray
	}

	cfgBytes, err := fetchConfig(ctx, d.Hiddify.HTTP, src)
	if err != nil {
		_ = c.Respond(&tb.CallbackResponse{Text: "Не удалось скачать"})
		return c.Send("Не удалось подготовить файл конфига, попробуйте позже.")
	}

	fileName := buildConfigFileName(c.Sender().Username, c.Sender().ID, protocol)
	doc := &tb.Document{
		File:     tb.FromReader(bytes.NewReader(cfgBytes)),
		FileName: fileName,
		MIME:     "text/plain",
	}
	_ = c.Respond(&tb.CallbackResponse{Text: "Готово"})
	return c.Send(doc)
}

func sendCopyLink(ctx context.Context, c tb.Context, d Deps, target string) error {
	if c.Sender() == nil {
		return c.Respond()
	}
	user, err := d.Store.GetUserByTelegramID(ctx, c.Sender().ID)
	if err != nil {
		_ = c.Respond(&tb.CallbackResponse{Text: "Пользователь не найден"})
		return nil
	}
	if user.Status != "active" || user.HiddifyUUID == "" {
		_ = c.Respond(&tb.CallbackResponse{Text: "Нет активного доступа"})
		return nil
	}
	if target == "tg" {
		mtproxyURL, err := d.Hiddify.MTProxyLinkByUUID(ctx, user.HiddifyUUID)
		if err != nil {
			_ = c.Respond(&tb.CallbackResponse{Text: "Ошибка ссылки"})
			return nil
		}
		mtproxyURL = normalizeMTProxyURL(mtproxyURL, d.Cfg.UsersProxyHost)
		_ = c.Respond(&tb.CallbackResponse{Text: "Ссылка отправлена"})
		_, err = d.Bot.Send(c.Recipient(), "Скопируйте ссылку Telegram Proxy:\n"+mtproxyURL)
		return err
	}
	profileURL, err := d.Hiddify.ProfileURLByUUID(ctx, user.HiddifyUUID)
	if err != nil {
		_ = c.Respond(&tb.CallbackResponse{Text: "Ошибка ссылки"})
		return nil
	}
	links := normalizeVPNLinks(buildVPNLinks(profileURL), d.Cfg.UsersProxyHost)
	_ = c.Respond(&tb.CallbackResponse{Text: "Ссылка отправлена"})
	_, err = d.Bot.Send(c.Recipient(), "Скопируйте ссылку «Все конфиги»:\n"+links.All)
	return err
}

func fetchConfig(ctx context.Context, client *http.Client, src string) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}

func buildConfigFileName(username string, telegramID int64, protocol string) string {
	userPart := strings.TrimSpace(username)
	if userPart == "" {
		userPart = strconv.FormatInt(telegramID, 10)
	}
	userPart = sanitizeFilePart(userPart)
	if protocol != "xray" {
		protocol = "wireguard"
	}
	return fmt.Sprintf("%s_%s.txt", userPart, protocol)
}

func sanitizeFilePart(in string) string {
	var b strings.Builder
	for _, r := range in {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "user"
	}
	return strings.ToLower(b.String())
}
