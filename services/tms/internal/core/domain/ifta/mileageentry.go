package ifta

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	MilesScale = 2
)

var (
	MaxMileageEntryMiles = decimal.NewFromInt(100_000)

	_ bun.BeforeAppendModelHook          = (*JurisdictionMileageEntry)(nil)
	_ pagination.CursorEntity            = (*JurisdictionMileageEntry)(nil)
	_ validationframework.TenantedEntity = (*JurisdictionMileageEntry)(nil)
)

type JurisdictionMileageEntry struct {
	bun.BaseModel             `bun:"table:ifta_jurisdiction_mileage_entries,alias:ifme" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                             json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	TractorID      pulid.ID `json:"tractorId"      bun:"tractor_id,type:VARCHAR(100),notnull"`
	JurisdictionID pulid.ID `json:"jurisdictionId" bun:"jurisdiction_id,type:VARCHAR(100),notnull"`

	TraveledAt int64 `json:"traveledAt" bun:"traveled_at,type:BIGINT,notnull"`
	Year       int   `json:"year"       bun:"year,type:SMALLINT,notnull"`
	Quarter    int   `json:"quarter"    bun:"quarter,type:SMALLINT,notnull"`

	Miles          decimal.Decimal `json:"miles"          bun:"miles,type:NUMERIC(12,2),notnull"`
	Loaded         bool            `json:"loaded"         bun:"loaded,type:BOOLEAN,notnull"`
	Source         MileageSource   `json:"source"         bun:"source,type:ifta_mileage_source_enum,notnull,default:'Manual'"`
	ShipmentMoveID *pulid.ID       `json:"shipmentMoveId" bun:"shipment_move_id,type:VARCHAR(100),nullzero"`
	Notes          string          `json:"notes"          bun:"notes,type:TEXT,nullzero"`
	CreatedByID    pulid.ID        `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Tractor      *tractor.Tractor `json:"tractor,omitempty"      bun:"rel:belongs-to,join:tractor_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Jurisdiction *Jurisdiction    `json:"jurisdiction,omitempty" bun:"rel:belongs-to,join:jurisdiction_id=id"`
	CreatedBy    *tenant.User     `json:"createdBy,omitempty"    bun:"rel:belongs-to,join:created_by_id=id"`
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
			"A "+e.Source.Label()+" entry must reference the move it came from",
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

func (e *JurisdictionMileageEntry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("ifme_")
		}
		if e.Source == "" {
			e.Source = MileageSourceManual
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
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
