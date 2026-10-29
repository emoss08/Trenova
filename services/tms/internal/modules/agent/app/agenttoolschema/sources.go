package agenttoolschema

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingbackfill"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringremittanceports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
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
		shipmentmoveports.StopActualActionValues(),
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
	CommentPriorities   = Source("shipment.commentPriority", shipment.CommentPriorityValues())
	CarrierRateMethods  = Source("shipment.carrierRateMethod", shipment.CarrierRateMethodValues())
	AccessorialMethods  = Source("accessorialCharge.method", accessorialcharge.MethodValues())
	TenderChannels      = Source("tender.channel", tender.ChannelValues())
	SpotTenderModes     = Source("tender.spotMode", tender.SpotModeValues())
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
		accountingprovider.AccountingSystems(),
	)
	FactoringFundingOutcomes = Source(
		"factoring.fundingOutcome",
		factoringledger.AllFundingOutcomes(),
	)
	FactoringRemittanceLineKinds = Source(
		"factoring.remittanceLineKind",
		factoringremittance.AllRemittanceLineKinds(),
	)
	FactoringRemittanceLineActions = Source(
		"factoring.remittanceLineAction",
		factoringremittanceports.AllRemittanceLineActions(),
	)
	FactoringOverrides = Source(
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
		accountingbackfill.BackfillObjectTypes(),
	)
	InboundChangeStatuses = Source(
		"accountingSync.inboundChangeStatus",
		accountinginbound.AllInboundChangeStatuses(),
	)
	InboundChangeKinds = Source(
		"accountingSync.inboundChangeKind",
		accountinginbound.AllInboundChangeKinds(),
	)
	InboundChangeReasons = Source(
		"accountingSync.inboundChangeReason",
		accountinginbound.AllInboundChangeReasons(),
	)
	DriftStatuses    = Source("accountingSync.driftStatus", accountingdrift.AllDriftStatuses())
	DriftKinds       = Source("accountingSync.driftKind", accountingdrift.AllDriftKinds())
	DriftDirections  = Source("accountingSync.driftDirection", accountingdrift.AllDriftDirections())
	DriftObjectTypes = Source(
		"accountingSync.driftObjectType",
		append(
			[]accountingsync.SyncObjectType{
				accountingsync.SyncObjectCustomer,
				accountingsync.DriftObjectGLAccount,
			},
			accountingdrift.DriftObjectTypes()...,
		),
	)
)

func ErrAccountingSystem() error {
	return fmt.Errorf("system must be one of %s", strings.Join(AccountingSystems.Names(), ", "))
}
