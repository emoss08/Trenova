package aicorrection

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ pagination.CursorEntity        = (*Correction)(nil)
	_ domaintypes.PostgresSearchable = (*Correction)(nil)
)

var ErrNothingPredicted = errors.New("the source holds no prediction to compare")

const (
	MaxFieldResults       = 500
	MaxDocumentKindLength = 100
	MaxFingerprintLength  = 255
	MaxModelLength        = 255
)

type StopSnapshot struct {
	Role                 string `json:"role"`
	Sequence             int    `json:"sequence"`
	Name                 string `json:"name,omitempty"`
	AddressLine1         string `json:"addressLine1,omitempty"`
	AddressLine2         string `json:"addressLine2,omitempty"`
	City                 string `json:"city,omitempty"`
	State                string `json:"state,omitempty"`
	PostalCode           string `json:"postalCode,omitempty"`
	Date                 string `json:"date,omitempty"`
	TimeWindow           string `json:"timeWindow,omitempty"`
	AppointmentRequired  bool   `json:"appointmentRequired"`
	ScheduledWindowStart int64  `json:"scheduledWindowStart,omitempty"`
	ScheduledWindowEnd   *int64 `json:"scheduledWindowEnd,omitempty"`
	Timezone             string `json:"timezone,omitempty"`
}

type Snapshot struct {
	Fields map[string]string `json:"fields"`
	Stops  []StopSnapshot    `json:"stops"`
}

type FieldResult struct {
	Key        string  `json:"key"`
	Predicted  string  `json:"predicted,omitempty"`
	Confirmed  string  `json:"confirmed,omitempty"`
	Outcome    Outcome `json:"outcome"`
	Source     string  `json:"source,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

type Tally struct {
	Scored      int
	Correct     int
	Corrected   int
	Missed      int
	Unconfirmed int
	Unscored    int
}

func TallyResults(results []FieldResult) Tally {
	var t Tally
	for i := range results {
		switch results[i].Outcome {
		case OutcomeCorrect:
			t.Correct++
		case OutcomeCorrected:
			t.Corrected++
		case OutcomeMissed:
			t.Missed++
		case OutcomeUnconfirmed:
			t.Unconfirmed++
		case OutcomeUnscored:
			t.Unscored++
		}
	}
	t.Scored = t.Correct + t.Corrected + t.Missed

	return t
}

type Correction struct {
	keel.Table `keel:"ai_corrections"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:aicr_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Task         Task        `json:"task"         keel:"task"`
	SourceType   SourceType  `json:"sourceType"   keel:"source_type"`
	SourceID     pulid.ID    `json:"sourceId"     keel:"source_id"`
	DocumentID   *pulid.ID   `json:"documentId"   keel:"document_id"`
	SubjectType  SubjectType `json:"subjectType"  keel:"subject_type"`
	SubjectID    pulid.ID    `json:"subjectId"    keel:"subject_id"`
	CapturedByID pulid.ID    `json:"capturedById" keel:"captured_by_id"`

	DocumentKind         string    `json:"documentKind"         keel:"document_kind,nullzero"`
	DocumentFingerprint  string    `json:"documentFingerprint"  keel:"document_fingerprint,nullzero"`
	ExtractionModel      string    `json:"extractionModel"      keel:"extraction_model,nullzero"`
	ExtractionProviderID *pulid.ID `json:"extractionProviderId" keel:"extraction_provider_id"`
	PredictedConfidence  float64   `json:"predictedConfidence"  keel:"predicted_confidence"`

	Predicted    *Snapshot     `json:"predicted"    keel:"predicted,jsonb"`
	Confirmed    *Snapshot     `json:"confirmed"    keel:"confirmed,jsonb"`
	FieldResults []FieldResult `json:"fieldResults" keel:"field_results,jsonb"`

	ScoredCount      int `json:"scoredCount"      keel:"scored_count"`
	CorrectCount     int `json:"correctCount"     keel:"correct_count"`
	CorrectedCount   int `json:"correctedCount"   keel:"corrected_count"`
	MissedCount      int `json:"missedCount"      keel:"missed_count"`
	UnconfirmedCount int `json:"unconfirmedCount" keel:"unconfirmed_count"`
	UnscoredCount    int `json:"unscoredCount"    keel:"unscored_count"`

	CapturedAt int64 `json:"capturedAt" keel:"captured_at"`
	Version    int64 `json:"version"    keel:"version,version"`
	CreatedAt  int64 `json:"createdAt"  keel:"created_at,autocreate"`
	UpdatedAt  int64 `json:"updatedAt"  keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (c *Correction) ApplyTally() {
	t := TallyResults(c.FieldResults)
	c.ScoredCount = t.Scored
	c.CorrectCount = t.Correct
	c.CorrectedCount = t.Corrected
	c.MissedCount = t.Missed
	c.UnconfirmedCount = t.Unconfirmed
	c.UnscoredCount = t.Unscored
}

func (c *Correction) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&c.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&c.Task,
			validation.Required.Error("Task is required"),
			domainvalidation.ValidEnum[Task]("Task is invalid"),
		),
		validation.Field(&c.SourceType,
			validation.Required.Error("Source type is required"),
			domainvalidation.ValidEnum[SourceType]("Source type is invalid"),
		),
		validation.Field(&c.SourceID, validation.Required.Error("Source is required")),
		validation.Field(&c.SubjectType,
			validation.Required.Error("Subject type is required"),
			domainvalidation.ValidEnum[SubjectType]("Subject type is invalid"),
		),
		validation.Field(&c.SubjectID, validation.Required.Error("Subject is required")),
		validation.Field(&c.CapturedByID, validation.Required.Error("Captured by is required")),
		validation.Field(&c.DocumentKind,
			validation.Length(0, MaxDocumentKindLength).Error(
				fmt.Sprintf("Document kind must be at most %d characters", MaxDocumentKindLength),
			),
		),
		validation.Field(&c.DocumentFingerprint,
			validation.Length(0, MaxFingerprintLength).Error(
				fmt.Sprintf("Document fingerprint must be at most %d characters", MaxFingerprintLength),
			),
		),
		validation.Field(&c.ExtractionModel,
			validation.Length(0, MaxModelLength).Error(
				fmt.Sprintf("Extraction model must be at most %d characters", MaxModelLength),
			),
		),
		validation.Field(&c.PredictedConfidence,
			validation.Min(0.0).Error("Predicted confidence must be between 0 and 1"),
			validation.Max(1.0).Error("Predicted confidence must be between 0 and 1"),
		),
		validation.Field(&c.Predicted, validation.NotNil.Error("Predicted snapshot is required")),
		validation.Field(&c.Confirmed, validation.NotNil.Error("Confirmed snapshot is required")),
		validation.Field(&c.FieldResults,
			validation.Length(0, MaxFieldResults).Error(
				fmt.Sprintf("At most %d field results are kept", MaxFieldResults),
			),
		),
		validation.Field(&c.CapturedAt, validation.Required.Error("Captured at is required")),
	))

	for i := range c.FieldResults {
		if !c.FieldResults[i].Outcome.IsValid() {
			multiErr.Add(
				fmt.Sprintf("fieldResults[%d].outcome", i),
				errortypes.ErrInvalid,
				"Outcome is invalid",
			)
		}
		if c.FieldResults[i].Key == "" {
			multiErr.Add(
				fmt.Sprintf("fieldResults[%d].key", i),
				errortypes.ErrRequired,
				"Key is required",
			)
		}
	}
}

func (c *Correction) GetID() pulid.ID { return c.ID }

func (c *Correction) GetCreatedAt() int64 { return c.CreatedAt }

func (c *Correction) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *Correction) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *Correction) GetTableName() string { return "ai_corrections" }

func (c *Correction) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "aicr",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "document_kind", Type: domaintypes.FieldTypeText},
			{Name: "document_fingerprint", Type: domaintypes.FieldTypeText},
			{Name: "extraction_model", Type: domaintypes.FieldTypeText},
			{Name: "task", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (c *Correction) BeforeCreate(_ context.Context) error {
	if c.CapturedAt == 0 {
		c.CapturedAt = timeutils.NowUnix()
	}

	return nil
}
