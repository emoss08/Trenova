package watchtowersources

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtoweraccounting"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowerfactoring"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowershipment"
	"go.uber.org/fx"
)

// Module registers every source the watchtower reconciles against. A
// deployment without one of these repositories simply has no source for
// that kind, and the tower carries only what its projections put there.
var Module = fx.Module("watchtower-sources",
	fx.Provide(
		asSource(NewInsightSource),
		asSource(NewProposalSource),
		asSource(NewPlanSource),
		asSource(NewFailedRunSource),
		asSource(NewExceptionSource),
		asSource(NewServiceFailureSource),
		asSource(NewCarrierIntelSource),
		asSource(NewWeatherSource),
		asSource(NewEDIQuarantineSource),
		asSource(NewBillingExceptionSource),
		asSource(NewDetentionSource),
		asSource(NewInboundMessageSource),
		asSource(NewQualityRegressionSource),
		asSource(watchtoweraccounting.NewAccountingSyncSource),
		asSource(watchtowershipment.NewTelematicsStopVisitSource),
		asSource(watchtowershipment.NewNoCheckCallSource),
		asSource(watchtowershipment.NewEtaAtRiskSource),
		asSource(watchtowershipment.NewMissedPickupSource),
		asSource(watchtowershipment.NewMissedDeliverySource),
		asSource(watchtowershipment.NewPodMissingSource),
		asSource(watchtowershipment.NewTenderExhaustedSource),
		asSource(watchtowerfactoring.NewFactoringSource),
	),
	fx.Provide(fx.Annotate(
		watchtowershipment.NewShipmentExceptionObserver,
		fx.ResultTags(`group:"shipment_event_observers"`),
	)),
	fx.Provide(fx.Annotate(
		watchtowershipment.NewNoCheckCallObserver,
		fx.ResultTags(`group:"check_call_observers"`),
	)),
	fx.Provide(fx.Annotate(
		watchtowershipment.NewEtaAtRiskObserver,
		fx.ResultTags(`group:"stop_eta_observers"`),
	)),
)

func asSource(constructor any) any {
	return fx.Annotate(constructor, fx.ResultTags(`group:"watchtower_sources"`))
}
