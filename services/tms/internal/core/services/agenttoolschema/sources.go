package agenttoolschema

import (
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

// The sources more than one tool package spends, or that name a domain type
// the tools share. A source only one tool uses lives beside that tool.
var (
	StopTypes         = Source("shipment.stopType", shipment.StopTypeValues())
	StopScheduleTypes = Source("shipment.stopScheduleType", shipment.StopScheduleTypeValues())
	StopActualActions = Source(
		"shipmentMove.stopActualAction",
		repositories.StopActualActionValues(),
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
	SubjectTypes = Source("agent.subjectType", agent.AllSubjectTypes())

	AccountingSystems = Source(
		"integration.accountingSystem",
		[]integration.Type{integration.TypeQuickBooksOnline},
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
