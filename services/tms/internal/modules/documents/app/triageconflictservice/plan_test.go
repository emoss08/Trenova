package triageconflictservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func duplicateConflict() inboundmessage.Conflict {
	return inboundmessage.Conflict{
		ID:                 "dup:imsga_rc",
		Kind:               inboundmessage.ConflictDuplicateDocument,
		AttachmentID:       "imsga_rc",
		ExistingShipmentID: "shp_104251",
	}
}

/*
A plan settles a copy of the conflict with the closure the write uses and saves
nothing: the store's mock expects no call, so a plan that wrote would fail.
*/
func TestPlanResolve_SettlesACopyTheWayResolveWould(t *testing.T) {
	t.Parallel()

	message := rateConMessage()
	message.Conflicts = []inboundmessage.Conflict{duplicateConflict()}
	r := newRig(t, message)

	plan, err := r.service.PlanResolve(t.Context(), &ResolveRequest{
		TenantInfo: tenantInfo,
		ActorID:    "usr_dana",
		MessageID:  "imsg_rc",
		ConflictID: "dup:imsga_rc",
		Resolution: inboundmessage.ConflictKeepBoth,
	})
	require.NoError(t, err)
	assert.Empty(t, plan.Before.Resolution)
	assert.Equal(t, inboundmessage.ConflictKeepBoth, plan.After.Resolution)
	assert.Equal(t, "usr_dana", plan.After.ResolvedBy.String())
	assert.Empty(t, message.Conflicts[0].Resolution, "the stored message is untouched")
	assert.Empty(t, r.activities.recorded)
	assert.Zero(t, r.notifier.changed)

	_, err = r.service.PlanResolve(t.Context(), &ResolveRequest{
		TenantInfo: tenantInfo,
		MessageID:  "imsg_rc",
		ConflictID: "dup:imsga_rc",
		Resolution: inboundmessage.ConflictSkipDuplicate,
	})
	require.Error(t, err, "a load's resolution does not settle a document conflict")

	_, err = r.service.PlanResolve(t.Context(), &ResolveRequest{
		TenantInfo: tenantInfo,
		MessageID:  "imsg_rc",
		ConflictID: "dup:nothing",
		Resolution: inboundmessage.ConflictKeepBoth,
	})
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestPlanReopen_RefusesAConflictStillOpen(t *testing.T) {
	t.Parallel()

	message := rateConMessage()
	message.Conflicts = []inboundmessage.Conflict{duplicateConflict()}
	r := newRig(t, message)
	req := &ReopenRequest{
		TenantInfo: tenantInfo,
		ActorID:    "usr_dana",
		MessageID:  "imsg_rc",
		ConflictID: "dup:imsga_rc",
	}

	_, err := r.service.PlanReopen(t.Context(), req)
	require.Error(t, err, "an open conflict has nothing to reopen")

	message.Conflicts[0].Resolution = inboundmessage.ConflictReplaceExisting
	message.Conflicts[0].ResolvedBy = "usr_dana"
	plan, err := r.service.PlanReopen(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, inboundmessage.ConflictReplaceExisting, plan.Before.Resolution)
	assert.Empty(t, plan.After.Resolution)
	assert.True(t, plan.After.ResolvedBy.IsNil())
	assert.Empty(t, r.activities.recorded)
}
