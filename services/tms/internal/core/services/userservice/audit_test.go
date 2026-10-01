package userservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestReplaceOrganizationMembershipsRecordsWhatChanged(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	recorder := &securityaudittest.Recorder{}
	deps.svc.auditor = recorder

	actorID, userID := pulid.MustNew("usr_"), pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	kept, removed, added := pulid.MustNew("org_"), pulid.MustNew("org_"), pulid.MustNew("org_")

	deps.repo.EXPECT().ListOrganizationMemberships(mock.Anything, userID, buID).
		Return([]*tenant.OrganizationMembership{{OrganizationID: kept}, {OrganizationID: removed}}, nil).Once()
	deps.repo.EXPECT().ReplaceOrganizationMemberships(mock.Anything, repositories.ReplaceOrganizationMembershipsRequest{
		ActorID:         actorID,
		UserID:          userID,
		BusinessUnitID:  buID,
		OrganizationIDs: []pulid.ID{kept, added},
	}).Return([]*tenant.OrganizationMembership{{OrganizationID: kept}, {OrganizationID: added}}, nil).Once()

	_, err := deps.svc.ReplaceOrganizationMemberships(t.Context(), actorID, userID, kept, buID, []pulid.ID{kept, added})
	require.NoError(t, err)

	change := recorder.Only(t)
	assert.Equal(t, permission.ResourceUser, change.Resource)
	assert.Equal(t, userID.String(), change.ResourceID)
	assert.Equal(t, actorID, change.Actor.UserID)
	assert.Equal(t, kept, change.OrganizationID)
	assert.Equal(t, []string{added.String()}, change.Metadata["addedOrganizationIds"])
	assert.Equal(t, []string{removed.String()}, change.Metadata["removedOrganizationIds"])
}
