package filtercatalog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCriteria_EmptyNoteNeverClaimsTheWholeSetFromAFilteredList(t *testing.T) {
	t.Parallel()

	filtered := NewCriteria("inbound EDI files")
	filtered.Field("status", "Quarantined")
	filtered.Field("partner", "edip_1")
	assert.Equal(t, "No inbound EDI files matched status Quarantined and partner edip_1. "+
		"That is the answer for these filters: report it as what they found, never as there "+
		"being no inbound EDI files at all, and search again with fewer filters only when the "+
		"question is about more than they cover.", filtered.EmptyNote())

	assert.Equal(t, "No inbound EDI files are visible to you in this organization. This is an "+
		"unfiltered list, so the result is the whole set, not a near miss.",
		NewCriteria("inbound EDI files").EmptyNote())
}
