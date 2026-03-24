package seeder

import (
	"fmt"
	"time"

	"github.com/fatih/color"
)

type ProgressReporter interface {
	OnStart(total int)
	OnSeedStart(name string)
	OnSeedSkip(name, reason string)
	OnSeedComplete(name string, duration time.Duration)
	OnSeedError(name string, err error)
	OnComplete(applied, skipped, failed int, duration time.Duration)
}

type ConsoleReporter struct {
	total   int
	current int
	verbose bool
}

func NewConsoleReporter(verbose bool) *ConsoleReporter {
	return &ConsoleReporter{
		verbose: verbose,
	}
}

func (r *ConsoleReporter) OnStart(total int) {
	r.total = total
	r.current = 0
	if total == 0 {
		color.Yellow("→ No seeds to apply")
		return
	}
	color.Cyan("🌱 Applying %d seed(s)...\n", total)
}

func (r *ConsoleReporter) OnSeedStart(name string) {
	r.current++
	fmt.Printf("  [%d/%d] %s ", r.current, r.total, name)
}

func (r *ConsoleReporter) OnSeedSkip(name, reason string) {
	r.current++
	color.Yellow("  [%d/%d] %s → skipped (%s)\n", r.current, r.total, name, reason)
}

func (r *ConsoleReporter) OnSeedComplete(name string, duration time.Duration) {
	color.Green("✓ (%s)\n", formatDuration(duration))
}

func (r *ConsoleReporter) OnSeedError(name string, err error) {
	color.Red("✗ %v\n", err)
}

func (r *ConsoleReporter) OnComplete(applied, skipped, failed int, duration time.Duration) {
	fmt.Println()
	if failed > 0 {
		color.Yellow("→ Seeding completed with errors: %d applied, %d skipped, %d failed (%s)\n",
			applied, skipped, failed, formatDuration(duration))
		return
	}
	if applied == 0 && skipped > 0 {
		color.Yellow("→ No seeds applied (%d already up to date)\n", skipped)
		return
	}
	color.Green("✓ Seeding complete: %d applied, %d skipped (%s)\n",
		applied, skipped, formatDuration(duration))
}

type SilentReporter struct{}

func NewSilentReporter() *SilentReporter {
	return &SilentReporter{}
}

func (r *SilentReporter) OnStart(total int)                                        {}
func (r *SilentReporter) OnSeedStart(name string)                                  {}
func (r *SilentReporter) OnSeedSkip(name, reason string)                           {}
func (r *SilentReporter) OnSeedComplete(name string, duration time.Duration)       {}
func (r *SilentReporter) OnSeedError(name string, err error)                       {}
func (r *SilentReporter) OnComplete(applied, skipped, failed int, _ time.Duration) {}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
