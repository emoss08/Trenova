package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func kindedIDSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"recordId": agenttoolschema.KindID(
				"The qualification record, from list_worker_qualifications. Never guess one.",
				permission.KindEmploymentVerification, permission.KindClearinghouseQuery,
			),
			"drawIds": agenttoolschema.KindIDs(
				"The rounds, from list_dot_random_draws.", 5, permission.KindDOTRandomDraw,
			),
		},
	}
}

// A qualification is a verification or a clearinghouse query, each with its
// own prefix, so its id parameter names both kinds rather than a resource. An
// id of either kind passes.
func TestContractCall_AcceptsAnIDOfAnyMarkedKind(t *testing.T) {
	t.Parallel()

	for _, id := range []string{
		"wemv_01J9Z3QK8M5T7V2X4Y6W0R1N8P",
		"wchq_01J9Z3QK8M5T7V2X4Y6W0R1N8P",
	} {
		_, err := contractCall(toolschema.NewValidator(), "stub", kindedIDSchema(),
			map[string]any{
				"recordId": id,
				"drawIds":  []any{"drdraw_01J9Z3QK8M5T7V2X4Y6W0R1N8P"},
			})
		require.NoError(t, err, id)
	}
}

// An id of another kind is refused naming what it is and every kind the
// parameter takes, and where the right one comes from.
func TestContractCall_RefusesAnIDOfNoMarkedKind(t *testing.T) {
	t.Parallel()

	_, err := contractCall(toolschema.NewValidator(), "stub", kindedIDSchema(),
		map[string]any{
			"recordId": "wcred_01J9Z3QK8M5T7V2X4Y6W0R1N8P",
			"drawIds": []any{
				"drdraw_01J9Z3QK8M5T7V2X4Y6W0R1N8P",
				"drpool_01J9Z3QK8M5T7V2X4Y6W0R1N8P",
			},
		})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	problems := argumentProblems(multiErr)
	require.Len(t, problems, 2)
	assert.Contains(t, problems[0], `drawIds[1]: "drpool_01J9Z3QK8M5T7V2X4Y6W0R1N8P" is a `+
		`DOT random pool id; this parameter takes a DOT random draw id, from `+
		`list_dot_random_draws.`)
	assert.Contains(t, problems[1], `recordId: "wcred_01J9Z3QK8M5T7V2X4Y6W0R1N8P" is a `+
		`worker credential id; this parameter takes an employment verification or a `+
		`clearinghouse query id, from list_worker_qualifications.`)
}

// A prefix no kind carries is named as neither kind, with the prefixes the
// parameter's kinds do carry.
func TestContractCall_RefusesAnUnknownPrefixNamingEveryKind(t *testing.T) {
	t.Parallel()

	_, err := contractCall(toolschema.NewValidator(), "stub", kindedIDSchema(),
		map[string]any{"recordId": "zzz_01J9Z3QK8M5T7V2X4Y6W0R1N8P"})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	problems := argumentProblems(multiErr)
	require.Len(t, problems, 1)
	assert.Contains(t, problems[0], `recordId: "zzz_01J9Z3QK8M5T7V2X4Y6W0R1N8P" is not an `+
		`employment verification or a clearinghouse query id: their ids start with "wemv_" `+
		`or "wchq_".`)
}

// The kinds mark is read from a schema decoded from JSON as well, where the
// list holds values of any type.
func TestMarkedKinds_ReadsADecodedList(t *testing.T) {
	t.Parallel()

	kinds := markedKinds(map[string]any{
		toolschema.KeyRecordKinds: []any{"employment_verification", "clearinghouse_query"},
	})

	assert.Equal(t, []permission.RecordKind{
		permission.KindEmploymentVerification, permission.KindClearinghouseQuery,
	}, kinds)
}
