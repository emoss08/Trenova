package audit

import (
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type ExportFormat string

const (
	ExportFormatCSV   = ExportFormat("CSV")
	ExportFormatJSONL = ExportFormat("JSONL")
)

func (f ExportFormat) Valid() bool {
	return f == ExportFormatCSV || f == ExportFormatJSONL
}

func (f ExportFormat) Extension() string {
	if f == ExportFormatJSONL {
		return "jsonl"
	}
	return "csv"
}

func (f ExportFormat) ContentType() string {
	if f == ExportFormatJSONL {
		return "application/x-ndjson"
	}
	return "text/csv"
}

type ExportStatus string

const (
	ExportStatusPending   = ExportStatus("Pending")
	ExportStatusRunning   = ExportStatus("Running")
	ExportStatusSucceeded = ExportStatus("Succeeded")
	ExportStatusFailed    = ExportStatus("Failed")
	ExportStatusExpired   = ExportStatus("Expired")
)

func (s ExportStatus) Terminal() bool {
	return s == ExportStatusSucceeded || s == ExportStatusFailed || s == ExportStatusExpired
}

const MaxExportRangeDays = 400

type ExportFilters struct {
	Resources     []string `json:"resources,omitempty"`
	Operations    []string `json:"operations,omitempty"`
	Categories    []string `json:"categories,omitempty"`
	Channels      []string `json:"channels,omitempty"`
	PrincipalID   string   `json:"principalId,omitempty"`
	ResourceID    string   `json:"resourceId,omitempty"`
	VisibleGroups uint8    `json:"visibleGroups"`
}

type Export struct {
	keel.Table `keel:"audit_exports"`

	ID                pulid.ID      `json:"id"                          keel:"id,pk,autoid:aex_"`
	BusinessUnitID    pulid.ID      `json:"businessUnitId"              keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID      `json:"organizationId"              keel:"organization_id,pk,tenant"`
	RequestedByID     pulid.ID      `json:"requestedById"               keel:"requested_by_id"`
	Format            ExportFormat  `json:"format"                      keel:"format"`
	Filters           ExportFilters `json:"filters"                     keel:"filters,jsonb"`
	RangeFrom         int64         `json:"rangeFrom"                   keel:"range_from"`
	RangeTo           int64         `json:"rangeTo"                     keel:"range_to"`
	IncludeSecurity   bool          `json:"includeSecurity"             keel:"include_security"`
	Status            ExportStatus  `json:"status"                      keel:"status"`
	RowCount          int64         `json:"rowCount"                    keel:"row_count"`
	ByteSize          int64         `json:"byteSize"                    keel:"byte_size"`
	SHA256            string        `json:"sha256,omitempty"            keel:"sha256,nullzero"`
	ArtifactKey       string        `json:"-"                           keel:"artifact_key,nullzero"`
	ArtifactExpiresAt *int64        `json:"artifactExpiresAt,omitempty" keel:"artifact_expires_at"`
	WorkflowID        string        `json:"workflowId,omitempty"        keel:"workflow_id,nullzero"`
	ErrorMessage      string        `json:"errorMessage,omitempty"      keel:"error_message,nullzero"`
	StartedAt         *int64        `json:"startedAt,omitempty"         keel:"started_at"`
	CompletedAt       *int64        `json:"completedAt,omitempty"       keel:"completed_at"`
	Version           int64         `json:"version"                     keel:"version,version"`
	CreatedAt         int64         `json:"createdAt"                   keel:"created_at,autocreate"`
	UpdatedAt         int64         `json:"updatedAt"                   keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (e *Export) GetTableName() string {
	return "audit_exports"
}

func (e *Export) GetID() pulid.ID {
	return e.ID
}

func (e *Export) GetCreatedAt() int64 {
	return e.CreatedAt
}

func (e *Export) Downloadable(now int64) bool {
	return e.Status == ExportStatusSucceeded && e.ArtifactKey != "" &&
		(e.ArtifactExpiresAt == nil || *e.ArtifactExpiresAt > now)
}

func (e *Export) FileName() string {
	return fmt.Sprintf(
		"audit-log-%s-to-%s.%s",
		time.Unix(e.RangeFrom, 0).UTC().Format("2006-01-02"),
		time.Unix(e.RangeTo, 0).UTC().Format("2006-01-02"),
		e.Format.Extension(),
	)
}

func (e *Export) Validate(multiErr *errortypes.MultiError) {
	if !e.Format.Valid() {
		multiErr.Add("format", errortypes.ErrInvalid, "Choose CSV or JSON Lines")
	}
	switch {
	case e.RangeFrom <= 0 || e.RangeTo <= 0:
		multiErr.Add("rangeFrom", errortypes.ErrRequired, "Choose the dates to export")
	case e.RangeTo <= e.RangeFrom:
		multiErr.Add(
			"rangeTo",
			errortypes.ErrInvalid,
			"The end of the range must be after its start",
		)
	case e.RangeTo-e.RangeFrom > int64(MaxExportRangeDays)*86400:
		multiErr.Add(
			"rangeTo",
			errortypes.ErrInvalid,
			"An export covers at most 400 days; split a longer range into several",
		)
	}
}
