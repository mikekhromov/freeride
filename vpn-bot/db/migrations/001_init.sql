CREATE TABLE IF NOT EXISTS users (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_id       INTEGER UNIQUE NOT NULL,
    telegram_username TEXT,
    hiddify_uuid      TEXT,
    status            TEXT DEFAULT 'pending',
                      -- pending / active / banned
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    approved_by       INTEGER,
    approved_at       DATETIME
);
