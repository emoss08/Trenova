package aidocumentservice

import (
	"strings"
	"testing"

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

func TestContractFormatReplyRoundTrips(t *testing.T) {
	t.Parallel()

	result := &serviceports.AIExtractResult{
		DocumentKind:      "RateConfirmation",
		OverallConfidence: 1.4,
		ReviewStatus:      "ready",
		Fields: map[string]serviceports.AIDocumentField{
			"rate":            {Label: "Rate", Value: "1500.00", Confidence: 0.9, PageNumber: 1, Source: "ai"},
			"shipper":         {Label: "Shipper", Value: "Harbor Supply", Confidence: 0.95},
			"notInSchema":     {Label: "Nope", Value: "dropped"},
			"referenceNumber": {Label: "Reference", Value: "", Confidence: 0.5},
		},
		Stops: []*serviceports.AIDocumentStop{
			{Sequence: 2, Role: "delivery", Name: "B"},
			nil,
			{Sequence: 1, Role: "pickup", Name: "A", EvidenceExcerpt: strings.Repeat("e", maxEvidenceRunes+10)},
		},
	}

	text, err := Contract{}.FormatReply(result)
	require.NoError(t, err)
	parsed, err := Contract{}.ParseReply(text)
	require.NoError(t, err)

	assert.Equal(t, "RateConfirmation", parsed.DocumentKind)
	assert.InDelta(t, 1.0, parsed.OverallConfidence, 1e-9)
	assert.Equal(t, "Ready", parsed.ReviewStatus)
	assert.Len(t, parsed.Fields, 2)
	assert.Equal(t, "1500.00", parsed.Fields["rate"].Value)
	assert.NotNil(t, parsed.Fields["rate"].AlternativeValues)
	require.Len(t, parsed.Stops, 2)
	assert.Equal(t, "A", parsed.Stops[0].Name)
	assert.Len(t, []rune(parsed.Stops[0].EvidenceExcerpt), maxEvidenceRunes)
	assert.Contains(t, text, `"fields":[{"key":"shipper"`)
}

func TestContractFormatReplyCapsFieldsAndStops(t *testing.T) {
	t.Parallel()

	fields := map[string]serviceports.AIDocumentField{}
	for _, key := range extractFieldKeys {
		fields[key] = serviceports.AIDocumentField{Value: "v"}
	}
	stops := make([]*serviceports.AIDocumentStop, 0, maxExtractStops+3)
	for i := range maxExtractStops + 3 {
		stops = append(stops, &serviceports.AIDocumentStop{Sequence: i})
	}

	text, err := Contract{}.FormatReply(&serviceports.AIExtractResult{Fields: fields, Stops: stops})
	require.NoError(t, err)
	parsed, err := Contract{}.ParseReply(text)
	require.NoError(t, err)
	assert.Len(t, parsed.Fields, maxExtractFields)
	assert.Len(t, parsed.Stops, maxExtractStops)
}

func TestContractParseReplyRejectsMalformedText(t *testing.T) {
	t.Parallel()

	_, err := Contract{}.ParseReply("not json")
	require.ErrorIs(t, err, serviceports.ErrModelSchemaValidation)
}
