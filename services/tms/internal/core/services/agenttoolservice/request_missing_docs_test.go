package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The agent writes prose; the organization's template writes the letter around
// it. This asserts the prose arrives as data rather than as markup or as a
// pre-assembled message, which is what lets a carrier put their own greeting and
// sign-off around everything an agent sends.
func TestAgentEmailContextCarriesTheAgentsProseAsData(t *testing.T) {
	t.Parallel()

	tool := &requestMissingDocsTool{}

	out := tool.agentEmailContext(
		t.Context(),
		pagination.TenantInfo{},
		"Missing paperwork for SHP-1001",
		"We are missing the signed bill of lading.",
		map[string]any{
			"customerName":       "Halstead Grocery Group",
			"shipmentProNumber":  "SHP-1001",
			"requestedDocuments": []any{"Signed bill of lading", "Lumper receipt"},
		},
	)

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
func TestAgentEmailContextToleratesMissingAndMalformedExtras(t *testing.T) {
	t.Parallel()

	tool := &requestMissingDocsTool{}

	bare := tool.agentEmailContext(
		t.Context(),
		pagination.TenantInfo{},
		"Subject",
		"Body",
		map[string]any{},
	)
	assert.Empty(t, bare.CustomerName)
	assert.Empty(t, bare.ShipmentProNumber)
	assert.Empty(t, bare.RequestedDocuments)
	assert.Equal(t, "Subject", bare.AgentSubject)

	malformed := tool.agentEmailContext(
		t.Context(),
		pagination.TenantInfo{},
		"Subject",
		"Body",
		map[string]any{
			"customerName":       42,
			"requestedDocuments": "a signed BOL",
		},
	)
	assert.Empty(t, malformed.CustomerName)
	assert.Empty(t, malformed.RequestedDocuments)
	assert.Equal(t, "Body", malformed.AgentBody)
}

// The schema is the agent's only description of what it may send. A parameter the
// template renders but the schema does not declare is a parameter no agent will
// ever fill.
func TestRequestMissingDocsSchemaDeclaresEveryTemplateInput(t *testing.T) {
	t.Parallel()

	schema := (&requestMissingDocsTool{}).ParamSchema()
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
