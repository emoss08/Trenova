package instancebootstrap

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	IDPrefix = "ibs_"

	DefaultTimezone = "America/New_York"
	DefaultState    = "NY"

	MaxOrganizationNameLength = 100
	MaxAdminNameLength        = 100
	MaxCityLength             = 100
	MaxAddressLength          = 150

	FieldOrganizationName = "organizationName"
	FieldAdminName        = "adminName"
	FieldAdminEmail       = "adminEmail"
	FieldTimezone         = "timezone"
	FieldState            = "state"
	FieldCity             = "city"
	FieldPostalCode       = "postalCode"
	FieldAddressLine1     = "addressLine1"
	FieldSCAC             = "scac"
	FieldDOTNumber        = "dotNumber"
)

var (
	_ bun.BeforeAppendModelHook = (*InstanceBootstrap)(nil)

	statePattern      = regexp.MustCompile(`^[A-Z]{2}$`)
	postalCodePattern = regexp.MustCompile(`^\d{5}(-\d{4})?$`)
	scacPattern       = regexp.MustCompile(`^[A-Z]{4}$`)
	dotNumberPattern  = regexp.MustCompile(`^\d{1,8}$`)
)

type Inputs struct {
	OrganizationName string `json:"organizationName"`
	AdminName        string `json:"adminName"`
	AdminEmail       string `json:"adminEmail"`
	Timezone         string `json:"timezone"`
	State            string `json:"state"`
	City             string `json:"city"`
	PostalCode       string `json:"postalCode"`
	AddressLine1     string `json:"addressLine1"`
	SCAC             string `json:"scac"`
	DOTNumber        string `json:"dotNumber"`
}

func (i *Inputs) Normalize() Inputs {
	return Inputs{
		OrganizationName: stringutils.CollapseWhitespace(i.OrganizationName),
		AdminName:        stringutils.CollapseWhitespace(i.AdminName),
		AdminEmail:       stringutils.NormalizeEmailAddress(i.AdminEmail),
		Timezone:         stringutils.WithDefault(i.Timezone, DefaultTimezone),
		State:            strings.ToUpper(stringutils.WithDefault(i.State, DefaultState)),
		City: stringutils.WithDefault(
			stringutils.CollapseWhitespace(i.City),
			onboarding.PlaceholderCity,
		),
		PostalCode:   stringutils.WithDefault(i.PostalCode, onboarding.PlaceholderPostalCode),
		AddressLine1: stringutils.CollapseWhitespace(i.AddressLine1),
		SCAC:         strings.ToUpper(stringutils.WithDefault(i.SCAC, onboarding.PlaceholderSCAC)),
		DOTNumber:    stringutils.WithDefault(i.DOTNumber, onboarding.PlaceholderDOTNumber),
	}
}

func (i *Inputs) Validate(multiErr *errortypes.MultiError) {
	switch {
	case i.OrganizationName == "":
		multiErr.Add(FieldOrganizationName, errortypes.ErrRequired, "Organization name is required")
	case utf8.RuneCountInString(i.OrganizationName) > MaxOrganizationNameLength:
		multiErr.Add(
			FieldOrganizationName,
			errortypes.ErrInvalidLength,
			"Organization name must be 100 characters or fewer",
		)
	}

	switch {
	case i.AdminName == "":
		multiErr.Add(FieldAdminName, errortypes.ErrRequired, "Administrator name is required")
	case utf8.RuneCountInString(i.AdminName) > MaxAdminNameLength:
		multiErr.Add(
			FieldAdminName,
			errortypes.ErrInvalidLength,
			"Administrator name must be 100 characters or fewer",
		)
	}

	if i.AdminEmail == "" {
		multiErr.Add(FieldAdminEmail, errortypes.ErrRequired, "Administrator email is required")
	} else if _, err := emailutils.Parse(i.AdminEmail); err != nil {
		multiErr.Add(
			FieldAdminEmail,
			errortypes.ErrInvalidFormat,
			"Administrator email must be a valid email address",
		)
	}

	if err := domainvalidation.ValidateTimezone(i.Timezone); err != nil {
		multiErr.Add(
			FieldTimezone,
			errortypes.ErrInvalid,
			"Timezone must be an IANA time zone such as America/Chicago",
		)
	}

	if !statePattern.MatchString(i.State) {
		multiErr.Add(
			FieldState,
			errortypes.ErrInvalidFormat,
			"State must be a two-letter US state abbreviation",
		)
	}

	if utf8.RuneCountInString(i.City) > MaxCityLength {
		multiErr.Add(FieldCity, errortypes.ErrInvalidLength, "City must be 100 characters or fewer")
	}

	if !postalCodePattern.MatchString(i.PostalCode) {
		multiErr.Add(
			FieldPostalCode,
			errortypes.ErrInvalidFormat,
			"Postal code must be a US ZIP code (12345 or 12345-6789)",
		)
	}

	if utf8.RuneCountInString(i.AddressLine1) > MaxAddressLength {
		multiErr.Add(
			FieldAddressLine1,
			errortypes.ErrInvalidLength,
			"Address line 1 must be 150 characters or fewer",
		)
	}

	if !scacPattern.MatchString(i.SCAC) {
		multiErr.Add(FieldSCAC, errortypes.ErrInvalidFormat, "SCAC code must be four letters")
	}

	if !dotNumberPattern.MatchString(i.DOTNumber) {
		multiErr.Add(
			FieldDOTNumber,
			errortypes.ErrInvalidFormat,
			"DOT number must be 1 to 8 digits",
		)
	}
}

func (i *Inputs) IdentityDifferences(other *Inputs) []string {
	differences := make([]string, 0, 3)
	if i.OrganizationName != other.OrganizationName {
		differences = append(differences, FieldOrganizationName)
	}
	if !strings.EqualFold(i.AdminEmail, other.AdminEmail) {
		differences = append(differences, FieldAdminEmail)
	}
	if i.AdminName != other.AdminName {
		differences = append(differences, FieldAdminName)
	}

	return differences
}

type InstanceBootstrap struct {
	bun.BaseModel `bun:"table:instance_bootstraps,alias:ibs" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	AdminUserID    pulid.ID `json:"adminUserId"    bun:"admin_user_id,type:VARCHAR(100),nullzero"`
	Inputs         Inputs   `json:"inputs"         bun:"inputs,type:JSONB,notnull"`
	CompletedAt    int64    `json:"completedAt"    bun:"completed_at,type:BIGINT,notnull"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (r *InstanceBootstrap) BeforeAppendModel(_ context.Context, q bun.Query) error {
	if _, ok := q.(*bun.InsertQuery); !ok {
		return nil
	}

	if r.ID.IsNil() {
		r.ID = pulid.MustNew(IDPrefix)
	}
	now := timeutils.NowUnix()
	if r.CreatedAt == 0 {
		r.CreatedAt = now
	}
	if r.CompletedAt == 0 {
		r.CompletedAt = now
	}

	return nil
}
