package archiveports_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/stretchr/testify/assert"
)

func TestARestoreNoteSaysWhatEachTrailEstablishedInOneLine(t *testing.T) {
	t.Parallel()

	assert.Empty(t, (&archiveports.TrailResult{
		AuditSeals: archiveports.ChainResult{Outcome: archiveports.ChainKept},
		AIAudit:    archiveports.ChainResult{Outcome: archiveports.ChainKept},
	}).RestoreNote(), "a restore that kept both trails has nothing to say")

	note := (&archiveports.TrailResult{
		AuditSeals: archiveports.ChainResult{Outcome: archiveports.ChainResealed, Attested: 40},
		AIAudit:    archiveports.ChainResult{Outcome: archiveports.ChainResealed, Marked: 6},
		Tail:       &archiveports.TailSeal{Rows: 3},
	}).RestoreNote()
	assert.Contains(t, note, "its 40 links were signed again here and are marked as verified at their source, as the archive states")
	assert.Contains(t, note, "its 6 links were signed again here and are marked as not verified at their source")
	assert.Contains(t, note, "3 audit log entries")
	assert.NotContains(t, note, "\n")
	assert.LessOrEqual(t, len(note), 500)
	assert.True(t, strings.HasPrefix(note, "Restored, but "))
}
