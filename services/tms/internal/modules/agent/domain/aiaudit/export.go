package aiaudit

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ pagination.CursorEntity        = (*AIAuditExport)(nil)
	_ domaintypes.PostgresSearchable = (*AIAuditExport)(nil)
)

const (
	ExportIDPrefix = "aiax_"
	// MaxExportRangeSeconds bounds one export to about seven years, the
	// longest the trail is kept by default.
	MaxExportRangeSeconds = int64(2556 * 24 * 60 * 60)
	maxExportFilters      = 20
)

// ExportFilter is the trail filter an export was asked for, kept on the
// export so its file can be explained later.
type ExportFilter struct {
	Query        string                    `json:"query,omitempty"`
	FieldFilters []domaintypes.FieldFilter `json:"fieldFilters,omitempty"`
	FilterGroups []domaintypes.FilterGroup `json:"filterGroups,omitempty"`
	Sort         []domaintypes.SortField   `json:"sort,omitempty"`
}

// Unfiltered reports a filter that keeps every row of its range, which is
// what an export needs before its chain can be called complete.
func (f *ExportFilter) Unfiltered() bool {
	if f == nil {
		return true
	}

	return f.Query == "" && len(f.FieldFilters) == 0 &&
		!slices.ContainsFunc(f.FilterGroups, func(group domaintypes.FilterGroup) bool {
			return len(group.Filters) > 0
		})
}

// AIAuditExport is one request to write the trail to a file, and the file.
type AIAuditExport struct {
	keel.Table `keel:"ai_audit_exports"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:aiax_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	RequestedByUserID pulid.ID      `json:"requestedByUserId" keel:"requested_by_user_id"`
	Format            ExportFormat  `json:"format"            keel:"format"`
	Filters           *ExportFilter `json:"filters"           keel:"filters,jsonb"`
	RangeFrom         int64         `json:"rangeFrom"         keel:"range_from"`
	RangeTo           int64         `json:"rangeTo"           keel:"range_to"`
	// SnapshotSeq is the end of the tenant's chain when the export was asked
	// for. The file holds nothing recorded after it, so a count taken first
	// and the rows written later agree.
	SnapshotSeq int64        `json:"snapshotSeq" keel:"snapshot_seq"`
	Status      ExportStatus `json:"status"      keel:"status"`

	RowCount          int64  `json:"rowCount"          keel:"row_count"`
	ByteSize          int64  `json:"byteSize"          keel:"byte_size"`
	SHA256            string `json:"sha256"            keel:"sha256,nullzero"`
	ArtifactKey       string `json:"artifactKey"       keel:"artifact_key,nullzero"`
	ArtifactExpiresAt *int64 `json:"artifactExpiresAt" keel:"artifact_expires_at"`

	ChainKeyID    string `json:"chainKeyId"    keel:"chain_key_id,nullzero"`
	ChainFirstSeq *int64 `json:"chainFirstSeq" keel:"chain_first_seq"`
	ChainLastSeq  *int64 `json:"chainLastSeq"  keel:"chain_last_seq"`
	ChainComplete bool   `json:"chainComplete" keel:"chain_complete"`

	WorkflowID   string `json:"workflowId"   keel:"workflow_id,nullzero"`
	ErrorMessage string `json:"errorMessage" keel:"error_message,nullzero"`
	StartedAt    *int64 `json:"startedAt"    keel:"started_at"`
	CompletedAt  *int64 `json:"completedAt"  keel:"completed_at"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (x *AIAuditExport) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		x,
		validation.Field(&x.RequestedByUserID,
			validation.Required.Error("An export must be requested by a person"),
		),
		validation.Field(&x.Format,
			validation.Required.Error("Format is required"),
			validation.By(func(any) error {
				if !x.Format.IsValid() {
					return validation.NewError("invalid_format", "Format must be CSV or JSON")
				}
				return nil
			}),
		),
		validation.Field(&x.RangeFrom,
			validation.Required.Error("The start of the range is required"),
			validation.Min(int64(1)).Error("The start of the range is required"),
		),
		validation.Field(&x.RangeTo,
			validation.Required.Error("The end of the range is required"),
		),
	))

	if x.RangeFrom > 0 && x.RangeTo > 0 {
		if x.RangeTo < x.RangeFrom {
			multiErr.Add(
				"to",
				errortypes.ErrInvalid,
				"The end of the range must not be before its start",
			)
		} else if x.RangeTo-x.RangeFrom > MaxExportRangeSeconds {
			multiErr.Add("to", errortypes.ErrInvalid, "An export can cover at most seven years")
		}
	}

	if x.Filters != nil &&
		len(x.Filters.FieldFilters)+len(x.Filters.FilterGroups) > maxExportFilters {
		multiErr.Add("fieldFilters", errortypes.ErrInvalid, "An export takes at most 20 filters")
	}
}

// Downloadable reports whether the export's file can still be fetched.
func (x *AIAuditExport) Downloadable(now int64) bool {
	if x.Status != ExportStatusSucceeded || x.ArtifactKey == "" {
		return false
	}

	return x.ArtifactExpiresAt == nil || *x.ArtifactExpiresAt > now
}

// FileName is what the download is saved as.
func (x *AIAuditExport) FileName() string {
	return "ai-audit-trail-" + x.ID.String() + "." + x.Format.Extension()
}

func (x *AIAuditExport) GetID() pulid.ID {
	return x.ID
}

func (x *AIAuditExport) GetCreatedAt() int64 {
	return x.CreatedAt
}

func (x *AIAuditExport) GetOrganizationID() pulid.ID {
	return x.OrganizationID
}

func (x *AIAuditExport) GetBusinessUnitID() pulid.ID {
	return x.BusinessUnitID
}

func (x *AIAuditExport) GetTableName() string {
	return "ai_audit_exports"
}

func (x *AIAuditExport) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "aiax",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "format", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (x *AIAuditExport) BeforeCreate(_ context.Context) error {
	if x.Status == "" {
		x.Status = ExportStatusPending
	}
	if x.Filters == nil {
		x.Filters = &ExportFilter{}
	}

	return nil
}
