package edi

import (
	"context"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	Repeated    bool                     `json:"repeated"    bun:"repeated,type:BOOLEAN,notnull"`
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

func (s *EDISourceContextSchema) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.Standard,
			validation.Required.Error("Standard is required"),
			domainvalidation.ValidEnum[EDIStandard]("Standard is invalid"),
		),
		validation.Field(&s.TransactionSet,
			validation.Required.Error("Transaction set is required"),
			domainvalidation.ValidEnum[TransactionSet]("Transaction set is invalid"),
		),
		validation.Field(&s.Direction,
			validation.Required.Error("Direction is required"),
			domainvalidation.ValidEnum[DocumentDirection]("Direction is invalid"),
		),
		validation.Field(&s.X12Version,
			validation.Required.Error("X12 version is required"),
			validation.Length(1, maxEDICodeLength).
				Error("X12 version cannot be longer than 20 characters"),
		),
		validation.Field(&s.ContextKey,
			validation.Required.Error("Context key is required"),
			validation.Length(1, maxContextKeyLength).
				Error("Context key cannot be longer than 100 characters"),
		),
		validation.Field(&s.SchemaVersion,
			validation.Required.Error("Schema version is required"),
			validation.Min(int64(1)).Error("Schema version must be at least one"),
		),
		validation.Field(&s.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&s.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[SourceContextFieldStatus]("Status is invalid"),
		),
	))

	for i, field := range s.Fields {
		if field == nil {
			continue
		}
		field.Validate(multiErr.WithIndex("fields", i))
	}
}

func (f *EDISourceContextField) Validate(multiErr *errortypes.MultiError) {
	// Tenancy and the owning schema are stamped when the parent is written, so
	// they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(f,
		validation.Field(&f.Path, validation.Required.Error("Path is required")),
		validation.Field(&f.SourceKind,
			validation.Required.Error("Source kind is required"),
			domainvalidation.ValidEnum[SourceContextKind]("Source kind is invalid"),
		),
		validation.Field(&f.DataType,
			validation.Required.Error("Data type is required"),
			domainvalidation.ValidEnum[SourceContextDataType]("Data type is invalid"),
		),
		// A repeated field is iterated over its repeat path, so without one the
		// template has nothing to loop on.
		validation.Field(&f.RepeatPath,
			validation.When(
				f.Repeated,
				validation.Required.Error("A repeated field must name its repeat path"),
			),
		),
		validation.Field(&f.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Display name cannot be longer than 200 characters"),
		),
		validation.Field(&f.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[SourceContextFieldStatus]("Status is invalid"),
		),
	))
}

const maxContextKeyLength = 100
