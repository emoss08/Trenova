package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/movetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/servicetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransferservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgstructureservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/recurringshipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/schedulingservice"
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
	return shipmenttools.NewUncancelShipmentTool(shipments)
}

func provideTransferShipmentOwnershipTool(
	shipments shipmentports.ShipmentService,
) agentports.AgentTool {
	return shipmenttools.NewTransferShipmentOwnershipTool(shipments)
}

func provideRerateShipmentTool(shipments shipmentports.ShipmentService) agentports.AgentTool {
	return shipmenttools.NewRerateShipmentTool(shipments)
}

func provideRecalculateShipmentDistanceTool(
	shipments shipmentports.ShipmentService,
) agentports.AgentTool {
	return shipmenttools.NewRecalculateShipmentDistanceTool(shipments)
}

func provideDuplicateShipmentTool(
	shipments shipmentports.ShipmentService,
	locations locationports.LocationRepository,
) agentports.AgentTool {
	return shipmenttools.NewDuplicateShipmentTool(shipmenttools.DuplicateShipmentDeps{
		Shipments: shipments,
		Sources:   shipments,
		Locations: locations,
	})
}

func provideUpdateShipmentHoldTool(holds shipmentports.ShipmentHoldService) agentports.AgentTool {
	return shipmenttools.NewUpdateShipmentHoldTool(holds)
}

func provideSplitMoveTool(moves shipmentports.ShipmentMoveService) agentports.AgentTool {
	return movetools.NewSplitMoveTool(moves)
}

func providePinShipmentCommentTool(
	comments shipmentports.ShipmentCommentService,
) agentports.AgentTool {
	return shipmenttools.NewPinShipmentCommentTool(comments)
}

func provideUnpinShipmentCommentTool(
	comments shipmentports.ShipmentCommentService,
) agentports.AgentTool {
	return shipmenttools.NewUnpinShipmentCommentTool(comments)
}

func provideResolveShipmentCommentTool(
	comments shipmentports.ShipmentCommentService,
) agentports.AgentTool {
	return shipmenttools.NewResolveShipmentCommentTool(comments)
}

func provideEditShipmentCommentTool(
	comments shipmentports.ShipmentCommentService,
	permissions permissionports.PermissionEngine,
) agentports.AgentTool {
	return shipmenttools.NewEditShipmentCommentTool(comments, permissions)
}

func provideDeleteShipmentCommentTool(
	comments shipmentports.ShipmentCommentService,
	permissions permissionports.PermissionEngine,
) agentports.AgentTool {
	return shipmenttools.NewDeleteShipmentCommentTool(comments, permissions)
}

func provideUpdateServiceFailureTool(
	failures servicefailureports.ServiceFailureService,
) agentports.AgentTool {
	return servicetools.NewUpdateServiceFailureTool(failures)
}

func provideReviewServiceFailureTool(
	failures servicefailureports.ServiceFailureService,
) agentports.AgentTool {
	return servicetools.NewReviewServiceFailureTool(failures)
}

func provideVoidServiceFailureTool(
	failures servicefailureports.ServiceFailureService,
) agentports.AgentTool {
	return servicetools.NewVoidServiceFailureTool(failures)
}

func provideDisputeDetentionTool(detentions *detentionservice.Service) agentports.AgentTool {
	return receivabletools.NewDisputeDetentionTool(detentions)
}

func provideCreateOrderTool(orders *orderservice.Service) agentports.AgentTool {
	return receivabletools.NewCreateOrderTool(orders)
}

func provideUpdateOrderTool(orders *orderservice.Service) agentports.AgentTool {
	return receivabletools.NewUpdateOrderTool(orders)
}

func provideAttachOrderShipmentsTool(orders *orderservice.Service) agentports.AgentTool {
	return receivabletools.NewAttachOrderShipmentsTool(orders)
}

func provideDetachOrderShipmentTool(orders *orderservice.Service) agentports.AgentTool {
	return receivabletools.NewDetachOrderShipmentTool(orders)
}

func provideCloseOrderTool(orders *orderservice.Service) agentports.AgentTool {
	return receivabletools.NewCloseOrderTool(orders)
}

func provideCancelOrderTool(orders *orderservice.Service) agentports.AgentTool {
	return receivabletools.NewCancelOrderTool(orders)
}

func provideAddOrderChargeTool(orders *orderservice.Service) agentports.AgentTool {
	return receivabletools.NewAddOrderChargeTool(orders)
}

func provideUpdateOrderChargeTool(orders *orderservice.Service) agentports.AgentTool {
	return receivabletools.NewUpdateOrderChargeTool(orders)
}

func provideSetOrderChargeAllocationsTool(orders *orderservice.Service) agentports.AgentTool {
	return receivabletools.NewSetOrderChargeAllocationsTool(orders)
}

func provideRemoveOrderChargeTool(orders *orderservice.Service) agentports.AgentTool {
	return receivabletools.NewRemoveOrderChargeTool(orders)
}

func provideCreateRecurringShipmentTool(
	series *recurringshipmentservice.Service,
) agentports.AgentTool {
	return receivabletools.NewCreateRecurringShipmentTool(series)
}

func provideUpdateRecurringShipmentTool(
	series *recurringshipmentservice.Service,
) agentports.AgentTool {
	return receivabletools.NewUpdateRecurringShipmentTool(series)
}

func provideSetRecurringShipmentStatusTool(
	series *recurringshipmentservice.Service,
) agentports.AgentTool {
	return receivabletools.NewSetRecurringShipmentStatusTool(series)
}

func provideGenerateRecurringShipmentTool(
	series *recurringshipmentservice.Service,
) agentports.AgentTool {
	return receivabletools.NewGenerateRecurringShipmentTool(series)
}

func provideAssignWorkerShiftTool(
	schedules *schedulingservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agentports.AgentTool {
	return receivabletools.NewAssignWorkerShiftTool(
		schedules,
		receivabletools.NewTeamScope(permissions, teams),
	)
}

func provideEndWorkerShiftAssignmentTool(
	schedules *schedulingservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agentports.AgentTool {
	return receivabletools.NewEndWorkerShiftAssignmentTool(
		schedules,
		receivabletools.NewTeamScope(permissions, teams),
	)
}

func provideSetWorkerAvailabilityPreferenceTool(
	schedules *schedulingservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agentports.AgentTool {
	return receivabletools.NewSetWorkerAvailabilityPreferenceTool(
		schedules,
		receivabletools.NewTeamScope(permissions, teams),
	)
}

func provideProposeShiftSwapTool(schedules *schedulingservice.Service) agentports.AgentTool {
	return receivabletools.NewProposeShiftSwapTool(schedules)
}

func provideApproveShiftSwapTool(schedules *schedulingservice.Service) agentports.AgentTool {
	return receivabletools.NewApproveShiftSwapTool(schedules)
}

func provideRejectShiftSwapTool(schedules *schedulingservice.Service) agentports.AgentTool {
	return receivabletools.NewRejectShiftSwapTool(schedules)
}

func provideWithdrawShiftSwapTool(schedules *schedulingservice.Service) agentports.AgentTool {
	return receivabletools.NewWithdrawShiftSwapTool(schedules)
}

func provideVetCarrierTool(intel *carrierintelservice.Service) agentports.AgentTool {
	return carriertools.NewVetCarrierTool(intel)
}

func provideVetCustomerBrokerTool(intel *carrierintelservice.Service) agentports.AgentTool {
	return carriertools.NewVetCustomerBrokerTool(intel)
}

func provideSetCarrierMonitoringTool(intel *carrierintelservice.Service) agentports.AgentTool {
	return carriertools.NewSetCarrierMonitoringTool(intel)
}

func provideMarkCarrierIntelReviewedTool(
	intel *carrierintelservice.Service,
) agentports.AgentTool {
	return carriertools.NewMarkCarrierIntelReviewedTool(intel)
}

func provideApplyCarrierIntelSuggestionsTool(
	intel *carrierintelservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentTool {
	return carriertools.NewApplyCarrierIntelSuggestionsTool(intel, permissions)
}

func provideImportSourcedCarrierTool(
	intel *carrierintelservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentTool {
	return carriertools.NewImportSourcedCarrierTool(intel, permissions)
}

func provideVerifyCarrierEquipmentTool(
	intel *carrierintelservice.Service,
) agentports.AgentTool {
	return carriertools.NewVerifyCarrierEquipmentTool(intel)
}

func provideReassignBillingChargeTool(
	billing billingqueueports.BillingQueueService,
) agentports.AgentTool {
	return billingtools.NewReassignBillingChargeTool(billing)
}

func provideManageBillingTransferRunTool(
	runs *billingtransferservice.Service,
) agentports.AgentTool {
	return billingtools.NewManageBillingTransferRunTool(runs)
}
