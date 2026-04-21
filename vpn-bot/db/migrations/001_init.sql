CREATE TABLE IF NOT EXISTS users (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_id       INTEGER UNIQUE NOT NULL,
    telegram_username TEXT,
    status            TEXT DEFAULT 'pending',
                      -- pending / active / banned
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    approved_by       INTEGER,
    approved_at       DATETIME
);

CREATE TABLE IF NOT EXISTS secrets (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id          INTEGER NOT NULL REFERENCES users(id),
    hiddify_user_id  TEXT,
    hiddify_link     TEXT,
    mtproxy_secret   TEXT,
    mtproxy_link     TEXT,
    is_active        BOOLEAN DEFAULT TRUE,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    revoked_at       DATETIME
);

CREATE TABLE IF NOT EXISTS secret_events (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    secret_id        INTEGER NOT NULL REFERENCES secrets(id),
    event_type       TEXT NOT NULL,
                     -- connected / disconnected / warning / revoked / reissued
    ip_address       TEXT,
    unique_ips_count INTEGER,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS warnings (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    secret_id    INTEGER NOT NULL REFERENCES secrets(id),
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at   DATETIME,
    status       TEXT DEFAULT 'pending',
                 -- pending / reissued / revoked / ignored / expired
    admin_action TEXT,
    resolved_at  DATETIME
);
