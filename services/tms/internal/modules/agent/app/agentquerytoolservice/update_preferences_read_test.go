package agentquerytoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func preferencesAnswer(
	preference, recipients any,
	withheld ...string,
) map[string]*graphqlreadports.GraphQLReadResult {
	return map[string]*graphqlreadports.GraphQLReadResult{"DeskCustomerUpdatePreferences": {
		Data: map[string]any{"customer": map[string]any{
			"id": "cus_1", "name": "FreshHaul", "statusUpdatePreference": preference,
			"statusUpdateRecipients": recipients,
			"emailProfile": map[string]any{
				"id": "cep_1", "toRecipients": "ops@freshhaul.test",
			},
		}},
		Withheld: withheld,
	}}
}

func readPreferences(
	t *testing.T,
	answers map[string]*graphqlreadports.GraphQLReadResult,
) (map[string]any, *keyedReader) {
	t.Helper()

	reader := &keyedReader{answers: answers}
	spec := customerUpdatePreferencesGetSpec()
	tool, err := newGraphQLGetTool(&spec, reader, &mocks.DenyResourcesPermissionEngine{})
	require.NoError(t, err)
	data, err := tool.Query(t.Context(), keyedCall(map[string]any{
		paramCustomerID: pulid.MustNew("cus_").String(),
	}))
	require.NoError(t, err)
	record, ok := data.(map[string]any)
	require.True(t, ok)

	return record, reader
}

// A customer set to None that names no recipients reads as the repository
// tool read it: no routine notices, the warning, and the notice profile's
// recipients as the fallback, without the profile itself.
func TestUpdatePreferences_WordWhatThePreferenceMeans(t *testing.T) {
	t.Parallel()

	record, reader := readPreferences(t, preferencesAnswer("None", nil))

	require.NotNil(t, reader.sent("DeskCustomerUpdatePreferences"))
	assert.Equal(t, false, record["wantsArrivals"])
	assert.Equal(t, false, record["wantsDepartures"])
	assert.Equal(t, updatePreferenceWarning(customer.StatusUpdateNone), record["warning"])
	assert.Equal(t, "ops@freshhaul.test", record["fallbackRecipients"])
	assert.NotContains(t, record, fieldEmailProfile)

	record, _ = readPreferences(t, preferencesAnswer("Arrivals", "dock@freshhaul.test"))
	assert.Equal(t, true, record["wantsArrivals"])
	assert.Equal(t, false, record["wantsDepartures"])
	assert.NotContains(t, record, "warning")
	assert.NotContains(t, record, "fallbackRecipients", "the customer's own list is used")
}

// A preference access withheld is not read as None, and a list it withheld
// is not read as empty.
func TestUpdatePreferences_SayNothingOfAWithheldPreference(t *testing.T) {
	t.Parallel()

	record, _ := readPreferences(t, preferencesAnswer(nil, nil,
		"customer.statusUpdatePreference", "customer.statusUpdateRecipients"))

	assert.NotContains(t, record, "warning")
	assert.NotContains(t, record, "wantsArrivals")
	assert.NotContains(t, record, "fallbackRecipients")
	assert.Equal(t, []string{"statusUpdatePreference", "statusUpdateRecipients"},
		record[outcomeWithheld])
}
