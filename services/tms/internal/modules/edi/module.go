package edi

import (
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicarrierinvoicerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicommunicationprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediconnectiondraftrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediconnectionrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicontrolnumberrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edidocumenttyperepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediinboundfilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediladingrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimappingprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimessagerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edipartnerrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edishipmentlinkrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edishipmenttenderrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editenderchangerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editenderrecipientrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransactionsetrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferchangerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferrepository"
	"github.com/emoss08/trenova/internal/modules/edi/app/edialert"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediconnectiondraft"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundapply"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundas2"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundpoll"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edimessageview"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioverview"
	"github.com/emoss08/trenova/internal/modules/edi/app/edipayloadbuild"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediprofilesecret"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservicefailure"
	"github.com/emoss08/trenova/internal/modules/edi/app/edisetup"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendermapping"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrecipient"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editendershipment"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransport"
	"github.com/emoss08/trenova/internal/modules/edi/app/edivalidation"
	"github.com/emoss08/trenova/internal/modules/edi/app/referencedataguard"
	"github.com/emoss08/trenova/internal/modules/edi/jobs/edijobs/edijobscontract"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			ediinboundservice.New,
			ediinboundas2.New,
			ediinboundpoll.New,
			referencedataguard.New,
			edipartnerrepository.New,
			edimappingprofilerepository.New,
			editransferrepository.New,
			ediconnectionrepository.New,
			edicommunicationprofilerepository.New,
			edishipmentlinkrepository.New,
			editransferchangerepository.New,
			editenderrecipientrepository.New,
			editenderchangerepository.New,
			edidocumenttyperepository.New,
			editransactionsetrepository.New,
			edicontrolnumberrepository.New,
			edimessagerepository.New,
			ediladingrepository.NewProjects,
			ediladingrepository.NewFlows,
			ediladingrepository.NewReplays,
			edilading.NewRuntime,
			edilading.NewFlows,
			edilading.NewProjects,
			edilading.NewReplayer,
			edijobscontract.NewTimerScheduler,
			fx.Annotate(
				edijobscontract.NewShipmentObserver,
				fx.ResultTags(`group:"shipment_event_observers"`),
			),
			ediinboundfilerepository.New,
			edicarrierinvoicerepository.New,
			edishipmenttenderrepository.New,
			edivalidation.NewValidator,
			edialert.New,
			ediprofilesecret.New,
			editenderrecipient.New,
			editendershipment.New,
			editendermapping.New,
			edipayloadbuild.New,
			edimessageview.New,
			edidelivery.New,
			edisetup.New,
			edioutbound.New,
			ediservicefailure.New,
			edioverview.New,
			ediinboundapply.New,
			editransferchangeservice.New,
			editenderservice.New,
			editenderchangeservice.New,
			editransferservice.New,
			func(s *edioutbound.Service) ediports.EDIDocumentService { return s },
			func(s *ediservicefailure.Service) ediports.ServiceFailure214Service { return s },
			func(s *edipayloadbuild.Service) ediports.ServiceFailurePayloadBuilder { return s },
			ediconnectiondraftrepository.New,
			ediconnectiondraft.New,
			func(s *ediconnectiondraft.Service) ediconnectiondraftports.EDIConnectionDraftService { return s },
		),
		editransport.Module,
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
