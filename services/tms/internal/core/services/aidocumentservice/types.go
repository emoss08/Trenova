package aidocumentservice

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

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
	Key             string  `json:"key"`
	Value           string  `json:"value"`
	Confidence      float64 `json:"confidence"`
	PageNumber      int     `json:"pageNumber"`
	ReviewRequired  bool    `json:"reviewRequired"`
	EvidenceExcerpt string  `json:"evidenceExcerpt"`
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
