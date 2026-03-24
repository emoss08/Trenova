package audit

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type DLQStatus string

const (
	DLQStatusPending   DLQStatus = "pending"
	DLQStatusRetrying  DLQStatus = "retrying"
	DLQStatusFailed    DLQStatus = "failed"
	DLQStatusRecovered DLQStatus = "recovered"
)

var _ bun.BeforeAppendModelHook = (*DLQEntry)(nil)

type DLQEntry struct {
	bun.BaseModel `bun:"table:audit_dlq_entries,alias:adle"`

	ID              pulid.ID       `json:"id"              bun:"id,pk,type:VARCHAR(100)"`
	OriginalEntryID pulid.ID       `json:"originalEntryId" bun:"original_entry_id,type:VARCHAR(100),notnull"`
	EntryData       map[string]any `json:"entryData"       bun:"entry_data,type:JSONB,notnull"`
	FailureTime     int64          `json:"failureTime"     bun:"failure_time,notnull"`
	RetryCount      int            `json:"retryCount"      bun:"retry_count,notnull,default:0"`
	LastError       string         `json:"lastError"       bun:"last_error,type:TEXT"`
	NextRetryAt     int64          `json:"nextRetryAt"     bun:"next_retry_at"`
	Status          DLQStatus      `json:"status"          bun:"status,type:VARCHAR(20),notnull,default:'pending'"`
	OrganizationID  pulid.ID       `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID       `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),notnull"`
	CreatedAt       int64          `json:"createdAt"       bun:"created_at,notnull"`
	UpdatedAt       int64          `json:"updatedAt"       bun:"updated_at,notnull"`
}

func (e *DLQEntry) Validate() error {
	return validation.ValidateStruct(
		e,
		validation.Field(
			&e.OriginalEntryID,
			validation.Required.Error("Original entry ID is required"),
		),
		validation.Field(
			&e.EntryData,
			validation.Required.Error("Entry data is required"),
		),
		validation.Field(
			&e.FailureTime,
			validation.Required.Error("Failure time is required"),
		),
		validation.Field(
			&e.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&e.BusinessUnitID,
			validation.Required.Error("Business unit ID is required"),
		),
	)
}

func (e *DLQEntry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("dlq_")
		}
		if e.Status == "" {
			e.Status = DLQStatusPending
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}
	return nil
}

func (e *DLQEntry) CanRetry(maxRetries int) bool {
	return e.RetryCount < maxRetries &&
		e.Status != DLQStatusRecovered &&
		e.Status != DLQStatusFailed
}

func (e *DLQEntry) IncrementRetry() {
	e.RetryCount++
	e.Status = DLQStatusRetrying
}

func (e *DLQEntry) MarkRecovered() {
	e.Status = DLQStatusRecovered
}

func (e *DLQEntry) MarkFailed(err string) {
	e.Status = DLQStatusFailed
	e.LastError = err
}
