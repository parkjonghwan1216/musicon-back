CREATE TABLE IF NOT EXISTS music_tracks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id   INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    provider    TEXT NOT NULL,
    external_id TEXT NOT NULL,
    title       TEXT NOT NULL,
    artist      TEXT NOT NULL,
    album_name  TEXT NOT NULL DEFAULT '',
    image_url   TEXT NOT NULL DEFAULT '',
    synced_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_music_tracks_device_provider_external
    ON music_tracks (device_id, provider, external_id);

CREATE INDEX IF NOT EXISTS idx_music_tracks_device_id
    ON music_tracks (device_id);
