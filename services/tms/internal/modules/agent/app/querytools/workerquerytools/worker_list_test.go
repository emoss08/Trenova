package workerquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgatetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
The question that started this: "which drivers hold a current hazmat
endorsement?" Endorsements live on the worker profile, not the worker, so
without a traversal filter the catalog could not express it at all — and the
model's only near-miss was the expiring-credentials list, which answers "whose is
about to lapse", a different question with a plausible-looking wrong answer.
*/
func TestListWorkers_CanFilterOnTheProfile(t *testing.T) {
	t.Parallel()

	spec := listtool.SpecOf(NewListWorkersTool(nil, nil))

	byName := make(map[string]filtercatalog.Field, len(spec.Fields))
	for _, field := range spec.Fields {
		byName[field.Name] = field
	}

	endorsement, ok := byName["profile.endorsement"]
	require.True(t, ok, "endorsements are on the profile, and the roster has to reach them")
	assert.Equal(t, filtercatalog.KindEnum, endorsement.Kind)

	for _, dated := range []string{"profile.hazmatExpiry", "profile.medicalCardExpiry"} {
		field, found := byName[dated]
		require.True(t, found, "%s is what makes 'current' answerable", dated)
		assert.Equal(t, filtercatalog.KindDate, field.Kind)
	}
}

/*
An endorsement is stored as a single letter: H is hazmat, X is tanker *and*
hazmat, N is tanker alone. No model guesses that, and one that guesses "Hazmat"
gets an empty page it will report as "nobody holds one". The values have to be
published with what they mean.
*/
func TestListWorkers_ExplainsTheEndorsementCodes(t *testing.T) {
	t.Parallel()

	spec := listtool.SpecOf(NewListWorkersTool(nil, nil))

	var endorsement filtercatalog.Field
	for _, field := range spec.Fields {
		if field.Name == "profile.endorsement" {
			endorsement = field
		}
	}

	assert.ElementsMatch(t, []string{"O", "N", "H", "X", "P", "T"}, endorsement.Values)
	assert.Contains(t, endorsement.Note, "hazmat")
	assert.Contains(t, endorsement.Note, "X",
		"the code that means both has to be spelled out or hazmat holders get undercounted")
}

func rosterWorker() *worker.Worker {
	return &worker.Worker{
		ID:        pulid.MustNew("wrk_"),
		FirstName: "Dana",
		LastName:  "Whitfield",
		City:      "Joliet",
		Profile:   &worker.WorkerProfile{Endorsement: worker.EndorsementType("X")},
	}
}

func TestListWorkers_WithholdsTheCityBelowTheCeiling(t *testing.T) {
	t.Parallel()

	tool := NewListWorkersTool(
		&querytooltest.FakeWorkerRepo{Items: []*worker.Worker{rosterWorker()}},
		&querytooltest.FakePermissions{Allowed: true},
	)

	result, err := tool.Query(t.Context(),
		querytooltest.AgentParams(map[string]any{}, permission.SensitivityInternal))
	require.NoError(t, err)
	rows, withheld := fieldgatetest.ListedRows[workerRow](t, result)
	require.Len(t, rows, 1)
	assert.Equal(t, "Dana Whitfield", rows[0].Name)
	assert.Empty(t, rows[0].City)
	assert.Equal(t, []string{"city"}, withheld)

	_, err = tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		"filters": []any{map[string]any{"field": "city", "operator": "eq", "value": "Joliet"}},
	}, permission.SensitivityInternal))
	require.Error(t, err, "filtering on a withheld field would reveal it")

	_, err = tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		"filters": []any{map[string]any{
			"field": "profile.endorsement", "operator": "eq", "value": "X",
		}},
	}, permission.SensitivityInternal))
	require.NoError(t, err, "a profile field is judged by the worker's own classification")

	result, err = tool.Query(t.Context(),
		querytooltest.AgentParams(map[string]any{}, permission.SensitivityRestricted))
	require.NoError(t, err)
	rows, withheld = fieldgatetest.ListedRows[workerRow](t, result)
	assert.Equal(t, "Joliet", rows[0].City)
	assert.Empty(t, withheld)
}
