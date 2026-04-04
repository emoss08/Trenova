package openaidocumentservice

import (
	"github.com/emoss08/trenova/internal/core/domain/ailog"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

type structuredResponseParams struct {
	orgID        pulid.ID
	buID         pulid.ID
	userID       pulid.ID
	documentID   pulid.ID
	operation    ailog.Operation
	model        ailog.Model
	systemPrompt string
	userPrompt   string
	schema       map[string]any
	out          any
}

type responsesRequest struct {
	Model           string              `json:"model"`
	Input           []responsesMessage  `json:"input"`
	Text            responsesTextConfig `json:"text"`
	MaxOutputTokens int                 `json:"max_output_tokens,omitempty"`
	Background      bool                `json:"background,omitempty"`
	Store           bool                `json:"store,omitempty"`
}

type responsesMessage struct {
	Role    string                 `json:"role"`
	Content []responsesMessagePart `json:"content"`
}

type responsesMessagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesTextConfig struct {
	Format responsesFormat `json:"format"`
}

type responsesFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

type responsesEnvelope struct {
	ID                string                      `json:"id"`
	Status            string                      `json:"status"`
	Model             string                      `json:"model"`
	OutputText        string                      `json:"output_text"`
	Output            []responsesOutputItem       `json:"output"`
	Usage             responsesUsage              `json:"usage"`
	ServiceTier       string                      `json:"service_tier"`
	Error             *responsesError             `json:"error"`
	IncompleteDetails *responsesIncompleteDetails `json:"incomplete_details"`
}

type responsesOutputItem struct {
	Type    string                 `json:"type"`
	Content []responsesContentPart `json:"content"`
}

type responsesContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesUsage struct {
	InputTokens         int                   `json:"input_tokens"`
	OutputTokens        int                   `json:"output_tokens"`
	TotalTokens         int                   `json:"total_tokens"`
	OutputTokensDetails responsesTokenDetails `json:"output_tokens_details"`
}

type responsesTokenDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

type responsesError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

type responsesIncompleteDetails struct {
	Reason string `json:"reason"`
}

type routeResponse struct {
	ShouldExtract       bool     `json:"shouldExtract"`
	DocumentKind        string   `json:"documentKind"`
	Confidence          float64  `json:"confidence"`
	Signals             []string `json:"signals"`
	ReviewStatus        string   `json:"reviewStatus"`
	ClassifierSource    string   `json:"classifierSource"`
	ProviderFingerprint string   `json:"providerFingerprint"`
	Reason              string   `json:"reason"`
}

type extractFieldResponse struct {
	Key               string   `json:"key"`
	Label             string   `json:"label"`
	Value             string   `json:"value"`
	Confidence        float64  `json:"confidence"`
	EvidenceExcerpt   string   `json:"evidenceExcerpt"`
	PageNumber        int      `json:"pageNumber"`
	ReviewRequired    bool     `json:"reviewRequired"`
	Conflict          bool     `json:"conflict"`
	Source            string   `json:"source"`
	AlternativeValues []string `json:"alternativeValues"`
}

type extractResponse struct {
	DocumentKind      string                             `json:"documentKind"`
	OverallConfidence float64                            `json:"overallConfidence"`
	ReviewStatus      string                             `json:"reviewStatus"`
	MissingFields     []string                           `json:"missingFields"`
	Signals           []string                           `json:"signals"`
	Fields            []extractFieldResponse             `json:"fields"`
	Stops             []*serviceports.AIDocumentStop     `json:"stops"`
	Conflicts         []*serviceports.AIDocumentConflict `json:"conflicts"`
}
