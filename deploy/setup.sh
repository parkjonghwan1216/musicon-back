#!/bin/bash
set -e

# 사전 조건 검증
command -v docker >/dev/null 2>&1 || { echo "Docker가 설치되지 않았습니다."; exit 1; }
docker compose version >/dev/null 2>&1 || { echo "Docker Compose V2가 설치되지 않았습니다."; exit 1; }

echo "=== Musicon Docker 초기 세팅 ==="

# 1. 기존 systemd 서비스 중지 및 제거
if systemctl is-active --quiet musicon 2>/dev/null; then
    echo "기존 systemd 서비스 중지..."
    sudo systemctl stop musicon
    sudo systemctl disable musicon
    sudo rm -f /etc/systemd/system/musicon.service
    sudo systemctl daemon-reload
    echo "  systemd 서비스 제거 완료"
fi

# 2. 디렉토리 생성
sudo mkdir -p /opt/musicon/data
sudo mkdir -p /opt/musicon/logs
echo "  /opt/musicon/data, logs 디렉토리 생성 완료"

# 3. deploy/.env 템플릿 생성
if [ ! -f /opt/musicon/deploy/.env ]; then
    sudo mkdir -p /opt/musicon/deploy
    sudo tee /opt/musicon/deploy/.env > /dev/null <<'EOF'
# Server
SERVER_PORT=7847
DATABASE_URL=/app/data/musicon.db
BASE_URL=http://<YOUR_SERVER_IP>:7847
BLEVE_INDEX_PATH=/app/data/bleve_index

# TJ Fetcher
TJ_API_BASE_URL=https://www.tjmedia.com/legacy/api/newSongOfMonth

# Spotify OAuth
SPOTIFY_CLIENT_ID=
SPOTIFY_CLIENT_SECRET=

# YouTube/Google OAuth
YOUTUBE_CLIENT_ID=
YOUTUBE_CLIENT_SECRET=
YTMUSIC_SCRIPT=/app/scripts/ytmusic_fetch.py
EOF
    echo "  deploy/.env 템플릿 생성 완료 — BASE_URL과 OAuth 값을 입력하세요!"
fi

# 4. docker-compose.prod.yml 배치
sudo cp docker-compose.prod.yml /opt/musicon/
echo "  docker-compose.prod.yml 복사 완료"

# 5. Docker Compose 실행 (ghcr.io 로그인은 GitHub Actions에서 매 배포마다 수행)
cd /opt/musicon
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d

echo ""
echo "=== 초기 세팅 완료 ==="
echo "헬스체크: curl http://localhost:7847/health"
echo "로그 확인: docker compose -f /opt/musicon/docker-compose.prod.yml logs -f"
