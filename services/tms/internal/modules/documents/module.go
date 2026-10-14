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
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/inboundreplyrepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/retrievalsourcerepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/triageinsightrepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/triagemessagerepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/triagerulerepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/triagetemplaterepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/triageworkrepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/redis/triagecache"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturereleaseservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/conversationfileservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentintelligenceservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentmailservice"
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
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundreplyservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagebriefingservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageconflictservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagecorrectionservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagepeekservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageruleservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/extractionevaljobs/extractionevalcontract"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/triagewakejobs/triagewakecontract"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentparsingruleports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Decorate(documentrepository.Observe),
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
			inboundmessagerepository.NewQueueRepository,
			inboundmessagerepository.NewMailboxStateRepository,
			inboundmessagerepository.NewActivityRepository,
			inboundmessagerepository.NewViewRepository,
			inboundmessagerepository.NewTargetRepository,
			inboundmessagerepository.NewSettingsRepository,
			fx.Annotate(
				inboundreplyrepository.New,
				fx.As(fx.Self()),
				fx.As(new(inboundmessageports.InboundReplyReader)),
			),
			triagetemplaterepository.New,
			triageworkrepository.New,
			fx.Annotate(
				inboundmessagerepository.NewRecordFinder,
				fx.As(new(inboundmessageports.InboundShipmentFinder)),
				fx.As(new(inboundmessageports.InboundPartyFinder)),
				fx.As(new(inboundmessageports.InboundDocumentTypeFinder)),
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
			conversationfileservice.New,
			func(s *documentservice.Service) documentports.DocumentStore { return s },
			fx.Annotate(
				documentmailservice.New,
				fx.As(new(documentports.DocumentMailAttachments)),
			),
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
			inboundmessageservice.NewNotifier,
			func(n *inboundmessageservice.Notifier) inboundmessageports.TriageMessageNotifier { return n },
			inboundmessageservice.NewActivityRecorder,
			func(r *inboundmessageservice.ActivityRecorder) inboundmessageports.TriageActivityRecorder {
				return r
			},
			inboundmessageservice.NewEventSink,
			func(e *inboundmessageservice.EventSink) inboundmessageports.TriageEventSink { return e },
			triagewakecontract.NewScheduler,
			func(s *triagewakecontract.Scheduler) inboundmessageports.TriageWakeScheduler { return s },
			triagenova.New,
			inboundreplyservice.New,
			fx.Annotate(
				inboundreplyservice.NewEmailObserver,
				fx.ResultTags(emailports.EmailMessageObserverGroup),
			),
			triageworkservice.New,
			func(s *triageworkservice.Service) inboundmessageports.TriageFiler { return s },
			func(s *triageworkservice.Service) inboundmessageports.TriageDrafter { return s },
			triagepeekservice.New,
			triagerulerepository.New,
			triagemessagerepository.New,
			triagemessagerepository.NewSenderHintRepository,
			triageinsightrepository.New,
			triagecache.New,
			triageruleservice.New,
			func(s *triageruleservice.Service) inboundmessageports.TriageRuleTeacher { return s },
			func(s *triageruleservice.Service) triageports.TriageRuleStats { return s },
			triageruleservice.NewEvaluator,
			func(e *triageruleservice.Evaluator) inboundmessageports.TriageRuleEvaluator { return e },
			triageconflictservice.New,
			func(s *triageconflictservice.Service) triageports.TriageConflictChecker { return s },
			triagecorrectionservice.New,
			triagecorrectionservice.NewHintProvider,
			func(p *triagecorrectionservice.HintProvider) inboundmessageports.SenderHintProvider {
				return p
			},
			triagebriefingservice.New,
			triagebriefingservice.NewInvalidator,
			func(i *triagebriefingservice.Invalidator) inboundmessageports.TriageBriefingInvalidator {
				return i
			},
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
