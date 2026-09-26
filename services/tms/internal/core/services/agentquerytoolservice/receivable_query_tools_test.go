package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAdjustmentReader struct {
	lineage   *serviceports.InvoiceAdjustmentLineage
	detail    *invoiceadjustment.InvoiceAdjustment
	approvals []*repositories.InvoiceAdjustmentApprovalQueueItem
	groupID   pulid.ID
}

func (f *fakeAdjustmentReader) GetDetail(
	context.Context,
	*serviceports.GetInvoiceAdjustmentDetailRequest,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	return f.detail, nil
}

func (f *fakeAdjustmentReader) GetLineage(
	_ context.Context,
	req *serviceports.GetInvoiceAdjustmentLineageRequest,
) (*serviceports.InvoiceAdjustmentLineage, error) {
	f.groupID = req.CorrectionGroupID

	return f.lineage, nil
}

func (f *fakeAdjustmentReader) ListApprovals(
	context.Context,
	*repositories.ListApprovalQueueRequest,
) (*pagination.CursorListResult[*repositories.InvoiceAdjustmentApprovalQueueItem], error) {
	return &pagination.CursorListResult[*repositories.InvoiceAdjustmentApprovalQueueItem]{
		Items: f.approvals,
	}, nil
}

type fakeDisputeLister struct {
	disputes map[pulid.ID][]*invoice.InvoiceDispute
}

func (f *fakeDisputeLister) ListByInvoiceIDs(
	context.Context,
	*repositories.ListInvoiceDisputesByInvoiceIDsRequest,
) (map[pulid.ID][]*invoice.InvoiceDispute, error) {
	return f.disputes, nil
}

type fakeCreditApplications struct {
	byInvoice map[pulid.ID][]*customerpayment.CreditMemoApplication
}

func (f *fakeCreditApplications) ListCreditMemoApplicationsByInvoiceIDs(
	context.Context,
	*repositories.ListApplicationsByInvoiceIDsRequest,
) (map[pulid.ID][]*customerpayment.CreditMemoApplication, error) {
	return f.byInvoice, nil
}

type fakeRunReader struct {
	run        *invoicerun.InvoiceRun
	runs       []*invoicerun.InvoiceRun
	statements []*serviceports.OpenStatement
	getReq     repositories.GetInvoiceRunByIDRequest
	stmtReq    *serviceports.ListOpenStatementsRequest
}

func (f *fakeRunReader) Get(
	_ context.Context,
	req repositories.GetInvoiceRunByIDRequest,
) (*invoicerun.InvoiceRun, error) {
	f.getReq = req

	return f.run, nil
}

func (f *fakeRunReader) List(
	context.Context,
	*repositories.ListInvoiceRunsRequest,
) (*pagination.ListResult[*invoicerun.InvoiceRun], error) {
	return &pagination.ListResult[*invoicerun.InvoiceRun]{Items: f.runs, Total: len(f.runs)}, nil
}

func (f *fakeRunReader) ListOpenStatements(
	_ context.Context,
	req *serviceports.ListOpenStatementsRequest,
) ([]*serviceports.OpenStatement, error) {
	f.stmtReq = req

	return f.statements, nil
}

type fakeShareCandidates struct {
	req        *serviceports.ListShareCandidatesRequest
	candidates []*serviceports.ShareCandidate
}

func (f *fakeShareCandidates) ListCandidates(
	_ context.Context,
	req *serviceports.ListShareCandidatesRequest,
) (*pagination.ListResult[*serviceports.ShareCandidate], error) {
	f.req = req

	return &pagination.ListResult[*serviceports.ShareCandidate]{
		Items: f.candidates,
		Total: len(f.candidates),
	}, nil
}

func TestGetInvoice_NamesEachLineByTheIDAnAdjustmentTakes(t *testing.T) {
	t.Parallel()

	inv := sensitiveInvoice()
	inv.Lines[0].ID = pulid.MustNew("invl_")
	tool := newGetInvoiceTool(&fakeInvoiceRepo{items: []*invoice.Invoice{inv}}, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"invoiceId": inv.ID.String(),
	}, ""))
	require.NoError(t, err)
	detail := result.(invoiceDetail)
	require.Len(t, detail.Lines, 1)
	assert.Equal(t, inv.Lines[0].ID.String(), detail.Lines[0].ID)
}

func TestListInvoiceAdjustments_ReadsAnInvoicesLineageWithItsDrafts(t *testing.T) {
	t.Parallel()

	inv := sensitiveInvoice()
	inv.CorrectionGroupID = pulid.MustNew("icg_")
	draft := &invoiceadjustment.InvoiceAdjustment{
		ID:                pulid.MustNew("iadj_"),
		OriginalInvoiceID: inv.ID,
		Kind:              invoiceadjustment.KindCreditOnly,
		Status:            invoiceadjustment.StatusDraft,
		CreditTotalAmount: decimal.RequireFromString("-150.00"),
	}
	reader := &fakeAdjustmentReader{lineage: &serviceports.InvoiceAdjustmentLineage{
		Invoices:    []*invoice.Invoice{inv},
		Adjustments: []*invoiceadjustment.InvoiceAdjustment{draft},
	}}
	tool := newListInvoiceAdjustmentsTool(
		reader, &fakeInvoiceRepo{items: []*invoice.Invoice{inv}}, &fakePermissions{},
	)

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"invoiceId": inv.ID.String(),
	}, permission.SensitivityRestricted))
	require.NoError(t, err)
	listed := result.(*receivableList[adjustmentRow])
	assert.Equal(t, inv.CorrectionGroupID, reader.groupID)
	require.Len(t, listed.Items, 1)
	assert.Equal(t, draft.ID.String(), listed.Items[0].ID)
	assert.Equal(t, "Draft", listed.Items[0].Status)
	assert.Equal(t, "INV-1042", listed.Items[0].InvoiceNumber)
	assert.Equal(t, "150.00", listed.Items[0].Credited)

	withheld, err := tool.Query(t.Context(), agentParams(map[string]any{
		"invoiceId": inv.ID.String(),
	}, ""))
	require.NoError(t, err)
	gated := withheld.(*receivableList[adjustmentRow])
	assert.Empty(t, gated.Items[0].Credited)
	assert.Equal(t, []string{withheldAmounts}, gated.Withheld)
}

func TestListInvoiceAdjustments_WithoutAnInvoiceListsWhatWaitsForApproval(t *testing.T) {
	t.Parallel()

	pending := &repositories.InvoiceAdjustmentApprovalQueueItem{
		AdjustmentID:          pulid.MustNew("iadj_"),
		OriginalInvoiceID:     pulid.MustNew("inv_"),
		OriginalInvoiceNumber: "INV-9",
		Kind:                  invoiceadjustment.KindFullReversal,
		Status:                invoiceadjustment.StatusPendingApproval,
	}
	tool := newListInvoiceAdjustmentsTool(
		&fakeAdjustmentReader{approvals: []*repositories.InvoiceAdjustmentApprovalQueueItem{pending}},
		&fakeInvoiceRepo{},
		&fakePermissions{},
	)

	result, err := tool.Query(t.Context(), agentParams(map[string]any{}, ""))
	require.NoError(t, err)
	listed := result.(*receivableList[adjustmentRow])
	require.Len(t, listed.Items, 1)
	assert.Equal(t, pending.AdjustmentID.String(), listed.Items[0].ID)
	assert.Equal(t, "PendingApproval", listed.Items[0].Status)
}

func TestGetInvoiceAdjustment_ShowsItsLines(t *testing.T) {
	t.Parallel()

	lineID := pulid.MustNew("invl_")
	adjustment := &invoiceadjustment.InvoiceAdjustment{
		ID:                pulid.MustNew("iadj_"),
		OriginalInvoiceID: pulid.MustNew("inv_"),
		Kind:              invoiceadjustment.KindCreditOnly,
		Status:            invoiceadjustment.StatusPendingApproval,
		Reason:            "Detention billed twice",
		Lines: []*invoiceadjustment.InvoiceAdjustmentLine{{
			ID:             pulid.MustNew("iadjl_"),
			OriginalLineID: lineID,
			Description:    "Detention",
			CreditAmount:   decimal.RequireFromString("-75.00"),
		}},
	}
	tool := newGetInvoiceAdjustmentTool(
		&fakeAdjustmentReader{detail: adjustment}, &fakePermissions{},
	)

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"adjustmentId": adjustment.ID.String(),
	}, permission.SensitivityRestricted))
	require.NoError(t, err)
	detail := result.(*adjustmentDetail)
	assert.Equal(t, "Detention billed twice", detail.Reason)
	require.Len(t, detail.Lines, 1)
	assert.Equal(t, lineID.String(), detail.Lines[0].InvoiceLineID)
	assert.Equal(t, "75.00", detail.Lines[0].Credit)
}

func TestListInvoiceDisputes_ListsEveryCaseOnTheInvoices(t *testing.T) {
	t.Parallel()

	invoiceID := pulid.MustNew("inv_")
	dispute := &invoice.InvoiceDispute{
		ID:             pulid.MustNew("idsp_"),
		InvoiceID:      invoiceID,
		Status:         invoice.DisputeCaseStatusOpen,
		ReasonCode:     invoice.DisputeReasonRateDiscrepancy,
		DisputedAmount: decimal.RequireFromString("200.00"),
	}
	tool := newListInvoiceDisputesTool(&fakeDisputeLister{
		disputes: map[pulid.ID][]*invoice.InvoiceDispute{invoiceID: {dispute}},
	}, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"invoiceIds": []any{invoiceID.String()},
	}, permission.SensitivityRestricted))
	require.NoError(t, err)
	listed := result.(*receivableList[disputeRow])
	require.Len(t, listed.Items, 1)
	assert.Equal(t, dispute.ID.String(), listed.Items[0].ID)
	assert.Equal(t, "200.00", listed.Items[0].DisputedAmount)
}

func TestListCreditMemoApplications_ListsEachApplicationOnce(t *testing.T) {
	t.Parallel()

	memoID, invoiceID := pulid.MustNew("inv_"), pulid.MustNew("inv_")
	application := &customerpayment.CreditMemoApplication{
		ID:                  pulid.MustNew("cma_"),
		CreditMemoInvoiceID: memoID,
		InvoiceID:           invoiceID,
		AppliedAmountMinor:  5000,
		Status:              customerpayment.CreditApplicationStatusApplied,
	}
	tool := newListCreditMemoApplicationsTool(&fakeCreditApplications{
		byInvoice: map[pulid.ID][]*customerpayment.CreditMemoApplication{
			memoID:    {application},
			invoiceID: {application},
		},
	}, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"invoiceIds": []any{memoID.String(), invoiceID.String()},
	}, permission.SensitivityRestricted))
	require.NoError(t, err)
	listed := result.(*receivableList[creditApplicationRow])
	require.Len(t, listed.Items, 1)
	assert.Equal(t, "50.00", listed.Items[0].Amount)
	assert.Equal(t, memoID.String(), listed.Items[0].CreditMemoInvoiceID)
}

func TestGetInvoiceRun_ShowsGroupsAndTheItemsInThem(t *testing.T) {
	t.Parallel()

	itemID := pulid.MustNew("invrgi_")
	run := &invoicerun.InvoiceRun{
		ID:     pulid.MustNew("invrun_"),
		Number: "RUN-12",
		Status: invoicerun.StatusReady,
		Groups: []*invoicerun.InvoiceRunGroup{{
			ID:          pulid.MustNew("invrg_"),
			CustomerID:  pulid.MustNew("cus_"),
			GroupLabel:  "Acme Foods",
			Status:      invoicerun.GroupStatusPending,
			TotalAmount: decimal.RequireFromString("900.00"),
			Items: []*invoicerun.InvoiceRunGroupItem{{
				ID:                 itemID,
				BillingQueueItemID: pulid.MustNew("bqi_"),
				ProNumber:          "PRO-1",
				Amount:             decimal.RequireFromString("900.00"),
			}},
		}},
	}
	runs := &fakeRunReader{run: run}
	tool := newGetInvoiceRunTool(runs, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"invoiceRunId": run.ID.String(),
	}, permission.SensitivityRestricted))
	require.NoError(t, err)
	assert.True(t, runs.getReq.IncludeGroups)
	assert.True(t, runs.getReq.IncludeItems)
	detail := result.(*invoiceRunDetail)
	require.Len(t, detail.Groups, 1)
	require.Len(t, detail.Groups[0].Items, 1)
	assert.Equal(t, itemID.String(), detail.Groups[0].Items[0].ID)
	assert.Equal(t, "900.00", detail.Groups[0].Total)
}

func TestListOpenStatements_ShowsTheShipmentsForOneCustomer(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	runs := &fakeRunReader{statements: []*serviceports.OpenStatement{{
		CustomerID:    customerID,
		CustomerName:  "Acme Foods",
		ShipmentCount: 1,
		TotalAmount:   decimal.RequireFromString("400.00"),
		Groups: []*serviceports.StatementGroup{{
			Label:       "Acme Foods",
			TotalAmount: decimal.RequireFromString("400.00"),
			Shipments: []*serviceports.StatementShipment{{
				BillingQueueItemID: pulid.MustNew("bqi_"),
				ProNumber:          "PRO-7",
				Amount:             decimal.RequireFromString("400.00"),
			}},
		}},
	}}}
	tool := newListOpenStatementsTool(runs, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"customerId": customerID.String(),
	}, permission.SensitivityRestricted))
	require.NoError(t, err)
	require.NotNil(t, runs.stmtReq)
	assert.True(t, runs.stmtReq.IncludeShipments)
	assert.Equal(t, customerID, runs.stmtReq.CustomerID)
	listed := result.(*receivableList[statementRow])
	require.Len(t, listed.Items, 1)
	require.Len(t, listed.Items[0].Groups, 1)
	require.Len(t, listed.Items[0].Groups[0].Shipments, 1)
	assert.Equal(t, "PRO-7", listed.Items[0].Groups[0].Shipments[0].ProNumber)
}

func TestListInvoiceShareCandidates_ListsWhoMayReadTheInvoice(t *testing.T) {
	t.Parallel()

	invoiceID := pulid.MustNew("inv_")
	shares := &fakeShareCandidates{candidates: []*serviceports.ShareCandidate{{
		ID: pulid.MustNew("usr_"), Name: "Dana Whitfield", Username: "dana",
	}}}
	tool := newListInvoiceShareCandidatesTool(shares)

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"invoiceId": invoiceID.String(),
		"query":     "dana",
	}, ""))
	require.NoError(t, err)
	require.NotNil(t, shares.req)
	assert.Equal(t, invoiceID, shares.req.InvoiceID)
	assert.Equal(t, "dana", shares.req.Query)
	listed := result.(*receivableList[shareCandidateRow])
	require.Len(t, listed.Items, 1)
	assert.Equal(t, "Dana Whitfield", listed.Items[0].Name)
}
