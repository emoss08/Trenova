//revive:disable-next-line:var-naming
package core

import (
	"context"

	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/infrastructure/inboundmail"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/workflowstarter"
	"github.com/emoss08/trenova/internal/modules/accounting/app/ledgersync"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/operations/app/internaledistatussync"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderderivation"
	"github.com/emoss08/trenova/internal/modules/operations/app/servicefailureservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/rating/app/carrierassignmentservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/driverportalservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeremploymentservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workeremploymentports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tablechangealertservice"
	"github.com/emoss08/trenova/pkg/seqgen"
	"go.uber.org/fx"
)

var ServiceModule = fx.Module("api-services", ledgersync.Module, fx.Provide(
	workflowstarter.New,
	workflowstarter.NewSignalStarter,
	newDetectorRegistry,
	newBriefingSources,
	crossref.VisibleRecordSources,
	crossref.OrderReferences,
	crossref.TemplateConsumers,
	crossref.RotaAssignments,
	crossref.LocationColumns,
	crossref.MoveColumns,
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
	inboundmail.New,
	fx.Annotate(
		func(s *ediservice.Service) shipmentserviceports.ShipmentMutationObserver { return s },
		fx.ResultTags(`group:"shipment_mutation_observers"`),
	),
	fx.Annotate(
		func(s *driversettlementservice.Service) shipmentserviceports.ShipmentMutationObserver { return s },
		fx.ResultTags(`group:"shipment_mutation_observers"`),
	),
	fx.Annotate(
		func(s *driversettlementservice.Service) shipmentserviceports.MoveStatusObserver { return s },
		fx.ResultTags(`group:"move_status_observers"`),
	),
	fx.Annotate(
		func(s *carriersettlementservice.Service) shipmentserviceports.ShipmentMutationObserver { return s },
		fx.ResultTags(`group:"shipment_mutation_observers"`),
	),
	fx.Annotate(
		func(s *carriersettlementservice.Service) shipmentserviceports.MoveStatusObserver { return s },
		fx.ResultTags(`group:"move_status_observers"`),
	),
	fx.Annotate(
		func(g seqgen.Generator) invoiceadjustmentports.InvoiceAdjustGenerator { return g },
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
	func(s *workerservice.Service, recorder workeremploymentports.EmploymentEventRecorder) {
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
			observers []shipmentserviceports.ShipmentMutationObserver,
		) {
			setter.SetShipmentMutationObservers(observers)
		},
		fx.ParamTags(``, `group:"shipment_mutation_observers"`),
	),
))
