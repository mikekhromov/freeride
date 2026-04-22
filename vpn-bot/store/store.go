package store

import (
	"context"
	"database/sql"
)

type Store struct {
	DB *sql.DB
}

type User struct {
	ID               int64
	TelegramID       int64
	TelegramUsername string
	HiddifyUUID      string
	Status           string
	ApprovedBy       sql.NullInt64
	ApprovedAt       sql.NullTime
}

func nullIfEmpty(u string) any {
	if u == "" {
		return nil
	}
	return u
}

func (s *Store) SetPendingRequest(ctx context.Context, telegramID int64, username string) error {
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO users (telegram_id, telegram_username, status)
		VALUES (?, ?, 'pending')
		ON CONFLICT(telegram_id) DO UPDATE SET
			telegram_username = COALESCE(excluded.telegram_username, users.telegram_username),
			status = CASE
				WHEN users.status = 'active' THEN users.status
				WHEN users.status = 'banned' THEN users.status
				ELSE 'pending' END
	`, telegramID, nullIfEmpty(username))
	return err
}

func (s *Store) GetUserByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, telegram_id, COALESCE(telegram_username,''), COALESCE(hiddify_uuid,''), status,
		       approved_by, approved_at
		FROM users WHERE telegram_id = ?`, telegramID)
	var u User
	var ap sql.NullInt64
	var aa sql.NullTime
	if err := row.Scan(&u.ID, &u.TelegramID, &u.TelegramUsername, &u.HiddifyUUID, &u.Status, &ap, &aa); err != nil {
		return nil, err
	}
	u.ApprovedBy = ap
	u.ApprovedAt = aa
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, telegram_id, COALESCE(telegram_username,''), COALESCE(hiddify_uuid,''), status,
		       approved_by, approved_at
		FROM users WHERE id = ?`, id)
	var u User
	var ap sql.NullInt64
	var aa sql.NullTime
	if err := row.Scan(&u.ID, &u.TelegramID, &u.TelegramUsername, &u.HiddifyUUID, &u.Status, &ap, &aa); err != nil {
		return nil, err
	}
	u.ApprovedBy = ap
	u.ApprovedAt = aa
	return &u, nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, telegram_id, COALESCE(telegram_username,''), COALESCE(hiddify_uuid,''), status,
		       approved_by, approved_at
		FROM users WHERE LOWER(telegram_username) = LOWER(?)`, username)
	var u User
	var ap sql.NullInt64
	var aa sql.NullTime
	if err := row.Scan(&u.ID, &u.TelegramID, &u.TelegramUsername, &u.HiddifyUUID, &u.Status, &ap, &aa); err != nil {
		return nil, err
	}
	u.ApprovedBy = ap
	u.ApprovedAt = aa
	return &u, nil
}

func (s *Store) RejectPendingUser(ctx context.Context, userID int64) (ok bool, err error) {
	res, err := s.DB.ExecContext(ctx, `UPDATE users SET status = 'banned' WHERE id = ? AND status = 'pending'`, userID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *Store) StatsByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT status, COUNT(*) FROM users GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]int)
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return nil, err
		}
		m[st] = n
	}
	return m, rows.Err()
}

func (s *Store) ListUsersRecent(ctx context.Context, limit int) ([]User, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, telegram_id, COALESCE(telegram_username,''), COALESCE(hiddify_uuid,''), status, approved_by, approved_at
		FROM users ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.TelegramUsername, &u.HiddifyUUID, &u.Status, &u.ApprovedBy, &u.ApprovedAt); err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

func (s *Store) SetUserBanned(ctx context.Context, userID int64) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET status = 'banned', hiddify_uuid = NULL WHERE id = ?`, userID)
	return err
}

func (s *Store) UserStatsByUsername(ctx context.Context, username string) (tgID int64, status string, active int, err error) {
	u, err := s.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, "", 0, err
	}
	active = 0
	if u.Status == "active" && u.HiddifyUUID != "" {
		active = 1
	}
	return u.TelegramID, u.Status, active, nil
}

func (s *Store) ActivateUser(ctx context.Context, userID int64, adminTelegramID int64, hiddifyUUID string) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE users
		SET status = 'active',
		    hiddify_uuid = ?,
		    approved_by = ?,
		    approved_at = datetime('now')
		WHERE id = ?`, hiddifyUUID, adminTelegramID, userID)
	return err
}

func (s *Store) ClearHiddifyUUID(ctx context.Context, userID int64) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET hiddify_uuid = NULL WHERE id = ?`, userID)
	return err
}

func (s *Store) CountByStatus(ctx context.Context, status string) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE status = ?`, status).Scan(&n)
	return n, err
}

func (s *Store) CountActiveUsers(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE status = 'active' AND hiddify_uuid IS NOT NULL`).Scan(&n)
	return n, err
}

func (s *Store) GetUserByHiddifyUUID(ctx context.Context, uuid string) (*User, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, telegram_id, COALESCE(telegram_username,''), COALESCE(hiddify_uuid,''), status,
		       approved_by, approved_at
		FROM users WHERE hiddify_uuid = ?`, uuid)
	var u User
	var ap sql.NullInt64
	var aa sql.NullTime
	if err := row.Scan(&u.ID, &u.TelegramID, &u.TelegramUsername, &u.HiddifyUUID, &u.Status, &ap, &aa); err != nil {
		return nil, err
	}
	u.ApprovedBy = ap
	u.ApprovedAt = aa
	return &u, nil
}

func (s *Store) SetUserStatus(ctx context.Context, userID int64, status string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET status = ? WHERE id = ?`, status, userID)
	return err
}

func (s *Store) EnsureUser(ctx context.Context, telegramID int64, username string) error {
	return s.SetPendingRequest(ctx, telegramID, username)
}

func (s *Store) HasUser(ctx context.Context, telegramID int64) (bool, error) {
	_, err := s.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
