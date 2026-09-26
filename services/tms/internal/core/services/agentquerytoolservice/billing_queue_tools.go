package agentquerytoolservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	paramBillingQueueItemID = "billingQueueItemId"
	fieldAllocatedTotal     = "allocatedTotalAmount"
	maxBillingQueueLines    = 50
	secondsPerDayInQueue    = 86400
	fieldNumber             = "number"
)

var (
	billingQueueStatuses = []string{
		string(billingqueue.StatusReadyForReview), string(billingqueue.StatusInReview),
		string(billingqueue.StatusOnHold), string(billingqueue.StatusException),
		string(billingqueue.StatusSentBackToOps), string(billingqueue.StatusApproved),
		string(billingqueue.StatusPosted), string(billingqueue.StatusCanceled),
	}
	billTypes = []string{
		string(billingqueue.BillTypeInvoice), string(billingqueue.BillTypeCreditMemo),
		string(billingqueue.BillTypeDebitMemo),
	}
	exceptionReasonCodes = []string{
		string(billingqueue.ExceptionMissingDocumentation),
		string(billingqueue.ExceptionIncorrectRates),
		string(billingqueue.ExceptionWeightDiscrepancy),
		string(billingqueue.ExceptionAccessorialDispute),
		string(billingqueue.ExceptionDuplicateCharge),
		string(billingqueue.ExceptionMissingReferenceNumber),
		string(billingqueue.ExceptionCustomerInformationError),
		string(billingqueue.ExceptionServiceFailure),
		string(billingqueue.ExceptionRateNotOnFile),
		string(billingqueue.ExceptionOther),
	}
)

type billingQueueLister interface {
	List(
		ctx context.Context,
		req *repositories.ListBillingQueueItemsRequest,
	) (*pagination.ListResult[*billingqueue.BillingQueueItem], error)
}

type billingQueueRow struct {
	ID              string       `json:"id"`
	Number          string       `json:"number"`
	Status          string       `json:"status"`
	BillType        string       `json:"billType"`
	ProNumber       string       `json:"proNumber,omitempty"`
	BillTo          string       `json:"billTo,omitempty"`
	AssignedBiller  string       `json:"assignedBiller,omitempty"`
	Amount          string       `json:"amount,omitempty"`
	ExceptionReason string       `json:"exceptionReason,omitempty"`
	AgeDays         int64        `json:"ageDays"`
	QueuedAt        optionalDate `json:"queuedAt"`
	InvoiceID       string       `json:"invoiceId,omitempty"`
}

// newListBillingQueueItemsTool reads the queue through the repository the
// billing queue page lists from. The page hides posted items unless asked;
// so does this, unless a status filter names Posted.
func newListBillingQueueItemsTool(
	repo repositories.BillingQueueRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return buildBillingQueueList(repo, newFieldAccess(permissions))
}

func buildBillingQueueList(
	lister billingQueueLister,
	access fieldAccess,
) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_billing_queue_items",
		entityPlural: "billing queue items",
		summary: "List billing queue items, the shipments waiting on a biller, by status, " +
			"bill type, exception reason, assigned biller, payer or age. Open one with " +
			"get_billing_queue_item before proposing a decision on it. Posted items are left " +
			"out unless you filter status to Posted.",
		resource: permission.ResourceBillingQueue,
		config:   querybuilder.GetFieldConfiguration((*billingqueue.BillingQueueItem)(nil)),
		fields: []listField{
			{
				Name:   paramStatus,
				Kind:   filterEnum,
				Values: billingQueueStatuses,
				Note: "ReadyForReview and InReview are waiting on a biller; OnHold, Exception " +
					"and SentBackToOps are blocked",
			},
			{Name: "billType", Kind: filterEnum, Values: billTypes},
			{Name: "exceptionReasonCode", Kind: filterEnum, Values: exceptionReasonCodes},
			{
				Name: "assignedBillerId",
				Kind: filterText,
				Note: "a user id; isnull for unassigned",
			},
			{Name: "billToCustomerId", Kind: filterText, Note: "the payer, from list_customers"},
			{Name: fieldNumber, Kind: filterText, Sortable: true},
			{Name: "shipment.proNumber", Kind: filterText},
			{
				Name:     agentRunFieldCreatedAt,
				Kind:     filterDate,
				Sortable: true,
				Note:     "when it was queued; age is how long ago",
			},
			{Name: "reviewStartedAt", Kind: filterDate, Sortable: true},
			{Name: fieldAllocatedTotal, Kind: filterNumber, Sortable: true},
		},
		access: access,
		fetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			gate *fieldGate,
		) ([]any, error) {
			if lister == nil {
				return nil, errortypes.NewConflictError("The billing queue is unavailable")
			}
			result, err := lister.List(ctx, &repositories.ListBillingQueueItemsRequest{
				Filter:        opts,
				IncludePosted: asksForPosted(opts.FieldFilters),
			})
			if err != nil {
				return nil, err
			}

			showAmount := gate.show(fieldAllocatedTotal, fieldAmount)
			now := timeutils.NowUnix()

			return listRows(result.Items, func(item *billingqueue.BillingQueueItem) any {
				return billingQueueRowFrom(item, showAmount, now)
			}), nil
		},
	})
}

// asksForPosted reports whether a status filter could match a posted item,
// which the queue leaves out unless asked.
func asksForPosted(filters []domaintypes.FieldFilter) bool {
	for _, filter := range filters {
		if filter.Field != paramStatus {
			continue
		}
		switch value := filter.Value.(type) {
		case string:
			if strings.EqualFold(value, string(billingqueue.StatusPosted)) {
				return filter.Operator == dbtype.OpEqual
			}
		case []any:
			if filter.Operator == dbtype.OpIn && slices.ContainsFunc(value, isPosted) {
				return true
			}
		case []string:
			if filter.Operator == dbtype.OpIn &&
				slices.Contains(value, string(billingqueue.StatusPosted)) {
				return true
			}
		}
	}

	return false
}

func isPosted(value any) bool {
	text, ok := value.(string)

	return ok && strings.EqualFold(text, string(billingqueue.StatusPosted))
}

func billingQueueRowFrom(
	item *billingqueue.BillingQueueItem,
	showAmount bool,
	now int64,
) billingQueueRow {
	row := billingQueueRow{
		ID:        item.ID.String(),
		Number:    item.Number,
		Status:    string(item.Status),
		BillType:  string(item.BillType),
		AgeDays:   max(now-item.CreatedAt, 0) / secondsPerDayInQueue,
		QueuedAt:  recordedDate(item.CreatedAt),
		InvoiceID: pulidString(item.InvoiceID),
	}
	if item.Shipment != nil {
		row.ProNumber = item.Shipment.ProNumber
	}
	if item.BillToCustomer != nil {
		row.BillTo = item.BillToCustomer.Name
	}
	if item.AssignedBiller != nil {
		row.AssignedBiller = item.AssignedBiller.Name
	}
	if item.ExceptionReasonCode != nil {
		row.ExceptionReason = string(*item.ExceptionReasonCode)
	}
	if showAmount {
		row.Amount = item.AllocatedTotalAmount.StringFixed(2)
	}

	return row
}

type billingQueueReader interface {
	GetByID(
		ctx context.Context,
		req *repositories.GetBillingQueueItemByIDRequest,
	) (*billingqueue.BillingQueueItem, error)
}

type billingReadinessReader interface {
	GetBillingReadiness(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*serviceports.ShipmentBillingReadiness, error)
}

type queueInvoiceReader interface {
	GetByBillingQueueItemID(
		ctx context.Context,
		req repositories.GetInvoiceByBillingQueueItemIDRequest,
	) (*invoice.Invoice, error)
}

type getBillingQueueItemTool struct {
	items     billingQueueReader
	readiness billingReadinessReader
	invoices  queueInvoiceReader
	access    fieldAccess
}

func newGetBillingQueueItemTool(
	items serviceports.BillingQueueService,
	shipments serviceports.ShipmentService,
	invoices repositories.InvoiceRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getBillingQueueItemTool{
		items:     items,
		readiness: shipments,
		invoices:  invoices,
		access:    newFieldAccess(permissions),
	}
}

func (t *getBillingQueueItemTool) Name() string { return "get_billing_queue_item" }

func (t *getBillingQueueItemTool) Description() string {
	return "Open one billing queue item: its payer, shipment, the charges this payer is " +
		"billed, the detention charges holding it, its exception and review notes, what its " +
		"shipment still lacks for billing, and the invoice it made once approved. Read it " +
		"before proposing approve_billing_queue_item, hold_billing_queue_item, " +
		"move_billing_item_to_exception, send_billing_item_back_to_ops or " +
		"cancel_billing_queue_item. Amounts are left out, and named in withheldByAccess, when " +
		"your data access does not reach them."
}

func (t *getBillingQueueItemTool) ParamSchema() map[string]any {
	return idSchema(paramBillingQueueItemID, "The item's id, from list_billing_queue_items, "+
		"this run's subject or "+onThePage)
}

func (t *getBillingQueueItemTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceBillingQueue})
}

type billingQueueChargeLine struct {
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Amount      string `json:"amount,omitempty"`
	ChargeTotal string `json:"chargeTotal,omitempty"`
}

type billingQueueHold struct {
	OccurrenceID string `json:"occurrenceId"`
	Location     string `json:"location,omitempty"`
	Reason       string `json:"reason"`
	Amount       string `json:"amount,omitempty"`
}

type billingQueueReadiness struct {
	MissingDocuments   []string `json:"missingDocuments"`
	ValidationFailures []string `json:"validationFailures"`
	Warnings           []string `json:"warnings"`
}

type billingQueueDetail struct {
	billingQueueRow

	ShipmentID                string                   `json:"shipmentId,omitempty"`
	ShipmentStatus            string                   `json:"shipmentStatus,omitempty"`
	BOL                       string                   `json:"bol,omitempty"`
	BillToCustomerID          string                   `json:"billToCustomerId"`
	AssignedBillerID          string                   `json:"assignedBillerId,omitempty"`
	ReviewNotes               string                   `json:"reviewNotes,omitempty"`
	ExceptionNotes            string                   `json:"exceptionNotes,omitempty"`
	CancelReason              string                   `json:"cancelReason,omitempty"`
	ReviewStartedAt           optionalDate             `json:"reviewStartedAt"`
	RequiresReplacementReview bool                     `json:"requiresReplacementReview,omitempty"`
	SplitBilled               bool                     `json:"splitBilled,omitempty"`
	Charges                   []billingQueueChargeLine `json:"charges"`
	ChargesTruncated          bool                     `json:"chargesTruncated,omitempty"`
	ChargeProblem             string                   `json:"chargeProblem,omitempty"`
	DetentionHolds            []billingQueueHold       `json:"detentionHolds"`
	Readiness                 *billingQueueReadiness   `json:"readiness,omitempty"`
	CanApprove                bool                     `json:"canApprove"`
	ApprovalBlockedBy         string                   `json:"approvalBlockedBy,omitempty"`
	Withheld                  []string                 `json:"withheldByAccess,omitempty"`
}

func (t *getBillingQueueItemTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	id, err := requirePulid(params.Params, paramBillingQueueItemID)
	if err != nil {
		return nil, err
	}
	tenant := tenantOf(params)

	item, err := t.items.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:                id,
		TenantInfo:            tenant,
		ExpandShipmentDetails: true,
	})
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceBillingQueue)
	showAmount := gate.show(fieldAllocatedTotal, withheldAmounts)
	detail := billingQueueDetail{
		billingQueueRow:           billingQueueRowFrom(item, showAmount, timeutils.NowUnix()),
		ShipmentID:                pulidString(item.ShipmentID),
		BillToCustomerID:          item.BillToCustomerID.String(),
		AssignedBillerID:          pointerIDString(item.AssignedBillerID),
		ReviewNotes:               item.ReviewNotes,
		ExceptionNotes:            item.ExceptionNotes,
		CancelReason:              item.CancelReason,
		ReviewStartedAt:           pointerDate(item.ReviewStartedAt),
		RequiresReplacementReview: item.RequiresReplacementReview,
		Charges:                   []billingQueueChargeLine{},
		DetentionHolds:            holdsOf(item.DetentionHolds, showAmount),
	}
	if item.Shipment != nil {
		detail.ShipmentStatus = string(item.Shipment.Status)
		detail.BOL = item.Shipment.BOL
	}
	applyPayerShare(&detail, item.PayerShare, showAmount)
	detail.CanApprove, detail.ApprovalBlockedBy = approvable(item)

	if detail.InvoiceID == "" && item.Status == billingqueue.StatusApproved && t.invoices != nil {
		if inv, invErr := t.invoices.GetByBillingQueueItemID(
			ctx,
			repositories.GetInvoiceByBillingQueueItemIDRequest{
				BillingQueueItemID: item.ID,
				TenantInfo:         tenant,
			},
		); invErr == nil && inv != nil {
			detail.InvoiceID = inv.ID.String()
		} else if invErr != nil && !errortypes.IsNotFoundError(invErr) {
			return nil, invErr
		}
	}

	if item.ShipmentID.IsNotNil() && t.readiness != nil &&
		!billingqueue.IsTerminalStatus(item.Status) {
		readiness, readErr := t.readiness.GetBillingReadiness(ctx, item.ShipmentID, tenant)
		if readErr != nil {
			return nil, readErr
		}
		detail.Readiness = readinessOf(readiness)
	}
	detail.Withheld = gate.Withheld()

	return detail, nil
}

// approvable says whether a person could approve the item now, and if not
// what stands in the way, by the rules approval enforces.
func approvable(item *billingqueue.BillingQueueItem) (canApprove bool, blockedBy string) {
	switch {
	case !billingqueue.IsAllowedTransition(item.Status, billingqueue.StatusApproved) ||
		item.Status == billingqueue.StatusApproved:
		return false, "an item in " + string(item.Status) + " cannot be approved; it must " +
			"be in review"
	case len(item.DetentionHolds) > 0:
		return false, "a detention charge on its shipment is still waiting on approval"
	default:
		return true, ""
	}
}

func applyPayerShare(detail *billingQueueDetail, share *billingqueue.PayerShare, showAmount bool) {
	if share == nil {
		return
	}

	detail.SplitBilled = share.IsSplit
	detail.ChargeProblem = share.ResolutionError
	lines := share.Lines
	if len(lines) > maxBillingQueueLines {
		lines = lines[:maxBillingQueueLines]
		detail.ChargesTruncated = true
	}
	for _, line := range lines {
		if line == nil {
			continue
		}
		charge := billingQueueChargeLine{Kind: string(line.Kind), Description: line.Description}
		if showAmount {
			charge.Amount = line.Amount.StringFixed(2)
			charge.ChargeTotal = line.ChargeTotal.StringFixed(2)
		}
		detail.Charges = append(detail.Charges, charge)
	}
}

func holdsOf(holds []*billingqueue.DetentionHold, showAmount bool) []billingQueueHold {
	out := make([]billingQueueHold, 0, len(holds))
	for _, hold := range holds {
		if hold == nil {
			continue
		}
		entry := billingQueueHold{
			OccurrenceID: hold.OccurrenceID.String(),
			Location:     hold.LocationName,
			Reason:       string(hold.Reason),
		}
		if showAmount {
			entry.Amount = hold.BillableAmount.StringFixed(2)
		}
		out = append(out, entry)
	}

	return out
}

func readinessOf(readiness *serviceports.ShipmentBillingReadiness) *billingQueueReadiness {
	if readiness == nil {
		return nil
	}

	out := &billingQueueReadiness{
		MissingDocuments:   make([]string, 0, len(readiness.MissingRequirements)),
		ValidationFailures: make([]string, 0, len(readiness.ValidationFailures)),
		Warnings:           make([]string, 0, len(readiness.Warnings)),
	}
	for _, requirement := range readiness.MissingRequirements {
		out.MissingDocuments = append(out.MissingDocuments, requirement.DocumentTypeName)
	}
	for _, failure := range readiness.ValidationFailures {
		out.ValidationFailures = append(out.ValidationFailures, failure.Message)
	}
	for _, warning := range readiness.Warnings {
		out.Warnings = append(out.Warnings, warning.Message)
	}

	return out
}
