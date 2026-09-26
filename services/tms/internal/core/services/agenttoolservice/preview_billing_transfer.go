package agenttoolservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

// transferView is one shipment as the transfer leaves it: its status, its
// billing stage, and what the transfer does with it and why.
type transferView struct {
	Status                string `json:"status"`
	BillingTransferStatus string `json:"billingTransferStatus"`
	Outcome               string `json:"outcome"`
	MissingDocuments      string `json:"missingDocuments"`
	Issues                string `json:"issues"`
}

var transferLabels = map[string]string{
	"billingTransferStatus": "Billing stage",
	"outcome":               "What the transfer does",
	"missingDocuments":      "Missing documents",
	"issues":                "Rate and validation issues",
}

func (t *transferToBillingTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	checked := request.shipmentIDs
	if request.background() {
		checked = checked[:serviceports.MaxBulkTransferToBillingShipments]
	}
	plan, err := t.shipments.PlanBillingTransfers(ctx, &serviceports.PlanBillingTransfersRequest{
		TenantInfo:                  tenantFrom(params),
		ShipmentIDs:                 checked,
		MarkCompletedReadyToInvoice: request.markReady,
	})
	if err != nil {
		return nil, err
	}

	changes := make([]*agent.RecordChange, 0, len(plan.Decisions))
	for _, decision := range refusalsFirst(plan.Decisions) {
		change, changeErr := transferChange(decision)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, change)
	}

	preview := toolpreview.Build(transferSummary(request, plan), changes...)
	if request.background() {
		preview.Partial = true
	}
	if plan.Transfer == 0 {
		toolpreview.Warn(preview, agent.PreviewWarningWouldFail,
			"None of the checked shipments would transfer as they stand.")
	}

	return preview, nil
}

// refusalsFirst orders the shipments a person most needs to see ahead of the
// ones that simply go, since a preview shows the first few records.
func refusalsFirst(
	decisions []serviceports.BillingTransferDecision,
) []*serviceports.BillingTransferDecision {
	ordered := make([]*serviceports.BillingTransferDecision, 0, len(decisions))
	for idx := range decisions {
		ordered = append(ordered, &decisions[idx])
	}
	slices.SortStableFunc(ordered, func(a, b *serviceports.BillingTransferDecision) int {
		switch {
		case a.Outcome.Transfers() == b.Outcome.Transfers():
			return 0
		case b.Outcome.Transfers():
			return -1
		default:
			return 1
		}
	})

	return ordered
}

func transferChange(decision *serviceports.BillingTransferDecision) (*agent.RecordChange, error) {
	stage := string(decision.BillingTransferStatus)
	if stage == "" {
		stage = "Not transferred"
	}
	before := &transferView{
		Status:                string(decision.Status),
		BillingTransferStatus: stage,
	}
	after := *before
	after.Outcome = transferOutcomeText(decision)
	after.MissingDocuments = missingDocuments(decision)
	after.Issues = validationIssues(decision)

	switch decision.Outcome {
	case serviceports.BillingTransferOutcomeMarkReadyAndTransfer:
		after.Status = string(shipment.StatusReadyToInvoice)
		after.BillingTransferStatus = string(shipment.BillingTransferReadyForReview)
	case serviceports.BillingTransferOutcomeTransfer:
		after.BillingTransferStatus = string(shipment.BillingTransferReadyForReview)
	case serviceports.BillingTransferOutcomeRefused,
		serviceports.BillingTransferOutcomeReturnToOperations:
	}

	label := decision.ProNumber
	if label == "" {
		label = decision.ShipmentID.String()
	}
	record := toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       decision.ShipmentID,
		Label:    label,
	}
	if decision.Version > 0 {
		record.Version = pinnedVersion(decision.Version)
	}

	return toolpreview.Changed(record, before, &after, toolpreview.Labels(transferLabels))
}

func transferOutcomeText(decision *serviceports.BillingTransferDecision) string {
	switch decision.Outcome {
	case serviceports.BillingTransferOutcomeTransfer,
		serviceports.BillingTransferOutcomeMarkReadyAndTransfer:
		text := "Queued for a biller's review"
		if decision.Outcome == serviceports.BillingTransferOutcomeMarkReadyAndTransfer {
			text = "Marked ready to invoice and queued for a biller's review"
		}
		if decision.AutoApprove {
			text += "; the organization's auto-approve rule clears it unless a detention " +
				"charge holds it"
		}

		return text
	case serviceports.BillingTransferOutcomeReturnToOperations:
		return "Left with operations to correct: " + decision.Reason
	case serviceports.BillingTransferOutcomeRefused:
		return fmt.Sprintf("Refused (%s): %s", decision.FailureCode, decision.Reason)
	default:
		return string(decision.Outcome)
	}
}

func missingDocuments(decision *serviceports.BillingTransferDecision) string {
	names := make([]string, 0, len(decision.MissingRequirements))
	for _, requirement := range decision.MissingRequirements {
		names = append(names, requirement.DocumentTypeName)
	}

	return strings.Join(names, ", ")
}

func validationIssues(decision *serviceports.BillingTransferDecision) string {
	messages := make([]string, 0, len(decision.ValidationFailures))
	for _, failure := range decision.ValidationFailures {
		messages = append(messages, failure.Message)
	}

	return strings.Join(messages, "; ")
}

func transferSummary(request transferRequest, plan *serviceports.BillingTransferPlan) string {
	counts := fmt.Sprintf("%d would transfer", plan.Transfer)
	if plan.Refused > 0 {
		counts += fmt.Sprintf(", %d would be refused", plan.Refused)
	}
	if plan.Returned > 0 {
		counts += fmt.Sprintf(", %d would stay with operations to correct", plan.Returned)
	}

	if request.background() {
		return fmt.Sprintf(
			"Would start a background transfer of %s to billing as %s. Of the first %d "+
				"checked, %s.",
			countOf(len(request.shipmentIDs), "shipment"),
			request.billType,
			len(plan.Decisions),
			counts,
		)
	}

	return fmt.Sprintf(
		"Would transfer %s to billing as %s: %s.",
		countOf(len(request.shipmentIDs), "shipment"),
		request.billType,
		counts,
	)
}
