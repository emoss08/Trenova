package formulatemplatestudioservice

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/formulatemplatetest"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreview_SendsEveryChoiceAndARefusalInTheModulesShape(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	chosen := "Flat"
	deps.rater.Result = &tariffports.RateResult{
		Refusal: &rules.TariffRefusal{
			Text:  "This rate needs the trip's miles",
			Range: &rules.TariffRange{From: 4, To: 20},
		},
		Choices: []rules.TariffChoice{{
			Label:  "Linehaul",
			Kind:   rules.TariffChoiceBest,
			Range:  &rules.TariffRange{From: 30, To: 90},
			Chosen: &chosen,
			Options: []rules.TariffChoiceOption{
				{
					Label:    "Flat",
					Range:    &rules.TariffRange{From: 40, To: 60},
					Status:   rules.TariffOptionChosen,
					Subtotal: rules.MoneyValue(decimal.RequireFromString("900")),
				},
				{
					Label:   "Volume",
					Range:   &rules.TariffRange{From: 61, To: 88},
					Status:  rules.TariffOptionRefused,
					Refusal: &rules.TariffRefusal{Text: "Too light"},
				},
			},
		}},
	}

	result, err := deps.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		Source:     formulatemplatetest.PerMileSource,
	})
	require.NoError(t, err)
	require.Len(t, result.Choices, 1)

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	var wire struct {
		Refusal map[string]any   `json:"refusal"`
		Choices []map[string]any `json:"choices"`
	}
	require.NoError(t, sonic.Unmarshal(encoded, &wire))
	assert.Equal(t, map[string]any{
		"message": "This rate needs the trip's miles",
		"range":   map[string]any{"from": float64(4), "to": float64(20)},
	}, wire.Refusal)
	require.Len(t, wire.Choices, 1)
	assert.Equal(t, "Flat", wire.Choices[0]["chosen"])
	options, ok := wire.Choices[0]["options"].([]any)
	require.True(t, ok)
	require.Len(t, options, 2)
	refused, ok := options[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "refused", refused["status"])
	assert.Nil(t, refused["subtotal"])
	assert.Equal(t, map[string]any{"message": "Too light", "range": nil}, refused["refusal"])
}

func TestPreview_ASourceThatDoesNotCheckSendsNoChoices(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	result, err := deps.svc.Preview(t.Context(), &PreviewRequest{
		TenantInfo: formulatemplatetest.NewTenantInfo(),
		Source:     formulatemplatetest.BrokenSource,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Choices)
	assert.Empty(t, result.Choices)
}
