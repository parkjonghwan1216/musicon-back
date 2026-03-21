# Stage 1: Go build (CGO_ENABLED=0 — modernc.org/sqlite is pure Go)
FROM golang:1.25-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o musicon-server ./cmd/server

# Stage 2: Python runtime + Go binary
FROM python:3.12-slim-bookworm
RUN apt-get update && apt-get install -y --no-install-recommends curl && \
    rm -rf /var/lib/apt/lists/* && \
    pip install --no-cache-dir ytmusicapi
RUN groupadd -r -g 997 appuser && useradd -r -u 997 -g appuser -d /app -s /sbin/nologin appuser
WORKDIR /app
COPY --from=builder /build/musicon-server .
COPY migrations/ migrations/
COPY scripts/ scripts/
RUN chown -R appuser:appuser /app
USER appuser
EXPOSE 7847
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD curl -sf http://localhost:7847/health || exit 1
CMD ["./musicon-server"]
