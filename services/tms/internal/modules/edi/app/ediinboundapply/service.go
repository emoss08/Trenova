package ediinboundapply

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendershipment"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	CarrierInvoiceRepo  ediports.EDICarrierInvoiceRepository
	Logger              *zap.Logger
	MappingProfileRepo  edipartnerports.EDIMappingProfileRepository
	ShipmentEventRepo   shipmenteventports.ShipmentEventRepository
	ShipmentMoveRepo    shipmentmoveports.ShipmentMoveRepository `optional:"true"`
	ShipmentSvc         shipmentserviceports.ShipmentLifecycleService
	Shipments           *editendershipment.Service
	TenderRecipientRepo editenderports.EDITenderRecipientRepository
	TenderMapping       *editendermapping.Service
	TrackingEvents      trackingeventports.TrackingEventService `optional:"true"`
	TransferRepo        editenderports.EDILoadTenderTransferRepository
}

type Service struct {
	carrierInvoiceRepo  ediports.EDICarrierInvoiceRepository
	l                   *zap.Logger
	mappingProfileRepo  edipartnerports.EDIMappingProfileRepository
	shipmentEventRepo   shipmenteventports.ShipmentEventRepository
	shipmentMoveRepo    shipmentmoveports.ShipmentMoveRepository
	shipmentSvc         shipmentserviceports.ShipmentLifecycleService
	tenderRecipientRepo editenderports.EDITenderRecipientRepository
	trackingEvents      trackingeventports.TrackingEventService
	transferRepo        editenderports.EDILoadTenderTransferRepository
	shipments           *editendershipment.Service
	tenderMapping       *editendermapping.Service
}

func New(p Params) *Service {
	return &Service{
		carrierInvoiceRepo:  p.CarrierInvoiceRepo,
		l:                   p.Logger.Named("service.edi"),
		mappingProfileRepo:  p.MappingProfileRepo,
		shipmentEventRepo:   p.ShipmentEventRepo,
		shipmentMoveRepo:    p.ShipmentMoveRepo,
		shipmentSvc:         p.ShipmentSvc,
		tenderRecipientRepo: p.TenderRecipientRepo,
		trackingEvents:      p.TrackingEvents,
		transferRepo:        p.TransferRepo,
		shipments:           p.Shipments,
		tenderMapping:       p.TenderMapping,
	}
}
