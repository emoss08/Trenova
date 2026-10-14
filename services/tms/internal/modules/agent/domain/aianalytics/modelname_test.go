package aianalytics_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/stretchr/testify/assert"
)

func TestCanonicalModelFoldsOneModelServedUnderDifferentNames(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"llama-3.3-70b-versatile":                    "llama-3.3-70b",
		"meta-llama/Llama-3.3-70B-Instruct":          "llama-3.3-70b",
		"us.meta.llama3-3-70b-instruct-v1:0":         "llama-3.3-70b",
		"qwen2.5:14b":                                "qwen-2.5-14b",
		"Qwen/Qwen2.5-14B-Instruct":                  "qwen-2.5-14b",
		"claude-sonnet-5":                            "claude-sonnet-5",
		"us.anthropic.claude-sonnet-5-20260101-v1:0": "claude-sonnet-5",
		"claude-3-5-sonnet-20241022":                 "claude-3-5-sonnet",
		"gpt-5.1-mini":                               "gpt-5.1-mini",
		"gpt-5.1-mini-2026-03-01":                    "gpt-5.1-mini",
		"gemini-3-flash":                             "gemini-3-flash",
		"o3-mini":                                    "o3-mini",
		"  ":                                         "",
		"mistralai/Mixtral-8x7B-Instruct-v0.1":       "mixtral-8x7b",
		"deepseek-r1:70b":                            "deepseek-r1-70b",
	}
	for raw, want := range cases {
		assert.Equal(t, want, aianalytics.CanonicalModel(raw), raw)
	}
}

func TestNameModelGivesTheLabelAndShortLabelPeopleRecognize(t *testing.T) {
	t.Parallel()

	cases := map[string]aianalytics.ModelName{
		"claude-sonnet-5":   {Label: "Claude Sonnet 5", Short: "Sonnet 5"},
		"claude-opus-4-5":   {Label: "Claude Opus 4.5", Short: "Opus 4.5"},
		"claude-3-5-sonnet": {Label: "Claude 3.5 Sonnet", Short: "Sonnet 3.5"},
		"gpt-5.1":           {Label: "GPT-5.1", Short: "GPT-5.1"},
		"gpt-5.1-mini":      {Label: "GPT-5.1 mini", Short: "5.1 mini"},
		"gemini-3-flash":    {Label: "Gemini 3 Flash", Short: "Gem. Flash"},
		"llama-3.3-70b":     {Label: "Llama 3.3 70B", Short: "Llama 70B"},
		"qwen-2.5-14b":      {Label: "Qwen 2.5 14B", Short: "Qwen 14B"},
		"mixtral-8x7b":      {Label: "Mixtral 8x7B", Short: "Mixtral 8x7B"},
		"o3-mini":           {Label: "o3 mini", Short: "o3 mini"},
		"custom":            {Label: "Custom", Short: "Custom"},
	}
	for key, want := range cases {
		assert.Equal(t, want, aianalytics.NameModel(key), key)
	}
}
