package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/internal/core/services/schedulingservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransferservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/recurringshipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
)

func operationsToolProviders() []any {
	return []any{
		provideUncancelShipmentTool,
		provideTransferShipmentOwnershipTool,
		provideRerateShipmentTool,
		provideRecalculateShipmentDistanceTool,
		provideDuplicateShipmentTool,
		provideUpdateShipmentHoldTool,
		provideSplitMoveTool,
		providePinShipmentCommentTool,
		provideUnpinShipmentCommentTool,
		provideResolveShipmentCommentTool,
		provideEditShipmentCommentTool,
		provideDeleteShipmentCommentTool,
		provideUpdateServiceFailureTool,
		provideReviewServiceFailureTool,
		provideVoidServiceFailureTool,
		provideDisputeDetentionTool,
		provideCreateOrderTool,
		provideUpdateOrderTool,
		provideAttachOrderShipmentsTool,
		provideDetachOrderShipmentTool,
		provideCloseOrderTool,
		provideCancelOrderTool,
		provideAddOrderChargeTool,
		provideUpdateOrderChargeTool,
		provideSetOrderChargeAllocationsTool,
		provideRemoveOrderChargeTool,
		provideCreateRecurringShipmentTool,
		provideUpdateRecurringShipmentTool,
		provideSetRecurringShipmentStatusTool,
		provideGenerateRecurringShipmentTool,
		provideAssignWorkerShiftTool,
		provideEndWorkerShiftAssignmentTool,
		provideSetWorkerAvailabilityPreferenceTool,
		provideProposeShiftSwapTool,
		provideApproveShiftSwapTool,
		provideRejectShiftSwapTool,
		provideWithdrawShiftSwapTool,
		provideVetCarrierTool,
		provideVetCustomerBrokerTool,
		provideSetCarrierMonitoringTool,
		provideMarkCarrierIntelReviewedTool,
		provideApplyCarrierIntelSuggestionsTool,
		provideImportSourcedCarrierTool,
		provideVerifyCarrierEquipmentTool,
		provideReassignBillingChargeTool,
		provideManageBillingTransferRunTool,
	}
}

func provideUncancelShipmentTool(shipments shipmentports.ShipmentService) agentports.AgentTool {
	return newUncancelShipmentTool(shipments)
}

func provideTransferShipmentOwnershipTool(
	shipments shipmentports.ShipmentService,
) agentports.AgentTool {
	return newTransferShipmentOwnershipTool(shipments)
}

func provideRerateShipmentTool(shipments shipmentports.ShipmentService) agentports.AgentTool {
	return newRerateShipmentTool(shipments)
}

func provideRecalculateShipmentDistanceTool(
	shipments shipmentports.ShipmentService,
) agentports.AgentTool {
	return newRecalculateShipmentDistanceTool(shipments)
}

func provideDuplicateShipmentTool(
	shipments shipmentports.ShipmentService,
	locations locationports.LocationRepository,
) agentports.AgentTool {
	return newDuplicateShipmentTool(duplicateShipmentDeps{
		Shipments: shipments,
		Sources:   shipments,
		Locations: locations,
	})
}

func provideUpdateShipmentHoldTool(holds shipmentports.ShipmentHoldService) agentports.AgentTool {
	return newUpdateShipmentHoldTool(holds)
}

func provideSplitMoveTool(moves shipmentports.ShipmentMoveService) agentports.AgentTool {
	return newSplitMoveTool(moves)
}

func providePinShipmentCommentTool(
	comments shipmentports.ShipmentCommentService,
) agentports.AgentTool {
	return newPinShipmentCommentTool(comments)
}

func provideUnpinShipmentCommentTool(
	comments shipmentports.ShipmentCommentService,
) agentports.AgentTool {
	return newUnpinShipmentCommentTool(comments)
}

func provideResolveShipmentCommentTool(
	comments shipmentports.ShipmentCommentService,
) agentports.AgentTool {
	return newResolveShipmentCommentTool(comments)
}

func provideEditShipmentCommentTool(
	comments shipmentports.ShipmentCommentService,
	permissions permissionports.PermissionEngine,
) agentports.AgentTool {
	return newEditShipmentCommentTool(comments, permissions)
}

func provideDeleteShipmentCommentTool(
	comments shipmentports.ShipmentCommentService,
	permissions permissionports.PermissionEngine,
) agentports.AgentTool {
	return newDeleteShipmentCommentTool(comments, permissions)
}

func provideUpdateServiceFailureTool(
	failures servicefailureports.ServiceFailureService,
) agentports.AgentTool {
	return newUpdateServiceFailureTool(failures)
}

func provideReviewServiceFailureTool(
	failures servicefailureports.ServiceFailureService,
) agentports.AgentTool {
	return newReviewServiceFailureTool(failures)
}

func provideVoidServiceFailureTool(
	failures servicefailureports.ServiceFailureService,
) agentports.AgentTool {
	return newVoidServiceFailureTool(failures)
}

func provideDisputeDetentionTool(detentions *detentionservice.Service) agentports.AgentTool {
	return newDisputeDetentionTool(detentions)
}

func provideCreateOrderTool(orders *orderservice.Service) agentports.AgentTool {
	return newCreateOrderTool(orders)
}

func provideUpdateOrderTool(orders *orderservice.Service) agentports.AgentTool {
	return newUpdateOrderTool(orders)
}

func provideAttachOrderShipmentsTool(orders *orderservice.Service) agentports.AgentTool {
	return newAttachOrderShipmentsTool(orders)
}

func provideDetachOrderShipmentTool(orders *orderservice.Service) agentports.AgentTool {
	return newDetachOrderShipmentTool(orders)
}

func provideCloseOrderTool(orders *orderservice.Service) agentports.AgentTool {
	return newCloseOrderTool(orders)
}

func provideCancelOrderTool(orders *orderservice.Service) agentports.AgentTool {
	return newCancelOrderTool(orders)
}

func provideAddOrderChargeTool(orders *orderservice.Service) agentports.AgentTool {
	return newAddOrderChargeTool(orders)
}

func provideUpdateOrderChargeTool(orders *orderservice.Service) agentports.AgentTool {
	return newUpdateOrderChargeTool(orders)
}

func provideSetOrderChargeAllocationsTool(orders *orderservice.Service) agentports.AgentTool {
	return newSetOrderChargeAllocationsTool(orders)
}

func provideRemoveOrderChargeTool(orders *orderservice.Service) agentports.AgentTool {
	return newRemoveOrderChargeTool(orders)
}

func provideCreateRecurringShipmentTool(
	series *recurringshipmentservice.Service,
) agentports.AgentTool {
	return newCreateRecurringShipmentTool(series)
}

func provideUpdateRecurringShipmentTool(
	series *recurringshipmentservice.Service,
) agentports.AgentTool {
	return newUpdateRecurringShipmentTool(series)
}

func provideSetRecurringShipmentStatusTool(
	series *recurringshipmentservice.Service,
) agentports.AgentTool {
	return newSetRecurringShipmentStatusTool(series)
}

func provideGenerateRecurringShipmentTool(
	series *recurringshipmentservice.Service,
) agentports.AgentTool {
	return newGenerateRecurringShipmentTool(series)
}

func provideAssignWorkerShiftTool(
	schedules *schedulingservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agentports.AgentTool {
	return newAssignWorkerShiftTool(schedules, newTeamScope(permissions, teams))
}

func provideEndWorkerShiftAssignmentTool(
	schedules *schedulingservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agentports.AgentTool {
	return newEndWorkerShiftAssignmentTool(
		schedules,
		newTeamScope(permissions, teams),
	)
}

func provideSetWorkerAvailabilityPreferenceTool(
	schedules *schedulingservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agentports.AgentTool {
	return newSetWorkerAvailabilityPreferenceTool(
		schedules,
		newTeamScope(permissions, teams),
	)
}

func provideProposeShiftSwapTool(schedules *schedulingservice.Service) agentports.AgentTool {
	return newProposeShiftSwapTool(schedules)
}

func provideApproveShiftSwapTool(schedules *schedulingservice.Service) agentports.AgentTool {
	return newApproveShiftSwapTool(schedules)
}

func provideRejectShiftSwapTool(schedules *schedulingservice.Service) agentports.AgentTool {
	return newRejectShiftSwapTool(schedules)
}

func provideWithdrawShiftSwapTool(schedules *schedulingservice.Service) agentports.AgentTool {
	return newWithdrawShiftSwapTool(schedules)
}

func provideVetCarrierTool(intel *carrierintelservice.Service) agentports.AgentTool {
	return newVetCarrierTool(intel)
}

func provideVetCustomerBrokerTool(intel *carrierintelservice.Service) agentports.AgentTool {
	return newVetCustomerBrokerTool(intel)
}

func provideSetCarrierMonitoringTool(intel *carrierintelservice.Service) agentports.AgentTool {
	return newSetCarrierMonitoringTool(intel)
}

func provideMarkCarrierIntelReviewedTool(
	intel *carrierintelservice.Service,
) agentports.AgentTool {
	return newMarkCarrierIntelReviewedTool(intel)
}

func provideApplyCarrierIntelSuggestionsTool(
	intel *carrierintelservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentTool {
	return newApplyCarrierIntelSuggestionsTool(intel, permissions)
}

func provideImportSourcedCarrierTool(
	intel *carrierintelservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentTool {
	return newImportSourcedCarrierTool(intel, permissions)
}

func provideVerifyCarrierEquipmentTool(
	intel *carrierintelservice.Service,
) agentports.AgentTool {
	return newVerifyCarrierEquipmentTool(intel)
}

func provideReassignBillingChargeTool(
	billing billingqueueports.BillingQueueService,
) agentports.AgentTool {
	return newReassignBillingChargeTool(billing)
}

func provideManageBillingTransferRunTool(
	runs *billingtransferservice.Service,
) agentports.AgentTool {
	return newManageBillingTransferRunTool(runs)
}
