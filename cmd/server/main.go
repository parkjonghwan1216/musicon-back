package main

import (
	"context"
	"database/sql"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	_ "modernc.org/sqlite"
	"gopkg.in/lumberjack.v2"

	_ "musicon-back/docs"
	"musicon-back/internal/config"
	"musicon-back/internal/fetcher"
	"musicon-back/internal/handler"
	"musicon-back/internal/migration"
	"musicon-back/internal/notification"
	"musicon-back/internal/provider"
	"musicon-back/internal/repository"
	"musicon-back/internal/scheduler"
	"musicon-back/internal/search"
	"musicon-back/internal/service"
)

// @title       Musicon API
// @version     1.0
// @description TJ 노래방 곡 검색 API
// @host        158.179.160.120:7847
// @BasePath    /
func main() {
	// 파일 + stdout 로그 설정
	fileLogger := &lumberjack.Logger{
		Filename:   "logs/musicon.log",
		MaxSize:    50, // MB
		MaxBackups: 10,
		MaxAge:     30, // 일
		Compress:   true,
	}
	multiWriter := io.MultiWriter(os.Stdout, fileLogger)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := sql.Open("sqlite", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Fatalf("Failed to enable WAL mode: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		log.Fatalf("Failed to enable foreign keys: %v", err)
	}

	if err := migration.RunAll(db, "migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database")

	songRepo := repository.NewSQLiteSongRepository(db)
	deviceRepo := repository.NewSQLiteDeviceRepository(db)
	reservationRepo := repository.NewSQLiteReservationRepository(db)
	musicAccountRepo := repository.NewSQLiteMusicAccountRepository(db)
	musicTrackRepo := repository.NewSQLiteMusicTrackRepository(db)
	trackMatchRepo := repository.NewSQLiteTrackMatchRepository(db)

	// Music providers
	spotifyProvider := provider.NewSpotifyProvider(cfg.SpotifyClientID, cfg.SpotifyClientSecret)
	youtubeProvider := provider.NewYouTubeProvider(cfg.YouTubeClientID, cfg.YouTubeClientSecret)
	providerRegistry := provider.NewRegistry(spotifyProvider, youtubeProvider)

	// Bleve 전문 검색 엔진 초기화
	var songSearcher search.SongSearcher
	var songIndexer search.SongIndexer
	var rebuildCancel context.CancelFunc
	var rebuildWg sync.WaitGroup

	bleveIndex, err := search.OpenOrCreateIndex(cfg.BleveIndexPath)
	if err != nil {
		log.Printf("[Search] Failed to open Bleve index, using SQL fallback: %v", err)
	} else {
		songSearcher = search.NewBleveSongSearcher(bleveIndex)
		songIndexer = search.NewBleveSongIndexer(bleveIndex, songRepo)

		// 인덱스가 비어 있으면 백그라운드에서 재구축
		docCount, _ := bleveIndex.DocCount()
		if docCount == 0 {
			var rebuildCtx context.Context
			rebuildCtx, rebuildCancel = context.WithCancel(context.Background())
			rebuildWg.Add(1)
			go func() {
				defer rebuildWg.Done()
				if err := songIndexer.RebuildFromDB(rebuildCtx); err != nil {
					if rebuildCtx.Err() == nil {
						log.Printf("[Search] Failed to rebuild Bleve index: %v", err)
					}
				}
			}()
		} else {
			log.Printf("[Search] Bleve index loaded: %d documents", docCount)
		}
	}

	songService := service.NewSongService(songRepo, songSearcher)
	deviceService := service.NewDeviceService(deviceRepo)
	reservationService := service.NewReservationService(reservationRepo, deviceRepo)
	musicAuthService := service.NewMusicAuthService(musicAccountRepo, musicTrackRepo, deviceRepo, providerRegistry)
	musicSyncService := service.NewMusicSyncService(musicAccountRepo, musicTrackRepo, trackMatchRepo, deviceRepo, providerRegistry, musicAuthService)
	musicQueryService := service.NewMusicQueryService(trackMatchRepo, deviceRepo)

	songHandler := handler.NewSongHandler(songService)
	deviceHandler := handler.NewDeviceHandler(deviceService)
	reservationHandler := handler.NewReservationHandler(reservationService)
	musicHandler := handler.NewMusicHandler(musicAuthService, musicSyncService, musicQueryService, cfg.BaseURL)

	app := fiber.New(fiber.Config{
		AppName: "Musicon API",
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Output: multiWriter,
	}))
	app.Use(cors.New())

	app.Get("/swagger/*", swagger.HandlerDefault)
	app.Get("/health", handler.HealthCheck)

	api := app.Group("/api")
	api.Get("/songs/search", songHandler.Search)
	api.Get("/songs/:number", songHandler.FindByTjNumber)

	api.Post("/devices/register", deviceHandler.Register)

	api.Post("/reservations", reservationHandler.Create)
	api.Get("/reservations", reservationHandler.List)
	api.Put("/reservations/:id", reservationHandler.Update)
	api.Delete("/reservations/:id", reservationHandler.Delete)

	api.Post("/music/spotify/connect", musicHandler.ConnectSpotify)
	api.Post("/music/youtube/connect", musicHandler.ConnectYouTube)
	api.Get("/auth/youtube/callback", musicHandler.YouTubeCallback)
	api.Get("/music/accounts", musicHandler.ListAccounts)
	api.Delete("/music/accounts/:provider", musicHandler.DisconnectAccount)
	api.Post("/music/sync", musicHandler.SyncTracks)
	api.Get("/music/matches", musicHandler.GetMatches)

	// TJ 최신곡 스케줄러 (12시간마다 = 하루 2회)
	tjFetcher := fetcher.NewTJFetcher(cfg.TJAPIBaseURL)
	pushService := notification.NewExpoPushService()
	matchingService := service.NewMatchingService(reservationRepo, pushService)
	songScheduler := scheduler.NewSongScheduler(tjFetcher, songRepo, matchingService, songIndexer, 12*time.Hour)
	songScheduler.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(":" + cfg.ServerPort); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server started on port %s", cfg.ServerPort)

	<-quit
	log.Println("Shutting down server...")

	// 1. HTTP 서버 종료 (진행 중인 요청 완료 대기)
	if err := app.Shutdown(); err != nil {
		log.Printf("Server shutdown failed: %v", err)
	}

	// 2. 스케줄러 종료
	songScheduler.Stop()

	// 3. Bleve 인덱스 재구축 취소 및 대기
	if rebuildCancel != nil {
		rebuildCancel()
	}
	rebuildWg.Wait()

	// 4. Bleve 인덱스 닫기 (모든 사용자가 완료된 후)
	if bleveIndex != nil {
		if err := bleveIndex.Close(); err != nil {
			log.Printf("[Search] Failed to close Bleve index: %v", err)
		}
	}

	log.Println("Server stopped")
}
