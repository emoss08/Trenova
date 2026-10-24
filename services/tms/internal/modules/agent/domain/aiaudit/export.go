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
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*AIAuditExport)(nil)
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
	bun.BaseModel `bun:"table:ai_audit_exports,alias:aiax" json:"-"`
	keel.Table    `keel:"ai_audit_exports"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:aiax_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)" keel:"organization_id,pk,tenant"`

	RequestedByUserID pulid.ID      `json:"requestedByUserId" bun:"requested_by_user_id,type:VARCHAR(100),notnull" keel:"requested_by_user_id"`
	Format            ExportFormat  `json:"format"            bun:"format,type:VARCHAR(10),notnull" keel:"format"`
	Filters           *ExportFilter `json:"filters"           bun:"filters,type:JSONB,notnull,default:'{}'::jsonb" keel:"filters,jsonb"`
	RangeFrom         int64         `json:"rangeFrom"         bun:"range_from,type:BIGINT,notnull" keel:"range_from"`
	RangeTo           int64         `json:"rangeTo"           bun:"range_to,type:BIGINT,notnull" keel:"range_to"`
	// SnapshotSeq is the end of the tenant's chain when the export was asked
	// for. The file holds nothing recorded after it, so a count taken first
	// and the rows written later agree.
	SnapshotSeq int64        `json:"snapshotSeq" bun:"snapshot_seq,type:BIGINT,notnull,default:0" keel:"snapshot_seq"`
	Status      ExportStatus `json:"status"      bun:"status,type:VARCHAR(20),notnull,default:'Pending'" keel:"status"`

	RowCount          int64  `json:"rowCount"          bun:"row_count,type:BIGINT,notnull,default:0" keel:"row_count"`
	ByteSize          int64  `json:"byteSize"          bun:"byte_size,type:BIGINT,notnull,default:0" keel:"byte_size"`
	SHA256            string `json:"sha256"            bun:"sha256,type:VARCHAR(64),nullzero" keel:"sha256,nullzero"`
	ArtifactKey       string `json:"artifactKey"       bun:"artifact_key,type:VARCHAR(500),nullzero" keel:"artifact_key,nullzero"`
	ArtifactExpiresAt *int64 `json:"artifactExpiresAt" bun:"artifact_expires_at,type:BIGINT,nullzero" keel:"artifact_expires_at"`

	ChainKeyID    string `json:"chainKeyId"    bun:"chain_key_id,type:VARCHAR(40),nullzero" keel:"chain_key_id,nullzero"`
	ChainFirstSeq *int64 `json:"chainFirstSeq" bun:"chain_first_seq,type:BIGINT,nullzero" keel:"chain_first_seq"`
	ChainLastSeq  *int64 `json:"chainLastSeq"  bun:"chain_last_seq,type:BIGINT,nullzero" keel:"chain_last_seq"`
	ChainComplete bool   `json:"chainComplete" bun:"chain_complete,type:BOOLEAN,notnull,default:false" keel:"chain_complete"`

	WorkflowID   string `json:"workflowId"   bun:"workflow_id,type:VARCHAR(255),nullzero" keel:"workflow_id,nullzero"`
	ErrorMessage string `json:"errorMessage" bun:"error_message,type:TEXT,nullzero" keel:"error_message,nullzero"`
	StartedAt    *int64 `json:"startedAt"    bun:"started_at,type:BIGINT,nullzero" keel:"started_at"`
	CompletedAt  *int64 `json:"completedAt"  bun:"completed_at,type:BIGINT,nullzero" keel:"completed_at"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
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

func (x *AIAuditExport) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if x.ID.IsNil() {
			x.ID = pulid.MustNew(ExportIDPrefix)
		}
		if x.Status == "" {
			x.Status = ExportStatusPending
		}
		if x.Filters == nil {
			x.Filters = &ExportFilter{}
		}
		x.CreatedAt = now
		x.UpdatedAt = now
	case *bun.UpdateQuery:
		x.UpdatedAt = now
	}

	return nil
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
