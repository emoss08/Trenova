package seedhelpers

import (
	"time"

	"github.com/emoss08/trenova/shared/pulid"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

type SeedLogger interface {
	EntityCreated(table string, id pulid.ID, description string)
	EntityQueried(table string, id pulid.ID)
	CacheHit(key string)
	CacheMiss(key string)
	BulkInsert(table string, count int)
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	PrintStats()
}

type LogStats struct {
	CacheHits     int
	CacheMisses   int
	EntitiesCount map[string]int
	QueriesCount  int
	StartTime     time.Time
	DurationMs    int64
}

func NewLogStats() *LogStats {
	return &LogStats{
		EntitiesCount: make(map[string]int),
		StartTime:     time.Now(),
	}
}

func (s *LogStats) CacheHitRate() float64 {
	total := s.CacheHits + s.CacheMisses
	if total == 0 {
		return 0
	}
	return float64(s.CacheHits) / float64(total) * 100
}

func (s *LogStats) TotalEntities() int {
	total := 0
	for _, count := range s.EntitiesCount {
		total += count
	}
	return total
}

func (s *LogStats) Duration() time.Duration {
	return time.Since(s.StartTime)
}

func (s *LogStats) Finalize() {
	s.DurationMs = time.Since(s.StartTime).Milliseconds()
}

type NoOpLogger struct{}

var _ SeedLogger = (*NoOpLogger)(nil)

func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

func (l *NoOpLogger) EntityCreated(table string, id pulid.ID, description string) {}
func (l *NoOpLogger) EntityQueried(table string, id pulid.ID)                     {}
func (l *NoOpLogger) CacheHit(key string)                                         {}
func (l *NoOpLogger) CacheMiss(key string)                                        {}
func (l *NoOpLogger) BulkInsert(table string, count int)                          {}
func (l *NoOpLogger) Debug(format string, args ...any)                            {}
func (l *NoOpLogger) Info(format string, args ...any)                             {}
func (l *NoOpLogger) Warn(format string, args ...any)                             {}
func (l *NoOpLogger) Error(format string, args ...any)                            {}
func (l *NoOpLogger) PrintStats()                                                 {}
