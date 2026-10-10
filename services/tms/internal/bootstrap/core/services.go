//revive:disable-next-line:var-naming
package core

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/accountingconnectionservice"
	"github.com/emoss08/trenova/internal/core/services/accountinginboundservice"
	"github.com/emoss08/trenova/internal/core/services/accountingmappingservice"
	"github.com/emoss08/trenova/internal/core/services/accountsreceivableservice"
	"github.com/emoss08/trenova/internal/core/services/approvalwindow"
	"github.com/emoss08/trenova/internal/core/services/benefitsservice"
	"github.com/emoss08/trenova/internal/core/services/billingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/capturereleaseservice"
	"github.com/emoss08/trenova/internal/core/services/costingservice"
	"github.com/emoss08/trenova/internal/core/services/customerupdateservice"
	"github.com/emoss08/trenova/internal/core/services/detentionbillingservice"
	"github.com/emoss08/trenova/internal/core/services/detentionpolicyservice"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/internal/core/services/driverportalservice"
	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/emoss08/trenova/internal/core/services/editioninfo"
	"github.com/emoss08/trenova/internal/core/services/globalsearchservice"
	"github.com/emoss08/trenova/internal/core/services/invoiceshareservice"
	"github.com/emoss08/trenova/internal/core/services/orgholidayservice"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/internal/core/services/performancereviewservice"
	"github.com/emoss08/trenova/internal/core/services/ptopolicyservice"
	"github.com/emoss08/trenova/internal/core/services/realtimeservice"
	"github.com/emoss08/trenova/internal/core/services/referencedataguard"
	"github.com/emoss08/trenova/internal/core/services/routingguideservice"
	"github.com/emoss08/trenova/internal/core/services/schedulingservice"
	"github.com/emoss08/trenova/internal/core/services/selfserviceservice"
	"github.com/emoss08/trenova/internal/core/services/servicefailurereasoncodeservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentboardservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentcapacityservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommentservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentcommercial"
	"github.com/emoss08/trenova/internal/core/services/shipmentholdservice"
	"github.com/emoss08/trenova/internal/core/services/shipmentinvalidation"
	"github.com/emoss08/trenova/internal/core/services/tableexportservice"
	"github.com/emoss08/trenova/internal/core/services/thumbnailservice"
	"github.com/emoss08/trenova/internal/core/services/timesheetservice"
	"github.com/emoss08/trenova/internal/core/services/versionservice"
	"github.com/emoss08/trenova/internal/core/services/workerchecklistservice"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/internal/core/services/workerdqfservice"
	"github.com/emoss08/trenova/internal/core/services/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/core/services/workeremploymentservice"
	"github.com/emoss08/trenova/internal/core/services/workerinjuryservice"
	"github.com/emoss08/trenova/internal/core/services/workerleaveservice"
	"github.com/emoss08/trenova/internal/core/services/workersafetyservice"
	"github.com/emoss08/trenova/internal/core/services/workertrainingservice"
	"github.com/emoss08/trenova/internal/core/services/workflowstarter"
	"github.com/emoss08/trenova/internal/infrastructure/inboundmail"
	"github.com/emoss08/trenova/internal/modules/accounting/app/ledgersync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/operations/app/internaledistatussync"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderderivation"
	"github.com/emoss08/trenova/internal/modules/operations/app/servicefailureservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/carrierassignmentservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeroverviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tablechangealertservice"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/seqgen"
	"go.uber.org/fx"
)

var ServiceModule = fx.Module("api-services", ledgersync.Module, fx.Provide(
	tableexportservice.New,
	workflowstarter.New,
	workflowstarter.NewSignalStarter,
	editioninfo.SelfHosted,
	shipmentcommercial.New,
	realtimeservice.New,
	shipmentinvalidation.New,
	globalsearchservice.New,
	thumbnailservice.NewGenerator,
	newDetectorRegistry,
	newBriefingSources,
	approvalwindow.New,
	dispatchcandidateservice.New,
	versionservice.New,
	capturereleaseservice.New,
	servicefailurereasoncodeservice.New,
	shipmentcommentservice.New,
	fx.Annotate(
		internaledistatussync.New,
		fx.As(new(shipmenteventports.ShipmentEventObserver)),
		fx.ResultTags(`group:"shipment_event_observers"`),
	),
	fx.Annotate(
		orderderivation.New,
		fx.As(new(shipmenteventports.ShipmentEventObserver)),
		fx.ResultTags(`group:"shipment_event_observers"`),
	),
	shipmentholdservice.New,
	ediinboundservice.New,
	inboundmail.New,
	fx.Annotate(
		func(s *ediservice.Service) shipmentports.ShipmentMutationObserver { return s },
		fx.ResultTags(`group:"shipment_mutation_observers"`),
	),
	drivernotificationservice.New,
	driverportalservice.New,
	fx.Annotate(
		func(s *driversettlementservice.Service) shipmentports.ShipmentMutationObserver { return s },
		fx.ResultTags(`group:"shipment_mutation_observers"`),
	),
	fx.Annotate(
		func(s *driversettlementservice.Service) shipmentports.MoveStatusObserver { return s },
		fx.ResultTags(`group:"move_status_observers"`),
	),
	fx.Annotate(
		func(s *carriersettlementservice.Service) shipmentports.ShipmentMutationObserver { return s },
		fx.ResultTags(`group:"shipment_mutation_observers"`),
	),
	fx.Annotate(
		func(s *carriersettlementservice.Service) shipmentports.MoveStatusObserver { return s },
		fx.ResultTags(`group:"move_status_observers"`),
	),
	routingguideservice.NewValidator,
	routingguideservice.New,
	accountsreceivableservice.New,
	detentionpolicyservice.New,
	referencedataguard.New,
	accountingconnectionservice.New,
	func(s *accountingconnectionservice.Service) accountingsyncports.AccountingConnectionService { return s },
	accountingmappingservice.New,
	func(s *accountingmappingservice.Service) accountingsyncports.AccountingMappingService { return s },
	accountinginboundservice.New,
	func(s *accountinginboundservice.Service) accountingsyncports.AccountingInboundService { return s },
	detentionbillingservice.New,
	fx.Annotate(
		func(g seqgen.Generator) invoiceadjustmentports.InvoiceAdjustGenerator { return g },
	),
	invoiceshareservice.New,
	func(s *invoiceshareservice.Service) invoiceports.InvoiceShareService { return s },
	costingservice.New,
	shipmentboardservice.New,
	customerupdateservice.New,
	shipmentcapacityservice.New,
	shipmentcapacityservice.NewCapacityReader,
	shipmentcapacityservice.NewCoverageSuggester,
	shipmentcapacityservice.NewTenderer,
	func(s *shipmentboardservice.Service) shipmentports.ShipmentBoardCapabilitiesReader { return s },
	func(s *shipmentboardservice.Service) shipmentports.ShipmentStageSummaryReader { return s },
	func(s *shipmentboardservice.Service) shipmentports.ShipmentBoardGroupSummarizer { return s },
	func(s *shipmentboardservice.Service) shipmentports.ShipmentQuickFilterCounter { return s },
	billingcontrolpolicyservice.New,
	ptopolicyservice.New,
	workercredentialservice.New,
	workertrainingservice.New,
	workerdqfservice.New,
	workerdrugalcoholservice.New,
	workerinjuryservice.New,
	workerleaveservice.New,
	orgstructureservice.New,
	benefitsservice.New,
	schedulingservice.New,
	selfserviceservice.New,
	selfserviceservice.NewDocumentChecksums,
	timesheetservice.New,
	workersafetyservice.New,
	performancereviewservice.New,
	func(s *workertrainingservice.Service) workerports.TrainingAssigner { return s },
	workerchecklistservice.New,
	func(s *workercredentialservice.Service) workeroverviewservice.CredentialSummarizer { return s },
	func(s *workertrainingservice.Service) workeroverviewservice.TrainingSummarizer { return s },
	func(s *workersafetyservice.Service) workeroverviewservice.SafetyScorer { return s },
	func(s *workercredentialservice.Service) workerdqfservice.CredentialSummarizer { return s },
	func(s *workerdrugalcoholservice.Service) workerdqfservice.DrugAlcoholStander { return s },
	func(s *workerchecklistservice.Service) workeroverviewservice.ChecklistLister { return s },
	func(s *performancereviewservice.Service) workeroverviewservice.ReviewLister { return s },
	func(s *workerleaveservice.Service) workeroverviewservice.LeaveCounter { return s },
	func(s *workerleaveservice.Service) driverportalservice.LeaveReader { return s },
	orgholidayservice.New,
	func(s *workerchecklistservice.Service) workerports.ChecklistSpawner { return s },
	workeremploymentservice.New,
	func(s *workeremploymentservice.Service) workerports.EmploymentEventRecorder { return s },
	fx.Annotate(
		func(s *fuelsurchargeservice.Service) formulatemplatetypes.ContextVariableProvider { return s },
		fx.ResultTags(`group:"formula_context_variables"`),
	),
), fx.Invoke(
	// The table-change-alert consumer reads the tca:events stream GTC fills
	// and runs for the API's lifetime. Its loops outlive OnStart, so they get
	// a context of their own; Stop cancels it.
	func(lc fx.Lifecycle, consumer *tablechangealertservice.Consumer) {
		lc.Append(fx.Hook{
			OnStart: func(context.Context) error {
				consumer.Start(context.Background())
				return nil
			},
			OnStop: func(context.Context) error {
				consumer.Stop()
				return nil
			},
		})
	},
	func(setter servicefailureservice.EDIServiceSetter, service ediports.EDIService) {
		setter.SetEDIService(service)
	},
	func(s *tenderservice.Service, assigner tenderports.CarrierMoveAssigner) {
		s.SetCarrierMoveAssigner(assigner)
	},
	// Carrier intelligence writes carrier records back through the carrier
	// service, so the gate and lifecycle observer are attached after every
	// consumer is built instead of through their constructors.
	func(
		tenders *tenderservice.Service,
		assignments *carrierassignmentservice.Service,
		gate carrierintelports.CarrierIntelGate,
		observer carrierintelports.CarrierLifecycleObserver,
	) {
		tenders.SetIntelGate(gate)
		tenders.SetLifecycleObserver(observer)
		assignments.SetIntelGate(gate)
		assignments.SetLifecycleObserver(observer)
	},
	// Injecting the recorder through FX would close the constructor cycle
	// workerservice -> EmploymentEventRecorder -> workeremploymentservice ->
	// workerservice, so it arrives after both services are built.
	func(s *workerservice.Service, recorder workerports.EmploymentEventRecorder) {
		s.SetEmploymentRecorder(recorder)
	},
	// The portal service reaches back into the employment service through the
	// safety service to record a termination, so injecting the revoker through
	// FX would close that cycle; it arrives after both are built.
	func(s *workeremploymentservice.Service, portal *driverportalservice.Service) {
		s.SetPortalRevoker(portal)
	},
	// Providing services.EDITenderChannel would close the constructor cycle
	// ediservice -> ShipmentService -> TenderGuard -> tenderservice ->
	// EDITenderChannel, so the channel arrives after both services are built.
	func(s *tenderservice.Service, ediSvc *ediservice.Service) {
		s.SetEDITenderChannel(ediSvc)
	},
	fx.Annotate(
		func(
			setter shipmentservice.MutationObserverSetter,
			observers []shipmentports.ShipmentMutationObserver,
		) {
			setter.SetShipmentMutationObservers(observers)
		},
		fx.ParamTags(``, `group:"shipment_mutation_observers"`),
	),
))
