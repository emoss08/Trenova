package ediservicefailure

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	ConnectionRepo     edipartnerports.EDIConnectionRepository
	Lading             *edilading.Runtime
	MappingProfileRepo edipartnerports.EDIMappingProfileRepository
	MessageRepo        edimessageports.EDIMessageRepository
	Outbound           *edioutbound.Service
	PartnerRepo        edipartnerports.EDIPartnerRepository
	ServiceFailureRepo servicefailureports.ServiceFailureRepository
	ShipmentSvc        shipmentserviceports.ShipmentService
}

type Service struct {
	connectionRepo     edipartnerports.EDIConnectionRepository
	lading             *edilading.Runtime
	mappingProfileRepo edipartnerports.EDIMappingProfileRepository
	messageRepo        edimessageports.EDIMessageRepository
	partnerRepo        edipartnerports.EDIPartnerRepository
	serviceFailureRepo servicefailureports.ServiceFailureRepository
	shipmentSvc        shipmentserviceports.ShipmentService
	outbound           *edioutbound.Service
}

func New(p Params) *Service {
	return &Service{
		connectionRepo:     p.ConnectionRepo,
		lading:             p.Lading,
		mappingProfileRepo: p.MappingProfileRepo,
		messageRepo:        p.MessageRepo,
		partnerRepo:        p.PartnerRepo,
		serviceFailureRepo: p.ServiceFailureRepo,
		shipmentSvc:        p.ShipmentSvc,
		outbound:           p.Outbound,
	}
}
