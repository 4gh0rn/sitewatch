package storage

import (
	"sync"

	"sitewatch/internal/models"
)

// MemoryStorage implements in-memory storage with ring buffer
type MemoryStorage struct {
	logs       []models.PingLog
	maxLogs    int
	logCounter int
	mu         sync.RWMutex
}

// NewMemoryStorage creates a new memory storage instance
func NewMemoryStorage(maxLogs int) *MemoryStorage {
	if maxLogs <= 0 {
		maxLogs = 1000 // Default
	}
	return &MemoryStorage{
		logs:    make([]models.PingLog, 0, maxLogs),
		maxLogs: maxLogs,
	}
}

func (m *MemoryStorage) AddPingLog(log models.PingLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logCounter++
	log.ID = m.logCounter

	// Ring buffer behavior
	if len(m.logs) >= m.maxLogs {
		// Remove oldest entry, add new one
		m.logs = append(m.logs[1:], log)
	} else {
		m.logs = append(m.logs, log)
	}

	return nil
}

func (m *MemoryStorage) GetFilteredLogs(siteID string, success *bool, limit int) ([]models.PingLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filtered []models.PingLog

	// Filter logs (newest first)
	for i := len(m.logs) - 1; i >= 0; i-- {
		log := m.logs[i]

		// Apply filters
		if siteID != "" && log.SiteID != siteID {
			continue
		}
		if success != nil && log.Success != *success {
			continue
		}

		filtered = append(filtered, log)

		// Apply limit
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}

	return filtered, nil
}

func (m *MemoryStorage) GetAllLogs() ([]models.PingLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return copy of all logs
	logs := make([]models.PingLog, len(m.logs))
	copy(logs, m.logs)
	return logs, nil
}

func (m *MemoryStorage) Close() error {
	// Nothing to close for memory storage
	return nil
}