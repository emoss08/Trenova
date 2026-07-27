package resolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// denyingPermissionEngine refuses every check, standing in for a role that lacks the
// permission under test.
type denyingPermissionEngine struct {
	mocks.AllowAllPermissionEngine
	request *servicesport.PermissionCheckRequest
}

func (e *denyingPermissionEngine) Check(
	_ context.Context,
	req *servicesport.PermissionCheckRequest,
) (*servicesport.PermissionCheckResult, error) {
	e.request = req
	return &servicesport.PermissionCheckResult{Allowed: false}, nil
}

func dispatchAuthContext(t *testing.T) context.Context {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")

	return gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})
}

// Every console entry point must be gated. A read-only role must not be able to move work
// onto a driver, and the assign and unassign operations are separately permissioned so an
// organization can grant one without the other.
func TestDispatchResolvers_RequireTheCorrectPermission(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		operation permission.Operation
		invoke    func(ctx context.Context, r *Resolver) error
	}{
		{
			name:      "board requires read",
			operation: permission.OpRead,
			invoke: func(ctx context.Context, r *Resolver) error {
				_, err := (&queryResolver{r}).DispatchBoard(ctx, gqlmodel.DispatchBoardInput{})
				return err
			},
		},
		{
			name:      "move candidates require read",
			operation: permission.OpRead,
			invoke: func(ctx context.Context, r *Resolver) error {
				_, err := (&queryResolver{r}).DispatchMoveCandidates(
					ctx,
					gqlmodel.DispatchMoveCandidatesInput{MoveID: pulid.MustNew("sm_").String()},
				)
				return err
			},
		},
		{
			name:      "driver moves require read",
			operation: permission.OpRead,
			invoke: func(ctx context.Context, r *Resolver) error {
				_, err := (&queryResolver{r}).DispatchDriverMoves(
					ctx,
					gqlmodel.DispatchDriverMovesInput{WorkerID: pulid.MustNew("wrk_").String()},
				)
				return err
			},
		},
		{
			name:      "assignment preview requires read",
			operation: permission.OpRead,
			invoke: func(ctx context.Context, r *Resolver) error {
				_, err := (&queryResolver{r}).DispatchAssignmentPreview(
					ctx,
					gqlmodel.DispatchAssignmentPreviewInput{
						MoveID:   pulid.MustNew("sm_").String(),
						WorkerID: pulid.MustNew("wrk_").String(),
					},
				)
				return err
			},
		},
		{
			name:      "assigning moves requires assign",
			operation: permission.OpAssign,
			invoke: func(ctx context.Context, r *Resolver) error {
				_, err := (&mutationResolver{r}).DispatchAssignMoves(
					ctx,
					[]*gqlmodel.DispatchAssignMoveInput{{
						MoveID:          pulid.MustNew("sm_").String(),
						PrimaryWorkerID: pulid.MustNew("wrk_").String(),
						TractorID:       pulid.MustNew("trac_").String(),
					}},
				)
				return err
			},
		},
		{
			name:      "unassigning moves requires unassign",
			operation: permission.OpUnassign,
			invoke: func(ctx context.Context, r *Resolver) error {
				_, err := (&mutationResolver{r}).DispatchUnassignMoves(
					ctx,
					[]string{pulid.MustNew("sm_").String()},
				)
				return err
			},
		},
		{
			// Planning is a read, but the same call applies the plan. Gating it on assign
			// stops a read-only role triggering writes through the apply flag.
			name:      "planning auto-assign requires assign",
			operation: permission.OpAssign,
			invoke: func(ctx context.Context, r *Resolver) error {
				_, err := (&mutationResolver{r}).DispatchPlanAutoAssign(
					ctx,
					gqlmodel.DispatchPlanInput{},
				)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := &denyingPermissionEngine{}
			err := tc.invoke(dispatchAuthContext(t), &Resolver{permissionEngine: engine})

			require.Error(t, err, "a denied permission must not fall through to the service")
			require.NotNil(t, engine.request)
			assert.Equal(t, permission.ResourceShipmentMove.String(), engine.request.Resource)
			assert.Equal(t, tc.operation, engine.request.Operation)
		})
	}
}

func TestDispatchResolvers_RejectUnauthenticatedCallers(t *testing.T) {
	t.Parallel()

	resolver := &Resolver{permissionEngine: &recordingPermissionEngine{}}

	_, err := (&queryResolver{resolver}).DispatchBoard(
		t.Context(),
		gqlmodel.DispatchBoardInput{},
	)
	require.Error(t, err)
}

func TestSummarizeBulkResults_CountsBothOutcomes(t *testing.T) {
	t.Parallel()

	summary := summarizeBulkResults([]*gqlmodel.DispatchAssignResult{
		{MoveID: "a", Success: true},
		{MoveID: "b", Success: false},
		{MoveID: "c", Success: true},
	})

	assert.Equal(t, 2, summary.Succeeded)
	assert.Equal(t, 1, summary.Failed)
	assert.Len(t, summary.Results, 3, "every move reports its own outcome")
}

// A failed assignment carries its validation errors back so the console renders them with
// the same badges it uses for a pre-flight.
func TestFindingsFromError_MapsValidationErrorsToFindings(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	multiErr.Add("hos", errortypes.ErrComplianceViolation, "Driver has no drive time remaining")
	multiErr.Add("mvrDueDate", errortypes.ErrInvalid, "MVR due date has passed")

	findings := findingsFromError(multiErr)

	require.Len(t, findings, 2)
	assert.Equal(t, string(dispatcheligibility.SeverityBlock), findings[0].Severity)
	assert.Equal(t, "Driver has no drive time remaining", findings[0].Message)
	assert.Equal(t, string(dispatcheligibility.SeverityWarn), findings[1].Severity)
}

func TestFindingsFromError_NonValidationErrorYieldsNoFindings(t *testing.T) {
	t.Parallel()

	assert.Empty(t, findingsFromError(errortypes.NewBusinessError("something broke")))
	assert.Empty(t, findingsFromError(nil))
}

func TestOptionalPulid(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("tr_")

	parsed, err := optionalPulid(nil)
	require.NoError(t, err)
	assert.Nil(t, parsed, "an absent optional ID is not an error")

	value := id.String()
	parsed, err = optionalPulid(&value)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, id, *parsed)

	empty := ""
	parsed, err = optionalPulid(&empty)
	require.NoError(t, err)
	assert.Nil(t, parsed)

	garbage := "not-an-id"
	_, err = optionalPulid(&garbage)
	assert.Error(t, err)
}
