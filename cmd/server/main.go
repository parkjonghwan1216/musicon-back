package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	_ "modernc.org/sqlite"

	_ "musicon-back/docs"
	"musicon-back/internal/config"
	"musicon-back/internal/handler"
	"musicon-back/internal/repository"
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

	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database")

	songRepo := repository.NewSQLiteSongRepository(db)
	songService := service.NewSongService(songRepo)
	songHandler := handler.NewSongHandler(songService)

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

	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped")
}

func runMigrations(db *sql.DB) error {
	migrationSQL, err := os.ReadFile("migrations/001_create_songs.sql")
	if err != nil {
		return err
	}

	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		return err
	}

	log.Println("Migrations applied successfully")
	return nil
}
