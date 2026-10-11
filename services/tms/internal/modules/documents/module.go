package documents

import (
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/capturerepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentaiextractionrepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentcontentrepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentpacketrulerepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentparsingrulerepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentrepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentsearchprojectionrepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documenttemplaterepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentuploadrepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/extractionevalrepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/extractionrolloutrepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/extractionshadowrepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/inboundmessagerepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/retrievalsourcerepository"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturereleaseservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentintelligenceservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentoperationsservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentpacketruleservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentparsingruleservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentsearchprojectionservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documenttemplateservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentuploadservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/extractionevalservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/extractionrolloutservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/extractionshadowservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/extractionevaljobs/extractionevalcontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentparsingruleports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			thumbnailservice.NewGenerator,
			capturereleaseservice.New,
			documentparsingrulerepository.New,
			documentpacketrulerepository.New,
			documenttemplaterepository.NewTemplateRepository,
			documenttemplaterepository.NewVersionRepository,
			documenttemplaterepository.NewAssignmentRepository,
			documenttemplaterepository.NewGeneratedRepository,
			documentrepository.New,
			documentaiextractionrepository.New,
			documentcontentrepository.New,
			documentsearchprojectionrepository.New,
			documentuploadrepository.New,
			extractionevalrepository.NewCases,
			extractionevalrepository.NewRuns,
			extractionevalrepository.NewResults,
			extractionrolloutrepository.NewRollouts,
			extractionrolloutrepository.NewAssignments,
			extractionshadowrepository.NewSettings,
			extractionshadowrepository.NewResults,
			retrievalsourcerepository.New,
			inboundmessagerepository.New,
			inboundmessagerepository.NewMailboxRepository,
			fx.Annotate(
				inboundmessagerepository.NewRecordFinder,
				fx.As(new(inboundmessageports.InboundShipmentFinder)),
				fx.As(new(inboundmessageports.InboundPartyFinder)),
			),
			capturerepository.NewDeviceRepository,
			capturerepository.NewPairingRepository,
			capturerepository.NewProfileRepository,
			capturerepository.NewRequestRepository,
			capturerepository.NewBatchRepository,
			capturerepository.NewPageRepository,
			capturerepository.NewItemRepository,
			capturerepository.NewCoverSheetRepository,
			capturerepository.NewRecordFinder,
			documentservice.NewValidator,
			documentparsingruleservice.NewValidator,
			documentpacketruleservice.NewValidator,
			documenttemplateservice.NewValidator,
			documentintelligenceservice.New,
			documentoperationsservice.New,
			documentsearchprojectionservice.New,
			documentservice.New,
			func(s *documentservice.Service) documentports.InvoiceDocumentService { return s },
			documentuploadservice.New,
			func(s *documentuploadservice.Service) documentuploadports.DocumentUploadService { return s },
			extractionevalservice.New,
			extractionevalservice.AsService,
			extractionevalservice.AsRunner,
			extractionevalservice.AsDriftChecker,
			extractionevalcontract.NewRunStarter,
			func(s *extractionevalcontract.RunStarter) extractionevalports.ExtractionEvalRunStarter { return s },
			extractionrolloutservice.New,
			extractionrolloutservice.AsService,
			extractionrolloutservice.AsRouter,
			extractionrolloutservice.AsGuard,
			extractionrolloutservice.AsRetention,
			extractionshadowservice.NewScorer,
			extractionshadowservice.New,
			extractionshadowservice.AsService,
			extractionshadowservice.AsSampler,
			extractionshadowservice.AsRetention,
			extractionshadowservice.NewRunner,
			extractionshadowservice.AsRunner,
			extractionshadowservice.AsScorer,
			inboundmessageservice.New,
			captureservice.New,
			documentparsingruleservice.New,
			func(s *documentparsingruleservice.Service) documentparsingruleports.DocumentParsingRuleRuntime {
				return s
			},
			func(s *documentparsingruleservice.Service) documentparsingruleports.DocumentParsingRuleAdminService {
				return s
			},
			documentpacketruleservice.New,
			documenttemplateservice.New,
			func(s *documenttemplateservice.Service) documenttemplateports.DocumentTemplateResolver { return s },
			func(s *documentservice.Service) workerdqfservice.PacketSummarizer { return s },
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
