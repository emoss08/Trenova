package edi

import (
	"errors"
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type EDIPartnerSettingSchema struct {
	keel.Table `keel:"edi_partner_setting_schemas"`

	ID             pulid.ID             `json:"id"             keel:"id,pk,autoid:edips_"`
	BusinessUnitID pulid.ID             `json:"businessUnitId" keel:"business_unit_id,nullzero"`
	OrganizationID pulid.ID             `json:"organizationId" keel:"organization_id,nullzero"`
	DocumentTypeID pulid.ID             `json:"documentTypeId" keel:"document_type_id,nullzero"`
	Standard       EDIStandard          `json:"standard"       keel:"standard"`
	TransactionSet TransactionSet       `json:"transactionSet" keel:"transaction_set"`
	Direction      DocumentDirection    `json:"direction"      keel:"direction"`
	X12Version     string               `json:"x12Version"     keel:"x12_version"`
	SchemaVersion  int64                `json:"schemaVersion"  keel:"schema_version"`
	Name           string               `json:"name"           keel:"name"`
	Description    string               `json:"description"    keel:"description,nullzero"`
	Status         PartnerSettingStatus `json:"status"         keel:"status"`
	CreatedAt      int64                `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64                `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Fields []*EDIPartnerSettingField `json:"fields,omitempty" keel:"many:schema_id"`
}

type EDIPartnerSettingField struct {
	keel.Table `keel:"edi_partner_setting_fields"`

	ID                pulid.ID               `json:"id"                keel:"id,pk,autoid:edipsf_"`
	SchemaID          pulid.ID               `json:"schemaId"          keel:"schema_id"`
	Path              string                 `json:"path"              keel:"path"`
	Label             string                 `json:"label"             keel:"label"`
	Description       string                 `json:"description"       keel:"description,nullzero"`
	DataType          PartnerSettingDataType `json:"dataType"          keel:"data_type"`
	Required          bool                   `json:"required"          keel:"required"`
	Nullable          bool                   `json:"nullable"          keel:"nullable"`
	DefaultValue      any                    `json:"defaultValue"      keel:"default_value,jsonb,nullzero"`
	AllowedValues     []string               `json:"allowedValues"     keel:"allowed_values,jsonb"`
	Secret            bool                   `json:"secret"            keel:"secret"`
	GroupKey          string                 `json:"groupKey"          keel:"group_key,nullzero"`
	DisplayOrder      int                    `json:"displayOrder"      keel:"display_order"`
	ValidationPattern string                 `json:"validationPattern" keel:"validation_pattern,nullzero"`
	MinLength         int                    `json:"minLength"         keel:"min_length"`
	MaxLength         int                    `json:"maxLength"         keel:"max_length"`
	UsageNotes        string                 `json:"usageNotes"        keel:"usage_notes,nullzero"`
	Status            PartnerSettingStatus   `json:"status"            keel:"status"`
	CreatedAt         int64                  `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64                  `json:"updatedAt"         keel:"updated_at,autoupdate"`

	Schema *EDIPartnerSettingSchema `json:"schema,omitempty" keel:"belongs:schema_id"`
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
	// Tenancy and the owning schema are stamped when the parent is written, so
	// they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(f,
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
var errInvalidValidationPattern = errors.New("validation pattern is not a valid expression")

const maxGroupKeyLength = 100
