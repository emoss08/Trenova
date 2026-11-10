package briefingwriter

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type cannedNova struct {
	text string
	err  error
}

func (c *cannedNova) Available(context.Context, pagination.TenantInfo) (bool, error) {
	return true, nil
}

func (c *cannedNova) Run(
	context.Context,
	*llmports.NovaTask,
	any,
) (*llmports.StructuredCompletionResult, error) {
	if c.err != nil {
		return nil, c.err
	}

	return &llmports.StructuredCompletionResult{Text: c.text, ModelIdentifier: "test-model"}, nil
}

func request() *Request {
	return &Request{
		Role: briefing.RoleDispatch,
		Sections: []briefing.Section{
			{
				Key:     briefing.SectionToday,
				Title:   "Today",
				Summary: "142 loads are moving.",
				Path:    "/dispatch/console",
			},
			{
				Key:     briefing.SectionExceptions,
				Title:   "Needs attention",
				Summary: "6 loads may miss their window.",
				Path:    "/desk/needs",
			},
		},
		Supported: []decimal.Decimal{decimal.NewFromInt(142), decimal.NewFromInt(6)},
	}
}

func writerWith(nova llmports.TaskRunner) *Service {
	return New(Params{Logger: zap.NewNop(), Nova: nova})
}

func TestWrite_LinksThePhrasesTheModelPointedAtSectionsOnThePage(t *testing.T) {
	nova := &cannedNova{text: `{
		"headline": "142 loads are moving and 6 may miss today's appointments.",
		"links": [
			{"phrase": "6 may miss today's appointments", "section": "Exceptions"},
			{"phrase": "142 loads are moving", "section": "Billing"}
		],
		"sections": []
	}`}

	result := writerWith(nova).Write(context.Background(), request())

	assert.True(t, result.Narrated)
	assert.Equal(t, "142 loads are moving and 6 may miss today's appointments.", result.Headline)
	assert.Equal(t, []briefing.HeadlineLink{{
		Text:    "6 may miss today's appointments",
		Section: briefing.SectionExceptions,
		Path:    "/desk/needs",
	}}, result.Links)
}

func TestWrite_KeepsTheComputedHeadlineAndItsLinkWhenTheModelCitesAnUnknownFigure(t *testing.T) {
	nova := &cannedNova{text: `{
		"headline": "9 loads may miss today's appointments.",
		"links": [{"phrase": "9 loads", "section": "Exceptions"}],
		"sections": []
	}`}

	result := writerWith(nova).Write(context.Background(), request())

	assert.False(t, result.Narrated)
	assert.Equal(t, "142 loads are moving.", result.Headline)
	assert.Equal(t, []briefing.HeadlineLink{{
		Text:    "142 loads are moving.",
		Section: briefing.SectionToday,
		Path:    "/dispatch/console",
	}}, result.Links)
}

func TestWrite_KeepsTheComputedHeadlineAndItsLinkWithoutAModel(t *testing.T) {
	nova := &cannedNova{err: errors.New("provider down")}

	result := writerWith(nova).Write(context.Background(), request())

	assert.False(t, result.Narrated)
	assert.Equal(t, "142 loads are moving.", result.Headline)
	assert.Len(t, result.Links, 1)
	assert.Equal(t, briefing.SectionToday, result.Links[0].Section)
}

func TestWrite_AsksForLinksAsPhrasesAndSectionKeysOnly(t *testing.T) {
	schema := outputSchema()

	properties, ok := schema["properties"].(map[string]any)
	assert.True(t, ok)
	links, ok := properties["links"].(map[string]any)
	assert.True(t, ok)
	items, ok := links["items"].(map[string]any)
	assert.True(t, ok)
	fields, ok := items["properties"].(map[string]any)
	assert.True(t, ok)

	assert.Len(t, fields, 2)
	assert.Contains(t, fields, "phrase")
	assert.Contains(t, fields, "section")
	assert.Contains(t, schema["required"], "links")
}
