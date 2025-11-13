package report

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
)

type Report struct {
	bun.BaseModel `bun:"table:reports,alias:rpt"`

	ID             pulid.ID                `json:"id"             bun:"id,pk,type:TEXT"`
	OrganizationID pulid.ID                `json:"organizationId" bun:"organization_id,notnull,type:TEXT"`
	BusinessUnitID pulid.ID                `json:"businessUnitId" bun:"business_unit_id,notnull,type:TEXT"`
	UserID         pulid.ID                `json:"userId"         bun:"user_id,notnull,type:TEXT"`
	ResourceType   string                  `json:"resourceType"   bun:"resource_type,notnull"`
	Name           string                  `json:"name"           bun:"name,notnull"`
	Format         Format                  `json:"format"         bun:"format,notnull,type:TEXT"`
	DeliveryMethod DeliveryMethod          `json:"deliveryMethod" bun:"delivery_method,notnull,type:TEXT"`
	Status         Status                  `json:"status"         bun:"status,notnull,type:TEXT,default:'PENDING'"`
	FilterState    pagination.QueryOptions `json:"filterState"    bun:"filter_state,type:jsonb"`
	FilePath       string                  `json:"filePath"       bun:"file_path"`
	FileSize       int64                   `json:"fileSize"       bun:"file_size"`
	RowCount       int                     `json:"rowCount"       bun:"row_count"`
	ErrorMessage   string                  `json:"errorMessage"   bun:"error_message"`
	CompletedAt    *int64                  `json:"completedAt"    bun:"completed_at"`
	ExpiresAt      *int64                  `json:"expiresAt"      bun:"expires_at"`

	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`
	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}
