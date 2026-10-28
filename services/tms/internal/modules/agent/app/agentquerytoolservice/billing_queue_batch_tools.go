package agentquerytoolservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramBillingQueueItemIDs  = "billingQueueItemIds"
	maxBatchBillingQueueItems = 50
	batchQueueItemsSearchFor  = "billing queue items by id"
	noBillerLabel             = "no biller"
	readinessSeparator        = "; "
	columnAmount              = "amount"
)

// billingQueueBatchRow is one item as a row of the table the batch read
// lands: the queue row, then what decides its next step.
type billingQueueBatchRow struct {
	ID                 string       `json:"id"`
	Number             string       `json:"number"`
	Status             string       `json:"status"`
	BillType           string       `json:"billType"`
	ProNumber          string       `json:"proNumber,omitempty"`
	BOL                string       `json:"bol,omitempty"`
	BillTo             string       `json:"billTo,omitempty"`
	AssignedBiller     string       `json:"assignedBiller"`
	Amount             string       `json:"amount,omitempty"`
	CanApprove         bool         `json:"canApprove"`
	ApprovalBlockedBy  string       `json:"approvalBlockedBy,omitempty"`
	MissingDocuments   string       `json:"missingDocuments,omitempty"`
	ValidationFailures string       `json:"validationFailures,omitempty"`
	Warnings           string       `json:"warnings,omitempty"`
	DetentionHolds     int          `json:"detentionHolds"`
	Charges            int          `json:"charges"`
	ExceptionReason    string       `json:"exceptionReason,omitempty"`
	AgeDays            int64        `json:"ageDays"`
	QueuedAt           optionalDate `json:"queuedAt"`
	InvoiceID          string       `json:"invoiceId,omitempty"`
	ShipmentID         string       `json:"shipmentId,omitempty"`
}

// several reads a set of items as rows of one table, each with what decides
// its next step, so a biller working a queue reads it once rather than an
// item at a time.
func (t *getBillingQueueItemTool) several(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	ids []pulid.ID,
	gate *fieldGate,
) (any, error) {
	tenant := tenantOf(params)
	reader := t.reader()
	rows := make([]billingQueueBatchRow, 0, len(ids))
	missing := make([]string, 0, len(ids))
	for _, id := range ids {
		detail, detailErr := reader.detail(ctx, tenant, id, gate)
		if detailErr != nil {
			if errortypes.IsNotFoundError(detailErr) {
				missing = append(missing, id.String())

				continue
			}

			return nil, detailErr
		}
		rows = append(rows, batchRowOf(detail))
	}

	withheld := gate.Withheld()
	outcome := searchOutcome{
		Count:       len(rows),
		SearchedFor: []string{batchQueueItemsSearchFor},
		Items:       rows,
		Columns:     columnsShown(columnsOf(rows), withheld),
	}
	if len(missing) > 0 {
		outcome.Note = "No billing queue item of this organization has these ids: " +
			strings.Join(missing, ", ")
	}

	return gatedResult(&outcome, gate), nil
}

func batchRowOf(detail *billingQueueDetail) billingQueueBatchRow {
	row := billingQueueBatchRow{
		ID:                detail.ID,
		Number:            detail.Number,
		Status:            detail.Status,
		BillType:          detail.BillType,
		ProNumber:         detail.ProNumber,
		BOL:               detail.BOL,
		BillTo:            detail.BillTo,
		AssignedBiller:    detail.AssignedBiller,
		Amount:            detail.Amount,
		CanApprove:        detail.CanApprove,
		ApprovalBlockedBy: detail.ApprovalBlockedBy,
		DetentionHolds:    len(detail.DetentionHolds),
		Charges:           len(detail.Charges),
		ExceptionReason:   detail.ExceptionReason,
		AgeDays:           detail.AgeDays,
		QueuedAt:          detail.QueuedAt,
		InvoiceID:         detail.InvoiceID,
		ShipmentID:        detail.ShipmentID,
	}
	if row.AssignedBiller == "" &&
		!billingqueue.IsTerminalStatus(billingqueue.Status(detail.Status)) {
		row.AssignedBiller = noBillerLabel
	}
	if detail.Readiness != nil {
		row.MissingDocuments = strings.Join(detail.Readiness.MissingDocuments, readinessSeparator)
		row.ValidationFailures = strings.Join(
			detail.Readiness.ValidationFailures,
			readinessSeparator,
		)
		row.Warnings = strings.Join(detail.Readiness.Warnings, readinessSeparator)
	}

	return row
}

// columnsShown drops the columns the reader's access withheld: a table that
// promises an amount column and leaves every cell empty says the wrong thing.
func columnsShown(columns, withheld []string) []string {
	if len(withheld) == 0 {
		return columns
	}
	hidden := map[string]struct{}{}
	for _, name := range withheld {
		if name == withheldAmounts {
			hidden[columnAmount] = struct{}{}
		}
		hidden[name] = struct{}{}
	}

	return slices.DeleteFunc(slices.Clone(columns), func(column string) bool {
		_, drop := hidden[column]

		return drop
	})
}
