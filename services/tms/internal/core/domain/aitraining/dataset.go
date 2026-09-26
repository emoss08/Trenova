package aitraining

import (
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DatasetFormat         = "trenova.extraction-dataset/v1"
	DatasetManifestFile   = "dataset-manifest.json"
	DatasetSchemaFile     = "schema.json"
	DatasetSFTTrainFile   = "sft-train.jsonl"
	DatasetSFTValidFile   = "sft-validation.jsonl"
	DatasetPreferenceFile = "preference-train.jsonl"
	DatasetEvaluationFile = "eval-validation.jsonl"
	RoleSystem            = "system"
	RoleUser              = "user"
	RoleAssistant         = "assistant"
	TargetVerifiedScore   = 0.95
	TargetUnverifiedScore = 0.7
	TargetOverallScore    = 0.9
	TargetReviewStatus    = "Ready"
	TargetSource          = "ai"
	DefaultTargetKind     = "RateConfirmation"
	EvidenceContextRunes  = 60
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type SFTRecord struct {
	ID         string        `json:"id"`
	Prompt     []ChatMessage `json:"prompt"`
	Completion []ChatMessage `json:"completion"`
}

type PreferenceRecord struct {
	ID       string        `json:"id"`
	Prompt   []ChatMessage `json:"prompt"`
	Chosen   []ChatMessage `json:"chosen"`
	Rejected []ChatMessage `json:"rejected"`
}

type EvaluationRecord struct {
	ID       string           `json:"id"`
	Prompt   []ChatMessage    `json:"prompt"`
	Expected *ExampleSnapshot `json:"expected"`
	Baseline *ExampleSnapshot `json:"baseline"`
}

type PredictionRecord struct {
	ID    string `json:"id"`
	Reply string `json:"reply"`
	Error string `json:"error,omitempty"`
}

type DatasetFile struct {
	Name    string `json:"name"`
	Records int    `json:"records"`
	Bytes   int64  `json:"bytes"`
	SHA256  string `json:"sha256"`
}

type DatasetCounts struct {
	Examples          int `json:"examples"`
	Train             int `json:"train"`
	Validation        int `json:"validation"`
	Preference        int `json:"preference"`
	WithdrawnExcluded int `json:"withdrawnExcluded"`
}

type DatasetManifest struct {
	Format               string                          `json:"format"`
	ExportID             pulid.ID                        `json:"exportId"`
	ExportManifestSHA256 string                          `json:"exportManifestSha256"`
	ExampleFormat        string                          `json:"exampleFormat"`
	Task                 aicorrection.Task               `json:"task"`
	StructuredOutputMode aiprovider.StructuredOutputMode `json:"structuredOutputMode"`
	SchemaName           string                          `json:"schemaName"`
	PromptSHA256         string                          `json:"promptSha256"`
	Temperature          *float64                        `json:"temperature,omitempty"`
	TopP                 *float64                        `json:"topP,omitempty"`
	KeepUnverified       bool                            `json:"keepUnverified"`
	Counts               DatasetCounts                   `json:"counts"`
	Files                []DatasetFile                   `json:"files"`
	RenderedAt           int64                           `json:"renderedAt"`
}

func (s *ExampleSnapshot) Correction() *aicorrection.Snapshot {
	out := &aicorrection.Snapshot{Fields: map[string]string{}, Stops: []aicorrection.StopSnapshot{}}
	if s == nil {
		return out
	}
	for key, value := range s.Fields {
		out.Fields[key] = value
	}
	for i := range s.Stops {
		stop := &s.Stops[i]
		out.Stops = append(out.Stops, aicorrection.StopSnapshot{
			Role:                stop.Role,
			Sequence:            stop.Sequence,
			Name:                stop.Name,
			AddressLine1:        stop.AddressLine1,
			AddressLine2:        stop.AddressLine2,
			City:                stop.City,
			State:               stop.State,
			PostalCode:          stop.PostalCode,
			Date:                stop.Date,
			TimeWindow:          stop.TimeWindow,
			AppointmentRequired: stop.AppointmentRequired,
		})
	}

	return out
}

func (e *Example) NeedsCorrection() bool {
	for _, outcome := range e.Outcomes {
		if outcome == aicorrection.OutcomeCorrected || outcome == aicorrection.OutcomeMissed {
			return true
		}
	}

	return false
}
