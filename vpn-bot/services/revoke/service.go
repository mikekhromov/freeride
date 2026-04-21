package revoke

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"freeride/vpn-bot/services/hiddify"
	"freeride/vpn-bot/services/mtproxy"
)

type Service struct {
	DB      *sql.DB
	Hiddify *hiddify.Client
	MTProxy *mtproxy.Manager
}

type Result struct {
	UserID        int64
	TelegramID    int64
	Username      string
	SecretRevoked bool
}

// RevokeByUsername — админ /revoke: отзыв секрета и статус banned.
func (s *Service) RevokeByUsername(ctx context.Context, username string) (*Result, error) {
	u := strings.TrimPrefix(strings.TrimSpace(username), "@")
	if u == "" {
		return nil, fmt.Errorf("пустой username")
	}
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, telegram_id, COALESCE(telegram_username,'')
		FROM users WHERE LOWER(telegram_username) = LOWER(?)`, u)
	var userID, tgID int64
	var tgUser string
	if err := row.Scan(&userID, &tgID, &tgUser); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("пользователь @%s не найден", u)
		}
		return nil, err
	}
	var secretID int64
	err := s.DB.QueryRowContext(ctx, `
		SELECT id FROM secrets WHERE user_id = ? AND is_active = 1 ORDER BY id DESC LIMIT 1`, userID).Scan(&secretID)
	if err == sql.ErrNoRows {
		return &Result{UserID: userID, TelegramID: tgID, Username: tgUser, SecretRevoked: false}, nil
	}
	if err != nil {
		return nil, err
	}
	res, err := s.RevokeBySecretID(ctx, secretID, true)
	if err != nil {
		return nil, err
	}
	res.Username = tgUser
	res.UserID = userID
	return res, nil
}

// RevokeBySecretID отзывает секрет по id. banUser=true — также users.status = banned (админский revoke).
func (s *Service) RevokeBySecretID(ctx context.Context, secretID int64, banUser bool) (*Result, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT s.id, s.user_id, s.is_active,
		       COALESCE(s.hiddify_user_id,''), COALESCE(s.mtproxy_secret,''),
		       u.telegram_id, COALESCE(u.telegram_username,'')
		FROM secrets s JOIN users u ON u.id = s.user_id
		WHERE s.id = ?`, secretID)
	var id, userID int64
	var active int
	var hUID, mtSec, tgUser string
	var tgID int64
	if err := row.Scan(&id, &userID, &active, &hUID, &mtSec, &tgID, &tgUser); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("секрет не найден")
		}
		return nil, err
	}
	if active == 0 {
		return &Result{UserID: userID, TelegramID: tgID, Username: tgUser, SecretRevoked: false}, nil
	}

	if err := s.MTProxy.RemoveSecret(mtSec); err != nil {
		return nil, fmt.Errorf("mtproxy: %w", err)
	}
	if err := s.Hiddify.DeactivateUser(ctx, hUID); err != nil {
		return nil, fmt.Errorf("hiddify: %w", err)
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		UPDATE secrets SET is_active = 0, revoked_at = datetime('now') WHERE id = ?`, secretID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secret_events (secret_id, event_type) VALUES (?, 'revoked')`, secretID); err != nil {
		return nil, err
	}
	if banUser {
		if _, err := tx.ExecContext(ctx, `
			UPDATE users SET status = 'banned' WHERE id = ?`, userID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Result{
		UserID:        userID,
		TelegramID:    tgID,
		Username:      tgUser,
		SecretRevoked: true,
	}, nil
}
