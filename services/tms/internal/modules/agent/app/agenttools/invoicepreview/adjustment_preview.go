package invoicepreview

import (
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func FiguresRefusal(figures []*invoiceadjustmentports.InvoiceAdjustmentPreview) error {
	multiErr := errortypes.NewMultiError()
	for idx, preview := range figures {
		if preview == nil || len(preview.Errors) == 0 {
			continue
		}
		fields := make([]string, 0, len(preview.Errors))
		for field := range preview.Errors {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		for _, field := range fields {
			for _, message := range preview.Errors[field] {
				path := field
				if len(figures) > 1 {
					path = fmt.Sprintf("%s[%d].%s", toolparams.ParamAdjustments, idx, field)
				}
				multiErr.Add(path, errortypes.ErrInvalid, message)
			}
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type AdjustmentView struct {
	InvoiceID      pulid.ID `json:"invoiceId"`
	Kind           string   `json:"kind"`
	RebillStrategy string   `json:"rebillStrategy"`
	Reason         string   `json:"reason"`
	Status         string   `json:"status"`
	ApprovalStatus string   `json:"approvalStatus,omitempty"`
	LineCount      int      `json:"lineCount"`
}

var AdjustmentRefs = map[string]permission.Resource{
	toolparams.ParamInvoiceID: permission.ResourceInvoice,
}

func adjustmentInvoiceLabel(number string) string {
	if strings.TrimSpace(number) == "" {
		return "the invoice"
	}

	return "Invoice " + number
}

func AdjustmentLabel(kind invoiceadjustment.Kind, invoiceNumber string) string {
	return string(kind) + " adjustment of " + adjustmentInvoiceLabel(invoiceNumber)
}

func AdjustmentMoney(
	change *agentpreview.RecordChange,
	figures *invoiceadjustmentports.InvoiceAdjustmentPreview,
	currency string,
) {
	if figures == nil {
		return
	}
	toolpreview.AttachMoney(change, toolpreview.MoneyBlock(
		currency,
		agentpreview.MoneyLine{
			Label: "Credited",
			After: changepreview.KnownAmount(figures.CreditTotalAmount.Abs()),
		},
		agentpreview.MoneyLine{
			Label: "Rebilled",
			After: changepreview.KnownAmount(figures.RebillTotalAmount),
		},
	), toolpreview.SensitiveAs("totalAmount"))
}

func OutcomeStatus(
	figures *invoiceadjustmentports.InvoiceAdjustmentPreview,
) invoiceadjustment.Status {
	if figures != nil && figures.RequiresApproval {
		return invoiceadjustment.StatusPendingApproval
	}

	return invoiceadjustment.StatusExecuted
}

func OutcomeSentence(figures *invoiceadjustmentports.InvoiceAdjustmentPreview) string {
	if OutcomeStatus(figures) == invoiceadjustment.StatusPendingApproval {
		return "waits for approval"
	}

	return "executes at once"
}

func FigureWarnings(figures *invoiceadjustmentports.InvoiceAdjustmentPreview) string {
	if figures == nil || len(figures.Warnings) == 0 {
		return ""
	}

	return " Policy notes: " + strings.Join(figures.Warnings, "; ") + "."
}
