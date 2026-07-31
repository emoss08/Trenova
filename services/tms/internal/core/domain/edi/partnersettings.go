package edi

import (
	"context"
	"errors"
	"regexp"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type EDIPartnerSettingSchema struct {
	bun.BaseModel `json:"-" bun:"table:edi_partner_setting_schemas,alias:epss"`

	ID             pulid.ID             `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID             `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),nullzero"`
	OrganizationID pulid.ID             `json:"organizationId" bun:"organization_id,type:VARCHAR(100),nullzero"`
	DocumentTypeID pulid.ID             `json:"documentTypeId" bun:"document_type_id,type:VARCHAR(100),nullzero"`
	Standard       EDIStandard          `json:"standard"       bun:"standard,type:edi_standard_enum,notnull"`
	TransactionSet TransactionSet       `json:"transactionSet" bun:"transaction_set,type:edi_transaction_set_enum,notnull"`
	Direction      DocumentDirection    `json:"direction"      bun:"direction,type:edi_document_direction_enum,notnull"`
	X12Version     string               `json:"x12Version"     bun:"x12_version,type:VARCHAR(20),notnull"`
	SchemaVersion  int64                `json:"schemaVersion"  bun:"schema_version,type:BIGINT,notnull"`
	Name           string               `json:"name"           bun:"name,type:VARCHAR(200),notnull"`
	Description    string               `json:"description"    bun:"description,type:TEXT,nullzero"`
	Status         PartnerSettingStatus `json:"status"         bun:"status,type:edi_partner_setting_status_enum,notnull"`
	CreatedAt      int64                `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64                `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Fields []*EDIPartnerSettingField `json:"fields,omitempty" bun:"rel:has-many,join:id=schema_id"`
}

func (s *EDIPartnerSettingSchema) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("edips_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}
	return nil
}

type EDIPartnerSettingField struct {
	bun.BaseModel `json:"-" bun:"table:edi_partner_setting_fields,alias:epsf"`

	ID                pulid.ID               `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	SchemaID          pulid.ID               `json:"schemaId"          bun:"schema_id,type:VARCHAR(100),notnull"`
	Path              string                 `json:"path"              bun:"path,type:TEXT,notnull"`
	Label             string                 `json:"label"             bun:"label,type:VARCHAR(200),notnull"`
	Description       string                 `json:"description"       bun:"description,type:TEXT,nullzero"`
	DataType          PartnerSettingDataType `json:"dataType"          bun:"data_type,type:edi_partner_setting_data_type_enum,notnull"`
	Required          bool                   `json:"required"          bun:"required,type:BOOLEAN,notnull,default:false"`
	Nullable          bool                   `json:"nullable"          bun:"nullable,type:BOOLEAN,notnull,default:false"`
	DefaultValue      any                    `json:"defaultValue"      bun:"default_value,type:JSONB,nullzero"`
	AllowedValues     []string               `json:"allowedValues"     bun:"allowed_values,type:JSONB,notnull,default:'[]'::jsonb"`
	Secret            bool                   `json:"secret"            bun:"secret,type:BOOLEAN,notnull,default:false"`
	GroupKey          string                 `json:"groupKey"          bun:"group_key,type:VARCHAR(100),nullzero"`
	DisplayOrder      int                    `json:"displayOrder"      bun:"display_order,type:INTEGER,notnull,default:0"`
	ValidationPattern string                 `json:"validationPattern" bun:"validation_pattern,type:TEXT,nullzero"`
	MinLength         int                    `json:"minLength"         bun:"min_length,type:INTEGER,notnull,default:0"`
	MaxLength         int                    `json:"maxLength"         bun:"max_length,type:INTEGER,notnull,default:0"`
	UsageNotes        string                 `json:"usageNotes"        bun:"usage_notes,type:TEXT,nullzero"`
	Status            PartnerSettingStatus   `json:"status"            bun:"status,type:edi_partner_setting_status_enum,notnull"`
	CreatedAt         int64                  `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64                  `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Schema *EDIPartnerSettingSchema `json:"schema,omitempty" bun:"rel:belongs-to,join:schema_id=id"`
}

func (f *EDIPartnerSettingField) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if f.ID.IsNil() {
			f.ID = pulid.MustNew("edipsf_")
		}
		f.CreatedAt = now
	case *bun.UpdateQuery:
		f.UpdatedAt = now
	}
	return nil
}

func (s *EDIPartnerSettingSchema) Validate(multiErr *errortypes.MultiError) {
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
		// A schema is looked up by version, so two rows sharing one would make
		// which of them applies undecidable.
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
			domainvalidation.ValidEnum[PartnerSettingStatus]("Status is invalid"),
		),
	))

	for i, field := range s.Fields {
		if field == nil {
			continue
		}
		field.Validate(multiErr.WithIndex("fields", i))
	}
}

func (f *EDIPartnerSettingField) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(f,
		validation.Field(&f.SchemaID, validation.Required.Error("Schema is required")),
		validation.Field(&f.Path, validation.Required.Error("Path is required")),
		validation.Field(&f.Label,
			validation.Required.Error("Label is required"),
			validation.Length(1, maxEDINameLength).
				Error("Label cannot be longer than 200 characters"),
		),
		validation.Field(&f.DataType,
			validation.Required.Error("Data type is required"),
			domainvalidation.ValidEnum[PartnerSettingDataType]("Data type is invalid"),
		),
		// An enum field with nothing to choose from renders an empty picker the
		// partner can never satisfy.
		validation.Field(&f.AllowedValues,
			validation.When(
				f.DataType == PartnerSettingDataTypeEnum,
				validation.Required.Error("An enum setting must list its allowed values"),
			),
		),
		validation.Field(&f.GroupKey,
			validation.Length(0, maxGroupKeyLength).
				Error("Group key cannot be longer than 100 characters"),
		),
		validation.Field(&f.DisplayOrder,
			validation.Min(0).Error("Display order cannot be negative"),
		),
		validation.Field(&f.MinLength,
			validation.Min(0).Error("Minimum length cannot be negative"),
		),
		validation.Field(&f.MaxLength,
			validation.Min(f.MinLength).
				Error("Maximum length cannot be less than the minimum"),
		),
		validation.Field(&f.ValidationPattern,
			validation.By(func(value any) error {
				pattern, _ := value.(string)
				if pattern == "" {
					return nil
				}
				if _, err := regexp.Compile(pattern); err != nil {
					return errInvalidValidationPattern
				}
				return nil
			}),
		),
		validation.Field(&f.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[PartnerSettingStatus]("Status is invalid"),
		),
	))
}

// errInvalidValidationPattern keeps an uncompilable pattern out of the column:
// it would reject every value a partner could enter and the reason would only
// surface at send time.
var errInvalidValidationPattern = errors.New("Validation pattern is not a valid expression")

const maxGroupKeyLength = 100
