# Musicon Backend

TJ 노래방 플레이리스트 앱의 백엔드 API 서버 (Go + Fiber + SQLite)

## 사전 요구사항

- Go 1.25+

## 빠른 시작

### 1. 환경변수 설정

```bash
cp .env.example .env
```

`.env` 파일은 기본값 그대로 사용 가능합니다. `DATABASE_URL`을 설정하지 않으면 `musicon.db` 파일이 자동 생성됩니다.

### 2. 서버 실행

```bash
make run
```

또는:

```bash
go run ./cmd/server
```

서버가 시작되면 SQLite DB 파일이 자동 생성되고 마이그레이션이 적용됩니다.
`Server started on port 3000` 로그가 출력되면 정상입니다.

### 3. 동작 확인

새 터미널에서:

```bash
# 헬스체크
curl localhost:3000/health
# 기대 응답: {"status":"ok"}

# 곡 검색 (아직 데이터가 없으므로 빈 배열 반환)
curl "localhost:3000/api/songs/search?q=test"
# 기대 응답: {"data":[],"meta":{"count":0,"limit":20,"offset":0,"query":"test"},"success":true}

# TJ 번호로 조회 (데이터 없음)
curl localhost:3000/api/songs/12345
# 기대 응답: {"error":"song not found","success":false}
```

세 요청 모두 정상 응답이 오면 서버가 올바르게 동작하는 것입니다.

## 프로젝트 구조

```
cmd/server/main.go          # 서버 엔트리포인트
internal/
  config/config.go           # 환경변수 기반 설정
  domain/song.go             # Song 도메인 모델
  handler/                   # HTTP 핸들러 (health, song)
  repository/                # DB 접근 계층 (SQLite)
  service/                   # 비즈니스 로직
  fetcher/tj_fetcher.go      # TJ 노래방 데이터 수집기
migrations/                  # SQL 마이그레이션 파일
```

## Makefile 명령어

| 명령 | 설명 |
|------|------|
| `make build` | 바이너리 빌드 |
| `make run` | 서버 실행 |
| `make test` | 테스트 실행 |
| `make vet` | 정적 분석 |
| `make fmt` | 코드 포맷팅 |
