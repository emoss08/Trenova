package edioverview

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	CarrierRepo        carrierports.CarrierRepository `optional:"true"`
	InboundFileRepo    edimessageports.EDIInboundFileRepository
	Lading             *edilading.Runtime
	MappingProfileRepo edipartnerports.EDIMappingProfileRepository
	MessageRepo        edimessageports.EDIMessageRepository
	PartnerRepo        edipartnerports.EDIPartnerRepository
	ProfileRepo        edipartnerports.EDICommunicationProfileRepository
	TransferRepo       editenderports.EDILoadTenderTransferRepository
}

type Service struct {
	carrierRepo        carrierports.CarrierRepository
	inboundFileRepo    edimessageports.EDIInboundFileRepository
	lading             *edilading.Runtime
	mappingProfileRepo edipartnerports.EDIMappingProfileRepository
	messageRepo        edimessageports.EDIMessageRepository
	partnerRepo        edipartnerports.EDIPartnerRepository
	profileRepo        edipartnerports.EDICommunicationProfileRepository
	transferRepo       editenderports.EDILoadTenderTransferRepository
}

func New(p Params) *Service {
	return &Service{
		carrierRepo:        p.CarrierRepo,
		inboundFileRepo:    p.InboundFileRepo,
		lading:             p.Lading,
		mappingProfileRepo: p.MappingProfileRepo,
		messageRepo:        p.MessageRepo,
		partnerRepo:        p.PartnerRepo,
		profileRepo:        p.ProfileRepo,
		transferRepo:       p.TransferRepo,
	}
}
