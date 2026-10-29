package billingtransfertools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
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

func (t *TransferToBillingTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	plan, err := t.previewPlan(ctx, params, &request)
	if err != nil {
		return nil, err
	}

	changes := make([]*agentpreview.RecordChange, 0, len(plan.Decisions))
	for _, decision := range refusalsFirst(plan.Decisions) {
		change, changeErr := transferChange(decision)
		if changeErr != nil {
			return nil, changeErr
		}
		changes = append(changes, change)
	}

	preview := toolpreview.Build(transferSummary(&request, plan), changes...)
	if request.background() && len(plan.Decisions) < len(request.shipmentIDs) {
		preview.Partial = true
	}
	if plan.Transfer == 0 {
		toolpreview.Warn(preview, agentpreview.PreviewWarningWouldFail,
			"None of the checked shipments would transfer as they stand.")
	}

	return preview, nil
}

func (t *TransferToBillingTool) previewPlan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
	request *transferRequest,
) (*shipmentbillingports.BillingTransferPlan, error) {
	if request.all {
		ids, plan, err := t.transferable(ctx, params, request)
		if err != nil {
			return nil, err
		}
		request.shipmentIDs = ids

		return plan, nil
	}

	checked := request.shipmentIDs
	if request.background() {
		checked = checked[:shipmentbillingports.MaxBulkTransferToBillingShipments]
	}

	return t.shipments.PlanBillingTransfers(ctx, &shipmentbillingports.PlanBillingTransfersRequest{
		TenantInfo:                  toolparams.TenantFrom(params),
		ShipmentIDs:                 checked,
		MarkCompletedReadyToInvoice: request.markReady,
	})
}

// refusalsFirst orders the shipments a person most needs to see ahead of the
// ones that simply go, since a preview shows the first few records.
func refusalsFirst(
	decisions []shipmentbillingports.BillingTransferDecision,
) []*shipmentbillingports.BillingTransferDecision {
	ordered := make([]*shipmentbillingports.BillingTransferDecision, 0, len(decisions))
	for idx := range decisions {
		ordered = append(ordered, &decisions[idx])
	}
	slices.SortStableFunc(ordered, func(a, b *shipmentbillingports.BillingTransferDecision) int {
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

func transferChange(
	decision *shipmentbillingports.BillingTransferDecision,
) (*agentpreview.RecordChange, error) {
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
	case shipmentbillingports.BillingTransferOutcomeMarkReadyAndTransfer:
		after.Status = string(shipmentvocab.StatusReadyToInvoice)
		after.BillingTransferStatus = string(shipmentvocab.BillingTransferReadyForReview)
	case shipmentbillingports.BillingTransferOutcomeTransfer:
		after.BillingTransferStatus = string(shipmentvocab.BillingTransferReadyForReview)
	case shipmentbillingports.BillingTransferOutcomeRefused,
		shipmentbillingports.BillingTransferOutcomeReturnToOperations:
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
		record.Version = changepreview.PinnedVersion(decision.Version)
	}

	return toolpreview.Changed(record, before, &after, toolpreview.Labels(transferLabels))
}

func transferOutcomeText(decision *shipmentbillingports.BillingTransferDecision) string {
	switch decision.Outcome {
	case shipmentbillingports.BillingTransferOutcomeTransfer,
		shipmentbillingports.BillingTransferOutcomeMarkReadyAndTransfer:
		text := "Queued for a biller's review"
		if decision.Outcome == shipmentbillingports.BillingTransferOutcomeMarkReadyAndTransfer {
			text = "Marked ready to invoice and queued for a biller's review"
		}
		if decision.AutoApprove {
			text += "; the organization's auto-approve rule clears it unless a detention " +
				"charge holds it"
		}

		return text
	case shipmentbillingports.BillingTransferOutcomeReturnToOperations:
		return "Left with operations to correct: " + decision.Reason
	case shipmentbillingports.BillingTransferOutcomeRefused:
		return fmt.Sprintf("Refused (%s): %s", decision.FailureCode, decision.Reason)
	default:
		return string(decision.Outcome)
	}
}

func missingDocuments(decision *shipmentbillingports.BillingTransferDecision) string {
	names := make([]string, 0, len(decision.MissingRequirements))
	for _, requirement := range decision.MissingRequirements {
		names = append(names, requirement.DocumentTypeName)
	}

	return strings.Join(names, ", ")
}

func validationIssues(decision *shipmentbillingports.BillingTransferDecision) string {
	messages := make([]string, 0, len(decision.ValidationFailures))
	for _, failure := range decision.ValidationFailures {
		messages = append(messages, failure.Message)
	}

	return strings.Join(messages, "; ")
}

func transferSummary(
	request *transferRequest,
	plan *shipmentbillingports.BillingTransferPlan,
) string {
	counts := fmt.Sprintf("%d would transfer", plan.Transfer)
	if plan.Refused > 0 {
		counts += fmt.Sprintf(", %d would be refused", plan.Refused)
	}
	if plan.Returned > 0 {
		counts += fmt.Sprintf(", %d would stay with operations to correct", plan.Returned)
	}

	if request.background() {
		return fmt.Sprintf(
			"Would start a background transfer of %s to billing as %s. Of the %d "+
				"checked, %s.",
			changepreview.CountOf(len(request.shipmentIDs), "shipment"),
			request.billType,
			len(plan.Decisions),
			counts,
		)
	}

	return fmt.Sprintf(
		"Would transfer %s to billing as %s: %s.",
		changepreview.CountOf(len(request.shipmentIDs), "shipment"),
		request.billType,
		counts,
	)
}
