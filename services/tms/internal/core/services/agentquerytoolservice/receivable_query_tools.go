package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/invoicerunservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	defaultReceivableRows   = 25
	maxReceivableRows       = 50
	maxReceivableInvoices   = 50
	maxRunItemsShown        = 500
	paramAdjustmentID       = "adjustmentId"
	paramInvoiceRunID       = "invoiceRunId"
	paramReceivableInvoice  = "invoiceId"
	paramReceivableInvoices = "invoiceIds"
)

type adjustmentReader interface {
	GetDetail(
		ctx context.Context,
		req *serviceports.GetInvoiceAdjustmentDetailRequest,
	) (*invoiceadjustment.InvoiceAdjustment, error)
	GetLineage(
		ctx context.Context,
		req *serviceports.GetInvoiceAdjustmentLineageRequest,
	) (*serviceports.InvoiceAdjustmentLineage, error)
	ListApprovals(
		ctx context.Context,
		req *repositories.ListApprovalQueueRequest,
	) (*pagination.CursorListResult[*repositories.InvoiceAdjustmentApprovalQueueItem], error)
}

type disputeLister interface {
	ListByInvoiceIDs(
		ctx context.Context,
		req *repositories.ListInvoiceDisputesByInvoiceIDsRequest,
	) (map[pulid.ID][]*invoice.InvoiceDispute, error)
}

type creditApplicationLister interface {
	ListCreditMemoApplicationsByInvoiceIDs(
		ctx context.Context,
		req *repositories.ListApplicationsByInvoiceIDsRequest,
	) (map[pulid.ID][]*customerpayment.CreditMemoApplication, error)
}

type invoiceRunReader interface {
	Get(
		ctx context.Context,
		req repositories.GetInvoiceRunByIDRequest,
	) (*invoicerun.InvoiceRun, error)
	List(
		ctx context.Context,
		req *repositories.ListInvoiceRunsRequest,
	) (*pagination.ListResult[*invoicerun.InvoiceRun], error)
	ListOpenStatements(
		ctx context.Context,
		req *serviceports.ListOpenStatementsRequest,
	) ([]*serviceports.OpenStatement, error)
}

type shareCandidateLister interface {
	ListCandidates(
		ctx context.Context,
		req *serviceports.ListShareCandidatesRequest,
	) (*pagination.ListResult[*serviceports.ShareCandidate], error)
}

type receivableList[T any] struct {
	Items    []T      `json:"items"`
	Count    int      `json:"count"`
	Note     string   `json:"note,omitempty"`
	Withheld []string `json:"withheldByAccess,omitempty"`
}

func newReceivableList[T any](items []T, gate *fieldGate) *receivableList[T] {
	list := &receivableList[T]{Items: items, Count: len(items)}
	if gate != nil {
		list.Withheld = gate.Withheld()
	}

	return list
}

type amounts struct {
	show bool
}

func amountsFor(
	ctx context.Context,
	access fieldAccess,
	params *serviceports.QueryToolParams,
) (amounts, *fieldGate) {
	gate := access.gate(ctx, params, permission.ResourceInvoice)

	return amounts{show: gate.show(fieldTotalAmount, withheldAmounts)}, gate
}

func (a amounts) of(value decimal.Decimal) string {
	if !a.show {
		return ""
	}

	return value.StringFixed(2)
}

func (a amounts) minor(value int64) string {
	return a.of(money.DecimalFromMinor(value))
}

func idListProperty(description string, limit int) map[string]any {
	return map[string]any{
		"type":        "array",
		"items":       map[string]any{"type": "string"},
		"minItems":    1,
		"maxItems":    limit,
		"description": description,
	}
}

func limitProperty() map[string]any {
	return map[string]any{
		"type":        "integer",
		"description": fmt.Sprintf("How many to return, at most %d.", maxReceivableRows),
	}
}

func receivableLimit(params map[string]any) int {
	limit := optionalInt(params, "limit", defaultReceivableRows)
	if limit <= 0 || limit > maxReceivableRows {
		return maxReceivableRows
	}

	return limit
}

func requireIDList(params map[string]any, key string, limit int) ([]pulid.ID, error) {
	ids, err := optionalPulidSlice(params, key, limit)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("parameter %q needs at least one id", key)
	}

	return ids, nil
}

type adjustmentRow struct {
	ID                   string       `json:"id"`
	InvoiceID            string       `json:"invoiceId"`
	InvoiceNumber        string       `json:"invoiceNumber,omitempty"`
	Kind                 string       `json:"kind"`
	Status               string       `json:"status"`
	ApprovalStatus       string       `json:"approvalStatus,omitempty"`
	RebillStrategy       string       `json:"rebillStrategy,omitempty"`
	Reason               string       `json:"reason,omitempty"`
	Credited             string       `json:"creditedAmount,omitempty"`
	Rebilled             string       `json:"rebilledAmount,omitempty"`
	CreditMemoInvoiceID  string       `json:"creditMemoInvoiceId,omitempty"`
	ReplacementInvoiceID string       `json:"replacementInvoiceId,omitempty"`
	SubmittedAt          optionalDate `json:"submittedAt"`
}

type listInvoiceAdjustmentsTool struct {
	adjustments adjustmentReader
	invoices    invoiceReader
	access      fieldAccess
}

func newListInvoiceAdjustmentsTool(
	adjustments adjustmentReader,
	invoices invoiceReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listInvoiceAdjustmentsTool{
		adjustments: adjustments,
		invoices:    invoices,
		access:      newFieldAccess(permissions),
	}
}

func (t *listInvoiceAdjustmentsTool) Name() string { return "list_invoice_adjustments" }

func (t *listInvoiceAdjustmentsTool) Description() string {
	return "List the invoice adjustments on one invoice, drafts included, or those waiting " +
		"for an approver. With invoiceId it lists every credit, rebill and reversal made or " +
		"drafted against the invoice and its replacements; without it, the approval queue. " +
		"Open one with get_invoice_adjustment."
}

func (t *listInvoiceAdjustmentsTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramReceivableInvoice: stringParam("Optional: the invoice, from list_invoices or " +
			"get_invoice. Leave it out for the approval queue."),
		"limit": limitProperty(),
	})
}

func (t *listInvoiceAdjustmentsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceInvoice})
}

func (t *listInvoiceAdjustmentsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	invoiceID, hasInvoice, err := optionalPulid(params.Params, paramReceivableInvoice)
	if err != nil {
		return nil, err
	}
	show, gate := amountsFor(ctx, t.access, params)
	if hasInvoice {
		return t.lineage(ctx, params, invoiceID, show, gate)
	}

	limit := receivableLimit(params.Params)
	cursor, err := pagination.NewCursorInfo(limit, "")
	if err != nil {
		return nil, err
	}
	queue, err := t.adjustments.ListApprovals(ctx, &repositories.ListApprovalQueueRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantOf(params),
			Pagination: pagination.Info{Limit: limit},
		},
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]adjustmentRow, 0, len(queue.Items))
	for _, item := range queue.Items {
		if item == nil {
			continue
		}
		rows = append(rows, adjustmentRow{
			ID:             item.AdjustmentID.String(),
			InvoiceID:      item.OriginalInvoiceID.String(),
			InvoiceNumber:  item.OriginalInvoiceNumber,
			Kind:           string(item.Kind),
			Status:         string(item.Status),
			ApprovalStatus: string(item.ApprovalStatus),
			RebillStrategy: string(item.RebillStrategy),
			Reason:         item.Reason,
			Credited:       show.of(item.CreditTotalAmount.Abs()),
			Rebilled:       show.of(item.RebillTotalAmount),
			SubmittedAt:    pointerDate(item.SubmittedAt),
		})
	}

	return newReceivableList(rows, gate), nil
}

func (t *listInvoiceAdjustmentsTool) lineage(
	ctx context.Context,
	params *serviceports.QueryToolParams,
	invoiceID pulid.ID,
	show amounts,
	gate *fieldGate,
) (any, error) {
	entity, err := t.invoices.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         invoiceID,
		TenantInfo: tenantOf(params),
	})
	if err != nil {
		return nil, err
	}
	if entity.CorrectionGroupID.IsNil() {
		list := newReceivableList([]adjustmentRow{}, gate)
		list.Note = "Nothing has been adjusted or drafted against this invoice."
		return list, nil
	}

	lineage, err := t.adjustments.GetLineage(ctx, &serviceports.GetInvoiceAdjustmentLineageRequest{
		CorrectionGroupID: entity.CorrectionGroupID,
		TenantInfo:        tenantOf(params),
	})
	if err != nil {
		return nil, err
	}

	numbers := make(map[pulid.ID]string, len(lineage.Invoices))
	for _, related := range lineage.Invoices {
		if related != nil {
			numbers[related.ID] = related.Number
		}
	}
	rows := make([]adjustmentRow, 0, len(lineage.Adjustments))
	for _, adjustment := range lineage.Adjustments {
		if adjustment == nil {
			continue
		}
		row := adjustmentRowFrom(adjustment, show)
		row.InvoiceNumber = numbers[adjustment.OriginalInvoiceID]
		rows = append(rows, row)
	}

	return newReceivableList(rows, gate), nil
}

func adjustmentRowFrom(
	adjustment *invoiceadjustment.InvoiceAdjustment,
	show amounts,
) adjustmentRow {
	return adjustmentRow{
		ID:                   adjustment.ID.String(),
		InvoiceID:            adjustment.OriginalInvoiceID.String(),
		Kind:                 string(adjustment.Kind),
		Status:               string(adjustment.Status),
		ApprovalStatus:       string(adjustment.ApprovalStatus),
		RebillStrategy:       string(adjustment.RebillStrategy),
		Reason:               adjustment.Reason,
		Credited:             show.of(adjustment.CreditTotalAmount.Abs()),
		Rebilled:             show.of(adjustment.RebillTotalAmount),
		CreditMemoInvoiceID:  pulidString(adjustment.CreditMemoInvoiceID),
		ReplacementInvoiceID: pulidString(adjustment.ReplacementInvoiceID),
		SubmittedAt:          pointerDate(adjustment.SubmittedAt),
	}
}

type adjustmentLineRow struct {
	ID              string `json:"id"`
	InvoiceLineID   string `json:"invoiceLineId"`
	LineNumber      int    `json:"lineNumber"`
	Description     string `json:"description"`
	CreditQuantity  string `json:"creditQuantity,omitempty"`
	Credit          string `json:"creditAmount,omitempty"`
	RebillQuantity  string `json:"rebillQuantity,omitempty"`
	Rebill          string `json:"rebillAmount,omitempty"`
	RemainingCredit string `json:"remainingEligibleAmount,omitempty"`
}

type adjustmentDetail struct {
	adjustmentRow

	PolicyReason    string              `json:"policyReason,omitempty"`
	RejectionReason string              `json:"rejectionReason,omitempty"`
	ExecutionError  string              `json:"executionError,omitempty"`
	ApprovalNeeded  bool                `json:"approvalRequired"`
	Lines           []adjustmentLineRow `json:"lines"`
	DocumentIDs     []string            `json:"supportingDocumentIds,omitempty"`
	Withheld        []string            `json:"withheldByAccess,omitempty"`
}

type getInvoiceAdjustmentTool struct {
	adjustments adjustmentReader
	access      fieldAccess
}

func newGetInvoiceAdjustmentTool(
	adjustments adjustmentReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getInvoiceAdjustmentTool{adjustments: adjustments, access: newFieldAccess(permissions)}
}

func (t *getInvoiceAdjustmentTool) Name() string { return "get_invoice_adjustment" }

func (t *getInvoiceAdjustmentTool) Description() string {
	return "Retrieve one invoice adjustment by id: its kind, status, approval, reason, the " +
		"invoices it made and each line it credits or rebills against the invoice's lines."
}

func (t *getInvoiceAdjustmentTool) ParamSchema() map[string]any {
	return idSchema(paramAdjustmentID, "The adjustment's id, from list_invoice_adjustments.")
}

func (t *getInvoiceAdjustmentTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceInvoice})
}

func (t *getInvoiceAdjustmentTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	id, err := requirePulid(params.Params, paramAdjustmentID)
	if err != nil {
		return nil, err
	}

	adjustment, err := t.adjustments.GetDetail(ctx, &serviceports.GetInvoiceAdjustmentDetailRequest{
		AdjustmentID: id,
		TenantInfo:   tenantOf(params),
	})
	if err != nil {
		return nil, err
	}

	show, gate := amountsFor(ctx, t.access, params)
	detail := &adjustmentDetail{
		adjustmentRow:   adjustmentRowFrom(adjustment, show),
		PolicyReason:    adjustment.PolicyReason,
		RejectionReason: adjustment.RejectionReason,
		ExecutionError:  adjustment.ExecutionError,
		ApprovalNeeded:  adjustment.ApprovalRequired,
		Lines:           make([]adjustmentLineRow, 0, len(adjustment.Lines)),
	}
	for _, line := range adjustment.Lines {
		if line == nil {
			continue
		}
		detail.Lines = append(detail.Lines, adjustmentLineRow{
			ID:              line.ID.String(),
			InvoiceLineID:   line.OriginalLineID.String(),
			LineNumber:      line.LineNumber,
			Description:     line.Description,
			CreditQuantity:  nonZero(line.CreditQuantity),
			Credit:          show.of(line.CreditAmount.Abs()),
			RebillQuantity:  nonZero(line.RebillQuantity),
			Rebill:          show.of(line.RebillAmount),
			RemainingCredit: show.of(line.RemainingEligibleAmount),
		})
	}
	for _, reference := range adjustment.DocumentReferences {
		if reference != nil {
			detail.DocumentIDs = append(detail.DocumentIDs, reference.DocumentID.String())
		}
	}
	detail.Withheld = gate.Withheld()

	return detail, nil
}

func nonZero(value decimal.Decimal) string {
	if value.IsZero() {
		return ""
	}

	return value.String()
}

type disputeRow struct {
	ID                     string       `json:"id"`
	InvoiceID              string       `json:"invoiceId"`
	Status                 string       `json:"status"`
	ReasonCode             string       `json:"reasonCode"`
	DisputedAmount         string       `json:"disputedAmount,omitempty"`
	Notes                  string       `json:"notes,omitempty"`
	OpenedAt               optionalDate `json:"openedAt"`
	Resolution             string       `json:"resolution,omitempty"`
	ResolutionAdjustmentID string       `json:"resolutionAdjustmentId,omitempty"`
	ResolutionNotes        string       `json:"resolutionNotes,omitempty"`
	ResolvedAt             optionalDate `json:"resolvedAt"`
}

type listInvoiceDisputesTool struct {
	disputes disputeLister
	access   fieldAccess
}

func newListInvoiceDisputesTool(
	disputes disputeLister,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listInvoiceDisputesTool{disputes: disputes, access: newFieldAccess(permissions)}
}

func (t *listInvoiceDisputesTool) Name() string { return "list_invoice_disputes" }

func (t *listInvoiceDisputesTool) Description() string {
	return "List the dispute cases on invoices, open and closed, newest first: the reason, " +
		"the amount in dispute, the notes and how each was resolved. Find disputed invoices " +
		"with list_invoices filtered to disputeStatus Disputed."
}

func (t *listInvoiceDisputesTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramReceivableInvoices: idListProperty("The invoices, from list_invoices or get_invoice.",
			maxReceivableInvoices),
	}, paramReceivableInvoices)
}

func (t *listInvoiceDisputesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceInvoiceDispute})
}

func (t *listInvoiceDisputesTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	ids, err := requireIDList(params.Params, paramReceivableInvoices, maxReceivableInvoices)
	if err != nil {
		return nil, err
	}

	byInvoice, err := t.disputes.ListByInvoiceIDs(ctx,
		&repositories.ListInvoiceDisputesByInvoiceIDsRequest{
			TenantInfo: tenantOf(params),
			InvoiceIDs: ids,
		})
	if err != nil {
		return nil, err
	}

	show, gate := amountsFor(ctx, t.access, params)
	rows := make([]disputeRow, 0, len(ids))
	for _, id := range ids {
		for _, dispute := range byInvoice[id] {
			if dispute == nil {
				continue
			}
			rows = append(rows, disputeRow{
				ID:                     dispute.ID.String(),
				InvoiceID:              dispute.InvoiceID.String(),
				Status:                 string(dispute.Status),
				ReasonCode:             string(dispute.ReasonCode),
				DisputedAmount:         show.of(dispute.DisputedAmount),
				Notes:                  dispute.Notes,
				OpenedAt:               recordedDate(dispute.OpenedAt),
				Resolution:             string(dispute.Resolution),
				ResolutionAdjustmentID: pulidString(dispute.ResolutionAdjustmentID),
				ResolutionNotes:        dispute.ResolutionNotes,
				ResolvedAt:             pointerDate(dispute.ResolvedAt),
			})
		}
	}

	return newReceivableList(rows, gate), nil
}

type creditApplicationRow struct {
	ID                  string       `json:"id"`
	CreditMemoInvoiceID string       `json:"creditMemoInvoiceId"`
	InvoiceID           string       `json:"invoiceId"`
	Amount              string       `json:"amount,omitempty"`
	Status              string       `json:"status"`
	AccountingDate      optionalDate `json:"accountingDate"`
	UnappliedAt         optionalDate `json:"unappliedAt"`
	UnappliedReason     string       `json:"unappliedReason,omitempty"`
}

type listCreditMemoApplicationsTool struct {
	applications creditApplicationLister
	access       fieldAccess
}

func newListCreditMemoApplicationsTool(
	applications creditApplicationLister,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listCreditMemoApplicationsTool{
		applications: applications,
		access:       newFieldAccess(permissions),
	}
}

func (t *listCreditMemoApplicationsTool) Name() string {
	return "list_credit_memo_applications"
}

func (t *listCreditMemoApplicationsTool) Description() string {
	return "List where credit memos were applied to the invoices or credit memos you name. " +
		"Each application shows the amount and whether it still stands or was taken back; " +
		"unapply_credit_memo takes one back."
}

func (t *listCreditMemoApplicationsTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramReceivableInvoices: idListProperty("Invoices or credit memos, from list_invoices "+
			"or get_invoice.", maxReceivableInvoices),
	}, paramReceivableInvoices)
}

func (t *listCreditMemoApplicationsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceInvoice})
}

func (t *listCreditMemoApplicationsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	ids, err := requireIDList(params.Params, paramReceivableInvoices, maxReceivableInvoices)
	if err != nil {
		return nil, err
	}

	byInvoice, err := t.applications.ListCreditMemoApplicationsByInvoiceIDs(ctx,
		&repositories.ListApplicationsByInvoiceIDsRequest{
			TenantInfo: tenantOf(params),
			InvoiceIDs: ids,
		})
	if err != nil {
		return nil, err
	}

	show, gate := amountsFor(ctx, t.access, params)
	seen := make(map[pulid.ID]struct{}, len(ids))
	rows := make([]creditApplicationRow, 0, len(ids))
	for _, id := range ids {
		for _, application := range byInvoice[id] {
			if application == nil {
				continue
			}
			if _, dup := seen[application.ID]; dup {
				continue
			}
			seen[application.ID] = struct{}{}
			rows = append(rows, creditApplicationRow{
				ID:                  application.ID.String(),
				CreditMemoInvoiceID: application.CreditMemoInvoiceID.String(),
				InvoiceID:           application.InvoiceID.String(),
				Amount:              show.minor(application.AppliedAmountMinor),
				Status:              string(application.Status),
				AccountingDate:      recordedDate(application.AccountingDate),
				UnappliedAt:         pointerDate(application.UnappliedAt),
				UnappliedReason:     application.UnappliedReason,
			})
		}
	}

	return newReceivableList(rows, gate), nil
}

type invoiceRunRow struct {
	ID            string       `json:"id"`
	Number        string       `json:"number"`
	Status        string       `json:"status"`
	Source        string       `json:"source"`
	Cycle         string       `json:"cycle,omitempty"`
	PeriodStart   optionalDate `json:"periodStart"`
	PeriodEnd     optionalDate `json:"periodEnd"`
	InvoiceDate   optionalDate `json:"invoiceDate"`
	GroupCount    int          `json:"groupCount"`
	ItemCount     int          `json:"itemCount"`
	ExcludedCount int          `json:"excludedCount"`
	InvoiceCount  int          `json:"invoiceCount"`
	Total         string       `json:"totalAmount,omitempty"`
	Currency      string       `json:"currency,omitempty"`
	FailureReason string       `json:"failureReason,omitempty"`
}

func invoiceRunRowFrom(run *invoicerun.InvoiceRun, show amounts) invoiceRunRow {
	return invoiceRunRow{
		ID:            run.ID.String(),
		Number:        run.Number,
		Status:        string(run.Status),
		Source:        string(run.Source),
		Cycle:         string(run.Cycle),
		PeriodStart:   recordedDate(run.PeriodStart),
		PeriodEnd:     recordedDate(run.PeriodEnd),
		InvoiceDate:   recordedDate(run.InvoiceDate),
		GroupCount:    run.GroupCount,
		ItemCount:     run.ItemCount,
		ExcludedCount: run.ExcludedCount,
		InvoiceCount:  run.InvoiceCount,
		Total:         show.of(run.TotalAmount),
		Currency:      run.CurrencyCode,
		FailureReason: run.FailureReason,
	}
}

func newListInvoiceRunsTool(
	runs invoiceRunReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	access := newFieldAccess(permissions)

	return newListTool(listSpec{
		name:         "list_invoice_runs",
		entityPlural: "invoice runs",
		summary: "List invoice runs, the reviewable batches of consolidated invoices for a " +
			"billing period, by status, number or period. Open one with get_invoice_run to " +
			"see its groups and items.",
		resource: permission.ResourceInvoiceRun,
		config:   querybuilder.GetFieldConfiguration((*invoicerun.InvoiceRun)(nil)),
		fields: []listField{
			{
				Name: "status",
				Kind: filterEnum,
				Values: []string{
					string(invoicerun.StatusBuilding),
					string(invoicerun.StatusReady),
					string(invoicerun.StatusCommitting),
					string(invoicerun.StatusCommitted),
					string(invoicerun.StatusFailed),
					string(invoicerun.StatusCanceled),
				},
				Note: "Ready runs are waiting to be committed or canceled",
			},
			{Name: "number", Kind: filterText, Sortable: true},
			{Name: "periodEnd", Kind: filterDate, Sortable: true},
			{Name: "createdAt", Kind: filterDate, Sortable: true},
		},
		access: access,
		fetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			gate *fieldGate,
		) ([]any, error) {
			result, err := runs.List(ctx, &repositories.ListInvoiceRunsRequest{Filter: opts})
			if err != nil {
				return nil, err
			}
			show := amounts{show: gate.show(fieldTotalAmount, fieldTotalAmount)}

			return listRows(result.Items, func(run *invoicerun.InvoiceRun) any {
				return invoiceRunRowFrom(run, show)
			}), nil
		},
	})
}

type invoiceRunItemRow struct {
	ID                 string `json:"id"`
	BillingQueueItemID string `json:"billingQueueItemId"`
	ShipmentID         string `json:"shipmentId"`
	ProNumber          string `json:"proNumber,omitempty"`
	Amount             string `json:"amount,omitempty"`
	Excluded           bool   `json:"excluded"`
	ExclusionReason    string `json:"exclusionReason,omitempty"`
}

type invoiceRunGroupRow struct {
	ID         string              `json:"id"`
	CustomerID string              `json:"customerId"`
	Label      string              `json:"label"`
	Status     string              `json:"status"`
	ItemCount  int                 `json:"itemCount"`
	Total      string              `json:"totalAmount,omitempty"`
	AutoBill   bool                `json:"autoBill"`
	InvoiceID  string              `json:"invoiceId,omitempty"`
	SkipReason string              `json:"skipReason,omitempty"`
	Items      []invoiceRunItemRow `json:"items"`
}

type invoiceRunDetail struct {
	invoiceRunRow

	OffCycleReason string               `json:"offCycleReason,omitempty"`
	Groups         []invoiceRunGroupRow `json:"groups"`
	ItemsTruncated bool                 `json:"itemsTruncated,omitempty"`
	Withheld       []string             `json:"withheldByAccess,omitempty"`
}

type getInvoiceRunTool struct {
	runs   invoiceRunReader
	access fieldAccess
}

func newGetInvoiceRunTool(
	runs invoiceRunReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getInvoiceRunTool{runs: runs, access: newFieldAccess(permissions)}
}

func (t *getInvoiceRunTool) Name() string { return "get_invoice_run" }

func (t *getInvoiceRunTool) Description() string {
	return "Retrieve one invoice run by id with each group, the invoice it would make or made, " +
		"and the items in it: the ids adjust_invoice_run_membership moves or excludes."
}

func (t *getInvoiceRunTool) ParamSchema() map[string]any {
	return idSchema(paramInvoiceRunID, "The run's id, from list_invoice_runs.")
}

func (t *getInvoiceRunTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceInvoiceRun})
}

func (t *getInvoiceRunTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	id, err := requirePulid(params.Params, paramInvoiceRunID)
	if err != nil {
		return nil, err
	}

	run, err := t.runs.Get(ctx, repositories.GetInvoiceRunByIDRequest{
		ID:            id,
		TenantInfo:    tenantOf(params),
		IncludeGroups: true,
		IncludeItems:  true,
	})
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceInvoiceRun)
	show := amounts{show: gate.show(fieldTotalAmount, withheldAmounts)}
	detail := &invoiceRunDetail{
		invoiceRunRow:  invoiceRunRowFrom(run, show),
		OffCycleReason: run.OffCycleReason,
		Groups:         make([]invoiceRunGroupRow, 0, len(run.Groups)),
	}
	shown := 0
	for _, group := range run.Groups {
		if group == nil {
			continue
		}
		row := invoiceRunGroupRow{
			ID:         group.ID.String(),
			CustomerID: group.CustomerID.String(),
			Label:      group.GroupLabel,
			Status:     string(group.Status),
			ItemCount:  group.ItemCount,
			Total:      show.of(group.TotalAmount),
			AutoBill:   group.AutoBill,
			InvoiceID:  pulidString(group.InvoiceID),
			SkipReason: group.SkipReason,
			Items:      make([]invoiceRunItemRow, 0, len(group.Items)),
		}
		for _, item := range group.Items {
			if item == nil {
				continue
			}
			if shown == maxRunItemsShown {
				detail.ItemsTruncated = true
				break
			}
			shown++
			row.Items = append(row.Items, invoiceRunItemRow{
				ID:                 item.ID.String(),
				BillingQueueItemID: item.BillingQueueItemID.String(),
				ShipmentID:         item.ShipmentID.String(),
				ProNumber:          item.ProNumber,
				Amount:             show.of(item.Amount),
				Excluded:           item.Excluded,
				ExclusionReason:    item.ExclusionReason,
			})
		}
		detail.Groups = append(detail.Groups, row)
	}
	detail.Withheld = gate.Withheld()

	return detail, nil
}

type statementShipmentRow struct {
	BillingQueueItemID string       `json:"billingQueueItemId"`
	ShipmentID         string       `json:"shipmentId"`
	ProNumber          string       `json:"proNumber,omitempty"`
	ServiceDate        optionalDate `json:"serviceDate"`
	Amount             string       `json:"amount,omitempty"`
}

type statementGroupRow struct {
	Label         string                 `json:"label"`
	ShipmentCount int                    `json:"shipmentCount"`
	Total         string                 `json:"totalAmount,omitempty"`
	BelowMinimum  bool                   `json:"belowMinimum"`
	Shipments     []statementShipmentRow `json:"shipments,omitempty"`
}

type statementRow struct {
	CustomerID    string              `json:"customerId"`
	CustomerName  string              `json:"customerName"`
	Cycle         string              `json:"cycle"`
	PeriodStart   optionalDate        `json:"periodStart"`
	PeriodEnd     optionalDate        `json:"periodEnd"`
	ShipmentCount int                 `json:"shipmentCount"`
	InvoiceCount  int                 `json:"invoiceCount"`
	Total         string              `json:"totalAmount,omitempty"`
	Currency      string              `json:"currency,omitempty"`
	AutoBill      bool                `json:"autoBill"`
	BelowMinimum  bool                `json:"belowMinimum"`
	HeldCount     int                 `json:"heldCount"`
	Groups        []statementGroupRow `json:"groups,omitempty"`
}

type listOpenStatementsTool struct {
	runs   invoiceRunReader
	access fieldAccess
}

func newListOpenStatementsTool(
	runs invoiceRunReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listOpenStatementsTool{runs: runs, access: newFieldAccess(permissions)}
}

func (t *listOpenStatementsTool) Name() string { return "list_open_statements" }

func (t *listOpenStatementsTool) Description() string {
	return "List the open billing periods of customers billed on statements, as they stand " +
		"now. Each shows the period, the freight on it, the invoices it would make and what " +
		"is held back. Name a customerId to see the shipments on theirs, which " +
		"bill_statement_now can exclude."
}

func (t *listOpenStatementsTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		"customerId": stringParam("Optional: one customer, from list_customers, with the " +
			"shipments on their statement."),
	})
}

func (t *listOpenStatementsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceInvoiceRun})
}

func (t *listOpenStatementsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	customerID, oneCustomer, err := optionalPulid(params.Params, "customerId")
	if err != nil {
		return nil, err
	}

	statements, err := t.runs.ListOpenStatements(ctx, &serviceports.ListOpenStatementsRequest{
		TenantInfo:       tenantOf(params),
		CustomerID:       customerID,
		IncludeShipments: oneCustomer,
	})
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceInvoiceRun)
	show := amounts{show: gate.show(fieldTotalAmount, withheldAmounts)}
	rows := make([]statementRow, 0, len(statements))
	for _, statement := range statements {
		if statement == nil {
			continue
		}
		rows = append(rows, statementRowFrom(statement, show))
	}

	return newReceivableList(rows, gate), nil
}

func statementRowFrom(statement *serviceports.OpenStatement, show amounts) statementRow {
	row := statementRow{
		CustomerID:    statement.CustomerID.String(),
		CustomerName:  statement.CustomerName,
		Cycle:         string(statement.Cycle),
		PeriodStart:   recordedDate(statement.PeriodStart),
		PeriodEnd:     recordedDate(statement.PeriodEnd),
		ShipmentCount: statement.ShipmentCount,
		InvoiceCount:  statement.InvoiceCount,
		Total:         show.of(statement.TotalAmount),
		Currency:      statement.CurrencyCode,
		AutoBill:      statement.AutoBill,
		BelowMinimum:  statement.BelowMinimum,
		HeldCount:     statement.HeldCount,
		Groups:        make([]statementGroupRow, 0, len(statement.Groups)),
	}
	for _, group := range statement.Groups {
		if group == nil {
			continue
		}
		groupRow := statementGroupRow{
			Label:         group.Label,
			ShipmentCount: group.ShipmentCount,
			Total:         show.of(group.TotalAmount),
			BelowMinimum:  group.BelowMinimum,
		}
		for _, shipment := range group.Shipments {
			if shipment == nil {
				continue
			}
			groupRow.Shipments = append(groupRow.Shipments, statementShipmentRow{
				BillingQueueItemID: shipment.BillingQueueItemID.String(),
				ShipmentID:         shipment.ShipmentID.String(),
				ProNumber:          shipment.ProNumber,
				ServiceDate:        pointerDate(shipment.ServiceDate),
				Amount:             show.of(shipment.Amount),
			})
		}
		row.Groups = append(row.Groups, groupRow)
	}

	return row
}

type shareCandidateRow struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
}

type listInvoiceShareCandidatesTool struct {
	shares shareCandidateLister
}

func newListInvoiceShareCandidatesTool(shares shareCandidateLister) serviceports.AgentQueryTool {
	return &listInvoiceShareCandidatesTool{shares: shares}
}

func (t *listInvoiceShareCandidatesTool) Name() string {
	return "list_invoice_share_candidates"
}

func (t *listInvoiceShareCandidatesTool) Description() string {
	return "List the teammates an invoice can be shared with: only people whose role lets " +
		"them read invoices. Search by name or username."
}

func (t *listInvoiceShareCandidatesTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramReceivableInvoice: stringParam("The invoice, from list_invoices or get_invoice."),
		"query":                stringParam("Optional: part of a name or username."),
		"limit":                limitProperty(),
	}, paramReceivableInvoice)
}

func (t *listInvoiceShareCandidatesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceInvoice})
}

func (t *listInvoiceShareCandidatesTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	invoiceID, err := requirePulid(params.Params, paramReceivableInvoice)
	if err != nil {
		return nil, err
	}

	result, err := t.shares.ListCandidates(ctx, &serviceports.ListShareCandidatesRequest{
		TenantInfo: tenantOf(params),
		InvoiceID:  invoiceID,
		Query:      strings.TrimSpace(optionalString(params.Params, "query")),
		Pagination: pagination.Info{Limit: receivableLimit(params.Params)},
	})
	if err != nil {
		return nil, err
	}

	rows := make([]shareCandidateRow, 0, len(result.Items))
	for _, candidate := range result.Items {
		if candidate == nil {
			continue
		}
		rows = append(rows, shareCandidateRow{
			ID:       candidate.ID.String(),
			Name:     candidate.Name,
			Username: candidate.Username,
		})
	}

	return newReceivableList(rows, nil), nil
}

func receivableToolProviders() []any {
	return []any{
		provideListInvoiceAdjustmentsTool,
		provideGetInvoiceAdjustmentTool,
		provideListInvoiceDisputesTool,
		provideListCreditMemoApplicationsTool,
		provideListInvoiceRunsTool,
		provideGetInvoiceRunTool,
		provideListOpenStatementsTool,
		provideListInvoiceShareCandidatesTool,
	}
}

func provideListInvoiceAdjustmentsTool(
	adjustments serviceports.InvoiceAdjustmentService,
	invoices repositories.InvoiceRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListInvoiceAdjustmentsTool(adjustments, invoices, permissions)
}

func provideGetInvoiceAdjustmentTool(
	adjustments serviceports.InvoiceAdjustmentService,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetInvoiceAdjustmentTool(adjustments, permissions)
}

func provideListInvoiceDisputesTool(
	disputes repositories.InvoiceDisputeRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListInvoiceDisputesTool(disputes, permissions)
}

func provideListCreditMemoApplicationsTool(
	payments repositories.CustomerPaymentRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListCreditMemoApplicationsTool(payments, permissions)
}

func provideListInvoiceRunsTool(
	runs *invoicerunservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListInvoiceRunsTool(runs, permissions)
}

func provideGetInvoiceRunTool(
	runs *invoicerunservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetInvoiceRunTool(runs, permissions)
}

func provideListOpenStatementsTool(
	runs *invoicerunservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListOpenStatementsTool(runs, permissions)
}

func provideListInvoiceShareCandidatesTool(
	shares serviceports.InvoiceShareService,
) serviceports.AgentQueryTool {
	return newListInvoiceShareCandidatesTool(shares)
}
