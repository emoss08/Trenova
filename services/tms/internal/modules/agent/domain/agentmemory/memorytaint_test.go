package agentmemory

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

/*
Three instruction memories a person had approved still tainted every turn of
every agent in the organization, so nothing ever wrote on its own and nothing
said why. A person who reviews a tainted memory makes it the organization's
own: it says where it came from, but taints nothing and is followed.
*/
func TestMemory_AReviewedMemoryNoLongerTaints(t *testing.T) {
	t.Parallel()

	reviewer := pulid.MustNew("usr_")
	at := int64(1_791_600_000)
	memory := &Memory{ID: pulid.MustNew("amem_"), Tainted: true}

	assert.True(t, memory.Taints())
	assert.True(t, memory.DrawnFromOutside())
	assert.Len(t, memory.TaintedRecords(), 1)

	memory.ReviewedByUserID = &reviewer
	memory.ReviewedAt = &at

	assert.True(t, memory.Reviewed())
	assert.True(t, memory.Tainted, "where it came from is kept")
	assert.False(t, memory.Taints())
	assert.False(t, memory.DrawnFromOutside(), "followed as the organization's own")
	assert.Empty(t, memory.TaintedRecords())
	assert.False(t, (&Memory{ReviewedByUserID: &reviewer}).Reviewed(), "a review has a time")
}
