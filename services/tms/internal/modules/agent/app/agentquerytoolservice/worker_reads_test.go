package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Fifteen drivers had an expired medical card. worker_list showed canBeAssigned
false for fourteen, folded from their credentials over the stored flag, and
Haiku proposed worker_place_dispatch_hold for each: thirteen were refused
"already off dispatch". canBeAssigned is now the flag the hold reads, and
dispatchable with notDispatchableBecause says the rest.
*/
func TestShapeWorkerRow(t *testing.T) {
	t.Parallel()

	t.Run("a NonCompliant driver with no hold keeps canBeAssigned and is not dispatchable",
		func(t *testing.T) {
			t.Parallel()

			row := map[string]any{
				workerAssignable:  true,
				workerStatusField: "Active",
				workerProfileField: map[string]any{
					workerComplianceKey: "NonCompliant",
					"licenseExpiry":     int64(1_800_000_000),
				},
			}
			shapeWorkerRow(row, nil, clock{})

			assert.Equal(t, true, row[workerAssignable], "the hold flag is what the hold reads")
			assert.Equal(t, false, row[workerDispatchable])
			assert.Equal(t, "credentials are not compliant", row[workerNotDispatchBy])
			assert.NotContains(t, row, workerBlockedField)
		})

	t.Run("a driver already off dispatch says so with its own reason", func(t *testing.T) {
		t.Parallel()

		row := map[string]any{
			workerAssignable:   false,
			workerStatusField:  "Active",
			workerBlockedField: "Medical certificate expired",
			workerProfileField: map[string]any{workerComplianceKey: "NonCompliant"},
		}
		shapeWorkerRow(row, nil, clock{})

		assert.Equal(t, false, row[workerAssignable])
		assert.Equal(t, false, row[workerDispatchable])
		assert.Equal(t,
			"already off dispatch (Medical certificate expired); credentials are not compliant",
			row[workerNotDispatchBy])
	})

	t.Run("an inactive driver is not dispatchable", func(t *testing.T) {
		t.Parallel()

		row := map[string]any{
			workerAssignable:   false,
			workerStatusField:  "Inactive",
			workerBlockedField: "Separated",
		}
		shapeWorkerRow(row, nil, clock{})

		assert.Equal(t, false, row[workerDispatchable])
		assert.Equal(t, "already off dispatch (Separated); not active (Inactive)",
			row[workerNotDispatchBy])
	})

	t.Run("a driver marked unavailable or blocked is not dispatchable", func(t *testing.T) {
		t.Parallel()

		row := map[string]any{
			workerAssignable:     true,
			workerAvailableField: false,
			workerBlockedField:   "Pending road test",
			workerStatusField:    "Active",
		}
		shapeWorkerRow(row, nil, clock{})

		assert.Equal(t, false, row[workerDispatchable])
		assert.Equal(t,
			"blocked from assignment (Pending road test); marked unavailable for dispatch",
			row[workerNotDispatchBy])
	})

	t.Run("a compliant active driver on dispatch is dispatchable", func(t *testing.T) {
		t.Parallel()

		row := map[string]any{
			workerAssignable:   true,
			workerStatusField:  "Active",
			workerProfileField: map[string]any{workerComplianceKey: "Compliant"},
		}
		shapeWorkerRow(row, nil, clock{})

		assert.Equal(t, true, row[workerDispatchable])
		assert.NotContains(t, row, workerNotDispatchBy)
	})

	t.Run("a withheld flag leaves the standing unsaid", func(t *testing.T) {
		t.Parallel()

		row := map[string]any{workerStatusField: "Active"}
		shapeWorkerRow(row, nil, clock{})

		assert.NotContains(t, row, workerDispatchable)
	})

	t.Run("a credential date with nothing on file is present and null", func(t *testing.T) {
		t.Parallel()

		profile := map[string]any{workerComplianceKey: "Compliant"}
		row := map[string]any{workerAssignable: true, workerProfileField: profile}
		shapeWorkerRow(row, map[string]struct{}{"profile.medicalCardExpiry": {}}, clock{})

		assert.Equal(t, true, row[workerAssignable])
		assert.Contains(t, profile, "hazmatExpiry")
		assert.Nil(t, profile["hazmatExpiry"])
		assert.NotContains(
			t,
			profile,
			"medicalCardExpiry",
			"a withheld date is not shown as missing",
		)
	})
}

func TestShapeCredentialWorker_SaysWhetherTheDriverIsOffDispatch(t *testing.T) {
	t.Parallel()

	row := map[string]any{
		"id": "wcred_1",
		fieldWorker: map[string]any{
			workerAssignable:   false,
			workerStatusField:  "Active",
			workerBlockedField: "Medical certificate expired",
		},
	}
	shapeCredentialWorker(row, nil, clock{})

	held, _ := row[fieldWorker].(map[string]any)
	assert.Equal(t, false, held[workerDispatchable])
	assert.Equal(t, "already off dispatch (Medical certificate expired)", held[workerNotDispatchBy])
}

func TestWorkerList_DispatchableFiltersByHoldEmploymentAndCompliance(t *testing.T) {
	t.Parallel()

	reader := &keyedReader{}
	tool, err := newGraphQLListTool(keyedSpec(t, toolWorkerList), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)

	assert.Contains(t, tool.ParamSchema()["properties"], workerDispatchable)
	assert.Contains(t, tool.Description(), workerDispatchable)

	_, err = tool.Query(t.Context(), dispatchCall(map[string]any{workerDispatchable: true}))
	require.NoError(t, err)
	input := sentInput(t, reader)
	assert.Equal(t, []map[string]any{
		{"field": "canBeAssigned", "operator": "eq", "value": true},
		{"field": "availableForDispatch", "operator": "eq", "value": true},
		{"field": "assignmentBlocked", "operator": "isnull"},
		{"field": "status", "operator": "eq", "value": "Active"},
		{"field": "profile.complianceStatus", "operator": "ne", "value": "NonCompliant"},
	}, input[inputFieldFilters])
	assert.NotContains(t, input, inputFilterGroups)

	reader = &keyedReader{}
	tool, err = newGraphQLListTool(keyedSpec(t, toolWorkerList), reader,
		&mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)
	_, err = tool.Query(t.Context(), dispatchCall(map[string]any{workerDispatchable: false}))
	require.NoError(t, err)
	input = sentInput(t, reader)
	assert.NotContains(t, input, inputFieldFilters)
	assert.Equal(t, []map[string]any{{"filters": []map[string]any{
		{"field": "canBeAssigned", "operator": "eq", "value": false},
		{"field": "availableForDispatch", "operator": "eq", "value": false},
		{"field": "assignmentBlocked", "operator": "isnotnull"},
		{"field": "status", "operator": "ne", "value": "Active"},
		{"field": "profile.complianceStatus", "operator": "eq", "value": "NonCompliant"},
	}}}, input[inputFilterGroups])

	_, err = tool.Query(t.Context(), dispatchCall(map[string]any{workerDispatchable: "yes"}))
	require.ErrorContains(t, err, "must be true or false")
}

func dispatchCall(params map[string]any) *agenttoolports.QueryToolParams {
	call := keyedCall(params)
	call.DataAccessCeiling = permission.SensitivityInternal

	return call
}

func sentInput(t *testing.T, reader *keyedReader) map[string]any {
	t.Helper()

	reader.mu.Lock()
	defer reader.mu.Unlock()
	require.NotEmpty(t, reader.requests)
	input, ok := reader.requests[0].Variables["input"].(map[string]any)
	require.True(t, ok)

	return input
}

func TestRankWorkerRowsPutsTheClosestNameFirst(t *testing.T) {
	t.Parallel()

	rows := []map[string]any{
		{workerFirstNameField: "Juana", workerLastNameField: "Smith"},
		{workerFirstNameField: "Juan", workerLastNameField: "Perez"},
		{workerFirstNameField: "Maria", workerLastNameField: "Perez"},
	}

	ranked := rankWorkerRows("juan perez", rows)

	assert.Equal(t, "Juan", ranked[0][workerFirstNameField])
	assert.Len(t, ranked, 3)
}
