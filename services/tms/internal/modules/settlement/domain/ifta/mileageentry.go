package ifta

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const (
	MilesScale = 2
)

var (
	MaxMileageEntryMiles                                    = decimal.NewFromInt(100_000)
	_                    pagination.CursorEntity            = (*JurisdictionMileageEntry)(nil)
	_                    validationframework.TenantedEntity = (*JurisdictionMileageEntry)(nil)
)

// Miles by state or province recorded outside routed shipment moves
type JurisdictionMileageEntry struct {
	keel.Table `                                                                  keel:"ifta_jurisdiction_mileage_entries,report=Equipment"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:ifme_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	TractorID      pulid.ID `json:"tractorId" keel:"tractor_id"`
	JurisdictionID pulid.ID `json:"jurisdictionId" keel:"jurisdiction_id"`

	TraveledAt int64 `json:"traveledAt" keel:"traveled_at"`
	Year       int   `json:"year" keel:"year,dimension"`
	Quarter    int   `json:"quarter" keel:"quarter,dimension"`

	Miles          decimal.Decimal `json:"miles" keel:"miles,format=distance"`
	Loaded         bool            `json:"loaded" keel:"loaded"`
	Source         MileageSource   `json:"source" keel:"source"`
	ShipmentMoveID *pulid.ID       `json:"shipmentMoveId" keel:"shipment_move_id"`
	Notes          string          `json:"notes" keel:"notes,nullzero"`
	CreatedByID    pulid.ID        `json:"createdById" keel:"created_by_id,nullzero"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Tractor      *tractor.Tractor `json:"tractor,omitempty" keel:"belongs:tractor_id"`
	Jurisdiction *Jurisdiction    `json:"jurisdiction,omitempty" keel:"belongs:jurisdiction_id,noreport"`
	CreatedBy    *tenant.User     `json:"createdBy,omitempty" keel:"belongs:created_by_id,noreport"`
}

func (e *JurisdictionMileageEntry) Normalize() {
	e.Notes = strings.TrimSpace(e.Notes)
	e.Miles = e.Miles.Round(MilesScale)
	if e.Source == "" {
		e.Source = MileageSourceManual
	}
}

func (e *JurisdictionMileageEntry) AssignPeriod(loc *time.Location) {
	period := PeriodOf(e.TraveledAt, loc)
	e.Year = period.Year
	e.Quarter = period.Quarter
}

func (e *JurisdictionMileageEntry) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.TractorID, validation.Required.Error("Tractor is required")),
		validation.Field(&e.JurisdictionID,
			validation.Required.Error("Jurisdiction is required"),
		),
		validation.Field(&e.Source,
			validation.Required.Error("Source is required"),
			domainvalidation.ValidEnum[MileageSource]("Source is not valid"),
		),
	))

	if e.TraveledAt <= 0 {
		multiErr.Add("traveledAt", errortypes.ErrRequired, "Travel date is required")
	}
	if err := e.Period().Validate(); err != nil {
		multiErr.Add("quarter", errortypes.ErrInvalid, err.Error())
	}

	if !e.Miles.IsPositive() {
		multiErr.Add("miles", errortypes.ErrInvalid, "Miles must be greater than zero")
	} else if e.Miles.GreaterThan(MaxMileageEntryMiles) {
		multiErr.Add("miles", errortypes.ErrInvalid, "Miles cannot exceed 100,000 in one entry")
	}

	if e.Source != MileageSourceManual && (e.ShipmentMoveID == nil || e.ShipmentMoveID.IsNil()) {
		multiErr.Add(
			"shipmentMoveId",
			errortypes.ErrRequired,
			"A {0} entry must reference the move it came from", e.Source.Label(),
		)
	}
}

func (e *JurisdictionMileageEntry) Period() Period {
	return Period{Year: e.Year, Quarter: e.Quarter}
}

func (e *JurisdictionMileageEntry) OverridesMove() bool {
	return e.ShipmentMoveID != nil && !e.ShipmentMoveID.IsNil()
}

func (e *JurisdictionMileageEntry) GetID() pulid.ID { return e.ID }

func (e *JurisdictionMileageEntry) GetCreatedAt() int64 { return e.CreatedAt }

func (e *JurisdictionMileageEntry) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *JurisdictionMileageEntry) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *JurisdictionMileageEntry) GetTableName() string {
	return "ifta_jurisdiction_mileage_entries"
}

func (e *JurisdictionMileageEntry) GetResourceType() string {
	return "ifta_jurisdiction_mileage_entry"
}

func (e *JurisdictionMileageEntry) GetResourceID() string { return e.ID.String() }

func (e *JurisdictionMileageEntry) BeforeCreate(ctx context.Context) error {
	if e.Source == "" {
		e.Source = MileageSourceManual
	}

	return nil
}

func (e *JurisdictionMileageEntry) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "ifme",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "notes", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}
