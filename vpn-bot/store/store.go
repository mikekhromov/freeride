package store

import (
	"context"
	"database/sql"
	"time"
)

type Store struct {
	DB *sql.DB
}

type User struct {
	ID               int64
	TelegramID       int64
	TelegramUsername string
	Status           string
	ApprovedBy       sql.NullInt64
	ApprovedAt       sql.NullTime
}

type Secret struct {
	ID            int64
	UserID        int64
	HiddifyUserID string
	HiddifyLink   string
	MTProxySecret string
	MTProxyLink   string
	IsActive      bool
}

type Warning struct {
	ID        int64
	SecretID  int64
	ExpiresAt time.Time
	Status    string
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
		SELECT id, telegram_id, COALESCE(telegram_username,''), status,
		       approved_by, approved_at
		FROM users WHERE telegram_id = ?`, telegramID)
	var u User
	var ap sql.NullInt64
	var aa sql.NullTime
	if err := row.Scan(&u.ID, &u.TelegramID, &u.TelegramUsername, &u.Status, &ap, &aa); err != nil {
		return nil, err
	}
	u.ApprovedBy = ap
	u.ApprovedAt = aa
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, telegram_id, COALESCE(telegram_username,''), status,
		       approved_by, approved_at
		FROM users WHERE id = ?`, id)
	var u User
	var ap sql.NullInt64
	var aa sql.NullTime
	if err := row.Scan(&u.ID, &u.TelegramID, &u.TelegramUsername, &u.Status, &ap, &aa); err != nil {
		return nil, err
	}
	u.ApprovedBy = ap
	u.ApprovedAt = aa
	return &u, nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, telegram_id, COALESCE(telegram_username,''), status,
		       approved_by, approved_at
		FROM users WHERE LOWER(telegram_username) = LOWER(?)`, username)
	var u User
	var ap sql.NullInt64
	var aa sql.NullTime
	if err := row.Scan(&u.ID, &u.TelegramID, &u.TelegramUsername, &u.Status, &ap, &aa); err != nil {
		return nil, err
	}
	u.ApprovedBy = ap
	u.ApprovedAt = aa
	return &u, nil
}

func (s *Store) RejectPendingUser(ctx context.Context, userID int64) (ok bool, err error) {
	res, err := s.DB.ExecContext(ctx, `UPDATE users SET status = 'rejected' WHERE id = ? AND status = 'pending'`, userID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *Store) GetActiveSecretByUserID(ctx context.Context, userID int64) (*Secret, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, user_id,
		       COALESCE(hiddify_user_id,''), COALESCE(hiddify_link,''),
		       COALESCE(mtproxy_secret,''), COALESCE(mtproxy_link,''),
		       is_active
		FROM secrets WHERE user_id = ? AND is_active = 1
		ORDER BY id DESC LIMIT 1`, userID)
	var sec Secret
	var active int
	if err := row.Scan(&sec.ID, &sec.UserID, &sec.HiddifyUserID, &sec.HiddifyLink, &sec.MTProxySecret, &sec.MTProxyLink, &active); err != nil {
		return nil, err
	}
	sec.IsActive = active != 0
	return &sec, nil
}

func (s *Store) GetSecretByID(ctx context.Context, id int64) (*Secret, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, user_id,
		       COALESCE(hiddify_user_id,''), COALESCE(hiddify_link,''),
		       COALESCE(mtproxy_secret,''), COALESCE(mtproxy_link,''),
		       is_active
		FROM secrets WHERE id = ?`, id)
	var sec Secret
	var active int
	if err := row.Scan(&sec.ID, &sec.UserID, &sec.HiddifyUserID, &sec.HiddifyLink, &sec.MTProxySecret, &sec.MTProxyLink, &active); err != nil {
		return nil, err
	}
	sec.IsActive = active != 0
	return &sec, nil
}

func (s *Store) ListActiveSecrets(ctx context.Context) ([]struct {
	SecretID      int64
	UserID        int64
	MTProxySecret string
	TelegramID    int64
	TelegramUser  string
}, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT s.id, s.user_id, COALESCE(s.mtproxy_secret,''), u.telegram_id, COALESCE(u.telegram_username,'')
		FROM secrets s
		JOIN users u ON u.id = s.user_id
		WHERE s.is_active = 1 AND u.status = 'active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		SecretID      int64
		UserID        int64
		MTProxySecret string
		TelegramID    int64
		TelegramUser  string
	}
	for rows.Next() {
		var r struct {
			SecretID      int64
			UserID        int64
			MTProxySecret string
			TelegramID    int64
			TelegramUser  string
		}
		if err := rows.Scan(&r.SecretID, &r.UserID, &r.MTProxySecret, &r.TelegramID, &r.TelegramUser); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CountPendingWarningForSecret(ctx context.Context, secretID int64) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM warnings WHERE secret_id = ? AND status = 'pending'`, secretID).Scan(&n)
	return n, err
}

func (s *Store) InsertWarning(ctx context.Context, secretID int64, expiresAt time.Time) (int64, error) {
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO warnings (secret_id, expires_at, status) VALUES (?, ?, 'pending')`,
		secretID, expiresAt.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListPendingWarnings(ctx context.Context) ([]struct {
	ID         int64
	SecretID   int64
	TelegramID int64
	Username   string
	ExpiresAt  string
}, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT w.id, w.secret_id, u.telegram_id, COALESCE(u.telegram_username,''), w.expires_at
		FROM warnings w
		JOIN secrets s ON s.id = w.secret_id
		JOIN users u ON u.id = s.user_id
		WHERE w.status = 'pending'
		ORDER BY w.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		ID         int64
		SecretID   int64
		TelegramID int64
		Username   string
		ExpiresAt  string
	}
	for rows.Next() {
		var r struct {
			ID         int64
			SecretID   int64
			TelegramID int64
			Username   string
			ExpiresAt  string
		}
		if err := rows.Scan(&r.ID, &r.SecretID, &r.TelegramID, &r.Username, &r.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetWarningByID(ctx context.Context, id int64) (*Warning, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, secret_id, expires_at, status FROM warnings WHERE id = ?`, id)
	var w Warning
	var exp string
	if err := row.Scan(&w.ID, &w.SecretID, &exp, &w.Status); err != nil {
		return nil, err
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z07:00"} {
		if t, err := time.Parse(layout, exp); err == nil {
			w.ExpiresAt = t
			return &w, nil
		}
	}
	w.ExpiresAt = time.Time{}
	return &w, nil
}

func (s *Store) UpdateWarningStatus(ctx context.Context, id int64, status, adminAction string) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE warnings SET status = ?, admin_action = ?, resolved_at = datetime('now')
		WHERE id = ?`, status, adminAction, id)
	return err
}

func (s *Store) ListExpiredPendingWarnings(ctx context.Context) ([]int64, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id FROM warnings
		WHERE status = 'pending' AND datetime(expires_at) < datetime('now')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
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
		SELECT id, telegram_id, COALESCE(telegram_username,''), status, approved_by, approved_at
		FROM users ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.TelegramUsername, &u.Status, &u.ApprovedBy, &u.ApprovedAt); err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

func (s *Store) SetUserBanned(ctx context.Context, userID int64) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET status = 'banned' WHERE id = ?`, userID)
	return err
}

func (s *Store) UserStatsByUsername(ctx context.Context, username string) (tgID int64, status string, activeSecrets int, err error) {
	u, err := s.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, "", 0, err
	}
	n, err := s.CountActiveSecretsForUserID(ctx, u.ID)
	if err != nil {
		return u.TelegramID, u.Status, 0, err
	}
	return u.TelegramID, u.Status, n, nil
}

func (s *Store) CountActiveSecretsForUserID(ctx context.Context, userID int64) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM secrets WHERE user_id = ? AND is_active = 1`, userID).Scan(&n)
	return n, err
}

func (s *Store) CountActiveSecretsForTelegram(ctx context.Context, telegramID int64) (int, error) {
	u, err := s.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return 0, err
	}
	return s.CountActiveSecretsForUserID(ctx, u.ID)
}
