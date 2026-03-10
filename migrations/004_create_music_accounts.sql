CREATE TABLE IF NOT EXISTS music_accounts (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id        INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    provider         TEXT NOT NULL,
    provider_user_id TEXT NOT NULL DEFAULT '',
    display_name     TEXT NOT NULL DEFAULT '',
    access_token     TEXT NOT NULL,
    refresh_token    TEXT NOT NULL DEFAULT '',
    token_expires_at DATETIME NOT NULL,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_music_accounts_device_provider
    ON music_accounts (device_id, provider);
