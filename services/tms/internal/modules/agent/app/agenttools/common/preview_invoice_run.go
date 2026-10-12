package common

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func RunLabel(run *invoicerun.InvoiceRun) string {
	if run.ID.IsNil() || run.Number == "" {
		return "New invoice run"
	}

	return "Invoice run " + run.Number
}

func RunRecord(run *invoicerun.InvoiceRun) toolpreview.Record {
	record := toolpreview.Record{Resource: permission.ResourceInvoiceRun, Label: RunLabel(run)}
	if run.ID.IsNotNil() {
		record.ID = run.ID
		record.Version = PinnedVersion(run.Version)
	}

	return record
}

func WithNotes(summary string, notes []string) string {
	if len(notes) == 0 {
		return summary
	}

	return summary + " " + strings.Join(notes, "; ") + "."
}

func RenderCancel(
	_ *invoicerunports.CancelInvoiceRunRequest,
	plan *invoicerunports.InvoiceRunChangePreview,
) (*agent.ToolPreview, error) {
	change, err := toolpreview.Changed(
		RunRecord(plan.Before),
		plan.Before,
		plan.After,
		toolpreview.Only(FieldStatus, "failureReason", "canceledById", "canceledAt"),
		toolpreview.WithRefs(
			map[string]permission.Resource{"canceledById": permission.ResourceUser},
		),
		toolpreview.Volatile("canceledAt"),
	)
	if err != nil {
		return nil, err
	}
	change.Operation = agent.PreviewOperationArchive

	return toolpreview.Build(fmt.Sprintf(
		"Would cancel %s (now %s); its %s stay approved for the next run.",
		RunLabel(plan.Before), plan.Before.Status, CountOf(plan.Before.ItemCount, "shipment"),
	), change), nil
}
