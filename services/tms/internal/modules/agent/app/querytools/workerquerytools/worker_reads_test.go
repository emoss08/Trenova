package workerquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqllisttool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/graphqlread"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/filtercatalog"
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
			shapeWorkerRow(row, nil, filtercatalog.Clock{})

			assert.Equal(t, true, row[workerAssignable], "the hold flag is what the hold reads")
			assert.Equal(t, false, row[DispatchableField])
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
		shapeWorkerRow(row, nil, filtercatalog.Clock{})

		assert.Equal(t, false, row[workerAssignable])
		assert.Equal(t, false, row[DispatchableField])
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
		shapeWorkerRow(row, nil, filtercatalog.Clock{})

		assert.Equal(t, false, row[DispatchableField])
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
		shapeWorkerRow(row, nil, filtercatalog.Clock{})

		assert.Equal(t, false, row[DispatchableField])
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
		shapeWorkerRow(row, nil, filtercatalog.Clock{})

		assert.Equal(t, true, row[DispatchableField])
		assert.NotContains(t, row, workerNotDispatchBy)
	})

	t.Run("a withheld flag leaves the standing unsaid", func(t *testing.T) {
		t.Parallel()

		row := map[string]any{workerStatusField: "Active"}
		shapeWorkerRow(row, nil, filtercatalog.Clock{})

		assert.NotContains(t, row, DispatchableField)
	})

	t.Run("a credential date with nothing on file is present and null", func(t *testing.T) {
		t.Parallel()

		profile := map[string]any{workerComplianceKey: "Compliant"}
		row := map[string]any{workerAssignable: true, workerProfileField: profile}
		shapeWorkerRow(
			row,
			map[string]struct{}{"profile.medicalCardExpiry": {}},
			filtercatalog.Clock{},
		)

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
	shapeCredentialWorker(row, nil, filtercatalog.Clock{})

	held, _ := row[fieldWorker].(map[string]any)
	assert.Equal(t, false, held[DispatchableField])
	assert.Equal(t, "already off dispatch (Medical certificate expired)", held[workerNotDispatchBy])
}

func TestShapeCredentialWorker_LeavesDispatchableUnsaidWithoutTheProfile(t *testing.T) {
	t.Parallel()

	withheld := map[string]struct{}{fieldWorker + "." + workerProfileField: {}}
	onDispatch := map[string]any{
		fieldWorker: map[string]any{workerAssignable: true, workerStatusField: "Active"},
	}
	shapeCredentialWorker(onDispatch, withheld, filtercatalog.Clock{})

	held, _ := onDispatch[fieldWorker].(map[string]any)
	assert.NotContains(t, held, DispatchableField,
		"compliance is on the withheld profile, so the driver is not called dispatchable")

	offDispatch := map[string]any{
		fieldWorker: map[string]any{workerAssignable: false, workerStatusField: "Active"},
	}
	shapeCredentialWorker(offDispatch, withheld, filtercatalog.Clock{})

	held, _ = offDispatch[fieldWorker].(map[string]any)
	assert.Equal(t, false, held[DispatchableField])
	assert.Equal(t, "already off dispatch", held[workerNotDispatchBy])
}

func TestWorkerList_DispatchableFiltersByHoldEmploymentAndCompliance(t *testing.T) {
	t.Parallel()

	reader := &querytooltest.KeyedReader{}
	tool, err := graphqllisttool.New(
		querytooltest.ListSpecNamed(t, ReadSpecs(), querytool.ToolWorkerList),
		reader,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)

	assert.Contains(t, tool.ParamSchema()["properties"], DispatchableField)
	assert.Contains(t, tool.Description(), DispatchableField)

	_, err = tool.Query(
		t.Context(),
		querytooltest.DispatchCall(map[string]any{DispatchableField: true}),
	)
	require.NoError(t, err)
	input := querytooltest.SentInput(t, reader)
	assert.Equal(t, []map[string]any{
		{"field": "canBeAssigned", "operator": "eq", "value": true},
		{"field": "availableForDispatch", "operator": "eq", "value": true},
		{"field": "assignmentBlocked", "operator": "isnull"},
		{"field": "status", "operator": "eq", "value": "Active"},
		{"field": "profile.complianceStatus", "operator": "ne", "value": "NonCompliant"},
	}, input[graphqlread.InputFieldFilters])
	assert.NotContains(t, input, graphqlread.InputFilterGroups)

	reader = &querytooltest.KeyedReader{}
	tool, err = graphqllisttool.New(
		querytooltest.ListSpecNamed(t, ReadSpecs(), querytool.ToolWorkerList),
		reader,
		&mocks.DenyResourcesPermissionEngine{},
	)
	require.NoError(t, err)
	_, err = tool.Query(
		t.Context(),
		querytooltest.DispatchCall(map[string]any{DispatchableField: false}),
	)
	require.NoError(t, err)
	input = querytooltest.SentInput(t, reader)
	assert.NotContains(t, input, graphqlread.InputFieldFilters)
	assert.Equal(t, []map[string]any{{"filters": []map[string]any{
		{"field": "canBeAssigned", "operator": "eq", "value": false},
		{"field": "availableForDispatch", "operator": "eq", "value": false},
		{"field": "assignmentBlocked", "operator": "isnotnull"},
		{"field": "status", "operator": "ne", "value": "Active"},
		{"field": "profile.complianceStatus", "operator": "eq", "value": "NonCompliant"},
	}}}, input[graphqlread.InputFilterGroups])

	_, err = tool.Query(
		t.Context(),
		querytooltest.DispatchCall(map[string]any{DispatchableField: "yes"}),
	)
	require.ErrorContains(t, err, "must be true or false")
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
