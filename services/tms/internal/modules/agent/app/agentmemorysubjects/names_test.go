package agentmemorysubjects

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A memory is never filed under a guess: a name resolves to the only record
// it finds or the only one it names exactly, and to nothing otherwise.
func TestNameResolver_ResolvesOnlyANameThatMeansOneRecord(t *testing.T) {
	t.Parallel()

	acme := agentmemoryports.MemorySubjectCandidate{
		ID:    pulid.MustNew("cus_"),
		Label: "Acme Manufacturing",
		Code:  "ACME",
	}
	foods := agentmemoryports.MemorySubjectCandidate{
		ID:    pulid.MustNew("cus_"),
		Label: "Acme Foods",
		Code:  "ACMF",
	}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	only := NewNameResolver(&fakeLinks{named: []agentmemoryports.MemorySubjectCandidate{acme}})
	id, err := only.Resolve(t.Context(), tenant, agentmemory.MemorySubjectCustomer, "acme")
	require.NoError(t, err)
	assert.Equal(t, acme.ID, id, "the only record found is the one meant")

	both := NewNameResolver(
		&fakeLinks{named: []agentmemoryports.MemorySubjectCandidate{acme, foods}},
	)
	id, err = both.Resolve(t.Context(), tenant, agentmemory.MemorySubjectCustomer, "Acme Foods")
	require.NoError(t, err)
	assert.Equal(t, foods.ID, id, "an exact name picks its record out of several")

	id, err = both.Resolve(t.Context(), tenant, agentmemory.MemorySubjectCustomer, "acme")
	require.NoError(t, err)
	assert.Equal(t, acme.ID, id, "a code names its record exactly")

	_, err = both.Resolve(t.Context(), tenant, agentmemory.MemorySubjectCustomer, "acm")
	require.ErrorContains(t, err, "Several customer records match")

	none := NewNameResolver(&fakeLinks{})
	_, err = none.Resolve(t.Context(), tenant, agentmemory.MemorySubjectCustomer, "Zenith")
	require.ErrorContains(t, err, "No customer is named")

	_, err = only.Resolve(t.Context(), tenant, "", "acme")
	require.Error(t, err, "a name needs the kind of record it names")
}
