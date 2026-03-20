-- 아티스트 전용 예약 (title = '') 지원을 위한 마이그레이션
-- reservations.title은 NOT NULL을 유지하되, 빈 문자열('')이 "아티스트 전용 구독"을 의미합니다.
-- SQLite에서 빈 문자열은 NOT NULL 제약에 위배되지 않으므로 스키마 변경 불필요.
--
-- reservation_notifications: 아티스트 전용 예약의 곡별 중복 알림 방지
CREATE TABLE IF NOT EXISTS reservation_notifications (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    reservation_id  INTEGER NOT NULL REFERENCES reservations(id) ON DELETE CASCADE,
    song_id         INTEGER NOT NULL,
    notified_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(reservation_id, song_id)
);
CREATE INDEX IF NOT EXISTS idx_reservation_notifications_reservation_id ON reservation_notifications (reservation_id);
