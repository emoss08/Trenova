package agentquerytoolservice

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/pkg/listquery"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	fieldAllocatedTotal  = "allocatedTotalAmount"
	maxBillingQueueLines = 50
	secondsPerDayInQueue = 86400
	fieldNumber          = "number"
	fieldCreatedAt       = "createdAt"
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
		req *billingqueueports.ListBillingQueueItemsRequest,
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
	repo billingqueueports.BillingQueueRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return buildBillingQueueList(repo, newFieldAccess(permissions))
}

func buildBillingQueueList(
	lister billingQueueLister,
	access fieldAccess,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "billing_list_queue_items",
		entityPlural: "billing queue items",
		summary: "List billing queue items, the shipments waiting on a biller. Open one with " +
			"billing_get_queue_item before deciding it; Posted items need status Posted.",
		resource: permission.ResourceBillingQueue,
		config: listquery.Describe(
			(*billingqueue.BillingQueueItem)(nil),
			billingqueuedb.BillingQueueItemsSchema(),
		),
		fields: []listField{
			{
				Name:   paramStatus,
				Kind:   filterEnum,
				Values: billingQueueStatuses,
				Note:   "ReadyForReview and InReview await a biller",
			},
			{Name: "billType", Kind: filterEnum, Values: billTypes},
			{Name: "exceptionReasonCode", Kind: filterEnum, Values: exceptionReasonCodes},
			{
				Name: "assignedBillerId",
				Kind: filterText,
				Note: "a user id; isnull for unassigned",
			},
			{Name: "billToCustomerId", Kind: filterText, Note: "the payer, from customer_list"},
			{
				Name: "id",
				Kind: filterText,
				Note: "in with several ids reads them together",
			},
			{Name: fieldNumber, Kind: filterText, Sortable: true},
			{Name: "shipment.proNumber", Kind: filterText},
			{
				Name:     fieldCreatedAt,
				Kind:     filterDate,
				Sortable: true,
				Note:     "when it was queued",
			},
			{Name: "reviewStartedAt", Kind: filterDate, Sortable: true},
			{Name: fieldAllocatedTotal, Kind: filterNumber, Sortable: true},
		},
		access: access,
		fetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			_ pagination.CursorInfo,
			gate *fieldGate,
		) (listPage, error) {
			if lister == nil {
				return listPage{}, errortypes.NewConflictError("The billing queue is unavailable")
			}
			result, err := lister.List(ctx, &billingqueueports.ListBillingQueueItemsRequest{
				Filter:        opts,
				IncludePosted: asksForPosted(opts.FieldFilters),
			})
			if err != nil {
				return listPage{}, err
			}

			showAmount := gate.show(fieldAllocatedTotal, fieldAmount)
			now := timeutils.NowUnix()

			return offsetListPage(result.Items, func(item *billingqueue.BillingQueueItem) any {
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
		req *billingqueueports.GetBillingQueueItemByIDRequest,
	) (*billingqueue.BillingQueueItem, error)
}

type billingReadinessReader interface {
	GetBillingReadiness(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*shipmentports.ShipmentBillingReadiness, error)
}

type queueInvoiceReader interface {
	GetByBillingQueueItemID(
		ctx context.Context,
		req invoiceports.GetInvoiceByBillingQueueItemIDRequest,
	) (*invoice.Invoice, error)
}

type getBillingQueueItemTool struct {
	items     billingQueueReader
	readiness billingReadinessReader
	invoices  queueInvoiceReader
	access    fieldAccess
}

func newGetBillingQueueItemTool(
	items billingqueueports.BillingQueueService,
	shipments shipmentports.ShipmentService,
	invoices invoiceports.InvoiceRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &getBillingQueueItemTool{
		items:     items,
		readiness: shipments,
		invoices:  invoices,
		access:    newFieldAccess(permissions),
	}
}

func (t *getBillingQueueItemTool) Name() string { return "billing_get_queue_item" }

func (t *getBillingQueueItemTool) Description() string {
	return "Open billing queue items by id: one in full, or up to 50 as a table of what " +
		"blocks each item's approval and what its shipment still lacks for billing. " +
		"One item comes with its payer, shipment and the charges this payer is billed, the " +
		"detention charges holding it, its exception and review notes, and the invoice it " +
		"made once approved. Its issues are the flagged checks, each with the issueId, " +
		"whether it is Open or Settled, the options a biller settles it with by optionKey, " +
		"and whether its settlement can be undone. Read an item before approving, holding, " +
		"releasing, canceling or sending it back, or settling one of its issues; read the set " +
		"before assigning, moving into review, approving or posting several. An id that is " +
		"not a queue item of this organization is named in the note. Amounts are left out, " +
		"and named in withheldByAccess, when your data access does not reach them."
}

func (t *getBillingQueueItemTool) SearchTerms() []string {
	return []string{"billing queue", "readiness"}
}

func (t *getBillingQueueItemTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramBillingQueueItemIDs: agenttoolschema.KindIDs("The items' ids, from "+
				"billing_list_queue_items, this run's subject, "+onThePage+" or the "+
				"proposals that named them.", maxBatchBillingQueueItems,
				permission.KindBillingQueueItem),
		},
		toolschema.KeyRequired:             []string{paramBillingQueueItemIDs},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *getBillingQueueItemTool) Policy() agentports.ToolPolicy {
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
	RatingReview              []string                 `json:"ratingReview,omitempty"`
	Issues                    []billingQueueIssue      `json:"issues"`
	Readiness                 *billingQueueReadiness   `json:"readiness,omitempty"`
	CanApprove                bool                     `json:"canApprove"`
	ApprovalBlockedBy         string                   `json:"approvalBlockedBy,omitempty"`
	Withheld                  []string                 `json:"withheldByAccess,omitempty"`
}

func (t *getBillingQueueItemTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	ids, err := requireIDList(params.Params, paramBillingQueueItemIDs, maxBatchBillingQueueItems)
	if err != nil {
		return nil, err
	}
	ids = sliceutils.Dedupe(ids)

	gate := t.access.gate(ctx, params, permission.ResourceBillingQueue)
	if len(ids) == 1 {
		detail, detailErr := t.reader().detail(ctx, tenantOf(params), ids[0], gate)
		if detailErr != nil {
			return nil, detailErr
		}
		detail.Withheld = gate.Withheld()

		return *detail, nil
	}

	return t.several(ctx, params, ids, gate)
}

func (t *getBillingQueueItemTool) reader() queueDetailReader {
	return queueDetailReader{items: t.items, readiness: t.readiness, invoices: t.invoices}
}

// queueDetailReader builds the detail of one item the way billing_get_queue_item
// shows it, for that tool and for the batch read that shows several.
type queueDetailReader struct {
	items     billingQueueReader
	readiness billingReadinessReader
	invoices  queueInvoiceReader
}

func (r queueDetailReader) detail(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	gate *fieldGate,
) (*billingQueueDetail, error) {
	item, err := r.items.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:                id,
		TenantInfo:            tenant,
		ExpandShipmentDetails: true,
	})
	if err != nil {
		return nil, err
	}

	showAmount := gate.show(fieldAllocatedTotal, withheldAmounts)
	detail := &billingQueueDetail{
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
		Issues:                    issuesOf(item, showAmount),
	}
	if hold := item.RatingReviewHold; hold != nil {
		detail.RatingReview = hold.Reasons
	}
	if item.Shipment != nil {
		detail.ShipmentStatus = string(item.Shipment.Status)
		detail.BOL = item.Shipment.BOL
	}
	applyPayerShare(detail, item.PayerShare, showAmount)
	detail.CanApprove, detail.ApprovalBlockedBy = approvable(item)

	if detail.InvoiceID == "" && item.Status == billingqueue.StatusApproved && r.invoices != nil {
		if inv, invErr := r.invoices.GetByBillingQueueItemID(
			ctx,
			invoiceports.GetInvoiceByBillingQueueItemIDRequest{
				BillingQueueItemID: item.ID,
				TenantInfo:         tenant,
			},
		); invErr == nil && inv != nil {
			detail.InvoiceID = inv.ID.String()
		} else if invErr != nil && !errortypes.IsNotFoundError(invErr) {
			return nil, invErr
		}
	}

	if item.ShipmentID.IsNotNil() && r.readiness != nil &&
		!billingqueue.IsTerminalStatus(item.Status) {
		readiness, readErr := r.readiness.GetBillingReadiness(ctx, item.ShipmentID, tenant)
		if readErr != nil {
			return nil, readErr
		}
		detail.Readiness = readinessOf(readiness)
	}

	return detail, nil
}

// approvable says whether a person could approve the item now, and if not
// what stands in the way, by the rules approval enforces.
func approvable(item *billingqueue.BillingQueueItem) (canApprove bool, blockedBy string) {
	switch {
	case item.Status == billingqueue.StatusApproved:
		return false, "it is already approved"
	case item.Status != billingqueue.StatusReadyForReview &&
		!billingqueue.IsAllowedTransition(item.Status, billingqueue.StatusApproved):
		return false, "an item in " + string(item.Status) + " cannot be approved"
	case len(item.DetentionHolds) > 0:
		return false, "a detention charge on its shipment is still waiting on approval"
	case item.RatingReviewHold != nil:
		return false, "the tariff held its shipment's rating for review; a person approves " +
			"the rating before it can be approved"
	case len(item.OpenIssues()) > 0:
		return false, "a flagged check on it has not been settled; a biller settles it in " +
			"the item's review before it can be approved"
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

type billingQueueIssueOption struct {
	OptionKey string `json:"optionKey"`
	Label     string `json:"label"`
	Amount    string `json:"amount,omitempty"`
}

type billingQueueIssue struct {
	IssueID    string                    `json:"issueId"`
	Check      string                    `json:"check"`
	Code       string                    `json:"code"`
	Summary    string                    `json:"summary"`
	Reasoning  string                    `json:"reasoning,omitempty"`
	State      string                    `json:"state"`
	Resolution string                    `json:"resolution,omitempty"`
	Undoable   bool                      `json:"undoable,omitempty"`
	AskedAt    optionalDate              `json:"driverAskedAt"`
	Options    []billingQueueIssueOption `json:"options"`
}

func issuesOf(item *billingqueue.BillingQueueItem, showAmount bool) []billingQueueIssue {
	if item.Review == nil {
		return []billingQueueIssue{}
	}

	out := make([]billingQueueIssue, 0, len(item.Review.Issues))
	for _, issue := range item.Review.Issues {
		if issue == nil {
			continue
		}
		priced := showAmount || issue.CheckKey != billingqueue.CheckCharges
		entry := billingQueueIssue{
			IssueID:  issue.ID.String(),
			Check:    string(issue.CheckKey),
			Code:     string(issue.Code),
			Summary:  issue.Summary,
			State:    issueState(issue),
			Undoable: issue.Undoable,
			AskedAt:  pointerDateOr(issue.RequestedAt, "not asked"),
			Options:  make([]billingQueueIssueOption, 0, len(issue.Options)),
		}
		if priced {
			entry.Reasoning = issue.Reasoning
		} else {
			entry.Summary = unpricedIssueSummary(issue.Code)
		}
		if !issue.IsOpen() {
			if priced {
				entry.Resolution = issue.ResolutionText
			}
			out = append(out, entry)
			continue
		}
		for idx := range issue.Options {
			option := &issue.Options[idx]
			choice := billingQueueIssueOption{OptionKey: option.Key, Label: option.Label}
			switch {
			case !priced:
				choice.Label = unpricedOptionLabel(option.Effect.Kind)
			case option.Amount.Valid:
				choice.Amount = option.Amount.Decimal.StringFixed(2)
			}
			entry.Options = append(entry.Options, choice)
		}
		out = append(out, entry)
	}

	return out
}

func unpricedIssueSummary(code billingqueue.IssueCode) string {
	switch code { //nolint:exhaustive // the charge checks whose wording carries amounts
	case billingqueue.IssueAccessorialNotOnRateCon:
		return "An accessorial isn't on the rate con"
	case billingqueue.IssueChargeOverRateCon:
		return "A charge is over the rate con"
	case billingqueue.IssueDetentionELDMismatch:
		return "Detention doesn't match the ELD"
	default:
		return "A charge needs a biller's decision"
	}
}

func unpricedOptionLabel(kind billingqueue.EffectKind) string {
	switch kind {
	case billingqueue.EffectKeep:
		return "Keep it as billed"
	case billingqueue.EffectDrop:
		return "Remove it"
	case billingqueue.EffectSet:
		return "Reprice it"
	case billingqueue.EffectRequest:
		return "Ask the driver for it"
	case billingqueue.EffectAccept:
		return "Bill it anyway"
	default:
		return string(kind)
	}
}

func issueState(issue *billingqueue.Issue) string {
	switch {
	case issue.IsOpen():
		return "Open"
	case issue.Settled():
		return "Settled"
	default:
		return "Cleared"
	}
}

func readinessOf(readiness *shipmentports.ShipmentBillingReadiness) *billingQueueReadiness {
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
