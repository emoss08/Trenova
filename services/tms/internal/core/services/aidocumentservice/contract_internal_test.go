package aidocumentservice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContractRequestIsTheProductionExtractionCall(t *testing.T) {
	t.Parallel()

	pages := []serviceports.AIDocumentPage{{PageNumber: 1, Text: strings.Repeat("x", extractPageLimit+500)}}
	req := Contract{}.CompletionRequest("document.pdf", pages)
	production := newExtractCall(&serviceports.AIExtractRequest{FileName: "document.pdf", Pages: pages}).request()

	assert.Equal(t, production.System, req.System)
	assert.Equal(t, production.Context, req.Context)
	assert.Equal(t, production.OutputSchema, req.OutputSchema)
	assert.Equal(t, schemaNameExtract, req.SchemaName)
	assert.Equal(t, aiprovider.TaskDocumentExtraction, req.Task)
	assert.Equal(t, extractPageLimit, Contract{}.PageLimit())
	assert.NotContains(t, req.Context.Sections[1].Content, strings.Repeat("x", extractPageLimit+1))
}

func TestContractFieldKeysMatchTheSchemaEnum(t *testing.T) {
	t.Parallel()

	schema := buildExtractSchema()
	items := schema["properties"].(map[string]any)["fields"].(map[string]any)["items"].(map[string]any)
	enum := items["properties"].(map[string]any)["key"].(map[string]any)["enum"].([]string)
	assert.Equal(t, enum, Contract{}.FieldKeys())
}

func TestContractParseReplyReadsTheWireFormat(t *testing.T) {
	t.Parallel()

	reply := `{"documentKind":"RateConfirmation","overallConfidence":1.4,"reviewStatus":"ready",` +
		`"missingFields":[],"signals":[],"conflicts":[],` +
		`"fields":[{"key":"rate","value":"1500.00","confidence":0.9,"evidenceExcerpt":"",` +
		`"pageNumber":1,"reviewRequired":false}],` +
		`"stops":[{"role":"pickup","name":"A","addressLine1":"","addressLine2":"","city":"",` +
		`"state":"","postalCode":"","date":"","timeWindow":"","appointmentRequired":false,"pageNumber":1,` +
		`"evidenceExcerpt":"","confidence":0.9,"reviewRequired":false}]}`

	parsed, err := Contract{}.ParseReply(reply)
	require.NoError(t, err)
	assert.Equal(t, "RateConfirmation", parsed.DocumentKind)
	assert.InDelta(t, 1.0, parsed.OverallConfidence, 1e-9)
	assert.Equal(t, "Ready", parsed.ReviewStatus)
	assert.Equal(t, "1500.00", parsed.Fields["rate"].Value)
	require.Len(t, parsed.Stops, 1)
	assert.Equal(t, "A", parsed.Stops[0].Name)
}

func TestContractParseReplyDerivesWhatTheModelNoLongerWrites(t *testing.T) {
	t.Parallel()

	reply := `{"documentKind":"RateConfirmation","overallConfidence":0.9,"reviewStatus":"Ready",` +
		`"missingFields":[],"signals":[],` +
		`"conflicts":[{"key":"pickupwindow","values":["08:00","09:00"],"pageNumbers":[1],"evidenceExcerpt":"x"},` +
		`{"key":"brokerNotes","values":["a"],"pageNumbers":[],"evidenceExcerpt":""},null],` +
		`"fields":[{"key":"pickupWindow","value":"08:00","confidence":0.6,"evidenceExcerpt":"",` +
		`"pageNumber":1,"reviewRequired":true},` +
		`{"key":"poNumber","value":"4500","confidence":0.9,"evidenceExcerpt":"","pageNumber":1,` +
		`"reviewRequired":false}],` +
		`"stops":[{"role":"pickup","name":"A","pageNumber":1,"evidenceExcerpt":"A"},null,` +
		`{"role":"delivery","name":"B","pageNumber":2,"evidenceExcerpt":"B"}]}`

	parsed, err := Contract{}.ParseReply(reply)
	require.NoError(t, err)

	window := parsed.Fields["pickupWindow"]
	assert.Equal(t, "Pickup Window", window.Label)
	assert.True(t, window.Conflict)
	assert.Equal(t, serviceports.AIDocumentSourceAI, window.Source)
	assert.Empty(t, window.AlternativeValues)

	po := parsed.Fields["poNumber"]
	assert.Equal(t, "PO Number", po.Label)
	assert.False(t, po.Conflict)

	require.Len(t, parsed.Stops, 2)
	for i, stop := range parsed.Stops {
		assert.Equal(t, i+1, stop.Sequence)
		assert.Equal(t, serviceports.AIDocumentSourceAI, stop.Source)
	}

	require.Len(t, parsed.Conflicts, 2)
	assert.Equal(t, "pickupWindow", parsed.Conflicts[0].Key)
	assert.Equal(t, "Pickup Window", parsed.Conflicts[0].Label)
	assert.Equal(t, "brokerNotes", parsed.Conflicts[1].Key)
	assert.Equal(t, "brokerNotes", parsed.Conflicts[1].Label)
	for _, conflict := range parsed.Conflicts {
		assert.Equal(t, serviceports.AIDocumentSourceAI, conflict.Source)
	}
}

func TestContractParseReplyReadsRepliesInTheFormerWireFormat(t *testing.T) {
	t.Parallel()

	reply := `{"documentKind":"RateConfirmation","overallConfidence":0.9,"reviewStatus":"Ready",` +
		`"missingFields":[],"signals":[],"conflicts":[],` +
		`"fields":[{"key":"rate","label":"Linehaul","value":"1500.00","confidence":0.9,` +
		`"evidenceExcerpt":"","pageNumber":1,"reviewRequired":false,"conflict":true,` +
		`"source":"model","alternativeValues":["1400.00"]}],` +
		`"stops":[{"sequence":7,"role":"pickup","name":"A","pageNumber":1,"evidenceExcerpt":"A",` +
		`"source":"model"}]}`

	parsed, err := Contract{}.ParseReply(reply)
	require.NoError(t, err)
	rate := parsed.Fields["rate"]
	assert.Equal(t, "1500.00", rate.Value)
	assert.Equal(t, "Rate", rate.Label)
	assert.False(t, rate.Conflict)
	assert.Equal(t, serviceports.AIDocumentSourceAI, rate.Source)
	require.Len(t, parsed.Stops, 1)
	assert.Equal(t, 1, parsed.Stops[0].Sequence)
	assert.Equal(t, serviceports.AIDocumentSourceAI, parsed.Stops[0].Source)
}

func TestExtractSchemaHoldsOnlyWhatTheModelMustDecide(t *testing.T) {
	t.Parallel()

	schema := buildExtractSchema()
	properties := schema["properties"].(map[string]any)
	objectProperties := func(name string) map[string]any {
		items := properties[name].(map[string]any)["items"].(map[string]any)
		return items["properties"].(map[string]any)
	}

	for _, derived := range []string{
		"label", "source", "conflict", "alternativeValues", "evidenceExcerpt",
	} {
		assert.NotContains(t, objectProperties("fields"), derived)
	}
	for _, derived := range []string{"sequence", "source"} {
		assert.NotContains(t, objectProperties("stops"), derived)
	}
	for _, derived := range []string{"label", "source"} {
		assert.NotContains(t, objectProperties("conflicts"), derived)
	}
}

func TestContractParseReplyRejectsMalformedText(t *testing.T) {
	t.Parallel()

	_, err := Contract{}.ParseReply("not json")
	require.ErrorIs(t, err, serviceports.ErrModelSchemaValidation)
}

const pipelineSchemaFixture = "../../../../../../ml/extraction-finetune/tests/fixtures/extraction-schema.json"

func TestFineTuningPipelineSchemaFixtureIsCurrent(t *testing.T) {
	t.Parallel()

	schema := Contract{}.CompletionRequest("", nil).OutputSchema
	encoded, err := sonic.ConfigStd.MarshalIndent(schema, "", "  ")
	require.NoError(t, err)
	encoded = append(encoded, '\n')

	if os.Getenv("TRENOVA_UPDATE_FIXTURES") == "1" {
		require.NoError(t, os.MkdirAll(filepath.Dir(pipelineSchemaFixture), 0o755))
		require.NoError(t, os.WriteFile(pipelineSchemaFixture, encoded, 0o644))
	}

	fixture, err := os.ReadFile(pipelineSchemaFixture)
	require.NoError(t, err, "regenerate with TRENOVA_UPDATE_FIXTURES=1 go test ./internal/core/services/aidocumentservice/")
	assert.JSONEq(t, string(encoded), string(fixture),
		"the fine-tuning pipeline's schema fixture is stale; regenerate it with "+
			"TRENOVA_UPDATE_FIXTURES=1 go test ./internal/core/services/aidocumentservice/")
}
