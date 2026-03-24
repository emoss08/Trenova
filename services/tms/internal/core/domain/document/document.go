package document

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Document)(nil)
	_ domaintypes.PostgresSearchable = (*Document)(nil)
)

type Status string

const (
	StatusDraft           Status = "Draft"
	StatusActive          Status = "Active"
	StatusArchived        Status = "Archived"
	StatusExpired         Status = "Expired"
	StatusPending         Status = "Pending"
	StatusRejected        Status = "Rejected"
	StatusPendingApproval Status = "PendingApproval"
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusDraft,
		StatusActive,
		StatusArchived,
		StatusExpired,
		StatusPending,
		StatusRejected,
		StatusPendingApproval:
		return true
	}
	return false
}

type Document struct {
	bun.BaseModel `bun:"table:documents,alias:doc" json:"-"`

	ID                 pulid.ID  `json:"id"                 bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID     pulid.ID  `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	BusinessUnitID     pulid.ID  `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	FileName           string    `json:"fileName"           bun:"file_name,type:VARCHAR(255),notnull"`
	OriginalName       string    `json:"originalName"       bun:"original_name,type:VARCHAR(255),notnull"`
	FileSize           int64     `json:"fileSize"           bun:"file_size,type:BIGINT,notnull"`
	FileType           string    `json:"fileType"           bun:"file_type,type:VARCHAR(100),notnull"`
	StoragePath        string    `json:"storagePath"        bun:"storage_path,type:VARCHAR(500),notnull"`
	Status             Status    `json:"status"             bun:"status,type:document_status_enum,notnull,default:'Active'"`
	Description        string    `json:"description"        bun:"description,type:TEXT,nullzero"`
	ResourceID         string    `json:"resourceId"         bun:"resource_id,type:VARCHAR(100),notnull"`
	ResourceType       string    `json:"resourceType"       bun:"resource_type,type:VARCHAR(100),notnull"`
	ExpirationDate     *int64    `json:"expirationDate"     bun:"expiration_date,type:BIGINT,nullzero"`
	Tags               []string  `json:"tags"               bun:"tags,type:VARCHAR(100)[],default:'{}'"`
	IsPublic           bool      `json:"isPublic"           bun:"is_public,type:BOOLEAN,notnull,default:false"`
	UploadedByID       pulid.ID  `json:"uploadedById"       bun:"uploaded_by_id,type:VARCHAR(100),notnull"`
	ApprovedByID       pulid.ID  `json:"approvedById"       bun:"approved_by_id,type:VARCHAR(100),nullzero"`
	ApprovedAt         *int64    `json:"approvedAt"         bun:"approved_at,type:BIGINT,nullzero"`
	PreviewStoragePath string    `json:"previewStoragePath" bun:"preview_storage_path,type:VARCHAR(500),nullzero"`
	DocumentTypeID     *pulid.ID `json:"documentTypeId"     bun:"document_type_id,type:VARCHAR(100),nullzero"`
	SearchVector       string    `json:"-"                  bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank               string    `json:"-"                  bun:"rank,type:VARCHAR(100),scanonly"`
	Version            int64     `json:"version"            bun:"version,type:BIGINT"`
	CreatedAt          int64     `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64     `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (d *Document) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("doc_")
		}
		d.CreatedAt = now
		d.UpdatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}

func (d *Document) GetID() pulid.ID {
	return d.ID
}

func (d *Document) GetOrganizationID() pulid.ID {
	return d.OrganizationID
}

func (d *Document) GetBusinessUnitID() pulid.ID {
	return d.BusinessUnitID
}

func (d *Document) GetTableName() string {
	return "documents"
}

func (d *Document) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "doc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "file_name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "original_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}
