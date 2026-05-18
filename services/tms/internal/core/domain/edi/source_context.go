package edi

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type EDISourceContextSchema struct {
	bun.BaseModel `json:"-" bun:"table:edi_source_context_schemas,alias:escs"`

	ID             pulid.ID                 `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID                 `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),nullzero"`
	OrganizationID pulid.ID                 `json:"organizationId" bun:"organization_id,type:VARCHAR(100),nullzero"`
	Standard       EDIStandard              `json:"standard"       bun:"standard,type:edi_standard_enum,notnull"`
	TransactionSet TransactionSet           `json:"transactionSet" bun:"transaction_set,type:edi_transaction_set_enum,notnull"`
	Direction      DocumentDirection        `json:"direction"      bun:"direction,type:edi_document_direction_enum,notnull"`
	X12Version     string                   `json:"x12Version"     bun:"x12_version,type:VARCHAR(20),notnull"`
	ContextKey     string                   `json:"contextKey"     bun:"context_key,type:VARCHAR(100),notnull"`
	SchemaVersion  int64                    `json:"schemaVersion"  bun:"schema_version,type:BIGINT,notnull"`
	Name           string                   `json:"name"           bun:"name,type:VARCHAR(200),notnull"`
	Description    string                   `json:"description"    bun:"description,type:TEXT,nullzero"`
	Status         SourceContextFieldStatus `json:"status"         bun:"status,type:edi_source_context_field_status_enum,notnull"`
	CreatedAt      int64                    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64                    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Fields []*EDISourceContextField `json:"fields,omitempty" bun:"rel:has-many,join:id=schema_id"`
}

func (s *EDISourceContextSchema) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("edisc_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}
	return nil
}

type EDISourceContextField struct {
	bun.BaseModel `json:"-" bun:"table:edi_source_context_fields,alias:escf"`

	ID          pulid.ID                 `json:"id"          bun:"id,pk,type:VARCHAR(100),notnull"`
	SchemaID    pulid.ID                 `json:"schemaId"    bun:"schema_id,type:VARCHAR(100),notnull"`
	Path        string                   `json:"path"        bun:"path,type:TEXT,notnull"`
	SourceKind  SourceContextKind        `json:"sourceKind"  bun:"source_kind,type:edi_source_context_kind_enum,notnull"`
	DataType    SourceContextDataType    `json:"dataType"    bun:"data_type,type:edi_source_context_data_type_enum,notnull"`
	Repeated    bool                     `json:"repeated"    bun:"repeated,type:BOOLEAN,notnull,default:false"`
	RepeatPath  string                   `json:"repeatPath"  bun:"repeat_path,type:TEXT,nullzero"`
	ParentPath  string                   `json:"parentPath"  bun:"parent_path,type:TEXT,nullzero"`
	DisplayName string                   `json:"displayName" bun:"display_name,type:VARCHAR(200),notnull"`
	Description string                   `json:"description" bun:"description,type:TEXT,nullzero"`
	Status      SourceContextFieldStatus `json:"status"      bun:"status,type:edi_source_context_field_status_enum,notnull"`
	CreatedAt   int64                    `json:"createdAt"   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt   int64                    `json:"updatedAt"   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Schema *EDISourceContextSchema `json:"schema,omitempty" bun:"rel:belongs-to,join:schema_id=id"`
}

func (f *EDISourceContextField) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if f.ID.IsNil() {
			f.ID = pulid.MustNew("ediscf_")
		}
		f.CreatedAt = now
	case *bun.UpdateQuery:
		f.UpdatedAt = now
	}
	return nil
}
