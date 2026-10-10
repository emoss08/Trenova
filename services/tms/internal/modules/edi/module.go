package edi

import (
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicarrierinvoicerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicommunicationprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediconnectionrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicontrolnumberrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edidocumenttyperepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediinboundfilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimappingprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimessagerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edipartnerdocumentprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edipartnerrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edipartnersettingrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edishipmentlinkrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edisourcecontextrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editemplaterepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editenderchangerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editenderrecipientrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editestcaserepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransactionsetrepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferchangerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/editransferrepository"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransport"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
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
			edisourcecontextrepository.New,
			edipartnersettingrepository.New,
			editemplaterepository.New,
			edipartnerdocumentprofilerepository.New,
			edicontrolnumberrepository.New,
			edimessagerepository.New,
			editestcaserepository.New,
			ediinboundfilerepository.New,
			edicarrierinvoicerepository.New,
			ediservice.NewValidator,
			ediservice.New,
			func(s *ediservice.Service) ediports.EDIService { return s },
		),
		editransport.Module,
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
