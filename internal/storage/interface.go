package storage

import "sitewatch/internal/models"

// Storage interface for pluggable storage backends
type Storage interface {
	AddPingLog(log models.PingLog) error
	GetFilteredLogs(siteID string, success *bool, limit int) ([]models.PingLog, error)
	GetAllLogs() ([]models.PingLog, error)
	Close() error
}

// CreateStorage creates a storage instance based on configuration
func CreateStorage(config models.Config) (Storage, error) {
	switch config.Storage.Type {
	case "sqlite":
		return NewSQLiteStorage(config.Storage.SQLitePath)
	case "memory":
		fallthrough
	default:
		maxLogs := config.Storage.MaxMemoryLogs
		if maxLogs <= 0 {
			maxLogs = 1000
		}
		return NewMemoryStorage(maxLogs), nil
	}
}