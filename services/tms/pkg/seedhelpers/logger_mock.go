package seedhelpers

import (
	"fmt"
	"maps"
	"strings"
	"sync"

	"github.com/emoss08/trenova/shared/pulid"
)

type MockSeedLogger struct {
	stats *LogStats
	logs  []string
	mu    sync.Mutex
}

var _ SeedLogger = (*MockSeedLogger)(nil)

func NewMockSeedLogger() *MockSeedLogger {
	return &MockSeedLogger{
		stats: NewLogStats(),
		logs:  make([]string, 0),
	}
}

func (l *MockSeedLogger) EntityCreated(table string, id pulid.ID, description string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.stats.EntitiesCount[table]++
	l.logs = append(l.logs, fmt.Sprintf("ENTITY_CREATED: %s %s %s", table, id, description))
}

func (l *MockSeedLogger) EntityQueried(table string, id pulid.ID) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.stats.QueriesCount++
	l.logs = append(l.logs, fmt.Sprintf("ENTITY_QUERIED: %s %s", table, id))
}

func (l *MockSeedLogger) CacheHit(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.stats.CacheHits++
	l.logs = append(l.logs, fmt.Sprintf("CACHE_HIT: %s", key))
}

func (l *MockSeedLogger) CacheMiss(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.stats.CacheMisses++
	l.logs = append(l.logs, fmt.Sprintf("CACHE_MISS: %s", key))
}

func (l *MockSeedLogger) BulkInsert(table string, count int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.stats.EntitiesCount[table] += count
	l.logs = append(l.logs, fmt.Sprintf("BULK_INSERT: %s %d", table, count))
}

func (l *MockSeedLogger) Debug(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, fmt.Sprintf("DEBUG: "+format, args...))
}

func (l *MockSeedLogger) Info(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, fmt.Sprintf("INFO: "+format, args...))
}

func (l *MockSeedLogger) Warn(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, fmt.Sprintf("WARN: "+format, args...))
}

func (l *MockSeedLogger) Error(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, fmt.Sprintf("ERROR: "+format, args...))
}

func (l *MockSeedLogger) PrintStats() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, "PRINT_STATS")
}

func (l *MockSeedLogger) GetStats() *LogStats {
	l.mu.Lock()
	defer l.mu.Unlock()

	statsCopy := &LogStats{
		CacheHits:     l.stats.CacheHits,
		CacheMisses:   l.stats.CacheMisses,
		EntitiesCount: make(map[string]int),
		QueriesCount:  l.stats.QueriesCount,
		StartTime:     l.stats.StartTime,
		DurationMs:    l.stats.DurationMs,
	}

	maps.Copy(statsCopy.EntitiesCount, l.stats.EntitiesCount)

	return statsCopy
}

func (l *MockSeedLogger) GetLogs() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	logsCopy := make([]string, len(l.logs))
	copy(logsCopy, l.logs)
	return logsCopy
}

func (l *MockSeedLogger) HasLog(substring string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, log := range l.logs {
		if strings.Contains(log, substring) {
			return true
		}
	}
	return false
}

func (l *MockSeedLogger) CountLogs(prefix string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	count := 0
	for _, log := range l.logs {
		if strings.HasPrefix(log, prefix) {
			count++
		}
	}
	return count
}

func (l *MockSeedLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logs = make([]string, 0)
	l.stats = NewLogStats()
}
