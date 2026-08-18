package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratesimulation"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetRateSimulationByIDRequest struct {
	RateSimulationID pulid.ID              `json:"rateSimulationId"`
	TenantInfo       pagination.TenantInfo `json:"-"`
	// IncludeResults loads the per-shipment rows. A status poll does not want
	// them and a run can carry a hundred thousand.
	IncludeResults bool `json:"includeResults"`
}

type ListRateSimulationsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	// RateAgreementID narrows to one contract's simulations, which is what the
	// agreement panel's tab shows.
	RateAgreementID *pulid.ID `json:"rateAgreementId"`
}

type ListRateSimulationResultsRequest struct {
	RateSimulationID pulid.ID                 `json:"rateSimulationId"`
	TenantInfo       pagination.TenantInfo    `json:"-"`
	Filter           *pagination.QueryOptions `json:"filter"`
	// ChangedOnly hides the shipments a change did not move, which is most of
	// them on a targeted amendment.
	ChangedOnly bool `json:"changedOnly"`
}

// SimulationShipmentPage is one batch of shipments to replay, with the cursor
// that continues the walk.
type SimulationShipmentPage struct {
	ShipmentIDs []pulid.ID `json:"shipmentIds"`
	// NextAfterID continues after the last shipment in this page. Empty when
	// the walk is finished.
	NextAfterID pulid.ID `json:"nextAfterId"`
}

// ListSimulationShipmentsRequest walks the shipments a simulation replays.
//
// It pages by id rather than by offset: the walk runs for minutes against a
// table that is still being written to, and an offset would silently skip or
// repeat rows as shipments are created underneath it.
type ListSimulationShipmentsRequest struct {
	TenantInfo pagination.TenantInfo
	PartyType  string
	// From and To bound the shipments by their own ship dates, half open.
	From int64
	To   int64
	// AfterID continues a previous page. Empty starts at the beginning.
	AfterID pulid.ID
	Limit   int
}

type RateSimulationRepository interface {
	GetByID(
		ctx context.Context,
		req *GetRateSimulationByIDRequest,
	) (*ratesimulation.RateSimulation, error)
	List(
		ctx context.Context,
		req *ListRateSimulationsRequest,
	) (*pagination.ListResult[*ratesimulation.RateSimulation], error)
	ListResults(
		ctx context.Context,
		req *ListRateSimulationResultsRequest,
	) (*pagination.ListResult[*ratesimulation.RateSimulationResult], error)

	Create(
		ctx context.Context,
		entity *ratesimulation.RateSimulation,
	) (*ratesimulation.RateSimulation, error)
	Update(
		ctx context.Context,
		entity *ratesimulation.RateSimulation,
	) (*ratesimulation.RateSimulation, error)

	// AppendResults writes one batch of replayed shipments.
	//
	// A run writes as it goes rather than at the end, so a simulation that
	// fails halfway still shows what it managed to price, and so a hundred
	// thousand rows are never held in memory at once.
	AppendResults(
		ctx context.Context,
		results []*ratesimulation.RateSimulationResult,
	) error

	// ListShipments walks the shipments a simulation replays, a page at a time.
	ListShipments(
		ctx context.Context,
		req *ListSimulationShipmentsRequest,
	) (*SimulationShipmentPage, error)
}
