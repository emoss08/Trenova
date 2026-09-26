package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInvoiceLifecycle struct {
	guard *writeGuard

	draftPlan  *serviceports.InvoiceDraftUpdatePreview
	updated    *serviceports.UpdateInvoiceDraftRequest
	pdfPlan    *serviceports.InvoicePDFGenerationPlan
	generated  *serviceports.InvoicePreviewRequest
	createPlan *serviceports.CreateInvoicesPlan
	planned    *serviceports.PlanCreateInvoicesRequest
	fromOrder  *serviceports.CreateInvoiceFromOrderRequest
	fromShips  *serviceports.CreateInvoiceFromShipmentsRequest
	memoPlan   *invoice.Invoice
	memo       *serviceports.CreateMemoRequest
	voidPlan   *serviceports.InvoiceVoidPreview
	voided     *serviceports.VoidInvoiceRequest
	ediPlan    *serviceports.InvoiceEDISendPreview
	ediSent    *serviceports.SendInvoiceEDIRequest
	refusal    error
}

func (f *fakeInvoiceLifecycle) plan() error { return f.refusal }

func (f *fakeInvoiceLifecycle) UpdateDraft(
	_ context.Context,
	req *serviceports.UpdateInvoiceDraftRequest,
	_ *serviceports.RequestActor,
) (*invoice.Invoice, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = req

	return f.draftPlan.After, nil
}

func (f *fakeInvoiceLifecycle) PreviewUpdateDraft(
	context.Context,
	*serviceports.UpdateInvoiceDraftRequest,
) (*serviceports.InvoiceDraftUpdatePreview, error) {
	return f.draftPlan, f.plan()
}

func (f *fakeInvoiceLifecycle) GeneratePDF(
	_ context.Context,
	req *serviceports.InvoicePreviewRequest,
	_ *serviceports.RequestActor,
) (*serviceports.GenerateInvoicePDFResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.generated = req

	return &serviceports.GenerateInvoicePDFResult{InvoiceID: req.InvoiceID, Status: "Queued"}, nil
}

func (f *fakeInvoiceLifecycle) PlanPDFGeneration(
	context.Context,
	*serviceports.InvoicePreviewRequest,
	*serviceports.RequestActor,
) (*serviceports.InvoicePDFGenerationPlan, error) {
	return f.pdfPlan, f.plan()
}

func (f *fakeInvoiceLifecycle) PlanCreateInvoices(
	_ context.Context,
	req *serviceports.PlanCreateInvoicesRequest,
) (*serviceports.CreateInvoicesPlan, error) {
	f.planned = req

	return f.createPlan, f.plan()
}

func (f *fakeInvoiceLifecycle) createdInvoices() *serviceports.CreateInvoicesResult {
	made := make([]*invoice.Invoice, 0, len(f.createPlan.Drafts))
	for _, draft := range f.createPlan.Drafts {
		created := *draft
		created.ID = pulid.MustNew("inv_")
		created.Number = "INV-7001"
		made = append(made, &created)
	}

	return &serviceports.CreateInvoicesResult{
		Invoices: made,
		Primary:  made[f.createPlan.PrimaryIndex],
	}
}

func (f *fakeInvoiceLifecycle) CreateInvoicesFromOrder(
	_ context.Context,
	req *serviceports.CreateInvoiceFromOrderRequest,
	_ *serviceports.RequestActor,
) (*serviceports.CreateInvoicesResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.fromOrder = req

	return f.createdInvoices(), nil
}

func (f *fakeInvoiceLifecycle) CreateInvoicesFromShipments(
	_ context.Context,
	req *serviceports.CreateInvoiceFromShipmentsRequest,
	_ *serviceports.RequestActor,
) (*serviceports.CreateInvoicesResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.fromShips = req

	return f.createdInvoices(), nil
}

func (f *fakeInvoiceLifecycle) CreateMemo(
	_ context.Context,
	req *serviceports.CreateMemoRequest,
	_ *serviceports.RequestActor,
) (*invoice.Invoice, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.memo = req
	created := *f.memoPlan
	created.ID = pulid.MustNew("inv_")
	created.Number = "CM-0042"

	return &created, nil
}

func (f *fakeInvoiceLifecycle) PreviewMemo(
	context.Context,
	*serviceports.CreateMemoRequest,
) (*invoice.Invoice, error) {
	return f.memoPlan, f.plan()
}

func (f *fakeInvoiceLifecycle) VoidInvoice(
	_ context.Context,
	req *serviceports.VoidInvoiceRequest,
	_ *serviceports.RequestActor,
) (*serviceports.VoidInvoiceResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.voided = req

	return &serviceports.VoidInvoiceResult{Invoice: f.voidPlan.After}, nil
}

func (f *fakeInvoiceLifecycle) PreviewVoid(
	context.Context,
	*serviceports.VoidInvoiceRequest,
	*serviceports.RequestActor,
) (*serviceports.InvoiceVoidPreview, error) {
	return f.voidPlan, f.plan()
}

func (f *fakeInvoiceLifecycle) SendEDI(
	_ context.Context,
	req *serviceports.SendInvoiceEDIRequest,
	_ *serviceports.RequestActor,
) (*serviceports.InvoiceEDISendResult, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.ediSent = req

	return &serviceports.InvoiceEDISendResult{InvoiceID: req.InvoiceID}, nil
}

func (f *fakeInvoiceLifecycle) PreviewSendEDI(
	context.Context,
	*serviceports.SendInvoiceEDIRequest,
	*serviceports.RequestActor,
) (*serviceports.InvoiceEDISendPreview, error) {
	return f.ediPlan, f.plan()
}

func lifecycleDraft(number string) *invoice.Invoice {
	inv := receivableInvoice(number, 165000, 0)
	inv.Status = invoice.StatusDraft
	inv.BillToName = "Acme Foods"
	inv.SubtotalAmount = decimal.RequireFromString("1500.00")
	inv.OtherAmount = decimal.RequireFromString("150.00")
	inv.TotalAmount = decimal.RequireFromString("1650.00")

	return inv
}

func approved(params serviceports.ToolExecuteParams) serviceports.ToolExecuteParams {
	params.ProposalID = pulid.MustNew("ap_")
	params.IdempotencyKey = params.ProposalID.String()

	return params
}

func TestUpdateInvoiceDraft_PreviewsTheChangeAndProposesNewRecipients(t *testing.T) {
	t.Parallel()

	before := lifecycleDraft("INV-5001")
	after := *before
	after.Memo = "Detention approved by the shipper"
	after.EmailToSnapshot = []string{"ap@acme.test"}
	docID := pulid.MustNew("doc_")
	fake := &fakeInvoiceLifecycle{guard: &writeGuard{}, draftPlan: &serviceports.InvoiceDraftUpdatePreview{
		Before:           before,
		After:            &after,
		AttachmentsAfter: []pulid.ID{docID},
	}}
	tool := newUpdateInvoiceDraftTool(fake)
	params := executeParams(map[string]any{
		paramInvoiceID:          before.ID.String(),
		"memo":                  "Detention approved by the shipper",
		"emailTo":               []any{"ap@acme.test"},
		"emailCc":               []any{},
		"attachmentDocumentIds": []any{docID.String()},
	})

	preview := previewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would update draft Invoice INV-5001")
	assert.Contains(t, preview.Summary, "1 attachment")
	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceInvoice, change.Resource)
	assert.Equal(t, before.ID, change.EntityID)
	assert.Equal(t, "Detention approved by the shipper", fieldByPath(t, change, "memo").After)

	policy := tool.Policy()
	require.NotNil(t, policy.Condition)
	assert.Equal(t, agent.TierPropose, policy.Condition.Limit(t.Context(), params))
	quiet := executeParams(map[string]any{paramInvoiceID: before.ID.String(), "memo": "x"})
	assert.Equal(t, agent.TierActWithApproval, policy.Condition.Limit(t.Context(), quiet))

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, fake.updated)
	require.NotNil(t, fake.updated.Memo)
	assert.Equal(t, "Detention approved by the shipper", *fake.updated.Memo)
	require.NotNil(t, fake.updated.EmailTo)
	assert.Equal(t, []string{"ap@acme.test"}, *fake.updated.EmailTo)
	require.NotNil(t, fake.updated.EmailCC, "an empty list clears the copies")
	assert.Empty(t, *fake.updated.EmailCC)
	assert.Nil(t, fake.updated.EmailBCC, "a field left out is left alone")
	assert.Nil(t, fake.updated.RemittanceInstructions)
	require.NotNil(t, fake.updated.AttachmentIDs)
	assert.Equal(t, []pulid.ID{docID}, *fake.updated.AttachmentIDs)
}

func TestUpdateInvoiceDraft_RefusesACallThatChangesNothing(t *testing.T) {
	t.Parallel()

	tool := newUpdateInvoiceDraftTool(&fakeInvoiceLifecycle{})
	err := tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(map[string]any{
		paramInvoiceID: pulid.MustNew("inv_").String(),
	}))
	require.ErrorIs(t, err, errNothingToUpdate)

	err = tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(map[string]any{
		paramInvoiceID: pulid.MustNew("inv_").String(),
		"emailTo":      []any{"not an address"},
	}))
	require.Error(t, err)
}

func TestGenerateInvoicePDF_RendersButRefusesWhenItWouldEmailTheCustomer(t *testing.T) {
	t.Parallel()

	inv := lifecycleDraft("INV-5002")
	fake := &fakeInvoiceLifecycle{
		guard:   &writeGuard{},
		pdfPlan: &serviceports.InvoicePDFGenerationPlan{Invoice: inv},
	}
	tool := newGenerateInvoicePDFTool(fake)
	params := executeParams(map[string]any{paramInvoiceID: inv.ID.String()})

	preview := previewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would render the PDF of Invoice INV-5002")
	assert.Empty(t, preview.Warnings)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, fake.generated)
	assert.Equal(t, inv.ID, fake.generated.InvoiceID)

	fake.pdfPlan.AutoSends = true
	fake.generated = nil
	err := tool.(serviceports.ToolValidator).Validate(t.Context(), params)
	require.ErrorIs(t, err, errPDFWouldSend)
	preview, err = tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	require.ErrorIs(t, tool.Execute(t.Context(), params), errPDFWouldSend)
	assert.Nil(t, fake.generated)
}

func TestCreateInvoice_PreviewsEachDraftAndOnlyAPersonCreatesThem(t *testing.T) {
	t.Parallel()

	shipper := lifecycleDraft("")
	consignee := lifecycleDraft("")
	consignee.BillToName = "Beta Farms"
	consignee.IsSplitBill = true
	fake := &fakeInvoiceLifecycle{guard: &writeGuard{}, createPlan: &serviceports.CreateInvoicesPlan{
		Drafts:       []*invoice.Invoice{shipper, consignee},
		PrimaryIndex: 1,
	}}
	tool := newCreateInvoiceTool(fake)
	shipmentID := pulid.MustNew("shp_")
	params := executeParams(map[string]any{
		paramShipmentIDs:  []any{shipmentID.String()},
		paramOffCycleNote: "Customer asked for this load on its own",
	})

	preview := previewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would create 2 draft invoices")
	assert.Contains(t, preview.Summary, "Acme Foods")
	assert.Contains(t, preview.Summary, "Beta Farms")
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 0).Operation)
	require.NotNil(t, previewChange(t, preview, 1).Money)
	require.NotNil(t, fake.planned)
	assert.Equal(t, []pulid.ID{shipmentID}, fake.planned.ShipmentIDs)
	assert.Equal(t, "Customer asked for this load on its own", fake.planned.OffCycleReason)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	result, err := tool.(serviceports.ToolResultReporter).
		ExecuteWithResult(t.Context(), approved(params))
	require.NoError(t, err)
	require.NotNil(t, fake.fromShips)
	assert.Nil(t, fake.fromOrder)
	require.NotNil(t, result.Record)
	assert.Equal(t, invoiceRecordEntity, result.Record.EntityType)

	orderID := pulid.MustNew("ord_")
	byOrder := approved(executeParams(map[string]any{paramOrderID: orderID.String()}))
	_, err = tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), byOrder)
	require.NoError(t, err)
	require.NotNil(t, fake.fromOrder)
	assert.Equal(t, orderID, fake.fromOrder.OrderID)

	both := executeParams(map[string]any{})
	err = tool.(serviceports.ToolValidator).Validate(t.Context(), both)
	require.ErrorIs(t, err, errOrderOrShipments)
}

func TestCreateInvoiceMemo_PreviewsTheMemoAndNeverPostsIt(t *testing.T) {
	t.Parallel()

	memo := lifecycleDraft("")
	memo.BillType = billingqueue.BillTypeCreditMemo
	memo.Scope = invoice.ScopeMemo
	memo.TotalAmount = decimal.RequireFromString("-75.50")
	memo.TotalAmountMinor = -7550
	fake := &fakeInvoiceLifecycle{guard: &writeGuard{}, memoPlan: memo}
	tool := newCreateInvoiceMemoTool(fake)
	customerID := pulid.MustNew("cus_")
	params := executeParams(map[string]any{
		paramCustomerID: customerID.String(),
		"billType":      string(billingqueue.BillTypeCreditMemo),
		paramReason:     "Goodwill credit after a late delivery",
		"invoiceDate":   "2026-09-25",
		"lines": []any{
			map[string]any{"description": "Late delivery goodwill", paramAmount: "50.00"},
			map[string]any{"description": "Detention waived", paramAmount: "25.50", "quantity": "1"},
		},
	})

	preview := previewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would raise a draft CreditMemo to Acme Foods for -75.50 USD")
	assert.Contains(t, preview.Summary, "post_invoice")
	require.Len(t, preview.Changes, 1)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	result, err := tool.(serviceports.ToolResultReporter).
		ExecuteWithResult(t.Context(), approved(params))
	require.NoError(t, err)
	require.NotNil(t, fake.memo)
	assert.False(t, fake.memo.AutoPost)
	assert.Equal(t, invoice.MemoKindManual, fake.memo.MemoKind)
	assert.Equal(t, customerID, fake.memo.CustomerID)
	assert.Equal(t, time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC).Unix(), fake.memo.InvoiceDate)
	require.Len(t, fake.memo.Lines, 2)
	assert.True(t, fake.memo.Lines[1].Amount.Equal(decimal.RequireFromString("25.5")))
	assert.True(t, fake.memo.Lines[1].Quantity.Equal(decimal.NewFromInt(1)))
	assert.Equal(t, "CM-0042", result.IDs["number"])

	bad := executeParams(map[string]any{
		paramCustomerID: customerID.String(),
		"billType":      string(billingqueue.BillTypeInvoice),
		paramReason:     "x",
		"lines":         []any{map[string]any{"description": "y", paramAmount: "1"}},
	})
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), bad))
}

func TestVoidInvoice_PreviewsADraftVoidAndAPostedReversal(t *testing.T) {
	t.Parallel()

	before := lifecycleDraft("INV-5003")
	after := *before
	after.Status = invoice.StatusVoided
	after.VoidReason = "Billed the wrong customer"
	after.VoidDisposition = invoice.VoidDispositionRebill
	fake := &fakeInvoiceLifecycle{guard: &writeGuard{}, voidPlan: &serviceports.InvoiceVoidPreview{
		Before: before, After: &after,
	}}
	tool := newVoidInvoiceTool(fake)
	params := executeParams(map[string]any{
		paramInvoiceID:       before.ID.String(),
		paramReason:          "Billed the wrong customer",
		paramVoidDisposition: string(invoice.VoidDispositionRebill),
	})

	preview := previewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would void draft Invoice INV-5003")
	assert.Contains(t, preview.Summary, "back to the billing queue")
	change := previewChange(t, preview, 0)
	assert.Equal(t, "Voided", fieldByPath(t, change, fieldStatus).After)

	posted := receivableInvoice("INV-5004", 100000, 0)
	postedAfter := *posted
	postedAfter.VoidDisposition = invoice.VoidDispositionDoNotRebill
	fake.voidPlan = &serviceports.InvoiceVoidPreview{
		Before: posted, After: &postedAfter, Posted: true,
		Reversal: &serviceports.InvoiceAdjustmentPreview{
			InvoiceID:         posted.ID,
			InvoiceNumber:     "INV-5004",
			Kind:              invoiceadjustment.KindFullReversal,
			CreditTotalAmount: decimal.NewFromInt(-1000),
			RequiresApproval:  true,
			CurrencyCode:      "USD",
		},
	}
	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "full reversal crediting 1000.00")
	assert.Contains(t, preview.Summary, "waits for approval")
	assert.Contains(t, preview.Summary, "canceled")
	require.NotNil(t, previewChange(t, preview, 0).Money)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	_, err = tool.(serviceports.ToolResultReporter).ExecuteWithResult(t.Context(), approved(params))
	require.NoError(t, err)
	require.NotNil(t, fake.voided)
	assert.Equal(t, invoice.VoidDispositionRebill, fake.voided.Disposition)

	policy := tool.Policy()
	assert.Equal(t, permission.OpCancel, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
}

func TestSendInvoiceEDI_PreviewsThePartnerAndOnlyAPersonSends(t *testing.T) {
	t.Parallel()

	inv := receivableInvoice("INV-5005", 100000, 0)
	fake := &fakeInvoiceLifecycle{guard: &writeGuard{}, ediPlan: &serviceports.InvoiceEDISendPreview{
		Invoice: inv,
		Plan: &serviceports.InvoiceEDISendPlan{
			Enabled:             true,
			PartnerName:         "Acme EDI",
			CommunicationMethod: "SFTP",
		},
	}}
	tool := newSendInvoiceEDITool(fake)
	params := executeParams(map[string]any{paramInvoiceID: inv.ID.String()})

	preview := previewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would send Invoice INV-5005 to Acme EDI as an EDI 210")
	change := previewChange(t, preview, 0)
	require.NotNil(t, change.Message)
	assert.Equal(t, agent.MessageChannelEDI, change.Message.Channel)
	assert.Equal(t, []string{"Acme EDI"}, change.Message.To)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	require.NoError(t, tool.Execute(t.Context(), approved(params)))
	require.NotNil(t, fake.ediSent)
	assert.False(t, fake.ediSent.Force)

	fake.refusal = errortypes.NewValidationError("invoiceId", errortypes.ErrInvalidOperation,
		"This invoice has already been sent by EDI; resend it with force")
	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)

	policy := tool.Policy()
	assert.Equal(t, permission.OpSubmit, policy.Operation)
	assert.Equal(t, []agent.EgressClass{agent.EgressExternalRecipient}, policy.Egress)
}

func TestReceivableTools_PinTheRecordTheyChange(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("x_")
	cases := []struct {
		tool     serviceports.AgentTool
		key      string
		resource permission.Resource
	}{
		{newResolveInvoiceDisputeTool(nil), paramDisputeID, permission.ResourceInvoiceDispute},
		{newWithdrawInvoiceDisputeTool(nil), paramDisputeID, permission.ResourceInvoiceDispute},
		{
			newApplyCustomerPaymentTool(nil),
			paramCustomerPaymentID,
			permission.ResourceCustomerPayment,
		},
		{
			newReverseCustomerPaymentTool(nil),
			paramCustomerPaymentID,
			permission.ResourceCustomerPayment,
		},
		{
			newUnapplyCreditMemoTool(nil),
			paramCreditMemoApplicationID,
			serviceports.RecordCreditMemoApplication,
		},
		{
			newApproveInvoiceAdjustmentTool(nil),
			paramAdjustmentID,
			serviceports.RecordInvoiceAdjustment,
		},
		{
			newRejectInvoiceAdjustmentTool(nil),
			paramAdjustmentID,
			serviceports.RecordInvoiceAdjustment,
		},
		{
			newSubmitInvoiceAdjustmentTool(nil),
			paramDraftAdjustmentID,
			serviceports.RecordInvoiceAdjustment,
		},
		{newCommitInvoiceRunTool(nil), paramInvoiceRunID, permission.ResourceInvoiceRun},
		{newCancelInvoiceRunTool(nil), paramInvoiceRunID, permission.ResourceInvoiceRun},
		{newUpdateInvoiceDraftTool(nil), paramInvoiceID, permission.ResourceInvoice},
		{newVoidInvoiceTool(nil), paramInvoiceID, permission.ResourceInvoice},
		{newSendInvoiceEDITool(nil), paramInvoiceID, permission.ResourceInvoice},
	}
	for _, tc := range cases {
		t.Run(tc.tool.Name(), func(t *testing.T) {
			t.Parallel()
			target, ok := tc.tool.(serviceports.TargetedTool).
				Target(map[string]any{tc.key: id.String()})
			require.True(t, ok)
			assert.Equal(t, serviceports.ToolTarget{Resource: tc.resource, ID: id}, target)
		})
	}
}
