package agenttoolschema

import (
	"fmt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/productguide"
)

// The sources more than one tool package spends, or that name a domain type
// the tools share. A source only one tool uses lives beside that tool.
var (
	StopTypes         = Source("shipment.stopType", shipment.StopTypeValues())
	StopScheduleTypes = Source("shipment.stopScheduleType", shipment.StopScheduleTypeValues())
	FreightTerms      = Source("shipment.freightTerms", shipment.FreightTermsValues())
	StopActualActions = Source(
		"shipmentMove.stopActualAction",
		shipmentports.StopActualActionValues(),
	)
	SettableMoveStatuses = Source("shipmentMove.settableStatus", []shipment.MoveStatus{
		shipment.MoveStatusInTransit,
		shipment.MoveStatusCompleted,
		shipment.MoveStatusCanceled,
	})
	CommentVisibilities = Source(
		"shipment.commentVisibility",
		shipment.CommentVisibilityValues(),
	)
	CommentPriorities  = Source("shipment.commentPriority", shipment.CommentPriorityValues())
	CarrierRateMethods = Source("shipment.carrierRateMethod", shipment.CarrierRateMethodValues())
	AccessorialMethods = Source("accessorialCharge.method", accessorialcharge.MethodValues())
	TenderChannels     = Source("tender.channel", tender.ChannelValues())
	SpotTenderModes    = Source("tender.spotMode", tender.SpotModeValues())
	TenderResponses    = Source("tender.responseAction", []tender.ResponseAction{
		tender.ResponseActionAccept,
		tender.ResponseActionDecline,
	})
	ExceptionCategories = Source(
		"agent.exceptionCategory",
		agent.AllExceptionCategories(),
	)
	Severities   = Source("agent.severity", agent.SeverityValues())
	PTOTypes     = Source("worker.ptoType", worker.PTOTypeValues())
	SubjectTypes = Source("agent.subjectType", agent.AllSubjectTypes())

	RecordEntities = Derived("guide.entity", productguide.Default.RecordEntities())

	AccountingSystems = Source(
		"integration.accountingSystem",
		accountingsync.AccountingSystems(),
	)
	FactoringQueueVerdicts = Source("factoring.verdict", factoring.QueueVerdicts())
	FactoringNoaStatuses   = Source("factoring.noaStatus", factoring.AllNoaStatuses())
	FactoringOverrides     = Source(
		"invoice.factoringOverride",
		invoice.FactoringOverrideValues(),
	)
	MappingTargetTypes = Source(
		"accountingSync.mappingTargetType",
		accountingsync.AllMappingTargetTypes(),
	)
	SyncStatuses    = Source("accountingSync.syncStatus", accountingsync.AllSyncStatuses())
	SyncObjectTypes = Source(
		"accountingSync.syncObjectType",
		accountingsync.AllSyncObjectTypes(),
	)
	SyncErrorCategories = Source(
		"accountingSync.syncErrorCategory",
		accountingsync.AllSyncErrorCategories(),
	)
	BackfillObjectTypes = Source(
		"accountingSync.backfillObjectType",
		accountingsync.BackfillObjectTypes(),
	)
	InboundChangeStatuses = Source(
		"accountingSync.inboundChangeStatus",
		accountingsync.AllInboundChangeStatuses(),
	)
	InboundChangeKinds = Source(
		"accountingSync.inboundChangeKind",
		accountingsync.AllInboundChangeKinds(),
	)
	InboundChangeReasons = Source(
		"accountingSync.inboundChangeReason",
		accountingsync.AllInboundChangeReasons(),
	)
	DriftStatuses    = Source("accountingSync.driftStatus", accountingsync.AllDriftStatuses())
	DriftKinds       = Source("accountingSync.driftKind", accountingsync.AllDriftKinds())
	DriftDirections  = Source("accountingSync.driftDirection", accountingsync.AllDriftDirections())
	DriftObjectTypes = Source(
		"accountingSync.driftObjectType",
		append(
			[]accountingsync.SyncObjectType{
				accountingsync.SyncObjectCustomer,
				accountingsync.DriftObjectGLAccount,
			},
			accountingsync.DriftObjectTypes()...,
		),
	)
)

func ErrAccountingSystem() error {
	return fmt.Errorf("system must be one of %s", strings.Join(AccountingSystems.Names(), ", "))
}
