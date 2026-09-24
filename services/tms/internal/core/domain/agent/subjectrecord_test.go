package agent_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A subject type without a prefix is one whose ids cannot be checked, so a
// case filed against it would take any id at all.
func TestEverySubjectTypeHasItsOwnIDPrefix(t *testing.T) {
	t.Parallel()

	seen := make(map[string]agent.SubjectType, len(agent.AllSubjectTypes()))
	for _, subject := range agent.AllSubjectTypes() {
		prefix := subject.IDPrefix()
		require.NotEmptyf(t, prefix, "subject type %q has no id prefix", subject)
		require.Truef(t, prefix[len(prefix)-1] == '_', "prefix %q must end in _", prefix)

		other, taken := seen[prefix]
		require.Falsef(t, taken, "%q and %q share the prefix %q", subject, other, prefix)
		seen[prefix] = subject

		id := pulid.MustNew(prefix)
		kind, ok := agent.SubjectTypeOfID(id)
		require.True(t, ok)
		assert.Equal(t, subject, kind)
		require.NoError(t, subject.CheckID(id))

		assert.NotEqual(t, string(subject), subject.Noun(), "%q reads as a word", subject)
		assert.Contains(t, subject.NounWithArticle(), subject.Noun())
	}
}

// The case that started it: a saved report's id filed as an insight. The
// refusal has to say what the id is and what to send instead, or the model
// retries the same call.
func TestCheckID_NamesTheKindAnIDBelongsTo(t *testing.T) {
	t.Parallel()

	reportID := pulid.MustNew("rd_")

	err := agent.SubjectInsight.CheckID(reportID)
	require.Error(t, err)
	assert.Equal(
		t,
		reportID.String()+" is a report, not an insight; use subjectType Report",
		err.Error(),
	)
}

func TestCheckID_RefusesAnIDOfNoKnownKind(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")

	err := agent.SubjectEDIInboundFile.CheckID(customerID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not the id of an EDI inbound file")
	assert.Contains(t, err.Error(), `"ediinf_"`)
}

func TestCheckID_RefusesAnUnknownSubjectType(t *testing.T) {
	t.Parallel()

	err := agent.SubjectType("Spreadsheet").CheckID(pulid.MustNew("shp_"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Spreadsheet")
}
