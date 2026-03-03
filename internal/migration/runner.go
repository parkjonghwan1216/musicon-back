package migration

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

// RunAll reads all .sql files from the migrations directory and executes them
// in filename-sorted order. Each migration uses IF NOT EXISTS, making them
// safe to re-run.
func RunAll(db *sql.DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".sql" {
			files = append(files, entry.Name())
		}
	}

	sort.Strings(files)

	for _, file := range files {
		path := filepath.Join(migrationsDir, file)

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}

		log.Printf("Migration applied: %s", file)
	}

	return nil
}
