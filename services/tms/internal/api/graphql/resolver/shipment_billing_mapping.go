package resolver

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func shipmentBillingReadinessToModel(
	readiness *services.ShipmentBillingReadiness,
) *gqlmodel.ShipmentBillingReadiness {
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
			ShipmentBillingRequirementEnforcement: string(
				readiness.Policy.ShipmentBillingRequirementEnforcement,
			),
			RateValidationEnforcement: string(
				readiness.Policy.RateValidationEnforcement,
			),
			BillingExceptionDisposition: string(
				readiness.Policy.BillingExceptionDisposition,
			),
			NotifyOnBillingExceptions: readiness.Policy.NotifyOnBillingExceptions,
			ReadyToBillAssignmentMode: string(
				readiness.Policy.ReadyToBillAssignmentMode,
			),
			BillingQueueTransferMode: string(
				readiness.Policy.BillingQueueTransferMode,
			),
		},
		Requirements:        shipmentBillingRequirementsToModel(readiness.Requirements),
		MissingRequirements: shipmentBillingRequirementsToModel(readiness.MissingRequirements),
		ValidationFailures:  shipmentBillingValidationsToModel(readiness.ValidationFailures),
		Warnings:            warnings,
		ServiceFailureContext: &gqlmodel.ShipmentServiceFailureBillingContext{
			HasUnresolved:     readiness.ServiceFailureContext.HasUnresolved,
			UnresolvedCount:   readiness.ServiceFailureContext.UnresolvedCount,
			ServiceFailureIds: readiness.ServiceFailureContext.ServiceFailureIDs,
		},
		CanMarkReadyToInvoice:        readiness.CanMarkReadyToInvoice,
		ShouldAutoMarkReadyToInvoice: readiness.ShouldAutoMarkReadyToInvoice,
		ShouldAutoTransferToBilling:  readiness.ShouldAutoTransferToBilling,
		ShouldAutoApproveBilling:     readiness.ShouldAutoApproveBilling,
		Payers:                       shipmentBillingPayersToModel(readiness.Payers),
	}
}

func shipmentBillingPayersToModel(
	payers []services.ShipmentBillingPayerReadiness,
) []*gqlmodel.ShipmentBillingPayerReadiness {
	out := make([]*gqlmodel.ShipmentBillingPayerReadiness, 0, len(payers))
	for _, payer := range payers {
		out = append(out, &gqlmodel.ShipmentBillingPayerReadiness{
			PayerID:                  payer.PayerID.String(),
			PayerName:                payer.PayerName,
			PayerCode:                payer.PayerCode,
			IsPrimary:                payer.IsPrimary,
			ShareAmount:              payer.ShareAmount.StringFixed(2),
			CreditStatus:             string(payer.CreditStatus),
			CreditHold:               payer.CreditHold,
			ShouldAutoApproveBilling: payer.ShouldAutoApproveBilling,
		})
	}
	return out
}

func transferToBillingResultToModel(
	result *services.TransferToBillingResult,
) (*gqlmodel.ShipmentTransferToBillingResult, error) {
	if result == nil {
		return nil, errortypes.NewDatabaseError("Billing queue transfer did not return a result").
			WithInternal(errors.New("shipment service returned nil transfer result"))
	}
	primary, err := requiredBillingQueueItemToModel(result.Primary)
	if err != nil {
		return nil, err
	}
	items := make([]*gqlmodel.BillingQueueItem, 0, len(result.Items))
	for _, item := range result.Items {
		model, itemErr := billingQueueItemToModel(item)
		if itemErr != nil {
			return nil, itemErr
		}
		if model != nil {
			items = append(items, model)
		}
	}
	return &gqlmodel.ShipmentTransferToBillingResult{Items: items, Primary: primary}, nil
}

func shipmentBillingWarningContextToModel(
	values map[string]any,
) *gqlmodel.ShipmentBillingWarningContext {
	if len(values) == 0 {
		return nil
	}
	return &gqlmodel.ShipmentBillingWarningContext{
		DocumentTypeID:          sliceutils.StringPtrValue(values["documentTypeId"]),
		DocumentTypeCode:        sliceutils.StringPtrValue(values["documentTypeCode"]),
		DocumentTypeName:        sliceutils.StringPtrValue(values["documentTypeName"]),
		DocumentCount:           intutils.IntPtrValue(values["documentCount"]),
		RequirementCount:        intutils.IntPtrValue(values["requirementCount"]),
		MissingRequirementCount: intutils.IntPtrValue(values["missingRequirementCount"]),
		ServiceFailureIds:       sliceutils.StringSliceValue(values["serviceFailureIds"]),
		UnresolvedCount:         intutils.IntPtrValue(values["unresolvedCount"]),
	}
}

func shipmentBillingRequirementsToModel(
	items []services.ShipmentBillingRequirement,
) []*gqlmodel.ShipmentBillingRequirement {
	out := make([]*gqlmodel.ShipmentBillingRequirement, 0, len(items))
	for _, item := range items {
		out = append(out, shipmentBillingRequirementToModel(item))
	}
	return out
}

func shipmentBillingValidationsToModel(
	items []services.ShipmentBillingValidation,
) []*gqlmodel.ShipmentBillingValidation {
	out := make([]*gqlmodel.ShipmentBillingValidation, 0, len(items))
	for _, item := range items {
		out = append(out, &gqlmodel.ShipmentBillingValidation{
			Field:   item.Field,
			Code:    item.Code,
			Message: item.Message,
		})
	}
	return out
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
	ctx context.Context,
	response *services.BulkTransferToBillingResponse,
) (*gqlmodel.ShipmentBulkTransferToBillingResponse, error) {
	locale := i18n.FromContext(ctx)
	classifier := helpers.NewDefaultClassifier()
	sanitizer := helpers.NewSanitizer(false)

	results := make([]*gqlmodel.ShipmentBulkTransferToBillingResult, 0, len(response.Results))
	for i := range response.Results {
		item := &response.Results[i]

		queueItem, err := billingQueueItemToModel(item.Item)
		if err != nil {
			return nil, err
		}
		queueItems := make([]*gqlmodel.BillingQueueItem, 0, len(item.Items))
		for _, created := range item.Items {
			model, itemErr := billingQueueItemToModel(created)
			if itemErr != nil {
				return nil, itemErr
			}
			if model != nil {
				queueItems = append(queueItems, model)
			}
		}

		result := &gqlmodel.ShipmentBulkTransferToBillingResult{
			ShipmentID:           item.ShipmentID.String(),
			ProNumber:            stringPtrFromValue(item.ProNumber),
			Success:              item.Success,
			MarkedReadyToInvoice: item.MarkedReadyToInvoice,
			BillingQueueItem:     queueItem,
			BillingQueueItems:    queueItems,
			MissingRequirements:  shipmentBillingRequirementsToModel(item.MissingRequirements),
			ValidationFailures:   shipmentBillingValidationsToModel(item.ValidationFailures),
		}
		if !item.Success {
			code := item.FailureCode
			result.FailureCode = &code
			result.Error = stringPtrFromValue(item.Error)
			if item.Err != nil {
				message := sanitizer.
					SanitizeMessage(item.Err, classifier.Classify(item.Err)).
					Localize(locale)
				result.Error = &message
			}
		}

		results = append(results, result)
	}

	return &gqlmodel.ShipmentBulkTransferToBillingResponse{
		Results:      results,
		TotalCount:   response.TotalCount,
		SuccessCount: response.SuccessCount,
		ErrorCount:   response.ErrorCount,
	}, nil
}

func billingTransferCandidateIDsToModel(
	response *services.BillingTransferCandidateIDsResponse,
) *gqlmodel.ShipmentBillingTransferCandidateIds {
	ids := make([]string, 0, len(response.IDs))
	for _, id := range response.IDs {
		ids = append(ids, id.String())
	}

	return &gqlmodel.ShipmentBillingTransferCandidateIds{
		Ids:        ids,
		TotalCount: response.TotalCount,
		Truncated:  response.Truncated,
	}
}

func parseBillType(value *billingqueue.BillType) billingqueue.BillType {
	if value == nil || *value == "" {
		return billingqueue.BillTypeInvoice
	}
	return *value
}
