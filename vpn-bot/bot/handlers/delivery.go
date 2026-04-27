package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
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

func normalizeMTProxyURL(rawURL, usersHost, hiddifyDomain string) string {
	if strings.TrimSpace(rawURL) == "" {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	// Hiddify can return tg://proxy?server=<ip>&port=... where host is "proxy".
	// For this format we must rewrite the "server" query param, not URL host.
	if strings.EqualFold(u.Scheme, "tg") {
		q := u.Query()
		serverHost := q.Get("server")
		targetHost := preferredUsersHost(serverHost, usersHost, hiddifyDomain)
		if targetHost == "" {
			return rawURL
		}
		q.Set("server", targetHost)
		normalizeKnownQueryHosts(q, targetHost)
		u.RawQuery = q.Encode()
		return u.String()
	}
	targetHost := preferredUsersHost(u.Hostname(), usersHost, hiddifyDomain)
	if targetHost == "" {
		return rawURL
	}
	port := u.Port()
	u.Host = targetHost
	if port != "" {
		u.Host = net.JoinHostPort(targetHost, port)
	}
	q := u.Query()
	normalizeKnownQueryHosts(q, targetHost)
	u.RawQuery = q.Encode()
	return u.String()
}

func sendConnectionPack(d Deps, recipient tb.Recipient, links vpnLinks, proxyURL string) error {
	links = normalizeVPNLinks(links, d.Cfg.UsersProxyHost, d.Cfg.HiddifyDomain)
	proxyURL = normalizeMTProxyURL(proxyURL, d.Cfg.UsersProxyHost, d.Cfg.HiddifyDomain)

	kb := &tb.ReplyMarkup{}
	kb.InlineKeyboard = [][]tb.InlineButton{
		{
			{Text: "WireGuard (.conf)", Data: "dl_wg"},
			{Text: "Full Xray (.txt)", Data: "dl_xr"},
		},
		{
			{Text: "Все конфиги (ссылка)", Data: "cp_all"},
		},
	}

	proxyKB := &tb.ReplyMarkup{}
	proxyKB.InlineKeyboard = [][]tb.InlineButton{
		{
			{Text: "Скопировать", Data: "cp_tg"},
			{Text: "Открыть", URL: proxyURL},
		},
	}
	clientKB := &tb.ReplyMarkup{}
	clientKB.InlineKeyboard = [][]tb.InlineButton{
		{
			{Text: "Happ", URL: "https://happ.su"},
			{Text: "Hiddify", URL: "https://github.com/hiddify/hiddify-next/releases"},
		},
	}

	if err := sendStaticCardOrText(d, recipient, "vpn", "VPN", "", &tb.SendOptions{ReplyMarkup: kb}); err != nil {
		return err
	}
	if err := sendStaticCardOrText(d, recipient, "telegram", "Telegram Proxy", "", &tb.SendOptions{ReplyMarkup: proxyKB}); err != nil {
		return err
	}
	if err := sendStaticCardOrText(d, recipient, "client", "VPN Client", "", &tb.SendOptions{ReplyMarkup: clientKB}); err != nil {
		return err
	}
	return nil
}

func sendGeneratedCardOrText(d Deps, recipient tb.Recipient, title, body string, opts *tb.SendOptions) error {
	if opts == nil {
		opts = &tb.SendOptions{}
	}
	fallbackText := strings.TrimSpace(body)
	if fallbackText == "" {
		fallbackText = title
	}
	card, err := media.RenderTitleCard(title)
	if err != nil {
		_, err = d.Bot.Send(recipient, fallbackText, opts)
		return err
	}
	f, err := os.CreateTemp("", "vpnbot-title-*.png")
	if err != nil {
		_, err = d.Bot.Send(recipient, fallbackText, opts)
		return err
	}
	tmpPath := f.Name()
	_, werr := f.Write(card)
	_ = f.Close()
	if werr != nil {
		_ = os.Remove(tmpPath)
		_, err = d.Bot.Send(recipient, fallbackText, opts)
		return err
	}
	defer os.Remove(tmpPath)

	photo := &tb.Photo{
		File:    tb.FromDisk(tmpPath),
		Caption: body,
	}
	_, err = d.Bot.Send(recipient, photo, opts)
	if err == nil {
		return nil
	}
	_, err = d.Bot.Send(recipient, fallbackText, opts)
	return err
}

func sendStaticCardOrText(d Deps, recipient tb.Recipient, cardName, fallbackTitle, body string, opts *tb.SendOptions) error {
	if opts == nil {
		opts = &tb.SendOptions{}
	}
	path, err := media.ResolveStaticCardPath(cardName)
	if err == nil {
		photo := &tb.Photo{
			File:    tb.FromDisk(path),
			Caption: body,
		}
		_, err = d.Bot.Send(recipient, photo, opts)
		if err == nil {
			return nil
		}
	}
	fallbackText := strings.TrimSpace(body)
	if fallbackText == "" {
		fallbackText = fallbackTitle
	}
	_, err = d.Bot.Send(recipient, fallbackText, opts)
	return err
}

func normalizeVPNLinks(links vpnLinks, usersHost, hiddifyDomain string) vpnLinks {
	return vpnLinks{
		WireGuard: normalizeURLHost(links.WireGuard, usersHost, hiddifyDomain),
		Xray:      normalizeURLHost(links.Xray, usersHost, hiddifyDomain),
		All:       normalizeURLHost(links.All, usersHost, hiddifyDomain),
	}
}

func normalizeURLHost(rawURL, usersHost, hiddifyDomain string) string {
	if strings.TrimSpace(rawURL) == "" {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	targetHost := preferredUsersHost(u.Hostname(), usersHost, hiddifyDomain)
	if targetHost == "" {
		return rawURL
	}
	port := u.Port()
	u.Host = targetHost
	if port != "" {
		u.Host = net.JoinHostPort(targetHost, port)
	}
	q := u.Query()
	normalizeKnownQueryHosts(q, targetHost)
	u.RawQuery = q.Encode()
	return u.String()
}

func preferredUsersHost(currentHost, usersHost, hiddifyDomain string) string {
	if strings.TrimSpace(usersHost) != "" {
		return usersHost
	}
	if currentHost == "" {
		return ""
	}
	if ip := net.ParseIP(currentHost); ip == nil {
		// URL already uses a domain name.
		return currentHost
	}
	trimmedDomain := strings.TrimSpace(hiddifyDomain)
	if trimmedDomain == "" {
		return currentHost
	}
	parsed, err := url.Parse(trimmedDomain)
	if err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	trimmedDomain = strings.TrimPrefix(trimmedDomain, "https://")
	trimmedDomain = strings.TrimPrefix(trimmedDomain, "http://")
	trimmedDomain = strings.Trim(trimmedDomain, "/")
	if trimmedDomain == "" {
		return currentHost
	}
	return trimmedDomain
}

func normalizeKnownQueryHosts(q url.Values, targetHost string) {
	if targetHost == "" {
		return
	}
	for _, key := range []string{"server", "host", "hostname", "sni", "domain"} {
		v := strings.TrimSpace(q.Get(key))
		if v == "" {
			continue
		}
		if ip := net.ParseIP(v); ip == nil {
			continue
		}
		q.Set(key, targetHost)
	}
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
	links = normalizeVPNLinks(links, d.Cfg.UsersProxyHost, d.Cfg.HiddifyDomain)
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
		mtproxyURL = normalizeMTProxyURL(mtproxyURL, d.Cfg.UsersProxyHost, d.Cfg.HiddifyDomain)
		_ = c.Respond(&tb.CallbackResponse{Text: "Ссылка отправлена"})
		_, err = d.Bot.Send(c.Recipient(), "Скопируйте ссылку Telegram Proxy:\n"+mtproxyURL)
		return err
	}
	profileURL, err := d.Hiddify.ProfileURLByUUID(ctx, user.HiddifyUUID)
	if err != nil {
		_ = c.Respond(&tb.CallbackResponse{Text: "Ошибка ссылки"})
		return nil
	}
	links := normalizeVPNLinks(buildVPNLinks(profileURL), d.Cfg.UsersProxyHost, d.Cfg.HiddifyDomain)
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
