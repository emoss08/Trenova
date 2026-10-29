package agenttoolstest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercreate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/shared/pulid"
)

type FakeShipmentWriter struct {
	Existing *shipment.Shipment                    `json:"-"`
	Created  *shipment.Shipment                    `json:"-"`
	Updated  *shipment.Shipment                    `json:"-"`
	LastGet  *shipmentports.GetShipmentByIDRequest `json:"-"`
	Actor    *permissionports.RequestActor         `json:"-"`
	// previewErr is what the create plan refuses with, when it refuses.
	PreviewErr error `json:"-"`
	Previews   int   `json:"-"`
}

func (f *FakeShipmentWriter) Get(
	_ context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	f.LastGet = req

	return f.Existing, nil
}

func (f *FakeShipmentWriter) Create(
	_ context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	f.Created = entity
	f.Actor = actor
	entity.ID = pulid.MustNew("shp_")
	entity.ProNumber = "S-10001"

	return entity, nil
}

func (f *FakeShipmentWriter) PreviewCreate(
	_ context.Context,
	entity *shipment.Shipment,
	_ *permissionports.RequestActor,
) (*shipmentserviceports.ShipmentCreatePlan, error) {
	f.Previews++
	if f.PreviewErr != nil {
		return nil, f.PreviewErr
	}

	return &shipmentserviceports.ShipmentCreatePlan{Shipment: entity}, nil
}

func (f *FakeShipmentWriter) Update(
	_ context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) (*shipment.Shipment, error) {
	f.Updated = entity
	f.Actor = actor

	return entity, nil
}

type FakeTenders struct {
	Waterfall *tendercreate.CreateWaterfallTenderRequest `json:"-"`
	Spot      *tendercreate.CreateSpotTenderRequest      `json:"-"`
}

func (f *FakeTenders) CreateWaterfall(
	_ context.Context,
	req *tendercreate.CreateWaterfallTenderRequest,
) (*tendercreate.CreateWaterfallResult, error) {
	f.Waterfall = req

	return &tendercreate.CreateWaterfallResult{
		Tender: &tender.Tender{ID: pulid.MustNew("tnd_")},
	}, nil
}

func (f *FakeTenders) CreateSpot(
	_ context.Context,
	req *tendercreate.CreateSpotTenderRequest,
) (*tender.Tender, error) {
	f.Spot = req

	return &tender.Tender{ID: pulid.MustNew("tnd_")}, nil
}
