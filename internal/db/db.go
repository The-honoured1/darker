package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Site represents an indexed onion website.
type Site struct {
	ID          int
	URL         string
	Title       string
	Description string
	LastSeen    time.Time
	IsActive    bool
}

// Store handles database operations.
type Store struct {
	db *sql.DB
}

// NewStore initializes a new SQLite database and creates the necessary tables.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS sites (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT UNIQUE NOT NULL,
		title TEXT,
		description TEXT,
		last_seen DATETIME,
		is_active BOOLEAN
	);
	CREATE INDEX IF NOT EXISTS idx_sites_url ON sites(url);

	CREATE VIRTUAL TABLE IF NOT EXISTS sites_fts USING fts5(
		url,
		title,
		description,
		content='sites',
		content_rowid='id'
	);

	-- Triggers to keep FTS table in sync
	CREATE TRIGGER IF NOT EXISTS sites_ai AFTER INSERT ON sites BEGIN
		INSERT INTO sites_fts(rowid, url, title, description) VALUES (new.id, new.url, new.title, new.description);
	END;
	CREATE TRIGGER IF NOT EXISTS sites_ad AFTER DELETE ON sites BEGIN
		INSERT INTO sites_fts(sites_fts, rowid, url, title, description) VALUES('delete', old.id, old.url, old.title, old.description);
	END;
	CREATE TRIGGER IF NOT EXISTS sites_au AFTER UPDATE ON sites BEGIN
		INSERT INTO sites_fts(sites_fts, rowid, url, title, description) VALUES('delete', old.id, old.url, old.title, old.description);
		INSERT INTO sites_fts(rowid, url, title, description) VALUES (new.id, new.url, new.title, new.description);
	END;
	`
	_, err = db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &Store{db: db}, nil
}

// UpsertSite adds a new site or updates an existing one.
func (s *Store) UpsertSite(site *Site) error {
	query := `
	INSERT INTO sites (url, title, description, last_seen, is_active)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(url) DO UPDATE SET
		title = excluded.title,
		description = excluded.description,
		last_seen = excluded.last_seen,
		is_active = excluded.is_active
	`
	_, err := s.db.Exec(query, site.URL, site.Title, site.Description, site.LastSeen, site.IsActive)
	if err != nil {
		return fmt.Errorf("failed to upsert site: %w", err)
	}
	return nil
}

// SearchSites returns sites matching the query using FTS5.
func (s *Store) SearchSites(queryText string) ([]Site, error) {
	if queryText == "" {
		return nil, nil
	}

	rows, err := s.db.Query(`
		SELECT sites.id, sites.url, sites.title, sites.description, sites.last_seen, sites.is_active 
		FROM sites 
		JOIN sites_fts ON sites.id = sites_fts.rowid
		WHERE sites_fts MATCH ?
		ORDER BY sites_fts.rank, sites.last_seen DESC
	`, queryText)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var site Site
		err := rows.Scan(&site.ID, &site.URL, &site.Title, &site.Description, &site.LastSeen, &site.IsActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan site: %w", err)
		}
		sites = append(sites, site)
	}
	return sites, nil
}

// GetUnscannedSites returns a list of sites that haven't been successfully scanned.
func (s *Store) GetUnscannedSites(limit int) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT url FROM sites 
		WHERE is_active = 0 OR title IS NULL OR title = ''
		ORDER BY last_seen ASC 
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
