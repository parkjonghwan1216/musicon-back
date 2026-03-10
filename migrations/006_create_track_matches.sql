CREATE TABLE IF NOT EXISTS track_matches (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    music_track_id INTEGER NOT NULL UNIQUE REFERENCES music_tracks(id) ON DELETE CASCADE,
    song_id        INTEGER REFERENCES songs(id),
    match_score    REAL NOT NULL DEFAULT 0.0,
    status         TEXT NOT NULL DEFAULT 'unmatched'
);

CREATE INDEX IF NOT EXISTS idx_track_matches_song_id
    ON track_matches (song_id);

CREATE INDEX IF NOT EXISTS idx_track_matches_status
    ON track_matches (status);
