package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const MaxAggregateMoveIDs = 50

type ListJurisdictionsRequest struct {
	MembersOnly bool                      `json:"membersOnly"`
	CountryCode string                    `json:"countryCode"`
	Statuses    []ifta.JurisdictionStatus `json:"statuses"`
}

type ListTaxRatesRequest struct {
	Filter              *pagination.QueryOptions `json:"filter"`
	Cursor              pagination.CursorInfo    `json:"cursor"`
	Year                int                      `json:"year"`
	Quarter             int                      `json:"quarter"`
	JurisdictionID      pulid.ID                 `json:"jurisdictionId"`
	FuelType            domaintypes.IFTAFuelType `json:"fuelType"`
	IncludeJurisdiction bool                     `json:"includeJurisdiction"`
}

type ResolveRatesRequest struct {
	Year    int `json:"year"`
	Quarter int `json:"quarter"`
}

type ListMileageEntriesRequest struct {
	Filter              *pagination.QueryOptions `json:"filter"`
	Cursor              pagination.CursorInfo    `json:"cursor"`
	TractorID           pulid.ID                 `json:"tractorId"`
	JurisdictionID      pulid.ID                 `json:"jurisdictionId"`
	Year                int                      `json:"year"`
	Quarter             int                      `json:"quarter"`
	Sources             []ifta.MileageSource     `json:"sources"`
	IncludeTractor      bool                     `json:"includeTractor"`
	IncludeJurisdiction bool                     `json:"includeJurisdiction"`
}

type GetMileageEntryByIDRequest struct {
	ID                  pulid.ID              `json:"id"`
	TenantInfo          pagination.TenantInfo `json:"tenantInfo"`
	IncludeTractor      bool                  `json:"includeTractor"`
	IncludeJurisdiction bool                  `json:"includeJurisdiction"`
}

type DeleteMileageEntryRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Version    int64                 `json:"version"`
}

type ListReturnsRequest struct {
	Filter   *pagination.QueryOptions `json:"filter"`
	Cursor   pagination.CursorInfo    `json:"cursor"`
	Year     int                      `json:"year"`
	Statuses []ifta.ReturnStatus      `json:"statuses"`
}

type GetReturnByIDRequest struct {
	ID                   pulid.ID              `json:"id"`
	TenantInfo           pagination.TenantInfo `json:"tenantInfo"`
	IncludeLines         bool                  `json:"includeLines"`
	IncludeJurisdictions bool                  `json:"includeJurisdictions"`
}

type GetReturnsByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	IDs        []pulid.ID            `json:"ids"`
}

type GetOpenReturnForPeriodRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Year       int                   `json:"year"`
	Quarter    int                   `json:"quarter"`
}

type GetLatestReturnForPeriodRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Year       int                   `json:"year"`
	Quarter    int                   `json:"quarter"`
}

type DeleteReturnRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Version    int64                 `json:"version"`
}

type AccumulateMilesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Start      int64                 `json:"start"`
	End        int64                 `json:"end"`
	Year       int                   `json:"year"`
	Quarter    int                   `json:"quarter"`
}

type MileRow struct {
	TractorID        pulid.ID        `json:"tractorId"        bun:"tractor_id"`
	CountryCode      string          `json:"countryCode"      bun:"country_code"`
	JurisdictionCode string          `json:"jurisdictionCode" bun:"jurisdiction_code"`
	JurisdictionID   pulid.ID        `json:"jurisdictionId"   bun:"jurisdiction_id"`
	Miles            decimal.Decimal `json:"miles"            bun:"miles"`
	LoadedMiles      decimal.Decimal `json:"loadedMiles"      bun:"loaded_miles"`
	EmptyMiles       decimal.Decimal `json:"emptyMiles"       bun:"empty_miles"`
	MoveCount        int             `json:"moveCount"        bun:"move_count"`
}

func (r *MileRow) HasTractor() bool { return !r.TractorID.IsNil() }

func (r *MileRow) HasJurisdiction() bool { return !r.JurisdictionID.IsNil() }

func (r *MileRow) Key() string { return ifta.JurisdictionKey(r.CountryCode, r.JurisdictionCode) }

type MoveAggregate struct {
	Miles     decimal.Decimal `json:"miles"`
	MoveCount int             `json:"moveCount"`
	MoveIDs   []pulid.ID      `json:"moveIds"`
}

func (a *MoveAggregate) Add(moveID pulid.ID, miles decimal.Decimal) {
	a.Miles = a.Miles.Add(miles)
	a.MoveCount++
	if len(a.MoveIDs) < MaxAggregateMoveIDs {
		a.MoveIDs = append(a.MoveIDs, moveID)
	}
}

type MoveMismatch struct {
	MoveID          pulid.ID        `json:"moveId"          bun:"move_id"`
	MoveDistance    decimal.Decimal `json:"moveDistance"    bun:"move_distance"`
	AttributedMiles decimal.Decimal `json:"attributedMiles" bun:"attributed_miles"`
}

type MileAccumulation struct {
	RouteRows    []*MileRow     `json:"routeRows"`
	ManualRows   []*MileRow     `json:"manualRows"`
	Unattributed MoveAggregate  `json:"unattributed"`
	NoTractor    MoveAggregate  `json:"noTractor"`
	Mismatches   []MoveMismatch `json:"mismatches"`
}

type IFTAJurisdictionCacheRepository interface {
	GetAll(ctx context.Context) ([]*ifta.Jurisdiction, error)
	Set(ctx context.Context, jurisdictions []*ifta.Jurisdiction) error
	Invalidate(ctx context.Context) error
}

type IFTARepository interface {
	ListJurisdictions(
		ctx context.Context,
		req *ListJurisdictionsRequest,
	) ([]*ifta.Jurisdiction, error)
	GetJurisdictionByID(ctx context.Context, id pulid.ID) (*ifta.Jurisdiction, error)
	GetJurisdictionsByIDs(ctx context.Context, ids []pulid.ID) ([]*ifta.Jurisdiction, error)
	GetJurisdictionByCode(
		ctx context.Context,
		countryCode, code string,
	) (*ifta.Jurisdiction, error)
	FindJurisdictionsByCodes(
		ctx context.Context,
		keys []string,
	) (map[string]*ifta.Jurisdiction, error)

	ListTaxRates(
		ctx context.Context,
		req *ListTaxRatesRequest,
	) (*pagination.CursorListResult[*ifta.TaxRate], error)
	GetTaxRateByID(ctx context.Context, id pulid.ID) (*ifta.TaxRate, error)
	UpsertTaxRates(ctx context.Context, rates []*ifta.TaxRate) ([]*ifta.TaxRate, error)
	DeleteTaxRate(ctx context.Context, id pulid.ID, version int64) error
	ResolveRates(
		ctx context.Context,
		req *ResolveRatesRequest,
	) (map[ifta.RateKey]*ifta.TaxRate, error)

	ListMileageEntries(
		ctx context.Context,
		req *ListMileageEntriesRequest,
	) (*pagination.CursorListResult[*ifta.JurisdictionMileageEntry], error)
	GetMileageEntryByID(
		ctx context.Context,
		req *GetMileageEntryByIDRequest,
	) (*ifta.JurisdictionMileageEntry, error)
	CreateMileageEntry(
		ctx context.Context,
		entity *ifta.JurisdictionMileageEntry,
	) (*ifta.JurisdictionMileageEntry, error)
	UpdateMileageEntry(
		ctx context.Context,
		entity *ifta.JurisdictionMileageEntry,
	) (*ifta.JurisdictionMileageEntry, error)
	DeleteMileageEntry(ctx context.Context, req *DeleteMileageEntryRequest) error

	ListReturns(
		ctx context.Context,
		req *ListReturnsRequest,
	) (*pagination.CursorListResult[*ifta.Return], error)
	GetReturnByID(ctx context.Context, req *GetReturnByIDRequest) (*ifta.Return, error)
	GetReturnsByIDs(ctx context.Context, req *GetReturnsByIDsRequest) ([]*ifta.Return, error)
	GetOpenReturnForPeriod(
		ctx context.Context,
		req *GetOpenReturnForPeriodRequest,
	) (*ifta.Return, error)
	GetLatestReturnForPeriod(
		ctx context.Context,
		req *GetLatestReturnForPeriodRequest,
	) (*ifta.Return, error)
	CreateReturn(ctx context.Context, entity *ifta.Return) (*ifta.Return, error)
	UpdateReturn(ctx context.Context, entity *ifta.Return) (*ifta.Return, error)
	ReplaceReturnLines(
		ctx context.Context,
		ret *ifta.Return,
		lines []*ifta.ReturnLine,
	) (*ifta.Return, error)
	DeleteReturn(ctx context.Context, req *DeleteReturnRequest) error

	AccumulateMiles(
		ctx context.Context,
		req *AccumulateMilesRequest,
	) (*MileAccumulation, error)
}
