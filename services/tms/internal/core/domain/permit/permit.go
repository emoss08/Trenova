package permit

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Permit)(nil)
	_ validationframework.TenantedEntity = (*Permit)(nil)
	_ domaintypes.PostgresSearchable     = (*Permit)(nil)
)

type Permit struct {
	bun.BaseModel             `bun:"table:permits,alias:pmt" json:"-"`
	pagination.CursorValueSet `bun:",embed"                  json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ShipmentID     pulid.ID `json:"shipmentId"     bun:"shipment_id,type:VARCHAR(100),notnull"`
	StateID        pulid.ID `json:"stateId"        bun:"state_id,type:VARCHAR(100),notnull"`

	PermitNumber string `json:"permitNumber" bun:"permit_number,type:VARCHAR(100),notnull"`
	Status       Status `json:"status"       bun:"status,type:permit_status_enum,notnull,default:'Pending'"`

	IssuedAt  *int64 `json:"issuedAt"  bun:"issued_at,type:BIGINT,nullzero"`
	ExpiresAt *int64 `json:"expiresAt" bun:"expires_at,type:BIGINT,nullzero"`

	Cost  decimal.NullDecimal `json:"cost"       bun:"cost,type:NUMERIC(19,4),nullzero"`
	Notes string              `json:"notes"      bun:"notes,type:TEXT,nullzero"`

	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`

	BusinessUnit *tenant.BusinessUnit `json:"-"               bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"               bun:"rel:belongs-to,join:organization_id=id"`
	State        *usstate.UsState     `json:"state,omitempty" bun:"rel:belongs-to,join:state_id=id"`
}

func (p *Permit) GetID() pulid.ID { return p.ID }

func (p *Permit) GetCreatedAt() int64 { return p.CreatedAt }

func (p *Permit) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *Permit) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *Permit) GetTableName() string { return "permits" }

func (p *Permit) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "pmt",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "permit_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{Name: "notes", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
		},
	}
}

// CoversAt reports whether this permit authorizes movement at a moment in time.
// A permit that expires mid-trip does not cover the trip, which is why the
// caller passes the last planned arrival rather than "now".
func (p *Permit) CoversAt(timestamp int64) bool {
	if !p.Status.Satisfies() {
		return false
	}
	if p.IssuedAt != nil && timestamp < *p.IssuedAt {
		return false
	}
	if p.ExpiresAt != nil && timestamp > *p.ExpiresAt {
		return false
	}
	return true
}

func (p *Permit) IsExpiredAt(timestamp int64) bool {
	return p.ExpiresAt != nil && timestamp > *p.ExpiresAt
}

func (p *Permit) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(p,
		validation.Field(&p.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(&p.StateID, validation.Required.Error("Issuing state is required")),
		validation.Field(&p.PermitNumber,
			validation.Required.Error("Permit number is required"),
			validation.Length(1, 100).
				Error("Permit number must be between 1 and 100 characters"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			validation.In(StatusPending, StatusActive, StatusExpired, StatusVoid).
				Error("Status is invalid"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	p.validateDates(multiErr)

	if p.Cost.Valid && p.Cost.Decimal.IsNegative() {
		multiErr.Add("cost", errortypes.ErrInvalid, "Permit cost cannot be negative")
	}
}

func (p *Permit) validateDates(multiErr *errortypes.MultiError) {
	if p.IssuedAt != nil && p.ExpiresAt != nil && *p.ExpiresAt <= *p.IssuedAt {
		multiErr.Add("expiresAt", errortypes.ErrInvalid,
			"Expiry must be after the issue date")
	}

	if p.Status == StatusActive && p.ExpiresAt == nil {
		multiErr.Add("expiresAt", errortypes.ErrRequired,
			"An active permit must record when it expires")
	}
}

func (p *Permit) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if p.Status == "" {
		p.Status = StatusPending
	}
	if p.Status == StatusActive && p.IsExpiredAt(now) {
		p.Status = StatusExpired
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("pmt_")
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
