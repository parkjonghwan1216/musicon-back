package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	_ "modernc.org/sqlite"

	_ "musicon-back/docs"
	"musicon-back/internal/config"
	"musicon-back/internal/fetcher"
	"musicon-back/internal/handler"
	"musicon-back/internal/migration"
	"musicon-back/internal/notification"
	"musicon-back/internal/provider"
	"musicon-back/internal/repository"
	"musicon-back/internal/scheduler"
	"musicon-back/internal/service"
)

// @title       Musicon API
// @version     1.0
// @description TJ 노래방 곡 검색 API
// @host        158.179.160.120:7847
// @BasePath    /
func main() {
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

	songService := service.NewSongService(songRepo)
	deviceService := service.NewDeviceService(deviceRepo)
	reservationService := service.NewReservationService(reservationRepo, deviceRepo)
	musicAuthService := service.NewMusicAuthService(musicAccountRepo, musicTrackRepo, deviceRepo, providerRegistry)
	musicSyncService := service.NewMusicSyncService(musicAccountRepo, musicTrackRepo, trackMatchRepo, deviceRepo, providerRegistry, musicAuthService)
	musicQueryService := service.NewMusicQueryService(trackMatchRepo, deviceRepo)

	songHandler := handler.NewSongHandler(songService)
	deviceHandler := handler.NewDeviceHandler(deviceService)
	reservationHandler := handler.NewReservationHandler(reservationService)
	musicHandler := handler.NewMusicHandler(musicAuthService, musicSyncService, musicQueryService)

	app := fiber.New(fiber.Config{
		AppName: "Musicon API",
	})

	app.Use(recover.New())
	app.Use(logger.New())
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
	api.Get("/music/accounts", musicHandler.ListAccounts)
	api.Delete("/music/accounts/:provider", musicHandler.DisconnectAccount)
	api.Post("/music/sync", musicHandler.SyncTracks)
	api.Get("/music/matches", musicHandler.GetMatches)

	// TJ 최신곡 스케줄러 (12시간마다 = 하루 2회)
	tjFetcher := fetcher.NewTJFetcher(cfg.TJAPIBaseURL)
	pushService := notification.NewExpoPushService()
	matchingService := service.NewMatchingService(reservationRepo, pushService)
	songScheduler := scheduler.NewSongScheduler(tjFetcher, songRepo, matchingService, 12*time.Hour)
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

	songScheduler.Stop()

	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped")
}

