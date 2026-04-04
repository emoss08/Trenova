package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AIDocumentPage struct {
	PageNumber int
	Text       string
}

type AIDocumentFeatureSet struct {
	TitleCandidates  []string `json:"titleCandidates"`
	SectionLabels    []string `json:"sectionLabels"`
	PartyLabels      []string `json:"partyLabels"`
	ReferenceLabels  []string `json:"referenceLabels"`
	MoneySignals     []string `json:"moneySignals"`
	StopSignals      []string `json:"stopSignals"`
	TermsSignals     []string `json:"termsSignals"`
	SignatureSignals []string `json:"signatureSignals"`
}

type AIDocumentFingerprintHint struct {
	Provider   string   `json:"provider"`
	KindHint   string   `json:"kindHint"`
	Confidence float64  `json:"confidence"`
	Signals    []string `json:"signals"`
}

type AIDocumentField struct {
	Label             string   `json:"label"`
	Value             string   `json:"value"`
	Confidence        float64  `json:"confidence"`
	EvidenceExcerpt   string   `json:"evidenceExcerpt"`
	PageNumber        int      `json:"pageNumber"`
	ReviewRequired    bool     `json:"reviewRequired"`
	Conflict          bool     `json:"conflict"`
	Source            string   `json:"source"`
	AlternativeValues []string `json:"alternativeValues,omitempty"`
}

type AIDocumentStop struct {
	Sequence            int     `json:"sequence"`
	Role                string  `json:"role"`
	Name                string  `json:"name"`
	AddressLine1        string  `json:"addressLine1"`
	AddressLine2        string  `json:"addressLine2"`
	City                string  `json:"city"`
	State               string  `json:"state"`
	PostalCode          string  `json:"postalCode"`
	Date                string  `json:"date"`
	TimeWindow          string  `json:"timeWindow"`
	AppointmentRequired bool    `json:"appointmentRequired"`
	PageNumber          int     `json:"pageNumber"`
	EvidenceExcerpt     string  `json:"evidenceExcerpt"`
	Confidence          float64 `json:"confidence"`
	ReviewRequired      bool    `json:"reviewRequired"`
	Source              string  `json:"source"`
}

type AIDocumentConflict struct {
	Key             string   `json:"key"`
	Label           string   `json:"label"`
	Values          []string `json:"values"`
	PageNumbers     []int    `json:"pageNumbers"`
	EvidenceExcerpt string   `json:"evidenceExcerpt"`
	Source          string   `json:"source"`
}

type AIRouteRequest struct {
	TenantInfo  pagination.TenantInfo
	DocumentID  pulid.ID
	FileName    string
	Text        string
	Pages       []AIDocumentPage
	Features    *AIDocumentFeatureSet
	Fingerprint *AIDocumentFingerprintHint
}

type AIRouteResult struct {
	ShouldExtract       bool     `json:"shouldExtract"`
	DocumentKind        string   `json:"documentKind"`
	Confidence          float64  `json:"confidence"`
	Signals             []string `json:"signals"`
	ReviewStatus        string   `json:"reviewStatus"`
	ClassifierSource    string   `json:"classifierSource"`
	ProviderFingerprint string   `json:"providerFingerprint"`
	Reason              string   `json:"reason"`
}

type AIExtractRequest struct {
	TenantInfo pagination.TenantInfo
	DocumentID pulid.ID
	FileName   string
	Text       string
	Pages      []AIDocumentPage
}

type AIExtractResult struct {
	DocumentKind      string                     `json:"documentKind"`
	OverallConfidence float64                    `json:"overallConfidence"`
	ReviewStatus      string                     `json:"reviewStatus"`
	MissingFields     []string                   `json:"missingFields"`
	Signals           []string                   `json:"signals"`
	Fields            map[string]AIDocumentField `json:"fields"`
	Stops             []AIDocumentStop           `json:"stops"`
	Conflicts         []AIDocumentConflict       `json:"conflicts"`
}

type AIBackgroundExtractionStatus string

const (
	AIBackgroundExtractionStatusPending   AIBackgroundExtractionStatus = "pending"
	AIBackgroundExtractionStatusCompleted AIBackgroundExtractionStatus = "completed"
	AIBackgroundExtractionStatusFailed    AIBackgroundExtractionStatus = "failed"
)

type AIBackgroundExtractSubmission struct {
	ResponseID string `json:"responseId"`
	Model      string `json:"model"`
	Status     string `json:"status"`
}

type AIBackgroundExtractPollRequest struct {
	TenantInfo pagination.TenantInfo
	DocumentID pulid.ID
	ResponseID string
}

type AIBackgroundExtractPollResult struct {
	ResponseID     string                       `json:"responseId"`
	Model          string                       `json:"model"`
	Status         AIBackgroundExtractionStatus `json:"status"`
	RawStatus      string                       `json:"rawStatus"`
	ExtractResult  *AIExtractResult             `json:"extractResult,omitempty"`
	FailureCode    string                       `json:"failureCode,omitempty"`
	FailureMessage string                       `json:"failureMessage,omitempty"`
}

type AIDocumentService interface {
	RouteDocument(ctx context.Context, req *AIRouteRequest) (*AIRouteResult, error)
	ExtractRateConfirmation(ctx context.Context, req *AIExtractRequest) (*AIExtractResult, error)
	SubmitRateConfirmationBackgroundExtraction(ctx context.Context, req *AIExtractRequest) (*AIBackgroundExtractSubmission, error)
	PollRateConfirmationBackgroundExtraction(ctx context.Context, req *AIBackgroundExtractPollRequest) (*AIBackgroundExtractPollResult, error)
}
