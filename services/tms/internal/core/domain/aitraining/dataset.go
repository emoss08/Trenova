package aitraining

import (
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DatasetFormat         = "trenova.extraction-dataset/v2"
	DatasetManifestFile   = "dataset-manifest.json"
	DatasetSchemaFile     = "schema.json"
	DatasetTrainFile      = "examples-train.jsonl"
	DatasetValidationFile = "examples-validation.jsonl"
	DatasetEvaluationFile = "eval-validation.jsonl"
	RoleSystem            = "system"
	RoleUser              = "user"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TrainingRecord struct {
	ID           string                          `json:"id"`
	Split        Split                           `json:"split"`
	DocumentKind string                          `json:"documentKind,omitempty"`
	Prompt       []ChatMessage                   `json:"prompt"`
	VisiblePages []ExamplePage                   `json:"visiblePages"`
	Target       *ExampleSnapshot                `json:"target"`
	Prediction   *ExampleSnapshot                `json:"prediction"`
	Outcomes     map[string]aicorrection.Outcome `json:"outcomes"`
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
	PageLimit            int                             `json:"pageLimit"`
	FieldKeys            []string                        `json:"fieldKeys"`
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
