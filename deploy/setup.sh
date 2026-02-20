#!/bin/bash
set -e

echo "=== Musicon 배포 시작 ==="

# 사용자 생성
if ! id -u musicon &>/dev/null; then
    sudo useradd -r -s /usr/sbin/nologin musicon
    echo "  musicon 사용자 생성 완료"
fi

# 디렉토리 생성
sudo mkdir -p /opt/musicon/data
sudo mkdir -p /opt/musicon/migrations

# 파일 복사
sudo cp musicon-server /opt/musicon/
sudo cp musicon-fetch /opt/musicon/
sudo cp migrations/001_create_songs.sql /opt/musicon/migrations/
sudo chmod +x /opt/musicon/musicon-server
sudo chmod +x /opt/musicon/musicon-fetch

# 권한 설정
sudo chown -R musicon:musicon /opt/musicon

# systemd 서비스 등록
sudo cp musicon.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable musicon
sudo systemctl restart musicon

echo "=== 배포 완료 ==="
echo ""
sleep 2
sudo systemctl status musicon --no-pager
echo ""
echo "포트: 7847"
echo "헬스체크: curl localhost:7847/health"
