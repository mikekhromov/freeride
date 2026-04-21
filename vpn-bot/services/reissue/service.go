package reissue

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

// ReissueForUserID перевыпускает активный секрет пользователя (по внутреннему user id).
func (s *Service) ReissueForUserID(ctx context.Context, userID int64, warningID *int64) (tgID int64, username, hLink, mtLink string, err error) {
	u, err := s.Store.GetUserByID(ctx, userID)
	if err != nil {
		return 0, "", "", "", err
	}
	if u.Status != "active" {
		return 0, "", "", "", fmt.Errorf("пользователь не active")
	}
	old, err := s.Store.GetActiveSecretByUserID(ctx, userID)
	if err != nil {
		return 0, "", "", "", fmt.Errorf("нет активного секрета: %w", err)
	}

	if err := s.MT.RemoveSecret(old.MTProxySecret); err != nil {
		return 0, "", "", "", err
	}
	if err := s.Hiddify.DeactivateUser(ctx, old.HiddifyUserID); err != nil {
		return 0, "", "", "", err
	}

	name := fmt.Sprintf("tg-%d", u.TelegramID)
	if u.TelegramUsername != "" {
		name = u.TelegramUsername
	}
	hUID, hSub, err := s.Hiddify.CreateUser(ctx, name, s.Cfg.DeviceLimit)
	if err != nil {
		return 0, "", "", "", err
	}
	mtSec, err := s.MT.GenerateSecret()
	if err != nil {
		return 0, "", "", "", err
	}
	if err := s.MT.AddSecret(mtSec); err != nil {
		return 0, "", "", "", err
	}
	mtL := buildMTProxyLink(s.Cfg.MtproxyPublicHost, s.Cfg.MtproxyPublicPort, mtSec)

	tx, err := s.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, "", "", "", err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		UPDATE secrets SET is_active = 0, revoked_at = datetime('now') WHERE id = ?`, old.ID); err != nil {
		return 0, "", "", "", err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secret_events (secret_id, event_type) VALUES (?, 'reissued')`, old.ID); err != nil {
		return 0, "", "", "", err
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO secrets (user_id, hiddify_user_id, hiddify_link, mtproxy_secret, mtproxy_link, is_active)
		VALUES (?, ?, ?, ?, ?, 1)`, userID, hUID, hSub, mtSec, mtL)
	if err != nil {
		return 0, "", "", "", err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return 0, "", "", "", err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secret_events (secret_id, event_type) VALUES (?, 'connected')`, newID); err != nil {
		return 0, "", "", "", err
	}
	if warningID != nil {
		if _, err := tx.ExecContext(ctx, `
			UPDATE warnings SET status = 'reissued', admin_action = 'reissue', resolved_at = datetime('now')
			WHERE id = ?`, *warningID); err != nil {
			return 0, "", "", "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, "", "", "", err
	}

	if mtL == "" {
		mtL = "(задайте MTPROXY_PUBLIC_HOST и MTPROXY_PUBLIC_PORT в .env.local)"
	}
	return u.TelegramID, u.TelegramUsername, hSub, mtL, nil
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
