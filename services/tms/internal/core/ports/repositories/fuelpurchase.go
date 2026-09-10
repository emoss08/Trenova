package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ListFuelCardsRequest struct {
	Filter             *pagination.QueryOptions  `json:"filter"`
	Cursor             pagination.CursorInfo     `json:"cursor"`
	Statuses           []fuelpurchase.CardStatus `json:"statuses"`
	Provider           fuelpurchase.CardProvider `json:"provider"`
	AssignedTractorID  pulid.ID                  `json:"assignedTractorId"`
	AssignedWorkerID   pulid.ID                  `json:"assignedWorkerId"`
	UnassignedOnly     bool                      `json:"unassignedOnly"`
	DiscoveredOnly     bool                      `json:"discoveredOnly"`
	IncludeAssignments bool                      `json:"includeAssignments"`
}

type GetFuelCardByIDRequest struct {
	ID                 pulid.ID              `json:"id"`
	TenantInfo         pagination.TenantInfo `json:"tenantInfo"`
	IncludeAssignments bool                  `json:"includeAssignments"`
}

type GetFuelCardsByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	IDs        []pulid.ID            `json:"ids"`
}

type FindFuelCardByLastFourRequest struct {
	TenantInfo pagination.TenantInfo     `json:"tenantInfo"`
	Provider   fuelpurchase.CardProvider `json:"provider"`
	LastFour   string                    `json:"lastFour"`
}

type FindFuelCardsByLastFourRequest struct {
	TenantInfo pagination.TenantInfo     `json:"tenantInfo"`
	Provider   fuelpurchase.CardProvider `json:"provider"`
	LastFours  []string                  `json:"lastFours"`
}

type GetFuelFeedStateRequest struct {
	TenantInfo pagination.TenantInfo     `json:"tenantInfo"`
	Provider   fuelpurchase.CardProvider `json:"provider"`
	FeedType   fuelpurchase.FeedType     `json:"feedType"`
}

type ListActiveFuelCardsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Query      string                `json:"query"`
	Limit      int                   `json:"limit"`
}

type ListFuelPurchasesRequest struct {
	Filter              *pagination.QueryOptions      `json:"filter"`
	Cursor              pagination.CursorInfo         `json:"cursor"`
	TractorID           pulid.ID                      `json:"tractorId"`
	WorkerID            pulid.ID                      `json:"workerId"`
	JurisdictionID      pulid.ID                      `json:"jurisdictionId"`
	FuelCardID          pulid.ID                      `json:"fuelCardId"`
	ImportBatchID       pulid.ID                      `json:"importBatchId"`
	FuelTypes           []domaintypes.IFTAFuelType    `json:"fuelTypes"`
	Sources             []fuelpurchase.PurchaseSource `json:"sources"`
	TaxPaid             *bool                         `json:"taxPaid"`
	From                int64                         `json:"from"`
	To                  int64                         `json:"to"`
	IncludeTractor      bool                          `json:"includeTractor"`
	IncludeWorker       bool                          `json:"includeWorker"`
	IncludeJurisdiction bool                          `json:"includeJurisdiction"`
	IncludeFuelCard     bool                          `json:"includeFuelCard"`
}

type GetFuelPurchaseByIDRequest struct {
	ID                  pulid.ID              `json:"id"`
	TenantInfo          pagination.TenantInfo `json:"tenantInfo"`
	IncludeTractor      bool                  `json:"includeTractor"`
	IncludeWorker       bool                  `json:"includeWorker"`
	IncludeJurisdiction bool                  `json:"includeJurisdiction"`
	IncludeFuelCard     bool                  `json:"includeFuelCard"`
}

type GetFuelPurchasesByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	IDs        []pulid.ID            `json:"ids"`
}

type FindFuelPurchaseReferencesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	References []string              `json:"references"`
}

type DeleteFuelPurchaseRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type AccumulateFuelRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Start      int64                 `json:"start"`
	End        int64                 `json:"end"`
}

type FuelAccumulationRow struct {
	TractorID      pulid.ID                 `json:"tractorId"      bun:"tractor_id"`
	JurisdictionID pulid.ID                 `json:"jurisdictionId" bun:"jurisdiction_id"`
	FuelType       domaintypes.IFTAFuelType `json:"fuelType"       bun:"fuel_type"`
	Gallons        decimal.Decimal          `json:"gallons"        bun:"gallons"`
	TaxPaidGallons decimal.Decimal          `json:"taxPaidGallons" bun:"tax_paid_gallons"`
	PurchaseCount  int                      `json:"purchaseCount"  bun:"purchase_count"`
}

type GetImportBatchByIDRequest struct {
	ID          pulid.ID              `json:"id"`
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	IncludeRows bool                  `json:"includeRows"`
}

type ListImportBatchesRequest struct {
	Filter             *pagination.QueryOptions    `json:"filter"`
	Cursor             pagination.CursorInfo       `json:"cursor"`
	Origin             fuelpurchase.ImportOrigin   `json:"origin"`
	Provider           fuelpurchase.CardProvider   `json:"provider"`
	Statuses           []fuelpurchase.ImportStatus `json:"statuses"`
	HeldRowsOnly       bool                        `json:"heldRowsOnly"`
	IncludeDefaultCard bool                        `json:"includeDefaultCard"`
}

type ListImportRowsRequest struct {
	BatchID    pulid.ID                       `json:"batchId"`
	TenantInfo pagination.TenantInfo          `json:"tenantInfo"`
	Filter     *pagination.QueryOptions       `json:"filter"`
	Cursor     pagination.CursorInfo          `json:"cursor"`
	Statuses   []fuelpurchase.ImportRowStatus `json:"statuses"`
}

type CommitImportRequest struct {
	Batch            *fuelpurchase.ImportBatch    `json:"batch"`
	Purchases        []*fuelpurchase.FuelPurchase `json:"purchases"`
	RowIDByReference map[string]pulid.ID          `json:"rowIdByReference"`
	CommittedByID    pulid.ID                     `json:"committedById"`
	CommittedAt      int64                        `json:"committedAt"`

	// KeepOpen leaves the batch's status and commit stamp alone, so rows that
	// could not be resolved stay actionable after the ones that could have been
	// posted. A sync sets it when it queues anything for review; a person
	// committing an upload never does, because they have already seen the rows.
	KeepOpen bool `json:"keepOpen"`
}

type CommitImportResult struct {
	Committed       int `json:"committed"`
	AlreadyImported int `json:"alreadyImported"`
}

type FuelPurchaseRepository interface {
	ListCards(
		ctx context.Context,
		req *ListFuelCardsRequest,
	) (*pagination.CursorListResult[*fuelpurchase.FuelCard], error)
	GetCardByID(ctx context.Context, req *GetFuelCardByIDRequest) (*fuelpurchase.FuelCard, error)
	GetCardsByIDs(
		ctx context.Context,
		req *GetFuelCardsByIDsRequest,
	) ([]*fuelpurchase.FuelCard, error)
	FindCardByLastFour(
		ctx context.Context,
		req *FindFuelCardByLastFourRequest,
	) (*fuelpurchase.FuelCard, error)
	ListActiveCards(
		ctx context.Context,
		req *ListActiveFuelCardsRequest,
	) ([]*fuelpurchase.FuelCard, error)
	FindCardsByLastFour(
		ctx context.Context,
		req *FindFuelCardsByLastFourRequest,
	) (map[string]*fuelpurchase.FuelCard, error)
	CreateCard(ctx context.Context, entity *fuelpurchase.FuelCard) (*fuelpurchase.FuelCard, error)
	UpdateCard(ctx context.Context, entity *fuelpurchase.FuelCard) (*fuelpurchase.FuelCard, error)

	ListPurchases(
		ctx context.Context,
		req *ListFuelPurchasesRequest,
	) (*pagination.CursorListResult[*fuelpurchase.FuelPurchase], error)
	GetPurchaseByID(
		ctx context.Context,
		req *GetFuelPurchaseByIDRequest,
	) (*fuelpurchase.FuelPurchase, error)
	GetPurchasesByIDs(
		ctx context.Context,
		req *GetFuelPurchasesByIDsRequest,
	) ([]*fuelpurchase.FuelPurchase, error)
	FindReferences(
		ctx context.Context,
		req *FindFuelPurchaseReferencesRequest,
	) (map[string]pulid.ID, error)
	CreatePurchase(
		ctx context.Context,
		entity *fuelpurchase.FuelPurchase,
	) (*fuelpurchase.FuelPurchase, error)
	UpdatePurchase(
		ctx context.Context,
		entity *fuelpurchase.FuelPurchase,
	) (*fuelpurchase.FuelPurchase, error)
	DeletePurchase(ctx context.Context, req *DeleteFuelPurchaseRequest) error
	AccumulateFuel(ctx context.Context, req *AccumulateFuelRequest) ([]*FuelAccumulationRow, error)

	ListImportBatches(
		ctx context.Context,
		req *ListImportBatchesRequest,
	) (*pagination.CursorListResult[*fuelpurchase.ImportBatch], error)
	GetImportBatchByID(
		ctx context.Context,
		req *GetImportBatchByIDRequest,
	) (*fuelpurchase.ImportBatch, error)
	CreateImportBatch(
		ctx context.Context,
		entity *fuelpurchase.ImportBatch,
	) (*fuelpurchase.ImportBatch, error)
	UpdateImportBatch(
		ctx context.Context,
		entity *fuelpurchase.ImportBatch,
	) (*fuelpurchase.ImportBatch, error)
	ReplaceImportRows(
		ctx context.Context,
		batch *fuelpurchase.ImportBatch,
		rows []*fuelpurchase.ImportRow,
	) error
	ListImportRows(
		ctx context.Context,
		req *ListImportRowsRequest,
	) (*pagination.CursorListResult[*fuelpurchase.ImportRow], error)
	CommitImport(ctx context.Context, req *CommitImportRequest) (*CommitImportResult, error)

	GetFeedState(
		ctx context.Context,
		req *GetFuelFeedStateRequest,
	) (*fuelpurchase.CardFeedState, error)
	SaveFeedState(ctx context.Context, state *fuelpurchase.CardFeedState) error
}
