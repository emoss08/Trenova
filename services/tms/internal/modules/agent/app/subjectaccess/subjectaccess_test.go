package subjectaccess_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/subjectaccess"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errDenied = errors.New("no access to the subject")

// A conversation about a record whose kind the plan leaves out is refused in
// the plan's words, naming the add-on, not as a grant the person lacks.
func TestRefusal_NamesTheAddOnWhenThePlanLeavesTheSubjectOut(t *testing.T) {
	t.Parallel()

	engine := &agenttest.StubPermissions{OffPlan: map[string]permissionports.PlanDenial{
		permission.ResourceInsight.String(): permissionports.NewPlanDenial(
			platformcatalog.FeatureAIDesk,
		),
	}}

	err := subjectaccess.Refusal(
		t.Context(), engine, runtimetest.Actor(), agent.SubjectInsight, errDenied,
	)

	restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err)
	require.True(t, ok, "expected a plan restriction, got %v", err)
	assert.Equal(t, "Trenova AI", restriction.AddOn)
}

func TestRefusal_KeepsThePermissionWordingForAMissingGrant(t *testing.T) {
	t.Parallel()

	engine := &agenttest.StubPermissions{Denied: map[string]bool{
		permission.ResourceShipment.String() + ":" + string(permission.OpRead): true,
	}}

	err := subjectaccess.Refusal(
		t.Context(), engine, runtimetest.Actor(), agent.SubjectShipment, errDenied,
	)

	require.ErrorIs(t, err, errDenied)
	assert.False(t, errortypes.IsPlanRestrictionError(err))
}

func TestRefusal_AllowsAReadableSubject(t *testing.T) {
	t.Parallel()

	require.NoError(t, subjectaccess.Refusal(
		t.Context(), &agenttest.StubPermissions{}, runtimetest.Actor(),
		agent.SubjectShipment, errDenied,
	))
}
