package agentmemoryrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestLeadWords_AreTheWordsLongEnoughToBeginACompanysName(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]string{"deliver", "acme", "this", "friday", "30pm"},
		leadWords("Can we deliver to ACME this Friday at 4:30pm? acme!"),
	)
	assert.Empty(t, leadWords("is it ok"))
}

// "acme" reaches Acme Manufacturing when no other customer with memories
// begins with it, and "granite" reaches none of four Granite customers: a
// short name is followed only when it points at one record.
func TestSubjectsLedBy_KeepsOnlyAFirstWordOneRecordBeginsWith(t *testing.T) {
	t.Parallel()

	acme := pulid.MustNew("cus_")
	acmeDC := pulid.MustNew("loc_")
	graniteBrands, graniteTextiles := pulid.MustNew("cus_"), pulid.MustNew("cus_")

	refs := SubjectsLedBy([]ledSubjectRow{
		{SubjectType: agentmemory.MemorySubjectCustomer, SubjectID: acme, Lead: "acme"},
		{SubjectType: agentmemory.MemorySubjectLocation, SubjectID: acmeDC, Lead: "acme"},
		{SubjectType: agentmemory.MemorySubjectCustomer, SubjectID: graniteBrands, Lead: "granite"},
		{SubjectType: agentmemory.MemorySubjectCustomer, SubjectID: graniteTextiles, Lead: "granite"},
	})

	assert.Equal(t, []agentmemoryports.MemorySubjectRef{
		{Type: agentmemory.MemorySubjectCustomer, ID: acme},
		{Type: agentmemory.MemorySubjectLocation, ID: acmeDC},
	}, refs, "one customer and one location may share a first word; two customers may not")
}
