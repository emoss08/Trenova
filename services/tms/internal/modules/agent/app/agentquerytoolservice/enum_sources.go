package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/app/formulaassistantservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
)

const (
	paramCategory = "category"
	paramEntity   = "entity"
	paramField    = "field"
)

// The sources the read tools' enums are drawn from, where the list is not
// already a filter's own values.
var (
	accountingDriftSeverities = agenttoolschema.Source(
		"accountingDrift.severity",
		[]string{"Critical", "Warning", "Watch"},
	)
	listShipmentStatuses  = agenttoolschema.Source("shipment.status", shipmentStatuses)
	weatherSeverityLevels = agenttoolschema.Source("weather.severity", weatherSeverities)
	watchtowerKindSource  = agenttoolschema.Source(
		"watchtower.sourceKind",
		watchtowerKindValues(),
	)
	watchtowerSeveritySource = agenttoolschema.Source("watchtower.severity", []string{
		watchtower.SeverityCritical.String(),
		watchtower.SeverityWarning.String(),
		watchtower.SeverityInfo.String(),
	})
	guidePageActions     = agenttoolschema.Source("guide.pageAction", []string{createAction})
	insightCategories    = agenttoolschema.Source("insight.category", insightCategoryNames())
	insightSeverities    = agenttoolschema.Source("insight.severity", insightSeverityNames())
	insightStatuses      = agenttoolschema.Source("insight.status", insightStatusNames())
	importRequiredFields = agenttoolschema.Source(
		"importDraft.requiredField",
		pagedraft.AllRequiredFields(),
	)
	ptoStatuses = agenttoolschema.Source("worker.ptoStatus", []worker.PTOStatus{
		worker.PTOStatusRequested,
		worker.PTOStatusApproved,
		worker.PTOStatusRejected,
		worker.PTOStatusCancelled,
	})
	homeWidgetCategories = agenttoolschema.Source("homeLayout.category", []string{
		homelayout.CategoryWork,
		homelayout.CategoryPulse,
		homelayout.CategoryInsight,
		homelayout.CategoryOrientation,
		homelayout.CategoryComms,
	})
	shopStrategies = agenttoolschema.Source("carrierShop.strategy", []ratequoteports.ShopStrategy{
		ratequoteports.ShopStrategyLeastCost,
		ratequoteports.ShopStrategyBestMargin,
		ratequoteports.ShopStrategyGuideRank,
		ratequoteports.ShopStrategyFastestAccept,
	})
	formulaKinds = agenttoolschema.Source("formula.kind", formulaassistantservice.AllKinds())
	ratingSides  = agenttoolschema.Source(
		"rating.side",
		[]string{"Customer", "Carrier"},
	)
	reportRunFormats = agenttoolschema.Source("report.format", report.AllFormats())
	memoryKinds      = agenttoolschema.Source("agent.memoryKind", []agent.MemoryKind{
		agent.MemoryKindInstruction,
		agent.MemoryKindFact,
		agent.MemoryKindCorrection,
		agent.MemoryKindProcedure,
	})
	memoryFilterSubjectTypes = agenttoolschema.Source(
		"agent.memoryFilterSubjectType",
		[]agent.MemorySubjectType{
			agent.MemorySubjectCustomer,
			agent.MemorySubjectLocation,
			agent.MemorySubjectWorker,
			agent.MemorySubjectCarrier,
		},
	)
	listSortDirections   = agenttoolschema.Source("list.sortDirection", []string{"asc", "desc"})
	shipmentDetailLevels = agenttoolschema.Source("shipment.detail", []shipmentDetail{
		shipmentDetailSummary,
		shipmentDetailFull,
	})
)
