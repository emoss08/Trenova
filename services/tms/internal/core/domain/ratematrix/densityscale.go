package ratematrix

import (
	"context"
	"sort"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
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

var (
	_ bun.BeforeAppendModelHook          = (*DensityScale)(nil)
	_ validationframework.TenantedEntity = (*DensityScale)(nil)
)

// DensityScale maps freight density to a freight class.
//
// The 2025 NMFC restructure moved most commodities off fixed classifications
// and onto a density scale, so a shipment's class is now something the system
// derives from weight and cube rather than something it reads off a commodity
// record. The scale is stored rather than compiled in because it is revised,
// because a contract can name an older revision, and because carriers publish
// their own variants.
type DensityScale struct {
	bun.BaseModel             `bun:"table:rate_density_scales,alias:rds" json:"-"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Code          string             `json:"code"          bun:"code,type:VARCHAR(64),notnull"`
	Name          string             `json:"name"          bun:"name,type:VARCHAR(100),notnull"`
	Description   string             `json:"description"   bun:"description,type:TEXT,nullzero"`
	Status        domaintypes.Status `json:"status"        bun:"status,type:status_enum,notnull,default:'Active'"`
	EffectiveFrom int64              `json:"effectiveFrom" bun:"effective_from,type:BIGINT,notnull"`
	IsOrgDefault  bool               `json:"isOrgDefault"  bun:"is_org_default,type:BOOLEAN,notnull,default:false"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"-"               bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"               bun:"rel:belongs-to,join:organization_id=id"`
	Tiers        []*DensityScaleTier  `json:"tiers,omitempty" bun:"rel:has-many,join:id=rate_density_scale_id"`
}

// ClassFor returns the freight class for a density in pounds per cubic foot.
//
// Tiers are half open on their upper bound, and the densest tier carries no
// upper bound at all, so every non-negative density lands in exactly one tier.
func (ds *DensityScale) ClassFor(
	densityPcf decimal.Decimal,
) (commodity.FreightClass, *DensityScaleTier, bool) {
	for _, tier := range ds.sortedTiers() {
		if tier == nil {
			continue
		}

		if densityPcf.LessThan(tier.FromPcf) {
			continue
		}

		if tier.ToPcf.Valid && densityPcf.GreaterThanOrEqual(tier.ToPcf.Decimal) {
			continue
		}

		return tier.FreightClass, tier, true
	}

	return "", nil, false
}

func (ds *DensityScale) sortedTiers() []*DensityScaleTier {
	tiers := make([]*DensityScaleTier, 0, len(ds.Tiers))
	for _, tier := range ds.Tiers {
		if tier != nil {
			tiers = append(tiers, tier)
		}
	}

	sort.Slice(tiers, func(a, b int) bool {
		return tiers[a].FromPcf.LessThan(tiers[b].FromPcf)
	})

	return tiers
}

// applyDefaults fills in the values the database would have defaulted, so a
// payload that omits them validates the same way it will be stored.
func (ds *DensityScale) applyDefaults() {
	if ds.Status == "" {
		ds.Status = domaintypes.StatusActive
	}
}

func (ds *DensityScale) Validate(multiErr *errortypes.MultiError) {
	ds.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(ds,
		validation.Field(&ds.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, maxCodeLength).
				Error("Code must be between 1 and 64 characters"),
		),
		validation.Field(&ds.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxNameLength).
				Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&ds.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status is invalid"),
		),
		validation.Field(&ds.EffectiveFrom,
			validation.Required.Error("Effective from is required"),
		),
	))

	ds.validateTiers(multiErr)
}

// validateTiers rejects any scale that would leave a density unclassified or
// claimed twice. A gap means a shipment that cannot be rated at all; an overlap
// means the class depends on row order, which is the same shipment rating two
// ways on two servers.
func (ds *DensityScale) validateTiers(multiErr *errortypes.MultiError) {
	if len(ds.Tiers) == 0 {
		multiErr.Add("tiers", errortypes.ErrRequired, "A density scale must have at least one tier")
		return
	}

	indexByTier := make(map[*DensityScaleTier]int, len(ds.Tiers))
	for i, tier := range ds.Tiers {
		if tier == nil {
			continue
		}
		indexByTier[tier] = i
		tier.Validate(multiErr.WithIndex("tiers", i))
	}

	sorted := ds.sortedTiers()
	if len(sorted) == 0 {
		return
	}

	if !sorted[0].FromPcf.IsZero() {
		multiErr.WithIndex("tiers", indexByTier[sorted[0]]).Add(
			"fromPcf",
			errortypes.ErrInvalid,
			"The lightest tier must start at zero",
		)
	}

	for i := 1; i < len(sorted); i++ {
		previous := sorted[i-1]
		current := sorted[i]
		currentErr := multiErr.WithIndex("tiers", indexByTier[current])

		if !previous.ToPcf.Valid {
			currentErr.Add(
				"fromPcf",
				errortypes.ErrInvalid,
				"Only the densest tier may be left open ended",
			)
			continue
		}

		if !previous.ToPcf.Decimal.Equal(current.FromPcf) {
			currentErr.Add(
				"fromPcf",
				errortypes.ErrInvalid,
				"Tiers must be contiguous with no gaps or overlaps",
			)
		}
	}

	if densest := sorted[len(sorted)-1]; densest.ToPcf.Valid {
		multiErr.WithIndex("tiers", indexByTier[densest]).Add(
			"toPcf",
			errortypes.ErrInvalid,
			"The densest tier must be open ended",
		)
	}
}

func (ds *DensityScale) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ds.ID.IsNil() {
			ds.ID = pulid.MustNew("rds_")
		}
		ds.CreatedAt = now
		ds.UpdatedAt = now
	case *bun.UpdateQuery:
		ds.UpdatedAt = now
	}

	return nil
}

func (ds *DensityScale) GetID() pulid.ID {
	return ds.ID
}

func (ds *DensityScale) GetCreatedAt() int64 {
	return ds.CreatedAt
}

func (ds *DensityScale) GetOrganizationID() pulid.ID {
	return ds.OrganizationID
}

func (ds *DensityScale) GetBusinessUnitID() pulid.ID {
	return ds.BusinessUnitID
}

func (ds *DensityScale) GetTableName() string {
	return "rate_density_scales"
}

// GetPostgresSearchConfig declares no search vector. There are only ever a
// handful of scales per organization, so the list is browsed rather than
// searched; the fields here are what it filters on.
func (ds *DensityScale) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "rds",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText},
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

// StandardDensityScaleCode names the scale published with the 2025 NMFC
// restructure. Organizations may edit their copy or add carrier variants, so
// this is the code of the seeded default rather than a special case in code.
const StandardDensityScaleCode = "NMFC_2025"

// standardTier is one row of the published thirteen tier scale, in pounds per
// cubic foot against the class it carries.
type standardTier struct {
	fromPcf string
	toPcf   string
	class   commodity.FreightClass
}

// standardTiers is the thirteen tier density scale introduced by NMFC docket
// 2025-1, which replaced the eleven tier scale and moved most commodities off
// fixed classifications. Bands are half open and the densest is open ended, so
// together they classify every possible density exactly once.
var standardTiers = []standardTier{
	{"0", "1", commodity.FreightClass400},
	{"1", "2", commodity.FreightClass300},
	{"2", "4", commodity.FreightClass250},
	{"4", "6", commodity.FreightClass175},
	{"6", "8", commodity.FreightClass125},
	{"8", "10", commodity.FreightClass100},
	{"10", "12", commodity.FreightClass92_5},
	{"12", "15", commodity.FreightClass85},
	{"15", "22.5", commodity.FreightClass70},
	{"22.5", "30", commodity.FreightClass65},
	{"30", "35", commodity.FreightClass60},
	{"35", "50", commodity.FreightClass55},
	{"50", "", commodity.FreightClass50},
}

// StandardDensityTiers builds the published scale for an organization. The
// seeder and the "reset to published values" action both call it, so the two
// can never drift.
func StandardDensityTiers(orgID, buID pulid.ID) []*DensityScaleTier {
	tiers := make([]*DensityScaleTier, 0, len(standardTiers))

	for i, row := range standardTiers {
		tier := &DensityScaleTier{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			FromPcf:        decimal.RequireFromString(row.fromPcf),
			FreightClass:   row.class,
			SortOrder:      int16(i),
		}

		if row.toPcf != "" {
			tier.ToPcf = decimal.NewNullDecimal(decimal.RequireFromString(row.toPcf))
		}

		tiers = append(tiers, tier)
	}

	return tiers
}

var _ bun.BeforeAppendModelHook = (*DensityScaleTier)(nil)

// DensityScaleTier is one density band and the class it carries.
type DensityScaleTier struct {
	bun.BaseModel `bun:"table:rate_density_scale_tiers,alias:rdt" json:"-"`

	ID                 pulid.ID `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RateDensityScaleID pulid.ID `json:"rateDensityScaleId" bun:"rate_density_scale_id,type:VARCHAR(100),notnull"`

	FromPcf      decimal.Decimal        `json:"fromPcf"      bun:"from_pcf,type:NUMERIC(10,4),notnull"`
	ToPcf        decimal.NullDecimal    `json:"toPcf"        bun:"to_pcf,type:NUMERIC(10,4),nullzero"`
	FreightClass commodity.FreightClass `json:"freightClass" bun:"freight_class,type:freight_class_enum,notnull"`
	SortOrder    int16                  `json:"sortOrder"    bun:"sort_order,type:SMALLINT,notnull"`

	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	DensityScale *DensityScale `json:"-" bun:"rel:belongs-to,join:rate_density_scale_id=id"`
}

func (dst *DensityScaleTier) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(dst,
		validation.Field(&dst.FreightClass,
			validation.Required.Error("Freight class is required"),
			domainvalidation.ValidEnum[commodity.FreightClass]("Freight class is invalid"),
		),
	))

	if dst.FromPcf.IsNegative() {
		multiErr.Add("fromPcf", errortypes.ErrInvalid, "Density cannot be negative")
	}

	if dst.ToPcf.Valid && dst.ToPcf.Decimal.LessThanOrEqual(dst.FromPcf) {
		multiErr.Add(
			"toPcf",
			errortypes.ErrInvalid,
			"The upper density must be greater than the lower density",
		)
	}
}

func (dst *DensityScaleTier) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if dst.ID.IsNil() {
			dst.ID = pulid.MustNew("rdt_")
		}
		dst.CreatedAt = now
		dst.UpdatedAt = now
	case *bun.UpdateQuery:
		dst.UpdatedAt = now
	}

	return nil
}

func (dst *DensityScaleTier) GetID() pulid.ID {
	return dst.ID
}

func (dst *DensityScaleTier) GetOrganizationID() pulid.ID {
	return dst.OrganizationID
}

func (dst *DensityScaleTier) GetBusinessUnitID() pulid.ID {
	return dst.BusinessUnitID
}

func (dst *DensityScaleTier) GetTableName() string {
	return "rate_density_scale_tiers"
}
