package workerinjuryservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/workerinjuryservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanInjury_NumbersWithoutEntering(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	planned, err := h.svc.PlanRecordInjury(t.Context(), h.newCase(), h.userID)
	require.NoError(t, err)
	assert.Equal(t, int32(1), planned.CaseNumber)
	assert.Equal(t, h.userID, planned.RecordedByID)
	assert.Empty(t, h.repo.injuries)

	created, err := h.svc.RecordInjury(t.Context(), h.newCase(), h.userID)
	require.NoError(t, err)

	next, err := h.svc.PlanRecordInjury(t.Context(), h.newCase(), h.userID)
	require.NoError(t, err)
	assert.Equal(t, int32(2), next.CaseNumber)

	part := "Left wrist"
	change, err := h.svc.PlanUpdateInjury(t.Context(), &workerinjuryservice.UpdateInjuryRequest{
		TenantInfo: h.tenant,
		InjuryID:   created.ID,
		BodyPart:   &part,
	})
	require.NoError(t, err)
	assert.Equal(t, part, change.After.BodyPart)
	assert.NotEqual(t, part, h.repo.injuries[created.ID].BodyPart)

	gone, err := h.svc.PlanDeleteInjury(t.Context(), h.tenant, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, gone.ID)
	assert.Len(t, h.repo.injuries, 1)
}
