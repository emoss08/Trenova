package aicorrection

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*Correction)(nil)

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
	bun.BaseModel `bun:"table:ai_corrections,alias:aicr" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Task         Task        `json:"task"        bun:"task,type:VARCHAR(50),notnull"`
	SourceType   SourceType  `json:"sourceType"  bun:"source_type,type:VARCHAR(50),notnull"`
	SourceID     pulid.ID    `json:"sourceId"    bun:"source_id,type:VARCHAR(100),notnull"`
	DocumentID   *pulid.ID   `json:"documentId"  bun:"document_id,type:VARCHAR(100),nullzero"`
	SubjectType  SubjectType `json:"subjectType" bun:"subject_type,type:VARCHAR(50),notnull"`
	SubjectID    pulid.ID    `json:"subjectId"   bun:"subject_id,type:VARCHAR(100),notnull"`
	CapturedByID pulid.ID    `json:"capturedById" bun:"captured_by_id,type:VARCHAR(100),notnull"`

	DocumentKind         string    `json:"documentKind"         bun:"document_kind,type:VARCHAR(100),nullzero"`
	DocumentFingerprint  string    `json:"documentFingerprint"  bun:"document_fingerprint,type:VARCHAR(255),nullzero"`
	ExtractionModel      string    `json:"extractionModel"      bun:"extraction_model,type:VARCHAR(255),nullzero"`
	ExtractionProviderID *pulid.ID `json:"extractionProviderId" bun:"extraction_provider_id,type:VARCHAR(100),nullzero"`
	PredictedConfidence  float64   `json:"predictedConfidence"  bun:"predicted_confidence,type:DOUBLE PRECISION,notnull,default:0"`

	Predicted    *Snapshot     `json:"predicted"    bun:"predicted,type:JSONB,notnull"`
	Confirmed    *Snapshot     `json:"confirmed"    bun:"confirmed,type:JSONB,notnull"`
	FieldResults []FieldResult `json:"fieldResults" bun:"field_results,type:JSONB,notnull"`

	ScoredCount      int `json:"scoredCount"      bun:"scored_count,type:INTEGER,notnull,default:0"`
	CorrectCount     int `json:"correctCount"     bun:"correct_count,type:INTEGER,notnull,default:0"`
	CorrectedCount   int `json:"correctedCount"   bun:"corrected_count,type:INTEGER,notnull,default:0"`
	MissedCount      int `json:"missedCount"      bun:"missed_count,type:INTEGER,notnull,default:0"`
	UnconfirmedCount int `json:"unconfirmedCount" bun:"unconfirmed_count,type:INTEGER,notnull,default:0"`
	UnscoredCount    int `json:"unscoredCount"    bun:"unscored_count,type:INTEGER,notnull,default:0"`

	CapturedAt int64 `json:"capturedAt" bun:"captured_at,type:BIGINT,notnull"`
	Version    int64 `json:"version"    bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt  int64 `json:"createdAt"  bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt  int64 `json:"updatedAt"  bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
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

func (c *Correction) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *Correction) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *Correction) GetTableName() string { return "ai_corrections" }

func (c *Correction) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("aicr_")
		}
		if c.CreatedAt == 0 {
			c.CreatedAt = now
		}
		if c.CapturedAt == 0 {
			c.CapturedAt = now
		}
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
