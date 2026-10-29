package receivablequerytools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/gettool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicerunservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun/invoicerundb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	defaultReceivableRows  = 25
	maxReceivableRows      = 50
	maxReceivableInvoices  = 50
	maxRunItemsShown       = 500
	paramInvoiceRunID      = "invoiceRunId"
	paramReceivableInvoice = "invoiceId"
)

type adjustmentReader interface {
	GetLineage(
		ctx context.Context,
		req *invoiceadjustmentports.GetInvoiceAdjustmentLineageRequest,
	) (*invoiceadjustmentports.InvoiceAdjustmentLineage, error)
	ListApprovals(
		ctx context.Context,
		req *invoiceadjustmentports.ListApprovalQueueRequest,
	) (*pagination.CursorListResult[*invoiceadjustmentports.InvoiceAdjustmentApprovalQueueItem], error)
}

type creditApplicationLister interface {
	ListCreditMemoApplicationsByInvoiceIDs(
		ctx context.Context,
		req *customerpaymentports.ListApplicationsByInvoiceIDsRequest,
	) (map[pulid.ID][]*customerpayment.CreditMemoApplication, error)
}

type invoiceRunReader interface {
	Get(
		ctx context.Context,
		req invoicerunports.GetInvoiceRunByIDRequest,
	) (*invoicerun.InvoiceRun, error)
	List(
		ctx context.Context,
		req *invoicerunports.ListInvoiceRunsRequest,
	) (*pagination.ListResult[*invoicerun.InvoiceRun], error)
	ListOpenStatements(
		ctx context.Context,
		req *invoicerunports.ListOpenStatementsRequest,
	) ([]*invoicerunports.OpenStatement, error)
}

type shareCandidateLister interface {
	ListCandidates(
		ctx context.Context,
		req *invoiceports.ListShareCandidatesRequest,
	) (*pagination.ListResult[*invoiceports.ShareCandidate], error)
}

type amounts struct {
	show bool
}

func amountsFor(
	ctx context.Context,
	access fieldgate.Access,
	params *agenttoolports.QueryToolParams,
) (amounts, *fieldgate.Gate) {
	gate := access.Gate(ctx, params, permission.ResourceInvoice)

	return amounts{show: gate.Show(querytool.FieldTotalAmount, querytool.WithheldAmounts)}, gate
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

func limitProperty() map[string]any {
	return querytool.LimitSchema(defaultReceivableRows, maxReceivableRows)
}

func receivableLimit(params map[string]any) int {
	limit := querytool.OptionalInt(params, querytool.ParamLimit, defaultReceivableRows)
	if limit <= 0 || limit > maxReceivableRows {
		return maxReceivableRows
	}

	return limit
}

type adjustmentRow struct {
	ID                   string                 `json:"id"`
	InvoiceID            string                 `json:"invoiceId"`
	InvoiceNumber        string                 `json:"invoiceNumber,omitempty"`
	Kind                 string                 `json:"kind"`
	Status               string                 `json:"status"`
	ApprovalStatus       string                 `json:"approvalStatus,omitempty"`
	RebillStrategy       string                 `json:"rebillStrategy,omitempty"`
	Reason               string                 `json:"reason,omitempty"`
	Credited             string                 `json:"creditedAmount,omitempty"`
	Rebilled             string                 `json:"rebilledAmount,omitempty"`
	CreditMemoInvoiceID  string                 `json:"creditMemoInvoiceId,omitempty"`
	ReplacementInvoiceID string                 `json:"replacementInvoiceId,omitempty"`
	SubmittedAt          querytool.OptionalDate `json:"submittedAt"`
}

type listInvoiceAdjustmentsTool struct {
	adjustments adjustmentReader
	invoices    invoiceReader
	access      fieldgate.Access
}

func newListInvoiceAdjustmentsTool(
	adjustments adjustmentReader,
	invoices invoiceReader,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return &listInvoiceAdjustmentsTool{
		adjustments: adjustments,
		invoices:    invoices,
		access:      fieldgate.NewAccess(permissions),
	}
}

func (t *listInvoiceAdjustmentsTool) Name() string { return "invoice_list_adjustments" }

func (t *listInvoiceAdjustmentsTool) Description() string {
	return "List the invoice adjustments on one invoice, drafts included, or those waiting " +
		"for an approver. With invoiceId it lists every credit, rebill and reversal made or " +
		"drafted against the invoice and its replacements; without it, the approval queue. " +
		"Open one with invoice_get_adjustment."
}

func (t *listInvoiceAdjustmentsTool) ParamSchema() map[string]any {
	return querytool.ObjectSchema(map[string]any{
		paramReceivableInvoice: agenttoolschema.RecordIDText(permission.ResourceInvoice,
			"Optional: the invoice, from invoice_list or "+
				"invoice_get. Leave it out for the approval queue."),
		querytool.ParamLimit: limitProperty(),
	})
}

func (t *listInvoiceAdjustmentsTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{Resource: permission.ResourceInvoice})
}

func (t *listInvoiceAdjustmentsTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	invoiceID, hasInvoice, err := querytool.OptionalPulid(params.Params, paramReceivableInvoice)
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
	queue, err := t.adjustments.ListApprovals(ctx, &invoiceadjustmentports.ListApprovalQueueRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: querytool.TenantOf(params),
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
			SubmittedAt:    querytool.PointerDate(item.SubmittedAt),
		})
	}

	return fieldgate.NewList(rows, gate), nil
}

func (t *listInvoiceAdjustmentsTool) lineage(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	invoiceID pulid.ID,
	show amounts,
	gate *fieldgate.Gate,
) (any, error) {
	entity, err := t.invoices.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         invoiceID,
		TenantInfo: querytool.TenantOf(params),
	})
	if err != nil {
		return nil, err
	}
	if entity.CorrectionGroupID.IsNil() {
		list := fieldgate.NewList([]adjustmentRow{}, gate)
		list.Note = "Nothing has been adjusted or drafted against this invoice."
		return list, nil
	}

	lineage, err := t.adjustments.GetLineage(
		ctx,
		&invoiceadjustmentports.GetInvoiceAdjustmentLineageRequest{
			CorrectionGroupID: entity.CorrectionGroupID,
			TenantInfo:        querytool.TenantOf(params),
		},
	)
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

	return fieldgate.NewList(rows, gate), nil
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
		CreditMemoInvoiceID:  querytool.PulidString(adjustment.CreditMemoInvoiceID),
		ReplacementInvoiceID: querytool.PulidString(adjustment.ReplacementInvoiceID),
		SubmittedAt:          querytool.PointerDate(adjustment.SubmittedAt),
	}
}

type creditApplicationRow struct {
	ID                  string                 `json:"id"`
	CreditMemoInvoiceID string                 `json:"creditMemoInvoiceId"`
	InvoiceID           string                 `json:"invoiceId"`
	Amount              string                 `json:"amount,omitempty"`
	Status              string                 `json:"status"`
	AccountingDate      querytool.OptionalDate `json:"accountingDate"`
	UnappliedAt         querytool.OptionalDate `json:"unappliedAt"`
	UnappliedReason     string                 `json:"unappliedReason,omitempty"`
}

type listCreditMemoApplicationsTool struct {
	applications creditApplicationLister
	access       fieldgate.Access
}

func newListCreditMemoApplicationsTool(
	applications creditApplicationLister,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return &listCreditMemoApplicationsTool{
		applications: applications,
		access:       fieldgate.NewAccess(permissions),
	}
}

func (t *listCreditMemoApplicationsTool) Name() string {
	return "receivable_list_credit_memo_applications"
}

func (t *listCreditMemoApplicationsTool) Description() string {
	return "List where credit memos were applied to the invoices or credit memos you name. " +
		"Each application shows the amount and whether it still stands or was taken back; " +
		"receivable_unapply_credit_memo takes one back."
}

func (t *listCreditMemoApplicationsTool) ParamSchema() map[string]any {
	return querytool.ObjectSchema(map[string]any{
		querytool.ParamReceivableInvoices: agenttoolschema.RecordIDs(
			permission.ResourceInvoice,
			"Invoices or credit memos, from invoice_list "+
				"or invoice_get.",
			maxReceivableInvoices,
		),
	}, querytool.ParamReceivableInvoices)
}

func (t *listCreditMemoApplicationsTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{Resource: permission.ResourceInvoice})
}

func (t *listCreditMemoApplicationsTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}
	ids, err := querytool.RequireIDList(
		params.Params,
		querytool.ParamReceivableInvoices,
		maxReceivableInvoices,
	)
	if err != nil {
		return nil, err
	}

	byInvoice, err := t.applications.ListCreditMemoApplicationsByInvoiceIDs(ctx,
		&customerpaymentports.ListApplicationsByInvoiceIDsRequest{
			TenantInfo: querytool.TenantOf(params),
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
				AccountingDate:      querytool.RecordedDate(application.AccountingDate),
				UnappliedAt:         querytool.PointerDate(application.UnappliedAt),
				UnappliedReason:     application.UnappliedReason,
			})
		}
	}

	return fieldgate.NewList(rows, gate), nil
}

type invoiceRunRow struct {
	ID            string                 `json:"id"`
	Number        string                 `json:"number"`
	Status        string                 `json:"status"`
	Source        string                 `json:"source"`
	Cycle         string                 `json:"cycle,omitempty"`
	PeriodStart   querytool.OptionalDate `json:"periodStart"`
	PeriodEnd     querytool.OptionalDate `json:"periodEnd"`
	InvoiceDate   querytool.OptionalDate `json:"invoiceDate"`
	GroupCount    int                    `json:"groupCount"`
	ItemCount     int                    `json:"itemCount"`
	ExcludedCount int                    `json:"excludedCount"`
	InvoiceCount  int                    `json:"invoiceCount"`
	Total         string                 `json:"totalAmount,omitempty"`
	Currency      string                 `json:"currency,omitempty"`
	FailureReason string                 `json:"failureReason,omitempty"`
}

func invoiceRunRowFrom(run *invoicerun.InvoiceRun, show amounts) invoiceRunRow {
	return invoiceRunRow{
		ID:            run.ID.String(),
		Number:        run.Number,
		Status:        string(run.Status),
		Source:        string(run.Source),
		Cycle:         string(run.Cycle),
		PeriodStart:   querytool.RecordedDate(run.PeriodStart),
		PeriodEnd:     querytool.RecordedDate(run.PeriodEnd),
		InvoiceDate:   querytool.RecordedDate(run.InvoiceDate),
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
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	access := fieldgate.NewAccess(permissions)

	return listtool.New(&listtool.Spec{
		Name:         "invoice_list_runs",
		Row:          invoiceRunRow{},
		EntityPlural: "invoice runs",
		Summary: "List invoice runs, the reviewable batches of consolidated invoices for a " +
			"billing period, by status, number or period. Open one with invoice_get_run to " +
			"see its groups and items.",
		Resource: permission.ResourceInvoiceRun,
		Config: listquery.Describe(
			(*invoicerun.InvoiceRun)(nil),
			invoicerundb.InvoiceRunsSchema(),
		),
		Fields: []filtercatalog.Field{
			{
				Name: querytool.ParamStatus,
				Kind: filtercatalog.KindEnum,
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
			{Name: querytool.FieldNumber, Kind: filtercatalog.KindText, Sortable: true},
			{Name: "periodEnd", Kind: filtercatalog.KindDate, Sortable: true},
			{Name: querytool.FieldCreatedAt, Kind: filtercatalog.KindDate, Sortable: true},
		},
		Access: access,
		FetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			_ pagination.CursorInfo,
			gate *fieldgate.Gate,
		) (listtool.Page, error) {
			result, err := runs.List(ctx, &invoicerunports.ListInvoiceRunsRequest{Filter: opts})
			if err != nil {
				return listtool.Page{}, err
			}
			show := amounts{show: gate.Show(querytool.FieldTotalAmount, querytool.FieldTotalAmount)}

			return listtool.OffsetPage(result.Items, func(run *invoicerun.InvoiceRun) any {
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
	access fieldgate.Access
}

func newGetInvoiceRunTool(
	runs invoiceRunReader,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return &getInvoiceRunTool{runs: runs, access: fieldgate.NewAccess(permissions)}
}

func (t *getInvoiceRunTool) Name() string { return "invoice_get_run" }

func (t *getInvoiceRunTool) Description() string {
	return "Retrieve one invoice run by id with each group, the invoice it would make or made, " +
		"and the items in it: the ids invoice_adjust_run moves or excludes."
}

func (t *getInvoiceRunTool) ParamSchema() map[string]any {
	return gettool.IDSchema(
		paramInvoiceRunID,
		agenttoolschema.RecordIDText(
			permission.ResourceInvoiceRun,
			"The run's id, from invoice_list_runs.",
		),
	)
}

func (t *getInvoiceRunTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(
		t.Name(),
		&querytool.ReadSpec{Resource: permission.ResourceInvoiceRun},
	)
}

func (t *getInvoiceRunTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}
	id, err := querytool.RequirePulid(params.Params, paramInvoiceRunID)
	if err != nil {
		return nil, err
	}

	run, err := t.runs.Get(ctx, invoicerunports.GetInvoiceRunByIDRequest{
		ID:            id,
		TenantInfo:    querytool.TenantOf(params),
		IncludeGroups: true,
		IncludeItems:  true,
	})
	if err != nil {
		return nil, err
	}

	gate := t.access.Gate(ctx, params, permission.ResourceInvoiceRun)
	show := amounts{show: gate.Show(querytool.FieldTotalAmount, querytool.WithheldAmounts)}
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
			InvoiceID:  querytool.PulidString(group.InvoiceID),
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
	BillingQueueItemID string                 `json:"billingQueueItemId"`
	ShipmentID         string                 `json:"shipmentId"`
	ProNumber          string                 `json:"proNumber,omitempty"`
	ServiceDate        querytool.OptionalDate `json:"serviceDate"`
	Amount             string                 `json:"amount,omitempty"`
}

type statementGroupRow struct {
	Label         string                 `json:"label"`
	ShipmentCount int                    `json:"shipmentCount"`
	Total         string                 `json:"totalAmount,omitempty"`
	BelowMinimum  bool                   `json:"belowMinimum"`
	Shipments     []statementShipmentRow `json:"shipments,omitempty"`
}

type statementRow struct {
	CustomerID    string                 `json:"customerId"`
	CustomerName  string                 `json:"customerName"`
	Cycle         string                 `json:"cycle"`
	PeriodStart   querytool.OptionalDate `json:"periodStart"`
	PeriodEnd     querytool.OptionalDate `json:"periodEnd"`
	ShipmentCount int                    `json:"shipmentCount"`
	InvoiceCount  int                    `json:"invoiceCount"`
	Total         string                 `json:"totalAmount,omitempty"`
	Currency      string                 `json:"currency,omitempty"`
	AutoBill      bool                   `json:"autoBill"`
	BelowMinimum  bool                   `json:"belowMinimum"`
	HeldCount     int                    `json:"heldCount"`
	Groups        []statementGroupRow    `json:"groups,omitempty"`
}

type listOpenStatementsTool struct {
	runs   invoiceRunReader
	access fieldgate.Access
}

func newListOpenStatementsTool(
	runs invoiceRunReader,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return &listOpenStatementsTool{runs: runs, access: fieldgate.NewAccess(permissions)}
}

func (t *listOpenStatementsTool) Name() string { return "invoice_list_open_statements" }

func (t *listOpenStatementsTool) Description() string {
	return "List the open billing periods of customers billed on statements, as they stand " +
		"now. Each shows the period, the freight on it, the invoices it would make and what " +
		"is held back. Name a customerId to see the shipments on theirs, which " +
		"invoice_bill_statement can exclude."
}

func (t *listOpenStatementsTool) ParamSchema() map[string]any {
	return querytool.ObjectSchema(map[string]any{
		querytool.ParamCustomerID: agenttoolschema.RecordIDText(permission.ResourceCustomer,
			"Optional: one customer, from customer_list, with the "+
				"shipments on their statement."),
	})
}

func (t *listOpenStatementsTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(
		t.Name(),
		&querytool.ReadSpec{Resource: permission.ResourceInvoiceRun},
	)
}

func (t *listOpenStatementsTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}
	customerID, oneCustomer, err := querytool.OptionalPulid(
		params.Params,
		querytool.ParamCustomerID,
	)
	if err != nil {
		return nil, err
	}

	statements, err := t.runs.ListOpenStatements(ctx, &invoicerunports.ListOpenStatementsRequest{
		TenantInfo:       querytool.TenantOf(params),
		CustomerID:       customerID,
		IncludeShipments: oneCustomer,
	})
	if err != nil {
		return nil, err
	}

	gate := t.access.Gate(ctx, params, permission.ResourceInvoiceRun)
	show := amounts{show: gate.Show(querytool.FieldTotalAmount, querytool.WithheldAmounts)}
	rows := make([]statementRow, 0, len(statements))
	for _, statement := range statements {
		if statement == nil {
			continue
		}
		rows = append(rows, statementRowFrom(statement, show))
	}

	return fieldgate.NewList(rows, gate), nil
}

func statementRowFrom(statement *invoicerunports.OpenStatement, show amounts) statementRow {
	row := statementRow{
		CustomerID:    statement.CustomerID.String(),
		CustomerName:  statement.CustomerName,
		Cycle:         string(statement.Cycle),
		PeriodStart:   querytool.RecordedDate(statement.PeriodStart),
		PeriodEnd:     querytool.RecordedDate(statement.PeriodEnd),
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
				ServiceDate:        querytool.PointerDate(shipment.ServiceDate),
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

func newListInvoiceShareCandidatesTool(shares shareCandidateLister) agenttoolports.AgentQueryTool {
	return &listInvoiceShareCandidatesTool{shares: shares}
}

func (t *listInvoiceShareCandidatesTool) Name() string {
	return "invoice_list_share_candidates"
}

func (t *listInvoiceShareCandidatesTool) Description() string {
	return "List the teammates an invoice can be shared with: only people whose role lets " +
		"them read invoices. Search by name or username."
}

func (t *listInvoiceShareCandidatesTool) ParamSchema() map[string]any {
	return querytool.ObjectSchema(map[string]any{
		paramReceivableInvoice: agenttoolschema.RecordIDText(permission.ResourceInvoice,
			"The invoice, from invoice_list or invoice_get."),
		querytool.ParamQuery: querytool.StringParam("Optional: part of a name or username."),
		querytool.ParamLimit: limitProperty(),
	}, paramReceivableInvoice)
}

func (t *listInvoiceShareCandidatesTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{Resource: permission.ResourceInvoice})
}

func (t *listInvoiceShareCandidatesTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}
	invoiceID, err := querytool.RequirePulid(params.Params, paramReceivableInvoice)
	if err != nil {
		return nil, err
	}

	result, err := t.shares.ListCandidates(ctx, &invoiceports.ListShareCandidatesRequest{
		TenantInfo: querytool.TenantOf(params),
		InvoiceID:  invoiceID,
		Query: strings.TrimSpace(
			querytool.OptionalString(params.Params, querytool.ParamQuery),
		),
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

	return fieldgate.NewList(rows, nil), nil
}

func Providers() []any {
	return []any{
		provideListInvoiceAdjustmentsTool,
		provideListCreditMemoApplicationsTool,
		provideListInvoiceRunsTool,
		provideGetInvoiceRunTool,
		provideListOpenStatementsTool,
		provideListInvoiceShareCandidatesTool,
	}
}

func provideListInvoiceAdjustmentsTool(
	adjustments invoiceadjustmentports.InvoiceAdjustmentService,
	invoices invoiceports.InvoiceRepository,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return newListInvoiceAdjustmentsTool(adjustments, invoices, permissions)
}

func provideListCreditMemoApplicationsTool(
	payments customerpaymentports.CustomerPaymentRepository,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return newListCreditMemoApplicationsTool(payments, permissions)
}

func provideListInvoiceRunsTool(
	runs *invoicerunservice.Service,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return newListInvoiceRunsTool(runs, permissions)
}

func provideGetInvoiceRunTool(
	runs *invoicerunservice.Service,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return newGetInvoiceRunTool(runs, permissions)
}

func provideListOpenStatementsTool(
	runs *invoicerunservice.Service,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return newListOpenStatementsTool(runs, permissions)
}

func provideListInvoiceShareCandidatesTool(
	shares invoiceports.InvoiceShareService,
) agenttoolports.AgentQueryTool {
	return newListInvoiceShareCandidatesTool(shares)
}
