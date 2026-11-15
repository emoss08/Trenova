package auditservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/auditctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func supportContext(t *testing.T, grantID, principalID pulid.ID) *auditctx.Support {
	t.Helper()
	return &auditctx.Support{
		GrantID:         grantID.String(),
		PrincipalUserID: principalID.String(),
		StaffEmail:      "jane.doe@trenova.app",
		StaffName:       "Jane Doe",
	}
}

func TestLogAction_EveryEntryOfASupportSessionIsCreditedToTheGrant(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})
	grantID := pulid.MustNew("sag_")
	principalID := pulid.MustNew("sup_")
	ctx := auditctx.WithSupport(t.Context(), *supportContext(t, grantID, principalID))

	params := validLogActionParams()
	params.UserID = principalID
	require.NoError(t, srv.LogAction(ctx, params))

	entry := repo.entries()[0]
	assert.Equal(t, "support", entry.PrincipalType)
	assert.Equal(t, grantID, entry.PrincipalID, "the customer lists a grant's actions by its id")
	assert.Equal(t, principalID, entry.UserID)
	assert.Equal(t, audit.ChannelSupport, entry.Channel)
	assert.Equal(t, "Trenova support: Jane Doe (jane.doe@trenova.app)", entry.ActorName)
	assert.Equal(t, grantID.String(), entry.Metadata[audit.MetadataSupportGrantID])
	assert.Equal(t, "jane.doe@trenova.app", entry.Metadata[audit.MetadataSupportStaffEmail])
}

func TestLogAction_AnEntryAboutAnotherActorInASupportSessionStillNamesTheGrant(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})
	grantID := pulid.MustNew("sag_")
	ctx := auditctx.WithSupport(t.Context(), *supportContext(t, grantID, pulid.MustNew("sup_")))

	params := validLogActionParams()
	params.UserID = pulid.Nil
	params.PrincipalType = permissionports.PrincipalTypeSystem
	params.PrincipalID = permissionports.SystemPrincipalID
	require.NoError(t, srv.LogAction(ctx, params))

	entry := repo.entries()[0]
	assert.Equal(t, "system", entry.PrincipalType, "the system's own write stays the system's")
	assert.Equal(t, audit.ChannelSupport, entry.Channel)
	assert.Equal(t, grantID.String(), entry.Metadata[audit.MetadataSupportGrantID])
}

func TestLogAction_AnEntryOutsideASupportSessionIsNotMarked(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	require.NoError(t, srv.LogAction(t.Context(), validLogActionParams()))

	entry := repo.entries()[0]
	assert.Equal(t, "session_user", entry.PrincipalType)
	assert.NotContains(t, entry.Metadata, audit.MetadataSupportGrantID)
}

func TestLogAction_ABatchInASupportSessionMarksEveryEntry(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})
	grantID := pulid.MustNew("sag_")
	principalID := pulid.MustNew("sup_")
	ctx := auditctx.WithSupport(t.Context(), *supportContext(t, grantID, principalID))

	first, second := validLogActionParams(), validLogActionParams()
	first.UserID, second.UserID = principalID, principalID
	require.NoError(t, srv.LogActions(ctx, []auditports.BulkLogEntry{{Params: first}, {Params: second}}))

	require.Len(t, repo.entries(), 2)
	for _, entry := range repo.entries() {
		assert.Equal(t, grantID, entry.PrincipalID)
		assert.Equal(t, audit.ChannelSupport, entry.Channel)
	}
}
