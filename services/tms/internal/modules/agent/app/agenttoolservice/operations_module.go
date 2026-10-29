package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingtransfertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/carriertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/movetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ordertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/recurringshipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/servicetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shifttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentcommenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentopstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransferservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgstructureservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/recurringshipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelenrollmentservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelreviewservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelsourcingservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelverifyservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelvetservice"
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

func provideUncancelShipmentTool(
	shipments shipmentserviceports.ShipmentService,
) agenttoolports.AgentTool {
	return shipmentopstools.NewUncancelShipmentTool(shipments)
}

func provideTransferShipmentOwnershipTool(
	shipments shipmentserviceports.ShipmentService,
) agenttoolports.AgentTool {
	return shipmentopstools.NewTransferShipmentOwnershipTool(shipments)
}

func provideRerateShipmentTool(
	shipments shipmentserviceports.ShipmentService,
) agenttoolports.AgentTool {
	return shipmentopstools.NewRerateShipmentTool(shipments)
}

func provideRecalculateShipmentDistanceTool(
	shipments shipmentserviceports.ShipmentService,
) agenttoolports.AgentTool {
	return shipmentopstools.NewRecalculateShipmentDistanceTool(shipments)
}

func provideDuplicateShipmentTool(
	shipments shipmentserviceports.ShipmentService,
	locations locationports.LocationRepository,
) agenttoolports.AgentTool {
	return shipmentopstools.NewDuplicateShipmentTool(shipmentopstools.DuplicateShipmentDeps{
		Shipments: shipments,
		Sources:   shipments,
		Locations: locations,
	})
}

func provideUpdateShipmentHoldTool(
	holds shipmentholdports.ShipmentHoldService,
) agenttoolports.AgentTool {
	return shipmenttools.NewUpdateShipmentHoldTool(holds)
}

func provideSplitMoveTool(moves shipmentmoveports.ShipmentMoveService) agenttoolports.AgentTool {
	return movetools.NewSplitMoveTool(moves)
}

func providePinShipmentCommentTool(
	comments shipmentcommentports.ShipmentCommentService,
) agenttoolports.AgentTool {
	return shipmentcommenttools.NewPinShipmentCommentTool(comments)
}

func provideUnpinShipmentCommentTool(
	comments shipmentcommentports.ShipmentCommentService,
) agenttoolports.AgentTool {
	return shipmentcommenttools.NewUnpinShipmentCommentTool(comments)
}

func provideResolveShipmentCommentTool(
	comments shipmentcommentports.ShipmentCommentService,
) agenttoolports.AgentTool {
	return shipmentcommenttools.NewResolveShipmentCommentTool(comments)
}

func provideEditShipmentCommentTool(
	comments shipmentcommentports.ShipmentCommentService,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentTool {
	return shipmentcommenttools.NewEditShipmentCommentTool(comments, permissions)
}

func provideDeleteShipmentCommentTool(
	comments shipmentcommentports.ShipmentCommentService,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentTool {
	return shipmentcommenttools.NewDeleteShipmentCommentTool(comments, permissions)
}

func provideUpdateServiceFailureTool(
	failures servicefailureports.ServiceFailureService,
) agenttoolports.AgentTool {
	return servicetools.NewUpdateServiceFailureTool(failures)
}

func provideReviewServiceFailureTool(
	failures servicefailureports.ServiceFailureService,
) agenttoolports.AgentTool {
	return servicetools.NewReviewServiceFailureTool(failures)
}

func provideVoidServiceFailureTool(
	failures servicefailureports.ServiceFailureService,
) agenttoolports.AgentTool {
	return servicetools.NewVoidServiceFailureTool(failures)
}

func provideDisputeDetentionTool(detentions *detentionreview.Service) agenttoolports.AgentTool {
	return receivabletools.NewDisputeDetentionTool(detentions)
}

func provideCreateOrderTool(orders *orderservice.Service) agenttoolports.AgentTool {
	return ordertools.NewCreateOrderTool(orders)
}

func provideUpdateOrderTool(orders *orderservice.Service) agenttoolports.AgentTool {
	return ordertools.NewUpdateOrderTool(orders)
}

func provideAttachOrderShipmentsTool(orders *orderservice.Service) agenttoolports.AgentTool {
	return ordertools.NewAttachOrderShipmentsTool(orders)
}

func provideDetachOrderShipmentTool(orders *orderservice.Service) agenttoolports.AgentTool {
	return ordertools.NewDetachOrderShipmentTool(orders)
}

func provideCloseOrderTool(orders *orderservice.Service) agenttoolports.AgentTool {
	return ordertools.NewCloseOrderTool(orders)
}

func provideCancelOrderTool(orders *orderservice.Service) agenttoolports.AgentTool {
	return ordertools.NewCancelOrderTool(orders)
}

func provideAddOrderChargeTool(orders *orderservice.Service) agenttoolports.AgentTool {
	return ordertools.NewAddOrderChargeTool(orders)
}

func provideUpdateOrderChargeTool(orders *orderservice.Service) agenttoolports.AgentTool {
	return ordertools.NewUpdateOrderChargeTool(orders)
}

func provideSetOrderChargeAllocationsTool(orders *orderservice.Service) agenttoolports.AgentTool {
	return ordertools.NewSetOrderChargeAllocationsTool(orders)
}

func provideRemoveOrderChargeTool(orders *orderservice.Service) agenttoolports.AgentTool {
	return ordertools.NewRemoveOrderChargeTool(orders)
}

func provideCreateRecurringShipmentTool(
	series *recurringshipmentservice.Service,
) agenttoolports.AgentTool {
	return recurringshipmenttools.NewCreateRecurringShipmentTool(series)
}

func provideUpdateRecurringShipmentTool(
	series *recurringshipmentservice.Service,
) agenttoolports.AgentTool {
	return recurringshipmenttools.NewUpdateRecurringShipmentTool(series)
}

func provideSetRecurringShipmentStatusTool(
	series *recurringshipmentservice.Service,
) agenttoolports.AgentTool {
	return recurringshipmenttools.NewSetRecurringShipmentStatusTool(series)
}

func provideGenerateRecurringShipmentTool(
	series *recurringshipmentservice.Service,
) agenttoolports.AgentTool {
	return recurringshipmenttools.NewGenerateRecurringShipmentTool(series)
}

func provideAssignWorkerShiftTool(
	schedules *schedulingservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agenttoolports.AgentTool {
	return shifttools.NewAssignWorkerShiftTool(
		schedules,
		shifttools.NewTeamScope(permissions, teams),
	)
}

func provideEndWorkerShiftAssignmentTool(
	schedules *schedulingservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agenttoolports.AgentTool {
	return shifttools.NewEndWorkerShiftAssignmentTool(
		schedules,
		shifttools.NewTeamScope(permissions, teams),
	)
}

func provideSetWorkerAvailabilityPreferenceTool(
	schedules *schedulingservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agenttoolports.AgentTool {
	return shifttools.NewSetWorkerAvailabilityPreferenceTool(
		schedules,
		shifttools.NewTeamScope(permissions, teams),
	)
}

func provideProposeShiftSwapTool(schedules *schedulingservice.Service) agenttoolports.AgentTool {
	return shifttools.NewProposeShiftSwapTool(schedules)
}

func provideApproveShiftSwapTool(schedules *schedulingservice.Service) agenttoolports.AgentTool {
	return shifttools.NewApproveShiftSwapTool(schedules)
}

func provideRejectShiftSwapTool(schedules *schedulingservice.Service) agenttoolports.AgentTool {
	return shifttools.NewRejectShiftSwapTool(schedules)
}

func provideWithdrawShiftSwapTool(schedules *schedulingservice.Service) agenttoolports.AgentTool {
	return shifttools.NewWithdrawShiftSwapTool(schedules)
}

func provideVetCarrierTool(intel *carrierintelvetservice.Service) agenttoolports.AgentTool {
	return carriertools.NewVetCarrierTool(intel)
}

func provideVetCustomerBrokerTool(intel *carrierintelvetservice.Service) agenttoolports.AgentTool {
	return carriertools.NewVetCustomerBrokerTool(intel)
}

func provideSetCarrierMonitoringTool(
	intel *carrierintelenrollmentservice.Service,
) agenttoolports.AgentTool {
	return carriertools.NewSetCarrierMonitoringTool(intel)
}

func provideMarkCarrierIntelReviewedTool(
	intel *carrierintelreviewservice.Service,
) agenttoolports.AgentTool {
	return carriertools.NewMarkCarrierIntelReviewedTool(intel)
}

func provideApplyCarrierIntelSuggestionsTool(
	intel *carrierintelvetservice.Service,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentTool {
	return carriertools.NewApplyCarrierIntelSuggestionsTool(intel, permissions)
}

func provideImportSourcedCarrierTool(
	intel *carrierintelsourcingservice.Service,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentTool {
	return carriertools.NewImportSourcedCarrierTool(intel, permissions)
}

func provideVerifyCarrierEquipmentTool(
	intel *carrierintelverifyservice.Service,
) agenttoolports.AgentTool {
	return carriertools.NewVerifyCarrierEquipmentTool(intel)
}

func provideReassignBillingChargeTool(
	billing billingqueueports.BillingQueueService,
) agenttoolports.AgentTool {
	return billingtransfertools.NewReassignBillingChargeTool(billing)
}

func provideManageBillingTransferRunTool(
	runs *billingtransferservice.Service,
) agenttoolports.AgentTool {
	return billingtransfertools.NewManageBillingTransferRunTool(runs)
}
