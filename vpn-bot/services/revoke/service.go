package revoke

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"freeride/vpn-bot/services/hiddify"
)

type Service struct {
	DB      *sql.DB
	Hiddify *hiddify.Client
}

type Result struct {
	UserID     int64
	TelegramID int64
	Username   string
	Revoked    bool
}

func (s *Service) RevokeByUsername(ctx context.Context, username string) (*Result, error) {
	u := strings.TrimPrefix(strings.TrimSpace(username), "@")
	if u == "" {
		return nil, fmt.Errorf("пустой username")
	}
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, telegram_id, COALESCE(telegram_username,''), COALESCE(hiddify_uuid,'')
		FROM users WHERE LOWER(telegram_username) = LOWER(?)`, u)
	var userID, tgID int64
	var tgUser, hiddifyUUID string
	if err := row.Scan(&userID, &tgID, &tgUser, &hiddifyUUID); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("пользователь @%s не найден", u)
		}
		return nil, err
	}

	if hiddifyUUID != "" {
		if err := s.Hiddify.DeleteUser(ctx, hiddifyUUID); err != nil {
			return nil, err
		}
	}

	if _, err := s.DB.ExecContext(ctx, `
		UPDATE users
		SET status = 'banned', hiddify_uuid = NULL
		WHERE id = ?`, userID); err != nil {
		return nil, err
	}
	return &Result{UserID: userID, TelegramID: tgID, Username: tgUser, Revoked: hiddifyUUID != ""}, nil
}
