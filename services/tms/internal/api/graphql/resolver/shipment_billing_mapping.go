package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func shipmentBillingReadinessToModel(
	readiness *services.ShipmentBillingReadiness,
) *gqlmodel.ShipmentBillingReadiness {
	requirements := make([]*gqlmodel.ShipmentBillingRequirement, 0, len(readiness.Requirements))
	for _, item := range readiness.Requirements {
		requirements = append(requirements, shipmentBillingRequirementToModel(item))
	}
	missing := make([]*gqlmodel.ShipmentBillingRequirement, 0, len(readiness.MissingRequirements))
	for _, item := range readiness.MissingRequirements {
		missing = append(missing, shipmentBillingRequirementToModel(item))
	}
	validations := make([]*gqlmodel.ShipmentBillingValidation, 0, len(readiness.ValidationFailures))
	for _, item := range readiness.ValidationFailures {
		validations = append(validations, &gqlmodel.ShipmentBillingValidation{
			Field:   item.Field,
			Code:    item.Code,
			Message: item.Message,
		})
	}
	warnings := make([]*gqlmodel.ShipmentBillingWarning, 0, len(readiness.Warnings))
	for _, item := range readiness.Warnings {
		warnings = append(warnings, &gqlmodel.ShipmentBillingWarning{
			Code:    item.Code,
			Message: item.Message,
			Context: shipmentBillingWarningContextToModel(item.Context),
		})
	}
	return &gqlmodel.ShipmentBillingReadiness{
		ShipmentID:     readiness.ShipmentID,
		ShipmentStatus: gqlmodel.ShipmentStatus(readiness.ShipmentStatus),
		Policy: &gqlmodel.ShipmentBillingReadinessPolicy{
			ShipmentBillingRequirementEnforcement: string(readiness.Policy.ShipmentBillingRequirementEnforcement),
			RateValidationEnforcement:             string(readiness.Policy.RateValidationEnforcement),
			BillingExceptionDisposition:           string(readiness.Policy.BillingExceptionDisposition),
			NotifyOnBillingExceptions:             readiness.Policy.NotifyOnBillingExceptions,
			ReadyToBillAssignmentMode:             string(readiness.Policy.ReadyToBillAssignmentMode),
			BillingQueueTransferMode:              string(readiness.Policy.BillingQueueTransferMode),
		},
		Requirements:        requirements,
		MissingRequirements: missing,
		ValidationFailures:  validations,
		Warnings:            warnings,
		ServiceFailureContext: &gqlmodel.ShipmentServiceFailureBillingContext{
			HasUnresolved:     readiness.ServiceFailureContext.HasUnresolved,
			UnresolvedCount:   readiness.ServiceFailureContext.UnresolvedCount,
			ServiceFailureIds: readiness.ServiceFailureContext.ServiceFailureIDs,
		},
		CanMarkReadyToInvoice:        readiness.CanMarkReadyToInvoice,
		ShouldAutoMarkReadyToInvoice: readiness.ShouldAutoMarkReadyToInvoice,
		ShouldAutoTransferToBilling:  readiness.ShouldAutoTransferToBilling,
	}
}

func shipmentBillingWarningContextToModel(
	context map[string]any,
) *gqlmodel.ShipmentBillingWarningContext {
	if len(context) == 0 {
		return nil
	}
	return &gqlmodel.ShipmentBillingWarningContext{
		DocumentTypeID:          sliceutils.StringPtrValue(context["documentTypeId"]),
		DocumentTypeCode:        sliceutils.StringPtrValue(context["documentTypeCode"]),
		DocumentTypeName:        sliceutils.StringPtrValue(context["documentTypeName"]),
		DocumentCount:           intutils.IntPtrValue(context["documentCount"]),
		RequirementCount:        intutils.IntPtrValue(context["requirementCount"]),
		MissingRequirementCount: intutils.IntPtrValue(context["missingRequirementCount"]),
		ServiceFailureIds:       sliceutils.StringSliceValue(context["serviceFailureIds"]),
		UnresolvedCount:         intutils.IntPtrValue(context["unresolvedCount"]),
	}
}

func shipmentBillingRequirementToModel(
	item services.ShipmentBillingRequirement,
) *gqlmodel.ShipmentBillingRequirement {
	return &gqlmodel.ShipmentBillingRequirement{
		DocumentTypeID:   item.DocumentTypeID,
		DocumentTypeCode: item.DocumentTypeCode,
		DocumentTypeName: item.DocumentTypeName,
		Satisfied:        item.Satisfied,
		DocumentCount:    item.DocumentCount,
		DocumentIds:      item.DocumentIDs,
	}
}

func bulkTransferToBillingToModel(
	response *services.BulkTransferToBillingResponse,
) *gqlmodel.ShipmentBulkTransferToBillingResponse {
	results := make([]*gqlmodel.ShipmentBulkTransferToBillingResult, 0, len(response.Results))
	for _, item := range response.Results {
		results = append(results, &gqlmodel.ShipmentBulkTransferToBillingResult{
			ShipmentID: item.ShipmentID.String(),
			Success:    item.Success,
			Error:      stringPtrFromValue(item.Error),
		})
	}
	return &gqlmodel.ShipmentBulkTransferToBillingResponse{
		Results:      results,
		TotalCount:   response.TotalCount,
		SuccessCount: response.SuccessCount,
		ErrorCount:   response.ErrorCount,
	}
}

func parseBillType(value *billingqueue.BillType) billingqueue.BillType {
	if value == nil || *value == "" {
		return billingqueue.BillTypeInvoice
	}
	return *value
}
