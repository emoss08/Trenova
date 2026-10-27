package audit

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type DLQStatus string

const (
	DLQStatusPending   DLQStatus = "pending"
	DLQStatusRetrying  DLQStatus = "retrying"
	DLQStatusFailed    DLQStatus = "failed"
	DLQStatusRecovered DLQStatus = "recovered"
)

type DLQEntry struct {
	keel.Table `keel:"audit_dlq_entries"`

	ID              pulid.ID       `json:"id"              keel:"id,pk,autoid:dlq_"`
	OriginalEntryID pulid.ID       `json:"originalEntryId" keel:"original_entry_id"`
	EntryData       map[string]any `json:"entryData"       keel:"entry_data,jsonb"`
	FailureTime     int64          `json:"failureTime"     keel:"failure_time"`
	RetryCount      int            `json:"retryCount"      keel:"retry_count"`
	LastError       string         `json:"lastError"       keel:"last_error,nullzero"`
	NextRetryAt     int64          `json:"nextRetryAt"     keel:"next_retry_at,nullzero"`
	Status          DLQStatus      `json:"status"          keel:"status"`
	OrganizationID  pulid.ID       `json:"organizationId"  keel:"organization_id,tenant"`
	BusinessUnitID  pulid.ID       `json:"businessUnitId"  keel:"business_unit_id,tenant"`
	CreatedAt       int64          `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64          `json:"updatedAt"       keel:"updated_at,autoupdate"`
}

func (e *DLQEntry) BeforeCreate(_ context.Context) error {
	if e.Status == "" {
		e.Status = DLQStatusPending
	}

	return nil
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
