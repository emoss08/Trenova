package edi

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type EDISourceContextSchema struct {
	keel.Table `keel:"edi_source_context_schemas"`

	ID             pulid.ID                 `json:"id" keel:"id,pk,autoid:edisc_"`
	BusinessUnitID pulid.ID                 `json:"businessUnitId" keel:"business_unit_id,nullzero"`
	OrganizationID pulid.ID                 `json:"organizationId" keel:"organization_id,nullzero"`
	Standard       EDIStandard              `json:"standard" keel:"standard"`
	TransactionSet TransactionSet           `json:"transactionSet" keel:"transaction_set"`
	Direction      DocumentDirection        `json:"direction" keel:"direction"`
	X12Version     string                   `json:"x12Version" keel:"x12_version"`
	ContextKey     string                   `json:"contextKey" keel:"context_key"`
	SchemaVersion  int64                    `json:"schemaVersion" keel:"schema_version"`
	Name           string                   `json:"name" keel:"name"`
	Description    string                   `json:"description" keel:"description,nullzero"`
	Status         SourceContextFieldStatus `json:"status" keel:"status"`
	CreatedAt      int64                    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64                    `json:"updatedAt" keel:"updated_at,autoupdate"`

	Fields []*EDISourceContextField `json:"fields,omitempty" keel:"many:schema_id"`
}

type EDISourceContextField struct {
	keel.Table `keel:"edi_source_context_fields"`

	ID          pulid.ID                 `json:"id" keel:"id,pk,autoid:ediscf_"`
	SchemaID    pulid.ID                 `json:"schemaId" keel:"schema_id"`
	Path        string                   `json:"path" keel:"path"`
	SourceKind  SourceContextKind        `json:"sourceKind" keel:"source_kind"`
	DataType    SourceContextDataType    `json:"dataType" keel:"data_type"`
	Repeated    bool                     `json:"repeated" keel:"repeated"`
	RepeatPath  string                   `json:"repeatPath" keel:"repeat_path,nullzero"`
	ParentPath  string                   `json:"parentPath" keel:"parent_path,nullzero"`
	DisplayName string                   `json:"displayName" keel:"display_name"`
	Description string                   `json:"description" keel:"description,nullzero"`
	Status      SourceContextFieldStatus `json:"status" keel:"status"`
	CreatedAt   int64                    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt   int64                    `json:"updatedAt" keel:"updated_at,autoupdate"`

	Schema *EDISourceContextSchema `json:"schema,omitempty" keel:"belongs:schema_id"`
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
