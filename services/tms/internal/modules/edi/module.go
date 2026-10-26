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
	"github.com/emoss08/trenova/internal/modules/edi/app/ediconnectiondraft"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransport"
	"github.com/emoss08/trenova/internal/modules/edi/app/referencedataguard"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			ediinboundservice.New,
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
			ediinboundfilerepository.New,
			edicarrierinvoicerepository.New,
			edishipmenttenderrepository.New,
			ediservice.NewValidator,
			ediservice.New,
			func(s *ediservice.Service) ediports.EDIService { return s },
			ediconnectiondraftrepository.New,
			ediconnectiondraft.New,
			func(s *ediconnectiondraft.Service) ediports.EDIConnectionDraftService { return s },
		),
		editransport.Module,
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
