package operations

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/assignmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/carrierassignmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/checkcallrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/customertrackingrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/detentionrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/dispatchconsolerepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/dispatchcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/documentshipmentdraftrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/orderrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/recurringshipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/routingguiderepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/servicefailurereasoncoderepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/servicefailurerepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentadditionalchargerepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentboardrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentcommentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentcommodityrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmenteventrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentholdrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentmovejurisdictionmilerepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentmoverepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentsuggestionrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmenttrackingrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/stopetarepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/tenderrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/trackingeventrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/triagelookuprepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/triageoverlaprepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/weatheralertrepository"
	"github.com/emoss08/trenova/internal/modules/operations/app/assignmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/checkcallservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/customerstatustrigger"
	"github.com/emoss08/trenova/internal/modules/operations/app/customertrackingservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/customerupdateservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionpolicyservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcontrolservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/etatrigger"
	"github.com/emoss08/trenova/internal/modules/operations/app/globalsearchservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderderivation"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/recurringshipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/routingguideservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/servicefailurereasoncodeservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/servicefailureservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentboardservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcapacityservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommentservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentetaservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentholdservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentinvalidation"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentmoveservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentquickfilterservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentratingreviewservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentsuggestionservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentwatchlistservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/tenderservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/trackingeventservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/triagedeliveryrelay"
	"github.com/emoss08/trenova/internal/modules/operations/app/triageshipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/weatheralertservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/customertrackingjobs/customertrackingcontract"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/etajobs/etacontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/weatheralertports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/driverportalports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"go.uber.org/fx"
)

//nolint:funlen // the module's providers are its part of the application graph
func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			dispatchconsolerepository.New,
			detentionpolicyservice.NewValidator,
			shipmentcommercial.New,
			shipmentinvalidation.New,
			globalsearchservice.New,
			dispatchcandidateservice.New,
			servicefailurereasoncodeservice.New,
			shipmentcommentservice.New,
			shipmentholdservice.New,
			shipmentratingreviewservice.New,
			func(s *shipmentratingreviewservice.Service) shipmentports.RatingReviewService { return s },
			routingguideservice.NewValidator,
			routingguideservice.New,
			detentionpolicyservice.New,
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
			shipmentrepository.NewInsightSource,
			weatheralertrepository.New,
			dispatchcontrolrepository.New,
			documentshipmentdraftrepository.New,
			triageoverlaprepository.New,
			triagelookuprepository.New,
			assignmentrepository.New,
			orderrepository.New,
			servicefailurereasoncoderepository.New,
			servicefailurerepository.New,
			shipmentadditionalchargerepository.New,
			shipmentcommodityrepository.New,
			shipmentcommentrepository.New,
			shipmenteventrepository.New,
			shipmentholdrepository.New,
			shipmentmoverepository.New,
			shipmentrepository.New,
			shipmentrepository.NewRatingReviewRepository,
			shipmentrepository.NewSearchCardReader,
			shipmentrepository.NewOrderReferenceMatcher,
			shipmentrepository.NewActiveHoldReader,
			shipmentboardrepository.New,
			shipmentboardrepository.NewBoardRepository,
			shipmentboardrepository.NewWatchlistRepository,
			shipmentboardrepository.NewBriefingRepository,
			shipmentboardrepository.NewCarrierPerformanceRepository,
			shipmentsuggestionrepository.New,
			shipmenttrackingrepository.New,
			recurringshipmentrepository.New,
			carrierassignmentrepository.New,
			routingguiderepository.New,
			tenderrepository.New,
			trackingeventrepository.New,
			checkcallrepository.New,
			checkcallrepository.NewTrackingLinks,
			customertrackingrepository.NewProfiles,
			customertrackingrepository.NewLinks,
			customertrackingrepository.NewNotifications,
			customertrackingrepository.NewFacts,
			stopetarepository.New,
			shipmentmovejurisdictionmilerepository.New,
			detentionrepository.NewPolicyRepository,
			detentionrepository.NewOccurrenceRepository,
			detentionrepository.NewEvidenceRepository,
			detentionrepository.NewNoticeRepository,
			detentionrepository.NewAnalyticsRepository,
			orderservice.NewValidator,
			shipmentservice.NewValidator,
			recurringshipmentservice.NewValidator,
			dispatchcontrolservice.NewValidator,
			shipmentstate.NewCoordinator,
			assignmentservice.New,
			fx.Annotate(
				dispatchconsoleservice.New,
				fx.As(new(agentports.DispatchConsoleService)),
			),
			orderservice.New,
			shipmenteventservice.New,
			triagedeliveryrelay.New,
			triageshipmentservice.New,
			func(s *triageshipmentservice.Service) inboundmessageports.TriageShipmentPeeker { return s },
			func(s *triageshipmentservice.Service) inboundmessageports.TriageStopTimesReader { return s },
			func(s *triageshipmentservice.Service) inboundmessageports.TriageLoadCreator { return s },
			func(s *triageshipmentservice.Service) inboundmessageports.TriageExceptionLogger { return s },
			func(s *triageshipmentservice.Service) inboundmessageports.TriageDetentionSettler { return s },
			func(s *triageshipmentservice.Service) inboundmessageports.TriageDetentionEvidenceSource {
				return s
			},
			fx.Annotate(
				func(r *triagedeliveryrelay.Relay) shipmenteventports.ShipmentEventObserver { return r },
				fx.ResultTags(`group:"shipment_event_observers"`),
			),
			fx.Annotate(
				func(r *triagedeliveryrelay.Relay) shipmentports.MoveStatusObserver { return r },
				fx.ResultTags(`group:"move_status_observers"`),
			),
			fx.Annotate(
				orderderivation.New,
				fx.As(new(orderports.OrderDerivationService)),
			),
			fx.Annotate(
				servicefailureservice.New,
				fx.As(new(servicefailureservice.EDIServiceSetter)),
				fx.As(new(servicefailureports.ServiceFailureService)),
				fx.As(new(servicefailureports.ServiceFailureEvaluator)),
			),
			shipmentmoveservice.New,
			fx.Annotate(
				checkcallservice.New,
				fx.As(new(checkcallports.CheckCallService)),
			),
			fx.Annotate(
				trackingeventservice.New,
				fx.As(new(trackingeventports.TrackingEventService)),
				fx.As(new(telematicsports.StopVisitRecorder)),
				fx.As(new(driverportalports.StopReporter)),
			),
			fx.Annotate(
				shipmentservice.New,
				fx.As(new(shipmentports.ShipmentService)),
				fx.As(new(shipmentservice.MutationObserverSetter)),
			),
			tenderservice.New,
			func(s *tenderservice.Service) tenderports.TenderGuard { return s },
			func(s *tenderservice.Service) tenderports.TenderResponseRecorder { return s },
			func(s *tenderservice.Service) tenderports.TenderLifecycle { return s },
			fx.Annotate(
				weatheralertservice.New,
				fx.As(new(weatheralertports.WeatherAlertService)),
			),
			recurringshipmentservice.New,
			detentionservice.New,
			shipmentquickfilterservice.New,
			shipmentsuggestionservice.New,
			shipmentsuggestionservice.NewReader,
			shipmentsuggestionservice.NewDecider,
			shipmentsuggestionservice.NewCandidates,
			shipmentetaservice.New,
			shipmentetaservice.NewTrackingReader,
			shipmentetaservice.NewRefresher,
			etacontract.NewRequester,
			etatrigger.New,
			fx.Annotate(
				func(t *etatrigger.Trigger) shipmenteventports.ShipmentEventObserver { return t },
				fx.ResultTags(`group:"shipment_event_observers"`),
			),
			fx.Annotate(
				func(t *etatrigger.Trigger) checkcallports.CheckCallObserver { return t },
				fx.ResultTags(`group:"check_call_observers"`),
			),
			customertrackingservice.New,
			func(s *customertrackingservice.Service) customertrackingports.CustomerTrackingService {
				return s
			},
			func(s *customertrackingservice.Service) customertrackingports.StatusNotifier { return s },
			customertrackingcontract.NewRequester,
			customerstatustrigger.New,
			fx.Annotate(
				func(t *customerstatustrigger.Trigger) shipmenteventports.ShipmentEventObserver {
					return t
				},
				fx.ResultTags(`group:"shipment_event_observers"`),
			),
			fx.Annotate(
				func(t *customerstatustrigger.Trigger) stopetaports.Observer { return t },
				fx.ResultTags(`group:"stop_eta_observers"`),
			),
			shipmentwatchlistservice.New,
			dispatchcontrolservice.New,
			func(s *detentionservice.Service) detentionports.DetentionNoticeService { return s },
			detentionservice.NewContextBuilder,
			fx.Annotate(
				func(b *detentionservice.ContextBuilder) documenttemplateports.ContextBuilder { return b },
				fx.ResultTags(documenttemplateports.ContextBuilderGroup),
			),
			fx.Annotate(
				detentionservice.NewPDFContextBuilder,
				fx.ResultTags(documenttemplateports.ContextBuilderGroup),
			),
		),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
