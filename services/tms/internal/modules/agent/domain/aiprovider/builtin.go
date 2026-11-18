package aiprovider

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	BuiltInEmbeddingID   = pulid.ID("aiprv_trenova_search")
	BuiltInEmbeddingName = "Trenova search service"

	BuiltInEmbeddingDimensions     = 768
	BuiltInEmbeddingTimeoutSeconds = 120
	BuiltInEmbeddingMaxConcurrent  = 2

	builtInEmbeddingAPIPath = "/v1"
)

type BuiltInEmbeddingService struct {
	URL   string
	Model string
}

func BuiltInEmbedding(orgID, buID pulid.ID, service BuiltInEmbeddingService) *Provider {
	dimensions := BuiltInEmbeddingDimensions
	free := decimal.Zero

	return &Provider{
		ID:                   BuiltInEmbeddingID,
		OrganizationID:       orgID,
		BusinessUnitID:       buID,
		Name:                 BuiltInEmbeddingName,
		Kind:                 KindOpenAIChat,
		BaseURL:              strings.TrimRight(service.URL, "/") + builtInEmbeddingAPIPath,
		Model:                service.Model,
		Platform:             aiplatform.PlatformDirect,
		StructuredOutputMode: StructuredOutputPrompted,
		ReasoningEffort:      aireasoning.ReasoningOff,
		ThinkingStyle:        aireasoning.ThinkingStyleAuto,
		Tasks:                []aitask.Task{aitask.TaskEmbedding},
		EmbeddingDimensions:  &dimensions,
		EmbeddingInputStyle:  EmbeddingInputStyleNomicPrefix,
		InputCostPerMillion:  &free,
		OutputCostPerMillion: &free,
		Enabled:              true,
		TimeoutSeconds:       BuiltInEmbeddingTimeoutSeconds,
		MaxConcurrent:        BuiltInEmbeddingMaxConcurrent,
		OnCap:                CapActionNext,
		BuiltIn:              true,
	}
}
