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

// Miles by state or province recorded outside routed shipment moves
type JurisdictionMileageEntry struct {
	bun.BaseModel             `bun:"table:ifta_jurisdiction_mileage_entries,alias:ifme" json:"-"`
	keel.Table                `                                                                  keel:"ifta_jurisdiction_mileage_entries,report=Equipment"`
	pagination.CursorValueSet `bun:",embed"                                             json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"               keel:"id,pk,autoid:ifme_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"  keel:"organization_id,pk,tenant"`

	TractorID      pulid.ID `json:"tractorId"      bun:"tractor_id,type:VARCHAR(100),notnull"      keel:"tractor_id"`
	JurisdictionID pulid.ID `json:"jurisdictionId" bun:"jurisdiction_id,type:VARCHAR(100),notnull" keel:"jurisdiction_id"`

	TraveledAt int64 `json:"traveledAt" bun:"traveled_at,type:BIGINT,notnull" keel:"traveled_at"`
	Year       int   `json:"year"       bun:"year,type:SMALLINT,notnull"      keel:"year,dimension"`
	Quarter    int   `json:"quarter"    bun:"quarter,type:SMALLINT,notnull"   keel:"quarter,dimension"`

	Miles          decimal.Decimal `json:"miles"          bun:"miles,type:NUMERIC(12,2),notnull"                              keel:"miles,format=distance"`
	Loaded         bool            `json:"loaded"         bun:"loaded,type:BOOLEAN,notnull"                                   keel:"loaded"`
	Source         MileageSource   `json:"source"         bun:"source,type:ifta_mileage_source_enum,notnull,default:'Manual'" keel:"source"`
	ShipmentMoveID *pulid.ID       `json:"shipmentMoveId" bun:"shipment_move_id,type:VARCHAR(100),nullzero"                   keel:"shipment_move_id"`
	Notes          string          `json:"notes"          bun:"notes,type:TEXT,nullzero"                                      keel:"notes,nullzero"`
	CreatedByID    pulid.ID        `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero"                      keel:"created_by_id,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"                                                                  keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Tractor      *tractor.Tractor `json:"tractor,omitempty"      bun:"rel:belongs-to,join:tractor_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:tractor_id"`
	Jurisdiction *Jurisdiction    `json:"jurisdiction,omitempty" bun:"rel:belongs-to,join:jurisdiction_id=id"                                                                        keel:"belongs:jurisdiction_id,noreport"`
	CreatedBy    *tenant.User     `json:"createdBy,omitempty"    bun:"rel:belongs-to,join:created_by_id=id"                                                                          keel:"belongs:created_by_id,noreport"`
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

func (e *JurisdictionMileageEntry) BeforeCreate(ctx context.Context) error {
	return e.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}

func (e *JurisdictionMileageEntry) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "ifme",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "notes", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}
