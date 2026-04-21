package approve

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"freeride/vpn-bot/config"
	"freeride/vpn-bot/services/hiddify"
	"freeride/vpn-bot/services/mtproxy"
	"freeride/vpn-bot/store"
)

type Service struct {
	Store   *store.Store
	Hiddify *hiddify.Client
	MT      *mtproxy.Manager
	Cfg     config.Config
}

func (s *Service) ApproveUser(ctx context.Context, userID int64, adminTelegramID int64) (tgUser string, tgID int64, hLink, mtLink string, alreadyActive bool, err error) {
	u, err := s.Store.GetUserByID(ctx, userID)
	if err != nil {
		return "", 0, "", "", false, err
	}
	if u.Status == "banned" {
		return "", 0, "", "", false, fmt.Errorf("пользователь заблокирован")
	}
	if u.Status == "active" {
		sec, e := s.Store.GetActiveSecretByUserID(ctx, userID)
		if e == nil && sec != nil {
			return u.TelegramUsername, u.TelegramID, sec.HiddifyLink, sec.MTProxyLink, true, nil
		}
	}
	if u.Status != "pending" && u.Status != "rejected" && u.Status != "active" {
		return "", 0, "", "", false, fmt.Errorf("статус пользователя не позволяет одобрить: %s", u.Status)
	}

	name := fmt.Sprintf("tg-%d", u.TelegramID)
	if u.TelegramUsername != "" {
		name = u.TelegramUsername
	}
	hUID, hSub, err := s.Hiddify.CreateUser(ctx, name, s.Cfg.DeviceLimit)
	if err != nil {
		return "", 0, "", "", false, err
	}

	mtSec, err := s.MT.GenerateSecret()
	if err != nil {
		return "", 0, "", "", false, err
	}
	if err := s.MT.AddSecret(mtSec); err != nil {
		return "", 0, "", "", false, err
	}

	mtLink = buildMTProxyLink(s.Cfg.MtproxyPublicHost, s.Cfg.MtproxyPublicPort, mtSec)
	if mtLink == "" {
		mtLink = "(задайте MTPROXY_PUBLIC_HOST и MTPROXY_PUBLIC_PORT в .env.local)"
	}

	tx, err := s.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", 0, "", "", false, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		UPDATE users SET status = 'active', approved_by = ?, approved_at = datetime('now')
		WHERE id = ?`, adminTelegramID, userID); err != nil {
		return "", 0, "", "", false, err
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO secrets (user_id, hiddify_user_id, hiddify_link, mtproxy_secret, mtproxy_link, is_active)
		VALUES (?, ?, ?, ?, ?, 1)`, userID, hUID, hSub, mtSec, mtLink)
	if err != nil {
		return "", 0, "", "", false, err
	}
	sid, err := res.LastInsertId()
	if err != nil {
		return "", 0, "", "", false, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secret_events (secret_id, event_type) VALUES (?, 'connected')`, sid); err != nil {
		return "", 0, "", "", false, err
	}
	if err := tx.Commit(); err != nil {
		return "", 0, "", "", false, err
	}

	return u.TelegramUsername, u.TelegramID, hSub, mtLink, false, nil
}

func buildMTProxyLink(host string, port int, secret string) string {
	host = strings.TrimSpace(host)
	if host == "" || port <= 0 || secret == "" {
		return ""
	}
	u := url.URL{Scheme: "tg", Host: "proxy"}
	q := u.Query()
	q.Set("server", host)
	q.Set("port", fmt.Sprintf("%d", port))
	q.Set("secret", secret)
	u.RawQuery = q.Encode()
	return u.String()
}
