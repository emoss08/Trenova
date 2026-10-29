package editenderchangeservice

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrecipient"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB                  kernelports.Transactor
	Delivery            *edidelivery.Service
	Logger              *zap.Logger
	Outbound            *edioutbound.Service
	PartnerRepo         edipartnerports.EDIPartnerRepository
	Recipient           *editenderrecipient.Service
	ShipmentLinkRepo    editenderports.EDIShipmentLinkRepository
	ShipmentSvc         shipmentserviceports.ShipmentService
	TenderChangeRepo    editenderports.EDITenderChangeRepository
	TenderRecipientRepo editenderports.EDITenderRecipientRepository
	TenderMapping       *editendermapping.Service
	TransferRepo        editenderports.EDILoadTenderTransferRepository
}

type Service struct {
	db                  kernelports.Transactor
	l                   *zap.Logger
	partnerRepo         edipartnerports.EDIPartnerRepository
	shipmentLinkRepo    editenderports.EDIShipmentLinkRepository
	shipmentSvc         shipmentserviceports.ShipmentService
	tenderChangeRepo    editenderports.EDITenderChangeRepository
	tenderRecipientRepo editenderports.EDITenderRecipientRepository
	transferRepo        editenderports.EDILoadTenderTransferRepository
	delivery            *edidelivery.Service
	outbound            *edioutbound.Service
	recipient           *editenderrecipient.Service
	tenderMapping       *editendermapping.Service
}

func New(p Params) *Service {
	return &Service{
		db:                  p.DB,
		l:                   p.Logger.Named("service.edi"),
		partnerRepo:         p.PartnerRepo,
		shipmentLinkRepo:    p.ShipmentLinkRepo,
		shipmentSvc:         p.ShipmentSvc,
		tenderChangeRepo:    p.TenderChangeRepo,
		tenderRecipientRepo: p.TenderRecipientRepo,
		transferRepo:        p.TransferRepo,
		delivery:            p.Delivery,
		outbound:            p.Outbound,
		recipient:           p.Recipient,
		tenderMapping:       p.TenderMapping,
	}
}
