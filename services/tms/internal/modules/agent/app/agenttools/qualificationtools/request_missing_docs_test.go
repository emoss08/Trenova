package qualificationtools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/requestemail"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentceiling"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func docsTool() *RequestMissingDocsTool {
	return &RequestMissingDocsTool{requests: requestemail.New(requestemail.Params{
		Templates: &agenttoolstest.FakeRenderer{},
	})}
}

// The agent writes prose; the organization's template writes the letter around
// it. This asserts the prose arrives as data rather than as markup or as a
// pre-assembled message, which is what lets a carrier put their own greeting and
// sign-off around everything an agent sends.
func TestRequestMissingDocsCarriesTheAgentsProseAsData(t *testing.T) {
	t.Parallel()

	params := agenttoolstest.ExecuteParams(map[string]any{
		"to":                 []any{"pod@consignee.example"},
		"subject":            "Missing paperwork for SHP-1001",
		"body":               "We are missing the signed bill of lading.",
		"customerName":       "Halstead Grocery Group",
		"shipmentProNumber":  "SHP-1001",
		"requestedDocuments": []any{"Signed bill of lading", "Lumper receipt"},
	})
	params.IdempotencyKey = "idem-1"

	composed, err := docsTool().compose(t.Context(), params)
	require.NoError(t, err)
	out := composed.Data

	assert.Equal(t, "Missing paperwork for SHP-1001", out.AgentSubject)
	assert.Equal(t, "We are missing the signed bill of lading.", out.AgentBody)
	assert.Equal(t, "Halstead Grocery Group", out.CustomerName)
	assert.Equal(t, "SHP-1001", out.ShipmentProNumber)
	require.Len(t, out.RequestedDocuments, 2)
	assert.Equal(t, "Signed bill of lading", out.RequestedDocuments[0])
}

// The optional parameters are the agent's to supply and easy for it to get
// wrong. A malformed one must not stop a document request going out, because the
// agent's own words already say what is being asked for.
func TestRequestMissingDocsToleratesMissingAndMalformedExtras(t *testing.T) {
	t.Parallel()

	bareParams := agenttoolstest.ExecuteParams(map[string]any{
		"to":      []any{"pod@consignee.example"},
		"subject": "Subject",
		"body":    "Body",
	})
	bareParams.IdempotencyKey = "idem-2"
	composed, err := docsTool().compose(t.Context(), bareParams)
	require.NoError(t, err)
	bare := composed.Data
	assert.Empty(t, bare.CustomerName)
	assert.Empty(t, bare.ShipmentProNumber)
	assert.Empty(t, bare.RequestedDocuments)
	assert.Equal(t, "Subject", bare.AgentSubject)

	malformedParams := agenttoolstest.ExecuteParams(map[string]any{
		"to":                 []any{"pod@consignee.example"},
		"subject":            "Subject",
		"body":               "Body",
		"customerName":       42,
		"requestedDocuments": "a signed BOL",
	})
	malformedParams.IdempotencyKey = "idem-3"
	composed, err = docsTool().compose(t.Context(), malformedParams)
	require.NoError(t, err)
	malformed := composed.Data
	assert.Empty(t, malformed.CustomerName)
	assert.Empty(t, malformed.RequestedDocuments)
	assert.Equal(t, "Body", malformed.AgentBody)
}

// The schema is the agent's only description of what it may send. A parameter the
// template renders but the schema does not declare is a parameter no agent will
// ever fill.
func TestRequestMissingDocsSchemaDeclaresEveryTemplateInput(t *testing.T) {
	t.Parallel()

	schema := (&RequestMissingDocsTool{}).ParamSchema()
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)

	for _, key := range []string{
		"profileId", "to", "subject", "body",
		"customerName", "shipmentProNumber", "requestedDocuments",
	} {
		assert.Contains(t, properties, key)
	}

	// additionalProperties is false, so an undeclared parameter is rejected
	// outright rather than silently dropped.
	assert.Equal(t, false, schema["additionalProperties"])
}

// Asking an outside party for paperwork is an email to that party, so it is
// gated on sending customer communications, not on working the billing queue.
func TestRequestMissingDocs_IsGatedOnCustomerCommunication(t *testing.T) {
	t.Parallel()

	tool := &RequestMissingDocsTool{}
	assert.Equal(t, permission.ResourceCustomerCommunication, tool.Policy().Resource)
	assert.Equal(t, permission.OpCreate, tool.Policy().Operation)
	assert.True(t, agentceiling.Allows(tool.Policy().Resource, tool.Policy().Operation))
}
