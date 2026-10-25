package ifta

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	CountryCodeUS = "US"
	CountryCodeCA = "CA"
	CountryCodeMX = "MX"

	maxJurisdictionNameLength = 100
)

var (
	_ pagination.CursorEntity = (*Jurisdiction)(nil)
)

type Jurisdiction struct {
	keel.Table `keel:"ifta_jurisdictions"`

	ID           pulid.ID           `json:"id"           keel:"id,pk,autoid:ifj_"`
	CountryCode  string             `json:"countryCode"  keel:"country_code"`
	Code         string             `json:"code"         keel:"code"`
	Name         string             `json:"name"         keel:"name"`
	UsStateID    *pulid.ID          `json:"usStateId"    keel:"us_state_id"`
	IsIftaMember bool               `json:"isIftaMember" keel:"is_ifta_member"`
	HasSurcharge bool               `json:"hasSurcharge" keel:"has_surcharge"`
	SortOrder    int                `json:"sortOrder"    keel:"sort_order"`
	Status       JurisdictionStatus `json:"status"       keel:"status"`
	CreatedAt    int64              `json:"createdAt"    keel:"created_at,autocreate"`
	UpdatedAt    int64              `json:"updatedAt"    keel:"updated_at,autoupdate"`

	UsState *usstate.UsState `json:"usState,omitempty" keel:"belongs:us_state_id"`
}

func JurisdictionKey(countryCode, code string) string {
	return strings.ToUpper(strings.TrimSpace(countryCode)) + "_" +
		strings.ToUpper(strings.TrimSpace(code))
}

func (j *Jurisdiction) Normalize() {
	j.CountryCode = strings.ToUpper(strings.TrimSpace(j.CountryCode))
	j.Code = strings.ToUpper(strings.TrimSpace(j.Code))
	j.Name = strings.TrimSpace(j.Name)
	if j.Status == "" {
		j.Status = JurisdictionStatusActive
	}
}

func (j *Jurisdiction) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(j,
		validation.Field(&j.CountryCode,
			validation.Required.Error("Country code is required"),
			validation.Match(domaintypes.JurisdictionCodeRegex).
				Error("Country code must be two upper-case letters"),
		),
		validation.Field(&j.Code,
			validation.Required.Error("Code is required"),
			validation.Match(domaintypes.JurisdictionCodeRegex).
				Error("Code must be two upper-case letters"),
		),
		validation.Field(&j.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxJurisdictionNameLength).
				Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&j.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[JurisdictionStatus]("Status is not valid"),
		),
		validation.Field(&j.SortOrder,
			validation.Min(0).Error("Sort order cannot be negative"),
		),
	))

	if j.UsStateID != nil && !j.UsStateID.IsNil() && j.CountryCode != CountryCodeUS {
		multiErr.Add(
			"usStateId",
			errortypes.ErrInvalid,
			"Only a US jurisdiction can be linked to a US state",
		)
	}
}

func (j *Jurisdiction) IsMember() bool { return j.IsIftaMember }

func (j *Jurisdiction) IsActive() bool { return j.Status == JurisdictionStatusActive }

func (j *Jurisdiction) Label() string { return j.Code + " – " + j.Name }

func (j *Jurisdiction) Key() string { return JurisdictionKey(j.CountryCode, j.Code) }

func (j *Jurisdiction) GetID() pulid.ID { return j.ID }

func (j *Jurisdiction) GetCreatedAt() int64 { return j.CreatedAt }

func (j *Jurisdiction) GetTableName() string { return "ifta_jurisdictions" }

func (j *Jurisdiction) BeforeCreate(ctx context.Context) error {
	if j.Status == "" {
		j.Status = JurisdictionStatusActive
	}

	return nil
}
