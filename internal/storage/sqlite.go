package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"sitewatch/internal/logger"
	"sitewatch/internal/models"
)

// SQLiteStorage implements SQLite-based persistent storage
type SQLiteStorage struct {
	db         *sql.DB
	logCounter int64
	mu         sync.RWMutex
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	storage := &SQLiteStorage{db: db}

	// Initialize database schema
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Get current max ID
	if err := storage.loadMaxID(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to load max ID: %w", err)
	}

	log := logger.Default().WithComponent("storage-sqlite")
	log.Info("SQLite storage initialized", "path", dbPath)
	return storage, nil
}

func (s *SQLiteStorage) initSchema() error {
	// Create table with extended ping statistics
	query := `
	CREATE TABLE IF NOT EXISTS ping_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		site_id TEXT NOT NULL,
		site_name TEXT NOT NULL,
		target TEXT NOT NULL,
		ip TEXT NOT NULL,
		success BOOLEAN NOT NULL,
		latency REAL,
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		
		-- Extended ping statistics
		packets_sent INTEGER DEFAULT 0,
		packets_recv INTEGER DEFAULT 0,
		packets_duplicates INTEGER DEFAULT 0,
		packet_loss REAL,
		min_latency REAL,
		max_latency REAL,
		jitter REAL
	);

	CREATE INDEX IF NOT EXISTS idx_timestamp ON ping_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_site_id ON ping_logs(site_id);
	CREATE INDEX IF NOT EXISTS idx_site_timestamp ON ping_logs(site_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_success ON ping_logs(success);
	CREATE INDEX IF NOT EXISTS idx_packet_loss ON ping_logs(packet_loss);
	CREATE INDEX IF NOT EXISTS idx_latency ON ping_logs(latency);
	`

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	
	// Add new columns to existing tables (migration)
	migrationQueries := []string{
		"ALTER TABLE ping_logs ADD COLUMN packets_sent INTEGER DEFAULT 0",
		"ALTER TABLE ping_logs ADD COLUMN packets_recv INTEGER DEFAULT 0", 
		"ALTER TABLE ping_logs ADD COLUMN packets_duplicates INTEGER DEFAULT 0",
		"ALTER TABLE ping_logs ADD COLUMN packet_loss REAL",
		"ALTER TABLE ping_logs ADD COLUMN min_latency REAL",
		"ALTER TABLE ping_logs ADD COLUMN max_latency REAL",
		"ALTER TABLE ping_logs ADD COLUMN jitter REAL",
	}
	
	// Execute migrations (ignore errors for existing columns)
	for _, migration := range migrationQueries {
		s.db.Exec(migration) // Ignore errors - column may already exist
	}
	
	return nil
}

func (s *SQLiteStorage) loadMaxID() error {
	var maxID sql.NullInt64
	err := s.db.QueryRow("SELECT MAX(id) FROM ping_logs").Scan(&maxID)
	if err != nil {
		return err
	}

	if maxID.Valid {
		s.logCounter = maxID.Int64
	}

	return nil
}

func (s *SQLiteStorage) AddPingLog(log models.PingLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	INSERT INTO ping_logs (
		timestamp, site_id, site_name, target, ip, success, latency, error,
		packets_sent, packets_recv, packets_duplicates, packet_loss,
		min_latency, max_latency, jitter
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(query,
		log.Timestamp,
		log.SiteID,
		log.SiteName,
		log.Target,
		log.IP,
		log.Success,
		log.Latency,
		log.Error,
		log.PacketsSent,
		log.PacketsRecv,
		log.PacketsDuplicates,
		log.PacketLoss,
		log.MinLatency,
		log.MaxLatency,
		log.Jitter,
	)

	if err != nil {
		return fmt.Errorf("failed to insert ping log: %w", err)
	}

	// Update log counter
	id, err := result.LastInsertId()
	if err == nil && id > s.logCounter {
		s.logCounter = id
	}

	return nil
}

func (s *SQLiteStorage) GetFilteredLogs(siteID string, success *bool, limit int) ([]models.PingLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var args []interface{}
	query := `SELECT id, timestamp, site_id, site_name, target, ip, success, latency, error,
		packets_sent, packets_recv, packets_duplicates, packet_loss,
		min_latency, max_latency, jitter 
		FROM ping_logs WHERE 1=1`

	if siteID != "" {
		query += " AND site_id = ?"
		args = append(args, siteID)
	}

	if success != nil {
		query += " AND success = ?"
		args = append(args, *success)
	}

	query += " ORDER BY timestamp DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query ping logs: %w", err)
	}
	defer rows.Close()

	var logs []models.PingLog
	for rows.Next() {
		var log models.PingLog
		var latency, packetLoss, minLatency, maxLatency, jitter sql.NullFloat64
		var errorMsg sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.SiteID,
			&log.SiteName,
			&log.Target,
			&log.IP,
			&log.Success,
			&latency,
			&errorMsg,
			&log.PacketsSent,
			&log.PacketsRecv,
			&log.PacketsDuplicates,
			&packetLoss,
			&minLatency,
			&maxLatency,
			&jitter,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan ping log: %w", err)
		}

		// Handle nullable float fields
		if latency.Valid {
			log.Latency = &latency.Float64
		}
		if errorMsg.Valid {
			log.Error = errorMsg.String
		}
		if packetLoss.Valid {
			log.PacketLoss = &packetLoss.Float64
		}
		if minLatency.Valid {
			log.MinLatency = &minLatency.Float64
		}
		if maxLatency.Valid {
			log.MaxLatency = &maxLatency.Float64
		}
		if jitter.Valid {
			log.Jitter = &jitter.Float64
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

func (s *SQLiteStorage) GetAllLogs() ([]models.PingLog, error) {
	return s.GetFilteredLogs("", nil, 0)
}

func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}