package archivejobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func day(d int) time.Time {
	return time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC)
}

func TestPruningKeepsTheNewestCompleteArchivesAndDropsTheRestWhole(t *testing.T) {
	t.Parallel()

	entries := []nightlyEntry{
		{prefix: "n/2026-11-20/", day: day(20), complete: false},
		{prefix: "n/2026-11-19/", day: day(19), complete: true},
		{prefix: "n/2026-11-18/", day: day(18), complete: false},
		{prefix: "n/2026-11-17/", day: day(17), complete: true},
		{prefix: "n/2026-11-16/", day: day(16), complete: true},
		{prefix: "n/2026-11-15/", day: day(15), complete: true},
	}
	assert.Equal(t, []string{"n/2026-11-18/", "n/2026-11-15/"}, prunable(entries, 3),
		"the newest three complete archives stay, tonight's partial one is left to finish")
	assert.Equal(t, []string{"n/2026-11-18/", "n/2026-11-17/", "n/2026-11-16/", "n/2026-11-15/"},
		prunable(entries, 1))
}

func TestPruningDeletesNothingWithoutANewerCompleteArchive(t *testing.T) {
	t.Parallel()

	entries := []nightlyEntry{
		{prefix: "n/2026-11-20/", day: day(20), complete: false},
		{prefix: "n/2026-11-19/", day: day(19), complete: false},
	}
	assert.Empty(t, prunable(entries, 14))
}
