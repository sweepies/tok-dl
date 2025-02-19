package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"time"

	_ "modernc.org/sqlite"
)

const (
	StatusComplete       = "complete"
	StatusRateLimited    = "rate_limited"
	StatusParseFailed    = "parse_failed"
	StatusUnknownError   = "unknown_error"
	StatusDownloadFailed = "download_failed"
	StatusNoMedia        = "no_media_urls"
)

type Database struct {
	db *sql.DB
}

// New creates a new database connection and initializes tables
func New(dbDir string) (*Database, error) {
	dbPath := path.Join(dbDir, "tiktok.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Initialize tables
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("error creating tables: %w", err)
	}

	return &Database{db: db}, nil
}

func createTables(db *sql.DB) error {
	// Create metadata table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS metadata (
			id TEXT PRIMARY KEY,
			url TEXT NOT NULL,
			data JSON NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Create cache table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cache (
			url TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// SaveMetadata stores TikTok post metadata
func (d *Database) SaveMetadata(id string, url string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}

	normalizedURL, err := normalizeURL(url)
	if err != nil {
		return fmt.Errorf("error normalizing URL: %w", err)
	}

	_, err = d.db.Exec(
		"INSERT OR REPLACE INTO metadata (id, url, data) VALUES (?, ?, ?)",
		id, normalizedURL, string(jsonData),
	)
	return err
}

// GetMetadata retrieves metadata for a TikTok post
func (d *Database) GetMetadata(id string) ([]byte, error) {
	var jsonData string
	err := d.db.QueryRow("SELECT data FROM metadata WHERE id = ?", id).Scan(&jsonData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return []byte(jsonData), nil
}

// IsProcessed checks if a URL has been processed and returns its status
func (d *Database) IsProcessed(url string) (bool, string, error) {
	normalizedURL, err := normalizeURL(url)
	if err != nil {
		return false, "", fmt.Errorf("error normalizing URL: %w", err)
	}

	var status string
	err = d.db.QueryRow("SELECT status FROM cache WHERE url = ?", normalizedURL).Scan(&status)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, status, nil
}

// MarkComplete marks a URL as successfully processed
func (d *Database) MarkComplete(url string) error {
	return d.setStatus(url, StatusComplete)
}

// MarkFailed marks a URL as failed with a specific status
func (d *Database) MarkFailed(url string, status string) error {
	return d.setStatus(url, status)
}

func (d *Database) setStatus(url string, status string) error {
	normalizedURL, err := normalizeURL(url)
	if err != nil {
		return fmt.Errorf("error normalizing URL: %w", err)
	}

	_, err = d.db.Exec(
		"INSERT OR REPLACE INTO cache (url, status, timestamp) VALUES (?, ?, ?)",
		normalizedURL, status, time.Now(),
	)
	return err
}

// normalizeURL ensures URLs are stored in a consistent format
func normalizeURL(inputURL string) (string, error) {
	parsed, err := url.Parse(inputURL)
	if err != nil {
		return "", err
	}

	// Remove any query parameters
	parsed.RawQuery = ""

	// Ensure proper scheme
	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}

	return parsed.String(), nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}
