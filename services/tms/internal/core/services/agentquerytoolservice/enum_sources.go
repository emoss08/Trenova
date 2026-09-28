package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
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
	accessorialAppliesTo  = agenttoolschema.Source(
		"accessorialCharge.appliesTo",
		[]string{"Pickup", "Delivery"},
	)
	detentionUrgencies = agenttoolschema.Source("detention.urgency", []string{
		"Lost", "NoticeOverdue", "NoticeDueSoon", detentionservice.UrgencyAwaitingApproval,
	})
	watchtowerKindSource = agenttoolschema.Source(
		"watchtower.sourceKind",
		watchtowerKindValues(),
	)
	watchtowerSeveritySource = agenttoolschema.Source("watchtower.severity", []string{
		watchtower.SeverityCritical.String(),
		watchtower.SeverityWarning.String(),
		watchtower.SeverityInfo.String(),
	})
	briefingRoles       = agenttoolschema.Source("briefing.role", briefingRoleValues())
	guidePageActions    = agenttoolschema.Source("guide.pageAction", []string{createAction})
	insightCategories   = agenttoolschema.Source("insight.category", insightCategoryNames())
	insightSeverities   = agenttoolschema.Source("insight.severity", insightSeverityNames())
	insightStatuses     = agenttoolschema.Source("insight.status", insightStatusNames())
	inboundListStatuses = agenttoolschema.Source(
		"inboundMessage.listStatus",
		inboundStatusNames(),
	)
	inboundClassifications = agenttoolschema.Source(
		"inboundMessage.classification",
		inboundmessage.AllClassifications(),
	)
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
	ptoTypes = agenttoolschema.Source("worker.ptoType", []worker.PTOType{
		worker.PTOTypePersonal,
		worker.PTOTypeVacation,
		worker.PTOTypeSick,
		worker.PTOTypeHoliday,
		worker.PTOTypeBereavement,
		worker.PTOTypeMaternity,
		worker.PTOTypePaternity,
	})
	homeWidgetCategories = agenttoolschema.Source("homeLayout.category", []string{
		homelayout.CategoryWork,
		homelayout.CategoryPulse,
		homelayout.CategoryInsight,
		homelayout.CategoryOrientation,
		homelayout.CategoryComms,
	})
	shopStrategies = agenttoolschema.Source("carrierShop.strategy", []serviceports.ShopStrategy{
		serviceports.ShopStrategyLeastCost,
		serviceports.ShopStrategyBestMargin,
		serviceports.ShopStrategyGuideRank,
		serviceports.ShopStrategyFastestAccept,
	})
	formulaVariableTypes = agenttoolschema.Source(
		"formula.variableType",
		[]string{"Number", "String", "Boolean"},
	)
	ratingSides = agenttoolschema.Source(
		"rating.side",
		[]string{"Customer", "Carrier"},
	)
	reportRunFormats            = agenttoolschema.Source("report.format", report.AllFormats())
	customerPaymentListStatuses = agenttoolschema.Source(
		"customerPayment.listStatus",
		[]customerpayment.Status{customerpayment.StatusPosted, customerpayment.StatusReversed},
	)
	memoryKinds = agenttoolschema.Source("agent.memoryKind", []agent.MemoryKind{
		agent.MemoryKindInstruction,
		agent.MemoryKindFact,
		agent.MemoryKindCorrection,
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
	listSortDirections = agenttoolschema.Source("list.sortDirection", []string{"asc", "desc"})
)
