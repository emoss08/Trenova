package agentquerytoolservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	paramBillingQueueItemIDs  = "billingQueueItemIds"
	maxBatchBillingQueueItems = 50
	batchQueueItemsSearchFor  = "billing queue items by id"
	noBillerLabel             = "no biller"
	readinessSeparator        = "; "
	columnAmount              = "amount"
)

// getBillingQueueItemsTool reads a set of queue items in one call, each with
// what get_billing_queue_item would say blocks it, so a biller working a
// queue reads it once and lands one table rather than a card per item.
type getBillingQueueItemsTool struct {
	items     billingQueueReader
	readiness billingReadinessReader
	invoices  queueInvoiceReader
	access    fieldAccess
}

func newGetBillingQueueItemsTool(
	items serviceports.BillingQueueService,
	shipments serviceports.ShipmentService,
	invoices repositories.InvoiceRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getBillingQueueItemsTool{
		items:     items,
		readiness: shipments,
		invoices:  invoices,
		access:    newFieldAccess(permissions),
	}
}

func (t *getBillingQueueItemsTool) Name() string { return "get_billing_queue_items" }

func (t *getBillingQueueItemsTool) BatchOf() string { return "get_billing_queue_item" }

func (t *getBillingQueueItemsTool) SearchTerms() []string {
	return []string{"billing queue", "review", "approve", "post", "batch", "readiness"}
}

func (t *getBillingQueueItemsTool) Description() string {
	return "Open several billing queue items by id in one call, each with what blocks its " +
		"approval and what its shipment still lacks for billing. Each row also carries the " +
		"payer, shipment, charges, biller and the invoice it made once approved. Use it " +
		"instead of calling get_billing_queue_item once per item: to check a queue before " +
		"assigning, moving into review, approving or posting it. Up to 50; an id that is not " +
		"a queue item of this organization is named in the note. Amounts are left out, and " +
		"named in withheldByAccess, when your data access does not reach them."
}

func (t *getBillingQueueItemsTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramBillingQueueItemIDs: agenttoolschema.KindIDs("The items' ids, from "+
				"list_billing_queue_items, the page you are on or the proposals that named "+
				"them.", maxBatchBillingQueueItems, permission.KindBillingQueueItem),
		},
		toolschema.KeyRequired:             []string{paramBillingQueueItemIDs},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *getBillingQueueItemsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceBillingQueue})
}

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

func (t *getBillingQueueItemsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	ids, err := requireIDList(params.Params, paramBillingQueueItemIDs, maxBatchBillingQueueItems)
	if err != nil {
		return nil, err
	}
	ids = sliceutils.Dedupe(ids)

	tenant := tenantOf(params)
	gate := t.access.gate(ctx, params, permission.ResourceBillingQueue)
	reader := queueDetailReader{items: t.items, readiness: t.readiness, invoices: t.invoices}
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
