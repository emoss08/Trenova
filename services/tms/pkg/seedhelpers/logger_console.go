package seedhelpers

import (
	"fmt"
	"maps"
	"sync"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/fatih/color"
)

type ConsoleSeedLogger struct {
	stats    *LogStats
	verbose  bool
	minLevel LogLevel
	mu       sync.Mutex
}

var _ SeedLogger = (*ConsoleSeedLogger)(nil)

func NewConsoleSeedLogger(verbose bool) *ConsoleSeedLogger {
	return &ConsoleSeedLogger{
		stats:    NewLogStats(),
		verbose:  verbose,
		minLevel: LogLevelInfo,
	}
}

func (l *ConsoleSeedLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
}

func (l *ConsoleSeedLogger) EntityCreated(table string, id pulid.ID, description string) {
	l.mu.Lock()
	l.stats.EntitiesCount[table]++
	l.mu.Unlock()

	if l.verbose {
		color.Green("  ✓ Created %s: %s (%s)", table, description, id)
	}
}

func (l *ConsoleSeedLogger) EntityQueried(table string, id pulid.ID) {
	l.mu.Lock()
	l.stats.QueriesCount++
	l.mu.Unlock()

	if l.verbose {
		l.Debug("  → Queried %s: %s", table, id)
	}
}

func (l *ConsoleSeedLogger) CacheHit(key string) {
	l.mu.Lock()
	l.stats.CacheHits++
	l.mu.Unlock()

	if l.verbose {
		color.Cyan("  ⚡ Cache hit: %s", key)
	}
}

func (l *ConsoleSeedLogger) CacheMiss(key string) {
	l.mu.Lock()
	l.stats.CacheMisses++
	l.mu.Unlock()

	if l.verbose {
		l.Debug("  ○ Cache miss: %s", key)
	}
}

func (l *ConsoleSeedLogger) BulkInsert(table string, count int) {
	l.mu.Lock()
	l.stats.EntitiesCount[table] += count
	l.mu.Unlock()

	if l.verbose {
		color.Green("  ✓ Bulk insert %s: %d records", table, count)
	}
}

func (l *ConsoleSeedLogger) Debug(format string, args ...any) {
	if !l.verbose || l.minLevel > LogLevelDebug {
		return
	}
	fmt.Printf(format+"\n", args...)
}

func (l *ConsoleSeedLogger) Info(format string, args ...any) {
	if l.minLevel > LogLevelInfo {
		return
	}
	fmt.Printf(format+"\n", args...)
}

func (l *ConsoleSeedLogger) Warn(format string, args ...any) {
	if l.minLevel > LogLevelWarn {
		return
	}
	color.Yellow(format, args...)
}

func (l *ConsoleSeedLogger) Error(format string, args ...any) {
	if l.minLevel > LogLevelError {
		return
	}
	color.Red(format, args...)
}

func (l *ConsoleSeedLogger) PrintStats() {
	if !l.verbose {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	fmt.Println("  Statistics:")

	if l.stats.CacheHits > 0 || l.stats.CacheMisses > 0 {
		hitRate := l.stats.CacheHitRate()
		fmt.Printf("    Cache: %d hits, %d misses (%.1f%% hit rate)\n",
			l.stats.CacheHits, l.stats.CacheMisses, hitRate)
	}

	if len(l.stats.EntitiesCount) > 0 {
		fmt.Print("    Entities:")
		for table, count := range l.stats.EntitiesCount {
			fmt.Printf(" %s=%d", table, count)
		}
		fmt.Println()
	}

	if l.stats.QueriesCount > 0 {
		fmt.Printf("    Queries: %d\n", l.stats.QueriesCount)
	}
}

func (l *ConsoleSeedLogger) GetStats() *LogStats {
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
