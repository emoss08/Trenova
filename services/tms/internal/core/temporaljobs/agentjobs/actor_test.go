package agentjobs

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

func actorActivities(t *testing.T) (*Activities, *mocks.MockUserRepository) {
	t.Helper()

	users := mocks.NewMockUserRepository(t)

	return NewActivities(ActivitiesParams{Logger: zap.NewNop(), Users: users}), users
}

func TestUnattendedActor_NamesTheSystemUserAndStaysAnAgent(t *testing.T) {
	t.Parallel()

	activities, users := actorActivities(t)
	system := pulid.MustNew("usr_")
	users.On("GetSystemUser", mock.Anything, []string{"id"}).
		Return(&tenant.User{ID: system}, nil).
		Once()
	scope := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	actor, err := activities.unattendedActor(t.Context(), scope)
	require.NoError(t, err)

	assert.Equal(t, system, actor.UserID, "writes are attributed to the system user")
	assert.Equal(t, serviceports.PrincipalTypeAgent, actor.PrincipalType,
		"authorization stays the agent's")
	assert.Equal(t, serviceports.AgentPrincipalID, actor.PrincipalID)
	assert.True(t, actor.IsAgent())
	assert.False(t, actor.IsUser())
	assert.Equal(t, pulid.Nil, actor.PersonUserID(), "the system user is not a person")
	assert.Equal(t, scope.OrgID, actor.OrganizationID)
	assert.Equal(t, scope.BuID, actor.BusinessUnitID)

	audit := actor.AuditActor()
	assert.Equal(t, serviceports.PrincipalTypeAgent, audit.PrincipalType)
	assert.Equal(t, serviceports.AgentPrincipalID, audit.PrincipalID)
}

func TestUnattendedActor_AMissingSystemUserFailsTheAttemptRetryably(t *testing.T) {
	t.Parallel()

	activities, users := actorActivities(t)
	users.On("GetSystemUser", mock.Anything, []string{"id"}).
		Return(nil, errors.New("connection refused")).
		Once()

	actor, err := activities.unattendedActor(
		t.Context(),
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	)

	require.Error(t, err)
	assert.Nil(t, actor)
	var applicationErr *temporal.ApplicationError
	require.ErrorAs(t, err, &applicationErr)
	assert.False(t, applicationErr.NonRetryable(), "the run is retried rather than run unnamed")
}
