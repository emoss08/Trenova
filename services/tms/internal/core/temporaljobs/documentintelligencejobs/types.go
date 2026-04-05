package documentintelligencejobs

import (
	"regexp"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	services "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	reviewStatusReady       = "Ready"
	reviewStatusNeedsReview = "NeedsReview"
	reviewStatusUnavailable = "Unavailable"

	stopRoleShipper   = "shipper"
	stopRoleConsignee = "consignee"
	stopRolePickup    = "pickup"
	stopRoleDelivery  = "delivery"

	kindRateConfirmation = "RateConfirmation"
	kindBillOfLading     = "BillOfLading"
	kindProofOfDelivery  = "ProofOfDelivery"
	kindInvoice          = "Invoice"
	kindOther            = "Other"

	maxConfidence                   = 0.99
	classificationMinConfidence     = 0.55
	reviewRequiredConfidenceFloor   = 0.8
	reviewReadyConfidenceThreshold  = 0.82
	defaultLowConfidence            = 0.3
	ocrBaseStopConfidence           = 0.72
	nativeBaseStopConfidence        = 0.9
	stopMissingAddressPenalty       = 0.18
	stopMissingCityStatePenalty     = 0.08
	rawExcerptPreTruncateLen        = 2100
	rawExcerptMaxLen                = 2000
	heartbeatInterval               = 10 * time.Second
	ocrPreprocessingContrastBoost   = 25.0
	ocrPreprocessingSharpenSigma    = 1.5
	ocrPreprocessingBinaryThreshold = 170
	stopBlockMaxLines               = 36
	stopBlockMaxBlankRun            = 5
	sectionBlockMaxLines            = 6
	stopExcerptLines                = 4
)

var (
	rateRegex = regexp.MustCompile(
		`(?im)^(?:rate|freight charge|line haul|amount due|total)\s*[:#-]\s*([$]?[0-9,]+(?:\.[0-9]{2})?)$`,
	)
	referenceRegex = regexp.MustCompile(
		`(?im)^(?:load|reference|order|confirmation|pro|bol|invoice)(?:\s+(?:number|#))?\s*[:#-]\s*([A-Za-z0-9-]+)$`,
	)
	bolReferenceRegex = regexp.MustCompile(
		`(?im)^(?:bill of lading|bol|b\/l|pro|reference|order|load)(?:\s+(?:number|#|no\.?))\s*[:#-]?\s*([A-Za-z0-9-]+)$`,
	)
	podReferenceRegex = regexp.MustCompile(
		`(?im)^(?:pod|pro|reference|order|load)(?:\s+(?:number|#|no\.?))\s*[:#-]?\s*([A-Za-z0-9-]+)$`,
	)
	pickupRegex = regexp.MustCompile(
		`(?im)^(pickup|ship)(?:\s+(?:date|window|location|address))?\s*[:\-]\s*(.+)$`,
	)
	deliveryRegex = regexp.MustCompile(
		`(?im)^(delivery|drop)(?:\s+(?:date|window|location|address))?\s*[:\-]\s*(.+)$`,
	)
	shipperRegex   = regexp.MustCompile(`(?im)^shipper(?:\s+name)?\s*[:\-]\s*(.+)$`)
	consigneeRegex = regexp.MustCompile(`(?im)^(consignee|receiver)(?:\s+name)?\s*[:\-]\s*(.+)$`)
	commodityRegex = regexp.MustCompile(`(?im)^(commodity|product|description)\s*[:\-]\s*(.+)$`)
	equipmentRegex = regexp.MustCompile(
		`(?i)\b(van|reefer|flatbed|step deck|power only|hotshot|conestoga|dry van)\b`,
	)
	weightRegex     = regexp.MustCompile(`(?i)([0-9,]+)\s*(lbs|pounds)`)
	pieceCountRegex = regexp.MustCompile(
		`(?im)^(?:pieces?|piece count|total pieces|packages?|package count|units?|cartons?)\s*[:#-]?\s*([0-9,]+)\b`,
	)
	instructionsRegex = regexp.MustCompile(
		`(?im)^(instructions|notes|special instructions)\s*[:\-]\s*(.+)$`,
	)
	invoiceDateRegex = regexp.MustCompile(`(?im)^invoice date\s*[:\-]\s*(.+)$`)
	dueDateRegex     = regexp.MustCompile(`(?im)^due date\s*[:\-]\s*(.+)$`)
	totalDueRegex    = regexp.MustCompile(
		`(?im)^(?:amount due|total due|balance due)\s*[:#-]\s*([$]?[0-9,]+(?:\.[0-9]{2})?)$`,
	)
	dateLabelRegex  = regexp.MustCompile(`(?i)\b(?:date|pick\s*up|pickup|delivery)\b`)
	timeWindowRegex = regexp.MustCompile(
		`(?i)\b([0-9]{1,2}[:][0-9]{2}\s*(?:am|pm)?\s*[-–]\s*[0-9]{1,2}[:][0-9]{2}\s*(?:am|pm)?)\b`,
	)
	appointmentRegex = regexp.MustCompile(`(?i)\b(\d{1,2}:\d{2}\s*(?:am|pm)?\s*appt\.?)\b`)
	dateValueRegex   = regexp.MustCompile(
		`(?i)(?:^|[^0-9:])((?:\d{1,2}[/-]\d{1,2}(?:[/-]\d{2,4})?|(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\s+\d{1,2}(?:,\s*\d{4})?))\b`,
	)
	addressLineRegex  = regexp.MustCompile(`(?i)\b\d{1,6}\s+[a-z0-9][a-z0-9.\-# ]+\b`)
	cityStateZipRegex = regexp.MustCompile(
		`(?i)\b([a-z .'-]+),?\s*([a-z]{2})\s+(\d{5}(?:-\d{4})?)\b`,
	)
	stopSectionRegex = regexp.MustCompile(
		`(?i)^\s*(shipper|pickup|origin|receiver|consignee|delivery|drop|destination)\s*(?:#\s*\d+)?\s*:?\s*$`,
	)
	phoneLineRegex = regexp.MustCompile(`^\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}$`)
)

var Boundaries = []string{
	"ship from", "ship to", "shipper", "consignee", "receiver", "delivery to", "delivered to",
	"pickup", "delivery", "received by", "signed by", "receiver signature", "consignee signature",
	"bill of lading", "bol", "reference", "load", "pro", "commodity", "description", "weight",
	"pieces", "packages", "remarks", "exceptions", "carrier", "invoice", "bill to",
}

type ProcessDocumentIntelligencePayload struct {
	temporaltype.BasePayload

	DocumentID pulid.ID `json:"documentId"`
}

type ProcessDocumentIntelligenceResult struct {
	DocumentID pulid.ID `json:"documentId"`
	Status     string   `json:"status"`
	Kind       string   `json:"kind"`
}

type ProcessDocumentAIExtractionPayload struct {
	temporaltype.BasePayload

	DocumentID  pulid.ID `json:"documentId"`
	ExtractedAt int64    `json:"extractedAt"`
}

type ProcessDocumentAIExtractionResult struct {
	DocumentID      pulid.ID `json:"documentId"`
	ExtractedAt     int64    `json:"extractedAt"`
	AcceptanceState string   `json:"acceptanceState"`
}

type ReconcileDocumentIntelligencePayload struct {
	temporaltype.BasePayload

	OlderThanSeconds int64 `json:"olderThanSeconds"`
}

type ReconcileDocumentIntelligenceResult struct {
	Queued int `json:"queued"`
}

type PollPendingDocumentAIExtractionsPayload struct {
	temporaltype.BasePayload

	Limit int `json:"limit"`
}

type PollPendingDocumentAIExtractionsResult struct {
	Completed int `json:"completed"`
	Pending   int `json:"pending"`
	Failed    int `json:"failed"`
}

type ClassificationResult struct {
	Kind                string
	Confidence          float64
	Signals             []string
	ReviewRequired      bool
	Source              string
	ProviderFingerprint string
	Reason              string
}

type DocumentFeatureSet struct {
	TitleCandidates  []string
	SectionLabels    []string
	PartyLabels      []string
	ReferenceLabels  []string
	MoneySignals     []string
	StopSignals      []string
	TermsSignals     []string
	SignatureSignals []string
}

type ProviderFingerprint struct {
	Provider   string
	KindHint   string
	Confidence float64
	Signals    []string
}

type ReviewField struct {
	Label           string
	Value           string
	Confidence      float64
	Excerpt         string
	EvidenceExcerpt string
	PageNumber      int
	ReviewRequired  bool
	Conflict        bool
	Source          string
}

type ReviewConflict struct {
	Key             string
	Label           string
	Values          []string
	PageNumbers     []int
	EvidenceExcerpt string
	Source          string
}

type IntelligenceStop struct {
	Sequence            int
	Role                string
	Name                string
	AddressLine1        string
	AddressLine2        string
	City                string
	State               string
	PostalCode          string
	Date                string
	TimeWindow          string
	AppointmentRequired bool
	PageNumber          int
	EvidenceExcerpt     string
	Confidence          float64
	ReviewRequired      bool
	Source              string
}

type DocumentIntelligenceAnalysis struct {
	Kind                 string
	OverallConfidence    float64
	ReviewStatus         string
	MissingFields        []string
	Signals              []string
	ClassifierSource     string
	ProviderFingerprint  string
	ClassificationReason string
	ParsingRuleMetadata  *services.DocumentParsingRuleMetadata
	Conflicts            []*ReviewConflict
	Fields               map[string]*ReviewField
	Stops                []*IntelligenceStop
	RawExcerpt           string
}

func (a *DocumentIntelligenceAnalysis) ToMap() map[string]any {
	fields := make(map[string]any, len(a.Fields))
	for key, field := range a.Fields {
		fields[key] = map[string]any{
			"label":           field.Label,
			"value":           field.Value,
			"confidence":      field.Confidence,
			"excerpt":         field.Excerpt,
			"evidenceExcerpt": field.EvidenceExcerpt,
			"pageNumber":      field.PageNumber,
			"reviewRequired":  field.ReviewRequired,
			"conflict":        field.Conflict,
			"source":          field.Source,
		}
	}

	conflicts := make([]map[string]any, 0, len(a.Conflicts))
	for _, conflict := range a.Conflicts {
		conflicts = append(conflicts, map[string]any{
			"key":             conflict.Key,
			"label":           conflict.Label,
			"values":          conflict.Values,
			"pageNumbers":     conflict.PageNumbers,
			"evidenceExcerpt": conflict.EvidenceExcerpt,
			"source":          conflict.Source,
		})
	}

	stops := make([]map[string]any, 0, len(a.Stops))
	for _, stop := range a.Stops {
		stops = append(stops, map[string]any{
			"sequence":            stop.Sequence,
			"role":                stop.Role,
			"name":                stop.Name,
			"addressLine1":        stop.AddressLine1,
			"addressLine2":        stop.AddressLine2,
			"city":                stop.City,
			"state":               stop.State,
			"postalCode":          stop.PostalCode,
			"date":                stop.Date,
			"timeWindow":          stop.TimeWindow,
			"appointmentRequired": stop.AppointmentRequired,
			"pageNumber":          stop.PageNumber,
			"evidenceExcerpt":     stop.EvidenceExcerpt,
			"confidence":          stop.Confidence,
			"reviewRequired":      stop.ReviewRequired,
			"source":              stop.Source,
		})
	}

	return map[string]any{
		"kind":                 a.Kind,
		"overallConfidence":    a.OverallConfidence,
		"reviewStatus":         a.ReviewStatus,
		"missingFields":        a.MissingFields,
		"signals":              a.Signals,
		"classifierSource":     a.ClassifierSource,
		"providerFingerprint":  a.ProviderFingerprint,
		"classificationReason": a.ClassificationReason,
		"parsingRuleMetadata":  a.ParsingRuleMetadata,
		"conflicts":            conflicts,
		"fields":               fields,
		"stops":                stops,
		"rawExcerpt":           a.RawExcerpt,
	}
}

type aiAcceptanceStatus string

const (
	aiAcceptanceStatusNotAttempted aiAcceptanceStatus = "not_attempted"
	aiAcceptanceStatusPending      aiAcceptanceStatus = "pending"
	aiAcceptanceStatusAccepted     aiAcceptanceStatus = "accepted"
	aiAcceptanceStatusRejected     aiAcceptanceStatus = "rejected"
)

type AIDiagnostics struct {
	FallbackAnalysis  *DocumentIntelligenceAnalysis
	CandidateAnalysis *DocumentIntelligenceAnalysis
	AcceptanceStatus  aiAcceptanceStatus
	RejectionReason   string
	ResponseID        string
	SubmittedAt       *int64
	LastPolledAt      *int64
}

func (d *AIDiagnostics) ToMap() map[string]any {
	data := map[string]any{
		"acceptanceStatus": d.AcceptanceStatus,
		"rejectionReason":  d.RejectionReason,
		"fallbackAnalysis": d.FallbackAnalysis.ToMap(),
	}
	if d.CandidateAnalysis != nil {
		data["candidateAnalysis"] = d.CandidateAnalysis.ToMap()
	}
	if d.ResponseID != "" {
		data["responseId"] = d.ResponseID
	}
	if d.SubmittedAt != nil {
		data["submittedAt"] = *d.SubmittedAt
	}
	if d.LastPolledAt != nil {
		data["lastPolledAt"] = *d.LastPolledAt
	}
	return data
}

type EnrichmentPayload struct {
	Payload        *ProcessDocumentIntelligencePayload
	Document       *document.Document
	Control        *tenant.DocumentControl
	Extracted      *ExtractionResult
	Features       *DocumentFeatureSet
	Fingerprint    *ProviderFingerprint
	Classification *ClassificationResult
	Intelligence   *DocumentIntelligenceAnalysis
}

type EnrichmentResult struct {
	Classification       *ClassificationResult
	DocumentIntelligence *DocumentIntelligenceAnalysis
	AIDiagnostics        *AIDiagnostics
	EnqueueAsyncAI       bool
}

type ExtractionResult struct {
	Text       string
	PageCount  int
	SourceKind documentcontent.SourceKind
	Pages      []*PageExtractionResult
}

type ExtractionPipelineOutcome struct {
	Extracted      *ExtractionResult
	Classification *ClassificationResult
	Intelligence   *DocumentIntelligenceAnalysis
	AIDiagnostics  *AIDiagnostics
	EnqueueAsyncAI bool
}

type PageExtractionResult struct {
	PageNumber           int
	SourceKind           documentcontent.SourceKind
	Text                 string
	OCRConfidence        float64
	PreprocessingApplied bool
	Width                int
	Height               int
	Metadata             map[string]any
}

type PersistIndexedContentPayload struct {
	Document       *document.Document
	TenantInfo     pagination.TenantInfo
	Control        *tenant.DocumentControl
	Content        *documentcontent.Content
	Extracted      *ExtractionResult
	Classification *ClassificationResult
	Intelligence   *DocumentIntelligenceAnalysis
	AIDiagnostics  *AIDiagnostics
	Timestamp      int64
}

type PageSectionMatch struct {
	PageNumber int
	Value      string
	Excerpt    string
}

type AddFieldFromSectionLabelsParams struct {
	Fields         map[string]*ReviewField
	Signals        *[]string
	Key            string
	Label          string
	Pages          []*PageExtractionResult
	Labels         []string
	Confidence     float64
	Signal         string
	ReviewRequired bool
	Extractor      func(string, []string) string
}

type AddWeightFieldParams struct {
	Fields  map[string]*ReviewField
	Signals *[]string
	Text    string
	Signal  string
}

type AddStopTimingFieldParams struct {
	Fields     map[string]*ReviewField
	Signals    *[]string
	Key        string
	Label      string
	Stop       *IntelligenceStop
	Confidence float64
}

type RegexValueFieldParams struct {
	Fields         map[string]*ReviewField
	Signals        *[]string
	Key            string
	Label          string
	Regex          *regexp.Regexp
	Pages          []*PageExtractionResult
	Confidence     float64
	Signal         string
	ReviewRequired bool
}

type FinalizeIntelligenceParams struct {
	Document       *document.Document
	Payload        *ProcessDocumentIntelligencePayload
	Content        *documentcontent.Content
	Extracted      *ExtractionResult
	Classification *ClassificationResult
	Intelligence   *DocumentIntelligenceAnalysis
	AIDiagnostics  *AIDiagnostics
	Control        *tenant.DocumentControl
	TenantInfo     pagination.TenantInfo
	EnqueueAsyncAI bool
	Timestamp      int64
}

type InferredDocumentType struct {
	Code           string
	Name           string
	Category       documenttype.DocumentCategory
	Classification documenttype.DocumentClassification
	Color          string
}
