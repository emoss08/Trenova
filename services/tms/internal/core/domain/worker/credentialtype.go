package worker

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	ErrInvalidCredentialCategory     = errors.New("invalid credential category")
	ErrInvalidCredentialProfileField = errors.New("invalid credential profile field")
)

type CredentialCategory string

const (
	CredentialCategoryLicense       = CredentialCategory("License")
	CredentialCategoryMedical       = CredentialCategory("Medical")
	CredentialCategoryEndorsement   = CredentialCategory("Endorsement")
	CredentialCategorySecurity      = CredentialCategory("Security")
	CredentialCategoryCertification = CredentialCategory("Certification")
	CredentialCategoryBackground    = CredentialCategory("Background")
	CredentialCategoryOther         = CredentialCategory("Other")
)

func (c CredentialCategory) String() string { return string(c) }

func (c CredentialCategory) IsValid() bool {
	switch c {
	case CredentialCategoryLicense, CredentialCategoryMedical, CredentialCategoryEndorsement,
		CredentialCategorySecurity, CredentialCategoryCertification, CredentialCategoryBackground,
		CredentialCategoryOther:
		return true
	default:
		return false
	}
}

func CredentialCategoryFromString(s string) (CredentialCategory, error) {
	category := CredentialCategory(s)
	if !category.IsValid() {
		return "", ErrInvalidCredentialCategory
	}
	return category, nil
}

// CredentialProfileField names the worker_profiles column a system credential
// type mirrors. The profile columns stay authoritative for dispatch eligibility
// rules, so every write to a mirrored credential is copied back to the column.
type CredentialProfileField string

const (
	CredentialProfileFieldNone              = CredentialProfileField("")
	CredentialProfileFieldLicenseExpiry     = CredentialProfileField("LicenseExpiry")
	CredentialProfileFieldHazmatExpiry      = CredentialProfileField("HazmatExpiry")
	CredentialProfileFieldMedicalCardExpiry = CredentialProfileField("MedicalCardExpiry")
	CredentialProfileFieldTWICExpiry        = CredentialProfileField("TWICExpiry")
	CredentialProfileFieldPhysicalDueDate   = CredentialProfileField("PhysicalDueDate")
	CredentialProfileFieldMVRDueDate        = CredentialProfileField("MVRDueDate")
)

func (f CredentialProfileField) String() string { return string(f) }

func (f CredentialProfileField) IsValid() bool {
	switch f {
	case CredentialProfileFieldNone, CredentialProfileFieldLicenseExpiry,
		CredentialProfileFieldHazmatExpiry, CredentialProfileFieldMedicalCardExpiry,
		CredentialProfileFieldTWICExpiry, CredentialProfileFieldPhysicalDueDate,
		CredentialProfileFieldMVRDueDate:
		return true
	default:
		return false
	}
}

func (f CredentialProfileField) IsSet() bool { return f != CredentialProfileFieldNone }

// CarriesNumber reports whether the mirrored profile column has a companion
// number column (only the CDL does).
func (f CredentialProfileField) CarriesNumber() bool {
	return f == CredentialProfileFieldLicenseExpiry
}

// Expiry reads the mirrored value off the profile; the CDL column is NOT NULL
// with zero meaning unset, every other column is nullable.
func (f CredentialProfileField) Expiry(profile *WorkerProfile) *int64 {
	if profile == nil {
		return nil
	}
	var value *int64
	switch f {
	case CredentialProfileFieldLicenseExpiry:
		if profile.LicenseExpiry > 0 {
			expiry := profile.LicenseExpiry
			value = &expiry
		}
	case CredentialProfileFieldHazmatExpiry:
		value = profile.HazmatExpiry
	case CredentialProfileFieldMedicalCardExpiry:
		value = profile.MedicalCardExpiry
	case CredentialProfileFieldTWICExpiry:
		value = profile.TWICExpiry
	case CredentialProfileFieldPhysicalDueDate:
		value = profile.PhysicalDueDate
	case CredentialProfileFieldMVRDueDate:
		value = profile.MVRDueDate
	case CredentialProfileFieldNone:
		return nil
	}
	if value != nil && *value <= 0 {
		return nil
	}
	return value
}

// Apply writes the mirrored value onto the profile. Clearing the CDL expiry is
// refused because the column is NOT NULL and the licence is always required.
func (f CredentialProfileField) Apply(profile *WorkerProfile, expiry *int64, number string) {
	if profile == nil {
		return
	}
	switch f {
	case CredentialProfileFieldLicenseExpiry:
		if expiry != nil {
			profile.LicenseExpiry = *expiry
		}
		if number != "" {
			profile.LicenseNumber = number
		}
	case CredentialProfileFieldHazmatExpiry:
		profile.HazmatExpiry = expiry
	case CredentialProfileFieldMedicalCardExpiry:
		profile.MedicalCardExpiry = expiry
	case CredentialProfileFieldTWICExpiry:
		profile.TWICExpiry = expiry
	case CredentialProfileFieldPhysicalDueDate:
		profile.PhysicalDueDate = expiry
	case CredentialProfileFieldMVRDueDate:
		profile.MVRDueDate = expiry
	case CredentialProfileFieldNone:
	}
}

const (
	credentialTypeCodeCDL         = "CDL"
	credentialTypeCodeMedicalCard = "MED_CARD"
	credentialTypeCodeDOTPhysical = "DOT_PHYSICAL"
	credentialTypeCodeMVR         = "MVR"
	credentialTypeCodeHazmat      = "HAZMAT"
	credentialTypeCodeTWIC        = "TWIC"
	credentialTypeCodeTanker      = "TANKER"
	credentialTypeCodeDoubles     = "DOUBLES"
	credentialTypeCodeForklift    = "FORKLIFT"

	// The three recurring obligations 49 CFR 391.51 puts in a driver
	// qualification file. They are credential types rather than a separate
	// mechanism because each is a dated record with a document and a renewal
	// clock — which is exactly what the credential registry already tracks,
	// sweeps nightly and rolls up onto the roster.
	credentialTypeCodeRoadTest      = "ROAD_TEST"
	credentialTypeCodeAnnualReview  = "ANNUAL_REVIEW"
	credentialTypeCodeViolationCert = "VIOLATION_CERT"
)

var (
	_ bun.BeforeAppendModelHook          = (*WorkerCredentialType)(nil)
	_ domaintypes.PostgresSearchable     = (*WorkerCredentialType)(nil)
	_ pagination.CursorEntity            = (*WorkerCredentialType)(nil)
	_ validationframework.TenantedEntity = (*WorkerCredentialType)(nil)
)

type WorkerCredentialType struct {
	bun.BaseModel             `bun:"table:worker_credential_types,alias:wct" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                 json:"-"`

	ID                     pulid.ID               `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID               `json:"businessUnitId"         bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID         pulid.ID               `json:"organizationId"         bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Code                   string                 `json:"code"                   bun:"code,type:VARCHAR(50),notnull"`
	Name                   string                 `json:"name"                   bun:"name,type:VARCHAR(100),notnull"`
	Description            string                 `json:"description"            bun:"description,type:TEXT,nullzero"`
	Category               CredentialCategory     `json:"category"               bun:"category,type:worker_credential_category_enum,notnull,default:'Other'"`
	Status                 domaintypes.Status     `json:"status"                 bun:"status,type:status_enum,notnull,default:'Active'"`
	IsRequired             bool                   `json:"isRequired"             bun:"is_required,type:BOOLEAN,notnull"`
	RequiredForDriverTypes []DriverType           `json:"requiredForDriverTypes" bun:"required_for_driver_types,type:JSONB,notnull,default:'[]'"`
	RenewalWindowDays      int32                  `json:"renewalWindowDays"      bun:"renewal_window_days,type:INTEGER,notnull,default:30"`
	ValidityMonths         *int32                 `json:"validityMonths"         bun:"validity_months,type:INTEGER,nullzero"`
	RequiresNumber         bool                   `json:"requiresNumber"         bun:"requires_number,type:BOOLEAN,notnull"`
	RequiresDocument       bool                   `json:"requiresDocument"       bun:"requires_document,type:BOOLEAN,notnull"`
	ProfileField           CredentialProfileField `json:"profileField"           bun:"profile_field,type:VARCHAR(40),nullzero"`
	IsSystem               bool                   `json:"isSystem"               bun:"is_system,type:BOOLEAN,notnull"`
	SortOrder              int32                  `json:"sortOrder"              bun:"sort_order,type:INTEGER,notnull"`
	Version                int64                  `json:"version"                bun:"version,type:BIGINT"`
	CreatedAt              int64                  `json:"createdAt"              bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64                  `json:"updatedAt"              bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector           string                 `json:"-"                      bun:"search_vector,type:TSVECTOR,scanonly"`
}

func (t *WorkerCredentialType) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 50).Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(&t.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&t.Category,
			validation.Required.Error("Category is required"),
			domainvalidation.ValidEnum[CredentialCategory](
				"category must be one of: License, Medical, Endorsement, Security, Certification, Background, Other",
			),
		),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status must be either Active or Inactive"),
		),
		validation.Field(&t.RenewalWindowDays,
			validation.Min(int32(0)).Error("Renewal window cannot be negative"),
			validation.Max(int32(365)).Error("Renewal window cannot exceed 365 days"),
		),
	))

	if t.ValidityMonths != nil && *t.ValidityMonths <= 0 {
		multiErr.Add("validityMonths", errortypes.ErrInvalid, "Validity must be at least one month")
	}

	if !t.ProfileField.IsValid() {
		multiErr.Add("profileField", errortypes.ErrInvalid, "Unknown profile field")
	}

	for i, driverType := range t.RequiredForDriverTypes {
		if !driverType.IsValid() {
			multiErr.Add(
				"requiredForDriverTypes["+strconv.Itoa(i)+"]",
				errortypes.ErrInvalid,
				"Unknown driver type",
			)
		}
	}
}

// AppliesTo reports whether this type is required for the given worker: a
// required type with no driver-type filter applies to everyone, otherwise only
// to the listed driver types.
func (t *WorkerCredentialType) AppliesTo(wrk *Worker) bool {
	if t == nil || !t.IsRequired || t.Status != domaintypes.StatusActive {
		return false
	}
	if len(t.RequiredForDriverTypes) == 0 {
		return true
	}
	if wrk == nil {
		return false
	}
	return slices.Contains(t.RequiredForDriverTypes, wrk.DriverType)
}

func (t *WorkerCredentialType) NormalizeCode() {
	t.Code = strings.ToUpper(strings.TrimSpace(t.Code))
}

func (t *WorkerCredentialType) GetID() pulid.ID { return t.ID }

func (t *WorkerCredentialType) GetCreatedAt() int64 { return t.CreatedAt }

func (t *WorkerCredentialType) GetOrganizationID() pulid.ID { return t.OrganizationID }

func (t *WorkerCredentialType) GetBusinessUnitID() pulid.ID { return t.BusinessUnitID }

func (t *WorkerCredentialType) GetTableName() string { return "worker_credential_types" }

func (t *WorkerCredentialType) GetResourceType() string { return "worker_credential_type" }

func (t *WorkerCredentialType) GetResourceID() string { return t.ID.String() }

func (t *WorkerCredentialType) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "wct",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (t *WorkerCredentialType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("wct_")
		}
		if t.RequiredForDriverTypes == nil {
			t.RequiredForDriverTypes = []DriverType{}
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		if t.RequiredForDriverTypes == nil {
			t.RequiredForDriverTypes = []DriverType{}
		}
		t.UpdatedAt = now
	}

	return nil
}

// SystemCredentialTypes is the catalog every organisation starts with. Codes are
// stable identifiers; the migration that introduced the registry seeds the same
// rows so an organisation created before or after it looks identical.
func SystemCredentialTypes() []*WorkerCredentialType {
	months := func(n int32) *int32 { return &n }
	return []*WorkerCredentialType{
		{
			Code:              credentialTypeCodeCDL,
			Name:              "Commercial Driver's License",
			Description:       "State-issued CDL. Number and expiry mirror the worker profile.",
			Category:          CredentialCategoryLicense,
			IsRequired:        true,
			RenewalWindowDays: 30,
			RequiresNumber:    true,
			RequiresDocument:  true,
			ProfileField:      CredentialProfileFieldLicenseExpiry,
			SortOrder:         10,
		},
		{
			Code:              credentialTypeCodeMedicalCard,
			Name:              "DOT Medical Card",
			Description:       "Medical examiner's certificate (49 CFR 391.43).",
			Category:          CredentialCategoryMedical,
			IsRequired:        true,
			RenewalWindowDays: 30,
			ValidityMonths:    months(24),
			RequiresDocument:  true,
			ProfileField:      CredentialProfileFieldMedicalCardExpiry,
			SortOrder:         20,
		},
		{
			Code:              credentialTypeCodeDOTPhysical,
			Name:              "DOT Physical",
			Description:       "Next physical examination due date.",
			Category:          CredentialCategoryMedical,
			IsRequired:        true,
			RenewalWindowDays: 30,
			ValidityMonths:    months(24),
			ProfileField:      CredentialProfileFieldPhysicalDueDate,
			SortOrder:         30,
		},
		{
			Code:              credentialTypeCodeMVR,
			Name:              "Motor Vehicle Record Review",
			Description:       "Annual MVR pull and review (49 CFR 391.25).",
			Category:          CredentialCategoryBackground,
			IsRequired:        true,
			RenewalWindowDays: 30,
			ValidityMonths:    months(12),
			ProfileField:      CredentialProfileFieldMVRDueDate,
			SortOrder:         40,
		},
		{
			Code:             credentialTypeCodeRoadTest,
			Name:             "Road Test Certificate",
			Description:      "Road test and certificate, or the licence accepted in its place (49 CFR 391.31, 391.33).",
			Category:         CredentialCategoryCertification,
			IsRequired:       true,
			RequiresDocument: true,
			SortOrder:        45,
		},
		{
			Code:              credentialTypeCodeAnnualReview,
			Name:              "Annual Review of Driving Record",
			Description:       "The employer's annual review of the driver's record and finding that they remain qualified (49 CFR 391.25(c)).",
			Category:          CredentialCategoryBackground,
			IsRequired:        true,
			RenewalWindowDays: 30,
			ValidityMonths:    months(12),
			RequiresDocument:  true,
			SortOrder:         46,
		},
		{
			Code:              credentialTypeCodeViolationCert,
			Name:              "Annual Violation Certification",
			Description:       "The driver's own annual list of traffic convictions, or a signed statement that there were none (49 CFR 391.27).",
			Category:          CredentialCategoryBackground,
			IsRequired:        true,
			RenewalWindowDays: 30,
			ValidityMonths:    months(12),
			RequiresDocument:  true,
			SortOrder:         47,
		},
		{
			Code:              credentialTypeCodeHazmat,
			Name:              "Hazmat Endorsement",
			Description:       "H or X endorsement with TSA threat assessment.",
			Category:          CredentialCategoryEndorsement,
			RenewalWindowDays: 60,
			ValidityMonths:    months(60),
			RequiresDocument:  true,
			ProfileField:      CredentialProfileFieldHazmatExpiry,
			SortOrder:         50,
		},
		{
			Code:              credentialTypeCodeTWIC,
			Name:              "TWIC Card",
			Description:       "Transportation Worker Identification Credential for port access.",
			Category:          CredentialCategorySecurity,
			RenewalWindowDays: 60,
			ValidityMonths:    months(60),
			RequiresNumber:    true,
			RequiresDocument:  true,
			ProfileField:      CredentialProfileFieldTWICExpiry,
			SortOrder:         60,
		},
		{
			Code:              credentialTypeCodeTanker,
			Name:              "Tanker Endorsement",
			Description:       "N endorsement for liquid bulk.",
			Category:          CredentialCategoryEndorsement,
			RenewalWindowDays: 30,
			SortOrder:         70,
		},
		{
			Code:              credentialTypeCodeDoubles,
			Name:              "Doubles/Triples Endorsement",
			Description:       "T endorsement for multi-trailer combinations.",
			Category:          CredentialCategoryEndorsement,
			RenewalWindowDays: 30,
			SortOrder:         80,
		},
		{
			Code:              credentialTypeCodeForklift,
			Name:              "Forklift Certification",
			Description:       "OSHA powered industrial truck certification.",
			Category:          CredentialCategoryCertification,
			RenewalWindowDays: 30,
			ValidityMonths:    months(36),
			RequiresDocument:  true,
			SortOrder:         90,
		},
	}
}
