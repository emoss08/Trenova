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
		`"fields":[{"key":"rate","label":"Rate","value":"1500.00","confidence":0.9,"evidenceExcerpt":"",` +
		`"pageNumber":1,"reviewRequired":false,"conflict":false,"source":"ai","alternativeValues":[]}],` +
		`"stops":[{"sequence":1,"role":"pickup","name":"A","addressLine1":"","addressLine2":"","city":"",` +
		`"state":"","postalCode":"","date":"","timeWindow":"","appointmentRequired":false,"pageNumber":1,` +
		`"evidenceExcerpt":"","confidence":0.9,"reviewRequired":false,"source":"ai"}]}`

	parsed, err := Contract{}.ParseReply(reply)
	require.NoError(t, err)
	assert.Equal(t, "RateConfirmation", parsed.DocumentKind)
	assert.InDelta(t, 1.0, parsed.OverallConfidence, 1e-9)
	assert.Equal(t, "Ready", parsed.ReviewStatus)
	assert.Equal(t, "1500.00", parsed.Fields["rate"].Value)
	require.Len(t, parsed.Stops, 1)
	assert.Equal(t, "A", parsed.Stops[0].Name)
}

func TestContractParseReplyRejectsMalformedText(t *testing.T) {
	t.Parallel()

	_, err := Contract{}.ParseReply("not json")
	require.ErrorIs(t, err, serviceports.ErrModelSchemaValidation)
}

const pipelineSchemaFixture = "../../../../../../ml/extraction-finetune/tests/fixtures/extraction-schema.json"

func TestFineTuningPipelineSchemaFixtureIsCurrent(t *testing.T) {
	t.Parallel()

	encoded, err := sonic.MarshalIndent(Contract{}.CompletionRequest("", nil).OutputSchema, "", "  ")
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
