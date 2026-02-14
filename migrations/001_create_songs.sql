CREATE TABLE IF NOT EXISTS songs (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    tj_number      INTEGER NOT NULL UNIQUE,
    title          TEXT NOT NULL,
    artist         TEXT NOT NULL,
    lyricist       TEXT NOT NULL DEFAULT '',
    composer       TEXT NOT NULL DEFAULT '',
    title_chosung  TEXT NOT NULL DEFAULT '',
    artist_chosung TEXT NOT NULL DEFAULT '',
    has_mv         BOOLEAN NOT NULL DEFAULT 0,
    published_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_songs_tj_number ON songs (tj_number);
CREATE INDEX IF NOT EXISTS idx_songs_title ON songs (title);
CREATE INDEX IF NOT EXISTS idx_songs_artist ON songs (artist);
CREATE INDEX IF NOT EXISTS idx_songs_title_chosung ON songs (title_chosung);
CREATE INDEX IF NOT EXISTS idx_songs_artist_chosung ON songs (artist_chosung);
