package reporting

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportingtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreview_PassesTheCallersPolicyZoneAndSupersedeKey(t *testing.T) {
	t.Parallel()

	svc, engine := engineService(t, engineOptions{})
	request := planRequest(testUserID())

	_, err := svc.Preview(t.Context(), &PreviewRequest{
		Request:    request,
		Definition: storedRunDefinition(),
		Supersede:  true,
	})
	require.ErrorIs(t, err, reportingtest.ErrNotExecuted)

	previews := engine.Recorded("preview")
	require.Len(t, previews, 1)
	got := previews[0].Request
	access, ok := got.Policy.(*reportaccess.Access)
	require.True(t, ok)
	assert.Equal(t, reportaccess.ModeRead, access.Mode())
	assert.Equal(t, "America/Chicago", got.Location.String())
	assert.Equal(t, previewSupersedeKey+":"+request.TenantInfo.UserID.String(), got.Key,
		"a newer preview by the same person supersedes the last, and nobody else's")
}

func TestPreview_PagesWhenAPageSizeIsGiven(t *testing.T) {
	t.Parallel()

	svc, engine := engineService(t, engineOptions{})

	_, err := svc.Preview(t.Context(), &PreviewRequest{
		Request:    planRequest(testUserID()),
		Definition: storedRunDefinition(),
		Cursor:     "abc",
		PageSize:   25,
	})
	require.ErrorIs(t, err, reportingtest.ErrNotExecuted)

	pages := engine.Recorded("page")
	require.Len(t, pages, 1)
	assert.Equal(t, "abc", pages[0].Cursor)
	assert.Equal(t, 25, pages[0].PageSize)
	assert.Empty(t, engine.Recorded("preview"))

	_, err = svc.Preview(t.Context(), &PreviewRequest{
		Request:    planRequest(testUserID()),
		Definition: storedRunDefinition(),
		PageSize:   5000,
	})
	require.ErrorIs(t, err, reportingtest.ErrNotExecuted)
	pages = engine.Recorded("page")
	require.Len(t, pages, 2)
	assert.Equal(t, 40, pages[1].PageSize, "a page is never larger than a preview")
}

func TestPreview_RefusesAFieldOfAKindTheCallerCannotRead(t *testing.T) {
	t.Parallel()

	svc, _ := engineService(t, engineOptions{denied: []permission.Resource{
		permission.ResourceShipment,
	}})

	_, err := svc.Preview(t.Context(), &PreviewRequest{
		Request:    planRequest(testUserID()),
		Definition: storedRunDefinition(),
	})

	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
}

func TestDrillThrough_LeavesTheRowCapToThePreview(t *testing.T) {
	t.Parallel()

	for requested, want := range map[int]int{0: 0, 40: 0, 5000: 0, -1: 0, 10: 10} {
		svc, engine := engineService(t, engineOptions{})

		_, err := svc.DrillThrough(t.Context(), &DrillThroughRequest{
			Request:     planRequest(testUserID()),
			Definition:  storedRunDefinition(),
			Dimensions:  []DrillValue{{ColumnID: "c_customer", Value: "Acme"}},
			PivotValues: []string{"New"},
			Limit:       requested,
		})
		require.ErrorIs(t, err, reportingtest.ErrNotExecuted, requested)

		drills := engine.Recorded("drill")
		require.Len(t, drills, 1, requested)
		assert.Equal(t, want, drills[0].Drill.Limit, requested)
		assert.Equal(t, []string{"New"}, drills[0].Drill.PivotValues)
		assert.Equal(t, []ir.DrillValue{{Column: "c_customer", Value: "Acme"}}, drills[0].Row)
		assert.Same(t, drills[0].Request.Policy, drills[0].Drill.Policy,
			"the detail rows are decided by the same policy as the report")
	}
}

func TestEngineError_MapsEveryEngineCode(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cases := []struct {
		name  string
		err   error
		check func(t *testing.T, mapped error)
	}{
		{
			name: "invalid definitions name the part at fault and the closest names",
			err: &analytics.Error{Code: analytics.CodeInvalid, Details: ir.Errors{{
				Code:        ir.CodeUnknownField,
				Message:     `unknown field "statsu" on shipment; did you mean "status"?`,
				Pointer:     "/columns/2/dimension/field",
				Suggestions: []string{"status"},
			}}},
			check: func(t *testing.T, mapped error) {
				var multi *errortypes.MultiError
				require.ErrorAs(t, mapped, &multi)
				require.Len(t, multi.Errors, 1)
				assert.Equal(t, "definition.columns[2].dimension.field", multi.Errors[0].Field)
				assert.Contains(t, multi.Error(), "status")
			},
		},
		{
			name: "a busy organization is told to retry",
			err:  &analytics.Error{Code: analytics.CodeBusy},
			check: func(t *testing.T, mapped error) {
				var limited *errortypes.RateLimitError
				assert.ErrorAs(t, mapped, &limited)
			},
		},
		{
			name: "a superseded preview is a cancellation",
			err:  &analytics.Error{Code: analytics.CodeSuperseded},
			check: func(t *testing.T, mapped error) {
				assert.ErrorIs(t, mapped, context.Canceled)
			},
		},
		{
			name: "a report too costly to run is a business refusal",
			err:  &analytics.Error{Code: analytics.CodeTooCostly},
			check: func(t *testing.T, mapped error) {
				var business *errortypes.BusinessError
				assert.ErrorAs(t, mapped, &business)
			},
		},
		{
			name: "a denial without a policy still refuses",
			err:  &analytics.Error{Code: analytics.CodeDenied, Message: "no"},
			check: func(t *testing.T, mapped error) {
				var authz *errortypes.AuthorizationError
				assert.ErrorAs(t, mapped, &authz)
			},
		},
		{
			name: "anything else passes through",
			err:  errors.New("boom"),
			check: func(t *testing.T, mapped error) {
				assert.EqualError(t, mapped, "boom")
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.check(t, EngineError(ctx, nil, tc.err))
		})
	}
}

func TestDecodeDefinition_PointsAtTheFieldAtFault(t *testing.T) {
	t.Parallel()

	_, err := DecodeDefinition(map[string]any{"entity": "shipment", "columns": "nope"})

	var multi *errortypes.MultiError
	require.ErrorAs(t, err, &multi)
	require.NotEmpty(t, multi.Errors)

	def, err := DecodeDefinition(map[string]any{
		"entity":  "shipment",
		"columns": []any{map[string]any{"id": "n", "measure": map[string]any{"agg": "count"}}},
	})
	require.NoError(t, err)
	assert.Equal(t, ir.Version, def.Version)
}
