package shipmentbillingtransfer

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentbillingreadiness"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type markReadyToInvoiceParams struct {
	ShipmentID        pulid.ID
	TenantInfo        pagination.TenantInfo
	Actor             kernelports.AuditActor
	Comment           string
	Entity            *shipment.Shipment
	RecordStatusEvent bool
}

func billingTransferPolicyViolation(
	readiness *shipmentbillingports.ShipmentBillingReadiness,
) (shipmentbillingports.BillingTransferFailureCode, error) {
	if readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelBlock &&
		shipmentbillingreadiness.HasShipmentRequirementIssues(readiness) {
		return shipmentbillingports.BillingTransferFailureRequirementsUnmet, errortypes.NewValidationError(
			"billingReadiness",
			errortypes.ErrInvalidOperation,
			"Shipment billing requirements must be resolved before transfer to billing",
		)
	}

	if readiness.Policy.RateValidationEnforcement == tenant.EnforcementLevelBlock &&
		shipmentbillingreadiness.HasRateIssues(readiness) {
		return shipmentbillingports.BillingTransferFailureRateValidation, errortypes.NewValidationError(
			"rateValidation",
			errortypes.ErrInvalidOperation,
			"Rate validation must be resolved before transfer to billing",
		)
	}

	if readiness.Policy.BillingExceptionDisposition == billingcontrol.BillingExceptionDispositionReturnToOperations &&
		(readiness.Policy.ShipmentBillingRequirementEnforcement == tenant.EnforcementLevelRequireReview ||
			readiness.Policy.RateValidationEnforcement == tenant.EnforcementLevelRequireReview) &&
		(shipmentbillingreadiness.HasShipmentRequirementIssues(readiness) || shipmentbillingreadiness.HasRateIssues(readiness)) {
		return shipmentbillingports.BillingTransferFailureReturnToOperations, errortypes.NewValidationError(
			"billingExceptionDisposition",
			errortypes.ErrInvalidOperation,
			"Shipment must be corrected in operations before it can be transferred to billing",
		)
	}

	return "", nil
}
