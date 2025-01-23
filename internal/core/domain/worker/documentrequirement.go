package worker

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*DocumentRequirement)(nil)
	_ domain.Validatable        = (*DocumentRequirement)(nil)
)

type DocumentRequirement struct {
	bun.BaseModel `bun:"table:document_requirements,alias:dr" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull" json:"organizationId"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),notnull" json:"businessUnitId"`

	// Core Fields
	Name            string                  `bun:"name,type:VARCHAR(255),notnull" json:"name"`
	Description     string                  `bun:"description,type:TEXT" json:"description"`
	DocumentType    DocumentType            `bun:"document_type,type:document_type_enum,notnull" json:"documentType"`
	RequirementType DocumentRequirementType `bun:"requirement_type,type:document_requirement_type_enum,notnull" json:"requirementType"`

	// CFR Reference
	CFRTitle   string `bun:"cfr_title,type:VARCHAR(100)" json:"cfrTitle"`
	CFRPart    string `bun:"cfr_part,type:VARCHAR(100)" json:"cfrPart"`
	CFRSection string `bun:"cfr_section,type:VARCHAR(100)" json:"cfrSection"`
	CFRUrl     string `bun:"cfr_url,type:VARCHAR(255)" json:"cfrUrl"`

	// Timing and Retention
	RetentionPeriod     RetentionPeriod `bun:"retention_period,type:retention_period_enum,notnull" json:"retentionPeriod"`
	CustomRetentionDays *int            `bun:"custom_retention_days,type:INTEGER" json:"customRetentionDays,omitempty"`
	RenewalPeriodDays   *int            `bun:"renewal_period_days,type:INTEGER" json:"renewalPeriodDays,omitempty"`
	ReminderDays        []int           `bun:"reminder_days,type:INTEGER[]" json:"reminderDays,omitempty"`

	// Validation and Requirements
	IsRequired       bool           `bun:"is_required,type:BOOLEAN,notnull" json:"isRequired"`
	ValidationRules  map[string]any `bun:"validation_rules,type:JSONB" json:"validationRules,omitempty"`
	BlocksAssignment bool           `bun:"blocks_assignment,type:BOOLEAN,notnull" json:"blocksAssignment"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT,notnull" json:"version"`
	CreatedAt int64 `bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (dr *DocumentRequirement) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, dr,
		// Name is required and must be between 3 and 255 characters
		validation.Field(&dr.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 255).Error("Name must be between 3 and 255 characters"),
		),

		// Description is required and must be between 10 to 1000 characters
		validation.Field(&dr.Description,
			validation.Required.Error("Description is required"),
			validation.Length(10, 1000).Error("Description must be between 10 and 1000 characters"),
		),

		// Document type is required and must be a valid document type
		validation.Field(&dr.DocumentType,
			validation.Required.Error("Document type is required"),
			validation.In(
				DocumentTypeMVR,
				DocumentTypeMedicalCert,
				DocumentTypeCDL,
				DocumentTypeViolationCert,
				DocumentTypeEmploymentHistory,
				DocumentTypeDrugTest,
				DocumentTypeRoadTest,
				DocumentTypeTrainingCert,
			).Error("Invalid document type"),
		),

		// CFR Title, Part, Section, and URL are required
		validation.Field(&dr.CFRTitle,
			validation.Required.Error("CFR title is required"),
			validation.Length(1, 100).Error("CFR title must be between 1 and 100 characters"),
		),
		validation.Field(&dr.CFRPart,
			validation.Required.Error("CFR part is required"),
			validation.Length(1, 100).Error("CFR part must be between 1 and 100 characters"),
		),
		validation.Field(&dr.CFRSection,
			validation.Required.Error("CFR section is required"),
			validation.Length(1, 100).Error("CFR section must be between 1 and 100 characters"),
		),
		validation.Field(&dr.CFRUrl,
			validation.Required.Error("CFR URL is required"),
			is.URL.Error("Must be a valid URL"),
		),

		// Retention period is required and must be a valid retention period
		validation.Field(&dr.RetentionPeriod,
			validation.Required.Error("Retention period is required"),
			validation.In(
				RetentionPeriodThreeYears,
				RetentionPeriodLifeOfEmployment,
				RetentionPeriodCustom,
			).Error("Invalid retention period"),
		),

		// Custom Retention Days is required when the retention period is custom
		validation.Field(&dr.CustomRetentionDays,
			validation.When(dr.RetentionPeriod == RetentionPeriodCustom,
				validation.Required.Error("Custom retention days is required when retention period is custom"),
				validation.Min(1).Error("Custom retention days must be greater than 0"),
			),
		),

		// Renewal Period Days is required when the requirement type is ongoing
		validation.Field(&dr.RenewalPeriodDays,
			validation.When(dr.RequirementType == RequirementTypeOngoing,
				validation.Required.Error("Renewal period days is required for ongoing requirements"),
				validation.Min(1).Error("Renewal period days must be greater than 0"),
			),
		),

		// Reminder Days is required when the requirement type is ongoing
		validation.Field(&dr.ReminderDays,
			validation.When(dr.RequirementType == RequirementTypeOngoing,
				validation.Required.Error("Reminder days is required for ongoing requirements"),
				validation.Each(validation.Min(1).Error("Reminder days must be greater than 0")),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromValidationErrors(validationErrs, multiErr, "")
		}
	}
}

func (dr *DocumentRequirement) DBValidate(ctx context.Context, tx bun.IDB) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// Validate common fields
	dr.Validate(ctx, multiErr)

	// Validate document type specific rules
	dr.validateDocumentTypeSpecificRules(multiErr)

	// If this is an update, validate that system-controlled fields are not haven't changed
	if dr.ID.IsNotNil() {
		original := new(DocumentRequirement)
		err := tx.NewSelect().Model(original).
			Where("id = ? AND organization_id = ? AND business_unit_id = ?",
				dr.ID, dr.OrganizationID, dr.BusinessUnitID).
			Scan(ctx)
		if err == nil {
			dr.validateSystemControlledFields(original, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// validateSystemControlledFields validates the system controlled fields of the document requirement
func (dr *DocumentRequirement) validateSystemControlledFields(original *DocumentRequirement, multiErr *errors.MultiError) {
	// Document Type cannot be changed after creation
	if original.DocumentType == dr.DocumentType {
		multiErr.Add("documentType", errors.ErrInvalid, "Document type cannot be changed after creation")
	}

	// Requirement Type cannot be changed after creation
	if original.RequirementType == dr.RequirementType {
		multiErr.Add("requirementType", errors.ErrInvalid, "Requirement type cannot be changed after creation")
	}

	// CFR Title, Part, Section, and URL cannot be changed after creation
	if original.CFRTitle == dr.CFRTitle && original.CFRPart == dr.CFRPart && original.CFRSection == dr.CFRSection && original.CFRUrl == dr.CFRUrl {
		multiErr.Add("cfrTitle", errors.ErrInvalid, "CFR Title cannot be changed after creation")
		multiErr.Add("cfrPart", errors.ErrInvalid, "CFR Part cannot be changed after creation")
		multiErr.Add("cfrSection", errors.ErrInvalid, "CFR Section cannot be changed after creation")
		multiErr.Add("cfrUrl", errors.ErrInvalid, "CFR URL cannot be changed after creation")
	}

	// Retention Period cannot be changed after creation
	if original.RetentionPeriod == dr.RetentionPeriod {
		multiErr.Add("retentionPeriod", errors.ErrInvalid, "Retention period cannot be changed after creation")
	}
}

func (dr *DocumentRequirement) validateDocumentTypeSpecificRules(multiErr *errors.MultiError) {
	switch dr.DocumentType {
	case DocumentTypeMedicalCert:
		dr.validateMedicalCertRules(multiErr)
	case DocumentTypeMVR:
		dr.validateMVRRules(multiErr)
	case DocumentTypeCDL:
		dr.validateCDLRules(multiErr)
	case DocumentTypeDrugTest:
		dr.validateDrugTestRules(multiErr)
	case DocumentTypeEmploymentHistory:
		dr.validateEmploymentHistoryRules(multiErr)
	case DocumentTypeViolationCert:
		dr.validateViolationCertRules(multiErr)
	case DocumentTypeRoadTest, DocumentTypeTrainingCert:
	}
}

func (dr *DocumentRequirement) validateMedicalCertRules(multiErr *errors.MultiError) {
	// Medical certs must have 24-month renewal period
	if dr.RenewalPeriodDays == nil || *dr.RenewalPeriodDays != 730 {
		multiErr.Add("renewalPeriodDays", errors.ErrInvalid, "Medical certificates require a 24-month renewal period based on the CFR")
	}

	// Required validation rules for medical certificates
	requiredRules := []string{
		"examinerRegistryNumber",
		"examinerExpirationDate",
		"medicalExamDate",
		"certificationLevel",
		"restrictions",
	}

	for _, rule := range requiredRules {
		if _, exists := dr.ValidationRules[rule]; !exists {
			multiErr.Add("validationRules", errors.ErrInvalid,
				fmt.Sprintf("Medical certificates require %s validation rule", rule))
		}
	}

	// must block assignment when invalid
	if !dr.BlocksAssignment {
		multiErr.Add("blocksAssignment", errors.ErrInvalid, "Medical certificates must block assignment when invalid")
	}

	// Must set reminder days appropriately
	if len(dr.ReminderDays) == 0 {
		multiErr.Add("reminderDays", errors.ErrInvalid, "Medical certificates must have reminder days set")
	}
}

func (dr *DocumentRequirement) validateMVRRules(multiErr *errors.MultiError) {
	// Annaul renewal requirement
	if dr.RenewalPeriodDays == nil || *dr.RenewalPeriodDays != 365 {
		multiErr.Add("renewalPeriodDays", errors.ErrInvalid, "MVR records require an annual renewal period")
	}

	// Required validation rules for MVR
	requiredRules := []string{
		"reviewerID",
		"reviewDate",
		"violationCheck",
		"stateAgencySource",
		"requestDate",
	}

	for _, rule := range requiredRules {
		if _, exists := dr.ValidationRules[rule]; !exists {
			multiErr.Add("validationRules", errors.ErrInvalid,
				fmt.Sprintf("MVR records require %s validation rule", rule))
		}
	}

	// Ensure proper retention period
	if dr.RetentionPeriod != RetentionPeriodThreeYears {
		multiErr.Add("retentionPeriod", errors.ErrInvalid, "MVR records require a 3-year retention period")
	}
}

func (dr *DocumentRequirement) validateCDLRules(multiErr *errors.MultiError) {
	requiredRules := []string{
		"licenseClass",
		"endorsements",
		"restrictions",
		"stateIssued",
		"ageVerification",
		"expirationDate",
		"issueDate",
	}

	for _, rule := range requiredRules {
		if _, exists := dr.ValidationRules[rule]; !exists {
			multiErr.Add("validationRules", errors.ErrInvalid,
				fmt.Sprintf("CDL requires %s validation rule", rule))
		}
	}

	if !dr.BlocksAssignment {
		multiErr.Add("blocksAssignment", errors.ErrInvalid,
			"CDL must block assignment when invalid")
	}

	// Ensure proper retention period
	if dr.RetentionPeriod != RetentionPeriodLifeOfEmployment {
		multiErr.Add("retentionPeriod", errors.ErrInvalid,
			"CDL records must be retained for life of employment plus 3 years")
	}
}

func (dr *DocumentRequirement) validateDrugTestRules(multiErr *errors.MultiError) {
	requiredRules := []string{
		"testType", // Pre-employment, Random, Post-accident, etc.
		"testDate",
		"result",
		"collectionSite",
		"testingFacility",
		"mroVerification",
	}

	for _, rule := range requiredRules {
		if _, exists := dr.ValidationRules[rule]; !exists {
			multiErr.Add("validationRules", errors.ErrInvalid,
				fmt.Sprintf("Drug test records require %s validation rule", rule))
		}
	}

	// Must block assignment for positive results
	if !dr.BlocksAssignment {
		multiErr.Add("blocksAssignment", errors.ErrInvalid,
			"Drug test records must block assignment when positive or refused")
	}
}

func (dr *DocumentRequirement) validateEmploymentHistoryRules(multiErr *errors.MultiError) {
	requiredRules := []string{
		"employerName",
		"employmentStartDate",
		"employmentEndDate",
		"reasonForLeaving",
		"safetyPerformanceHistory",
		"accidentHistory",
		"drugTestHistory",
	}

	for _, rule := range requiredRules {
		if _, exists := dr.ValidationRules[rule]; !exists {
			multiErr.Add("validationRules", errors.ErrInvalid,
				fmt.Sprintf("Employment history requires %s validation rule", rule))
		}
	}

	// Ensure proper retention period
	if dr.RetentionPeriod != RetentionPeriodLifeOfEmployment {
		multiErr.Add("retentionPeriod", errors.ErrInvalid,
			"Employment history must be retained for life of employment plus 3 years")
	}
}

func (dr *DocumentRequirement) validateViolationCertRules(multiErr *errors.MultiError) {
	// Annual requirement
	if dr.RenewalPeriodDays == nil || *dr.RenewalPeriodDays != 365 {
		multiErr.Add("renewalPeriodDays", errors.ErrInvalid,
			"Violation certifications require annual renewal")
	}

	requiredRules := []string{
		"certificationDate",
		"violationsReported",
		"reviewerID",
		"reviewDate",
	}

	for _, rule := range requiredRules {
		if _, exists := dr.ValidationRules[rule]; !exists {
			multiErr.Add("validationRules", errors.ErrInvalid,
				fmt.Sprintf("Violation certification requires %s validation rule", rule))
		}
	}

	// Ensure proper retention period
	if dr.RetentionPeriod != RetentionPeriodThreeYears {
		multiErr.Add("retentionPeriod", errors.ErrInvalid,
			"Violation certifications must be retained for 3 years")
	}
}

func (dr *DocumentRequirement) GetTableName() string {
	return "document_requirements"
}

// BeforeAppendModel is a bun hook that sets the createdAt and updatedAt fields
func (dr *DocumentRequirement) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if dr.ID == "" {
			dr.ID = pulid.MustNew("dr_")
		}

		dr.CreatedAt = now
	case *bun.UpdateQuery:
		dr.UpdatedAt = now
	}

	return nil
}
