package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingtransferservice"
	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/orderservice"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/internal/core/services/recurringshipmentservice"
	"github.com/emoss08/trenova/internal/core/services/schedulingservice"
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

func provideUncancelShipmentTool(shipments serviceports.ShipmentService) serviceports.AgentTool {
	return newUncancelShipmentTool(shipments)
}

func provideTransferShipmentOwnershipTool(
	shipments serviceports.ShipmentService,
) serviceports.AgentTool {
	return newTransferShipmentOwnershipTool(shipments)
}

func provideRerateShipmentTool(shipments serviceports.ShipmentService) serviceports.AgentTool {
	return newRerateShipmentTool(shipments)
}

func provideRecalculateShipmentDistanceTool(
	shipments serviceports.ShipmentService,
) serviceports.AgentTool {
	return newRecalculateShipmentDistanceTool(shipments)
}

func provideDuplicateShipmentTool(
	shipments serviceports.ShipmentService,
	locations repositories.LocationRepository,
) serviceports.AgentTool {
	return newDuplicateShipmentTool(duplicateShipmentDeps{
		Shipments: shipments,
		Sources:   shipments,
		Locations: locations,
	})
}

func provideUpdateShipmentHoldTool(holds serviceports.ShipmentHoldService) serviceports.AgentTool {
	return newUpdateShipmentHoldTool(holds)
}

func provideSplitMoveTool(moves serviceports.ShipmentMoveService) serviceports.AgentTool {
	return newSplitMoveTool(moves)
}

func providePinShipmentCommentTool(
	comments serviceports.ShipmentCommentService,
) serviceports.AgentTool {
	return newPinShipmentCommentTool(comments)
}

func provideUnpinShipmentCommentTool(
	comments serviceports.ShipmentCommentService,
) serviceports.AgentTool {
	return newUnpinShipmentCommentTool(comments)
}

func provideResolveShipmentCommentTool(
	comments serviceports.ShipmentCommentService,
) serviceports.AgentTool {
	return newResolveShipmentCommentTool(comments)
}

func provideEditShipmentCommentTool(
	comments serviceports.ShipmentCommentService,
	permissions serviceports.PermissionEngine,
) serviceports.AgentTool {
	return newEditShipmentCommentTool(comments, permissions)
}

func provideDeleteShipmentCommentTool(
	comments serviceports.ShipmentCommentService,
	permissions serviceports.PermissionEngine,
) serviceports.AgentTool {
	return newDeleteShipmentCommentTool(comments, permissions)
}

func provideUpdateServiceFailureTool(
	failures serviceports.ServiceFailureService,
) serviceports.AgentTool {
	return newUpdateServiceFailureTool(failures)
}

func provideReviewServiceFailureTool(
	failures serviceports.ServiceFailureService,
) serviceports.AgentTool {
	return newReviewServiceFailureTool(failures)
}

func provideVoidServiceFailureTool(
	failures serviceports.ServiceFailureService,
) serviceports.AgentTool {
	return newVoidServiceFailureTool(failures)
}

func provideDisputeDetentionTool(detentions *detentionservice.Service) serviceports.AgentTool {
	return newDisputeDetentionTool(detentions)
}

func provideCreateOrderTool(orders *orderservice.Service) serviceports.AgentTool {
	return newCreateOrderTool(orders)
}

func provideUpdateOrderTool(orders *orderservice.Service) serviceports.AgentTool {
	return newUpdateOrderTool(orders)
}

func provideAttachOrderShipmentsTool(orders *orderservice.Service) serviceports.AgentTool {
	return newAttachOrderShipmentsTool(orders)
}

func provideDetachOrderShipmentTool(orders *orderservice.Service) serviceports.AgentTool {
	return newDetachOrderShipmentTool(orders)
}

func provideCloseOrderTool(orders *orderservice.Service) serviceports.AgentTool {
	return newCloseOrderTool(orders)
}

func provideCancelOrderTool(orders *orderservice.Service) serviceports.AgentTool {
	return newCancelOrderTool(orders)
}

func provideAddOrderChargeTool(orders *orderservice.Service) serviceports.AgentTool {
	return newAddOrderChargeTool(orders)
}

func provideUpdateOrderChargeTool(orders *orderservice.Service) serviceports.AgentTool {
	return newUpdateOrderChargeTool(orders)
}

func provideSetOrderChargeAllocationsTool(orders *orderservice.Service) serviceports.AgentTool {
	return newSetOrderChargeAllocationsTool(orders)
}

func provideRemoveOrderChargeTool(orders *orderservice.Service) serviceports.AgentTool {
	return newRemoveOrderChargeTool(orders)
}

func provideCreateRecurringShipmentTool(
	series *recurringshipmentservice.Service,
) serviceports.AgentTool {
	return newCreateRecurringShipmentTool(series)
}

func provideUpdateRecurringShipmentTool(
	series *recurringshipmentservice.Service,
) serviceports.AgentTool {
	return newUpdateRecurringShipmentTool(series)
}

func provideSetRecurringShipmentStatusTool(
	series *recurringshipmentservice.Service,
) serviceports.AgentTool {
	return newSetRecurringShipmentStatusTool(series)
}

func provideGenerateRecurringShipmentTool(
	series *recurringshipmentservice.Service,
) serviceports.AgentTool {
	return newGenerateRecurringShipmentTool(series)
}

func provideAssignWorkerShiftTool(
	schedules *schedulingservice.Service,
	permissions serviceports.PermissionEngine,
	teams *orgstructureservice.Service,
) serviceports.AgentTool {
	return newAssignWorkerShiftTool(schedules, newTeamScope(permissions, teams))
}

func provideEndWorkerShiftAssignmentTool(
	schedules *schedulingservice.Service,
	permissions serviceports.PermissionEngine,
	teams *orgstructureservice.Service,
) serviceports.AgentTool {
	return newEndWorkerShiftAssignmentTool(
		schedules,
		newTeamScope(permissions, teams),
	)
}

func provideSetWorkerAvailabilityPreferenceTool(
	schedules *schedulingservice.Service,
	permissions serviceports.PermissionEngine,
	teams *orgstructureservice.Service,
) serviceports.AgentTool {
	return newSetWorkerAvailabilityPreferenceTool(
		schedules,
		newTeamScope(permissions, teams),
	)
}

func provideProposeShiftSwapTool(schedules *schedulingservice.Service) serviceports.AgentTool {
	return newProposeShiftSwapTool(schedules)
}

func provideApproveShiftSwapTool(schedules *schedulingservice.Service) serviceports.AgentTool {
	return newApproveShiftSwapTool(schedules)
}

func provideRejectShiftSwapTool(schedules *schedulingservice.Service) serviceports.AgentTool {
	return newRejectShiftSwapTool(schedules)
}

func provideWithdrawShiftSwapTool(schedules *schedulingservice.Service) serviceports.AgentTool {
	return newWithdrawShiftSwapTool(schedules)
}

func provideVetCarrierTool(intel *carrierintelservice.Service) serviceports.AgentTool {
	return newVetCarrierTool(intel)
}

func provideVetCustomerBrokerTool(intel *carrierintelservice.Service) serviceports.AgentTool {
	return newVetCustomerBrokerTool(intel)
}

func provideSetCarrierMonitoringTool(intel *carrierintelservice.Service) serviceports.AgentTool {
	return newSetCarrierMonitoringTool(intel)
}

func provideMarkCarrierIntelReviewedTool(
	intel *carrierintelservice.Service,
) serviceports.AgentTool {
	return newMarkCarrierIntelReviewedTool(intel)
}

func provideApplyCarrierIntelSuggestionsTool(
	intel *carrierintelservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentTool {
	return newApplyCarrierIntelSuggestionsTool(intel, permissions)
}

func provideImportSourcedCarrierTool(
	intel *carrierintelservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentTool {
	return newImportSourcedCarrierTool(intel, permissions)
}

func provideVerifyCarrierEquipmentTool(
	intel *carrierintelservice.Service,
) serviceports.AgentTool {
	return newVerifyCarrierEquipmentTool(intel)
}

func provideReassignBillingChargeTool(
	billing serviceports.BillingQueueService,
) serviceports.AgentTool {
	return newReassignBillingChargeTool(billing)
}

func provideManageBillingTransferRunTool(
	runs *billingtransferservice.Service,
) serviceports.AgentTool {
	return newManageBillingTransferRunTool(runs)
}
