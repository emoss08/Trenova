package ifta

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	CountryCodeUS = "US"
	CountryCodeCA = "CA"
	CountryCodeMX = "MX"

	maxJurisdictionNameLength = 100
)

var (
	_ bun.BeforeAppendModelHook = (*Jurisdiction)(nil)
	_ pagination.CursorEntity   = (*Jurisdiction)(nil)
)

type Jurisdiction struct {
	bun.BaseModel `bun:"table:ifta_jurisdictions,alias:ifj" json:"-"`

	ID           pulid.ID           `json:"id"            bun:"id,pk,type:VARCHAR(100),notnull"`
	CountryCode  string             `json:"countryCode"   bun:"country_code,type:CHAR(2),notnull"`
	Code         string             `json:"code"          bun:"code,type:VARCHAR(2),notnull"`
	Name         string             `json:"name"          bun:"name,type:VARCHAR(100),notnull"`
	UsStateID    *pulid.ID          `json:"usStateId"     bun:"us_state_id,type:VARCHAR(100),nullzero"`
	IsIftaMember bool               `json:"isIftaMember"  bun:"is_ifta_member,type:BOOLEAN,notnull"`
	HasSurcharge bool               `json:"hasSurcharge"  bun:"has_surcharge,type:BOOLEAN,notnull"`
	SortOrder    int                `json:"sortOrder"     bun:"sort_order,type:INTEGER,notnull,default:0"`
	Status       JurisdictionStatus `json:"status"        bun:"status,type:ifta_jurisdiction_status_enum,notnull,default:'Active'"`
	CreatedAt    int64              `json:"createdAt"     bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64              `json:"updatedAt"     bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	UsState *usstate.UsState `json:"usState,omitempty" bun:"rel:belongs-to,join:us_state_id=id"`
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

func (j *Jurisdiction) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if j.ID.IsNil() {
			j.ID = pulid.MustNew("ifj_")
		}
		if j.Status == "" {
			j.Status = JurisdictionStatusActive
		}
		j.CreatedAt = now
		j.UpdatedAt = now
	case *bun.UpdateQuery:
		j.UpdatedAt = now
	}

	return nil
}
