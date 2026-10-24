package permit

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*Permit)(nil)
	_ domaintypes.PostgresSearchable     = (*Permit)(nil)
)

type Permit struct {
	keel.Table     `                                       keel:"permits"`
	ID             pulid.ID `json:"id" keel:"id,pk,autoid:pmt_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	ShipmentID     pulid.ID `json:"shipmentId" keel:"shipment_id"`
	StateID        pulid.ID `json:"stateId" keel:"state_id"`

	PermitNumber string `json:"permitNumber" keel:"permit_number"`
	Status       Status `json:"status" keel:"status"`

	IssuedAt  *int64 `json:"issuedAt" keel:"issued_at"`
	ExpiresAt *int64 `json:"expiresAt" keel:"expires_at"`

	Cost  decimal.NullDecimal `json:"cost" keel:"cost"`
	Notes string              `json:"notes" keel:"notes,nullzero"`

	Version      int64  `json:"version" keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector string `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-" keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
	State        *usstate.UsState     `json:"state,omitempty" keel:"belongs:state_id"`
}

func (p *Permit) BeforeUpdate(_ context.Context) error {
	now := timeutils.NowUnix()
	if p.Status == "" {
		p.Status = StatusPending
	}
	if p.Status == StatusActive && p.IsExpiredAt(now) {
		p.Status = StatusExpired
	}

	return nil
}

func (p *Permit) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if p.Status == "" {
		p.Status = StatusPending
	}
	if p.Status == StatusActive && p.IsExpiredAt(now) {
		p.Status = StatusExpired
	}

	return nil
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
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.ShipmentID, validation.Required.Error("Shipment is required")),
		validation.Field(&p.StateID, validation.Required.Error("Issuing state is required")),
		validation.Field(&p.PermitNumber,
			validation.Required.Error("Permit number is required"),
			validation.Length(1, 100).
				Error("Permit number must be between 1 and 100 characters"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
	))

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

func (p *Permit) BeforeSave(_ context.Context) error {
	now := timeutils.NowUnix()
	if p.Status == "" {
		p.Status = StatusPending
	}
	if p.Status == StatusActive && p.IsExpiredAt(now) {
		p.Status = StatusExpired
	}

	return nil
}
