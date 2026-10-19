package agenttoolservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/receivabletools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInvoiceLifecycle struct {
	guard *agenttoolstest.WriteGuard

	draftPlan  *invoiceports.InvoiceDraftUpdatePreview
	updated    *invoiceports.UpdateInvoiceDraftRequest
	pdfPlan    *invoiceports.InvoicePDFGenerationPlan
	generated  *invoiceports.InvoicePreviewRequest
	createPlan *invoiceports.CreateInvoicesPlan
	planned    *invoiceports.PlanCreateInvoicesRequest
	fromOrder  *invoiceports.CreateInvoiceFromOrderRequest
	fromShips  *invoiceports.CreateInvoiceFromShipmentsRequest
	memoPlan   *invoice.Invoice
	memo       *invoiceports.CreateMemoRequest
	voidPlan   *invoiceports.InvoiceVoidPreview
	voided     *invoiceports.VoidInvoiceRequest
	ediPlan    *invoiceports.InvoiceEDISendPreview
	ediSent    *invoiceports.SendInvoiceEDIRequest
	refusal    error
}

func (f *fakeInvoiceLifecycle) plan() error { return f.refusal }

func (f *fakeInvoiceLifecycle) UpdateDraft(
	_ context.Context,
	req *invoiceports.UpdateInvoiceDraftRequest,
	_ *permissionports.RequestActor,
) (*invoice.Invoice, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.updated = req

	return f.draftPlan.After, nil
}

func (f *fakeInvoiceLifecycle) PreviewUpdateDraft(
	context.Context,
	*invoiceports.UpdateInvoiceDraftRequest,
) (*invoiceports.InvoiceDraftUpdatePreview, error) {
	return f.draftPlan, f.plan()
}

func (f *fakeInvoiceLifecycle) GeneratePDF(
	_ context.Context,
	req *invoiceports.InvoicePreviewRequest,
	_ *permissionports.RequestActor,
) (*invoiceports.GenerateInvoicePDFResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.generated = req

	return &invoiceports.GenerateInvoicePDFResult{InvoiceID: req.InvoiceID, Status: "Queued"}, nil
}

func (f *fakeInvoiceLifecycle) PlanPDFGeneration(
	context.Context,
	*invoiceports.InvoicePreviewRequest,
	*permissionports.RequestActor,
) (*invoiceports.InvoicePDFGenerationPlan, error) {
	return f.pdfPlan, f.plan()
}

func (f *fakeInvoiceLifecycle) PlanCreateInvoices(
	_ context.Context,
	req *invoiceports.PlanCreateInvoicesRequest,
) (*invoiceports.CreateInvoicesPlan, error) {
	f.planned = req

	return f.createPlan, f.plan()
}

func (f *fakeInvoiceLifecycle) createdInvoices() *invoiceports.CreateInvoicesResult {
	made := make([]*invoice.Invoice, 0, len(f.createPlan.Drafts))
	for _, draft := range f.createPlan.Drafts {
		created := *draft
		created.ID = pulid.MustNew("inv_")
		created.Number = "INV-7001"
		made = append(made, &created)
	}

	return &invoiceports.CreateInvoicesResult{
		Invoices: made,
		Primary:  made[f.createPlan.PrimaryIndex],
	}
}

func (f *fakeInvoiceLifecycle) CreateInvoicesFromOrder(
	_ context.Context,
	req *invoiceports.CreateInvoiceFromOrderRequest,
	_ *permissionports.RequestActor,
) (*invoiceports.CreateInvoicesResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.fromOrder = req

	return f.createdInvoices(), nil
}

func (f *fakeInvoiceLifecycle) CreateInvoicesFromShipments(
	_ context.Context,
	req *invoiceports.CreateInvoiceFromShipmentsRequest,
	_ *permissionports.RequestActor,
) (*invoiceports.CreateInvoicesResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.fromShips = req

	return f.createdInvoices(), nil
}

func (f *fakeInvoiceLifecycle) CreateMemo(
	_ context.Context,
	req *invoiceports.CreateMemoRequest,
	_ *permissionports.RequestActor,
) (*invoice.Invoice, error) {
	if err := f.guard.Write(); err != nil {
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
	*invoiceports.CreateMemoRequest,
) (*invoice.Invoice, error) {
	return f.memoPlan, f.plan()
}

func (f *fakeInvoiceLifecycle) VoidInvoice(
	_ context.Context,
	req *invoiceports.VoidInvoiceRequest,
	_ *permissionports.RequestActor,
) (*invoiceports.VoidInvoiceResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.voided = req

	return &invoiceports.VoidInvoiceResult{Invoice: f.voidPlan.After}, nil
}

func (f *fakeInvoiceLifecycle) PreviewVoid(
	context.Context,
	*invoiceports.VoidInvoiceRequest,
	*permissionports.RequestActor,
) (*invoiceports.InvoiceVoidPreview, error) {
	return f.voidPlan, f.plan()
}

func (f *fakeInvoiceLifecycle) SendEDI(
	_ context.Context,
	req *invoiceports.SendInvoiceEDIRequest,
	_ *permissionports.RequestActor,
) (*invoiceports.InvoiceEDISendResult, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.ediSent = req

	return &invoiceports.InvoiceEDISendResult{InvoiceID: req.InvoiceID}, nil
}

func (f *fakeInvoiceLifecycle) PreviewSendEDI(
	context.Context,
	*invoiceports.SendInvoiceEDIRequest,
	*permissionports.RequestActor,
) (*invoiceports.InvoiceEDISendPreview, error) {
	return f.ediPlan, f.plan()
}

func lifecycleDraft(number string) *invoice.Invoice {
	inv := agenttoolstest.ReceivableInvoice(number, 165000, 0)
	inv.Status = invoice.StatusDraft
	inv.BillToName = "Acme Foods"
	inv.SubtotalAmount = decimal.RequireFromString("1500.00")
	inv.OtherAmount = decimal.RequireFromString("150.00")
	inv.TotalAmount = decimal.RequireFromString("1650.00")

	return inv
}

func TestUpdateInvoiceDraft_PreviewsTheChangeAndProposesNewRecipients(t *testing.T) {
	t.Parallel()

	before := lifecycleDraft("INV-5001")
	after := *before
	after.Memo = "Detention approved by the shipper"
	after.EmailToSnapshot = []string{"ap@acme.test"}
	docID := pulid.MustNew("doc_")
	fake := &fakeInvoiceLifecycle{guard: &agenttoolstest.WriteGuard{}, draftPlan: &invoiceports.InvoiceDraftUpdatePreview{
		Before:           before,
		After:            &after,
		AttachmentsAfter: []pulid.ID{docID},
	}}
	tool := invoicetools.NewUpdateInvoiceDraftTool(fake)
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamInvoiceID:   before.ID.String(),
		"memo":                  "Detention approved by the shipper",
		"emailTo":               []any{"ap@acme.test"},
		"emailCc":               []any{},
		"attachmentDocumentIds": []any{docID.String()},
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would update draft Invoice INV-5001")
	assert.Contains(t, preview.Summary, "1 attachment")
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceInvoice, change.Resource)
	assert.Equal(t, before.ID, change.EntityID)
	assert.Equal(t, "Detention approved by the shipper", agenttoolstest.FieldByPath(t, change, "memo").After)

	policy := tool.Policy()
	require.NotNil(t, policy.Condition)
	assert.Equal(t, agent.TierPropose, policy.Condition.Limit(t.Context(), params))
	quiet := agenttoolstest.ExecuteParams(map[string]any{common.ParamInvoiceID: before.ID.String(), "memo": "x"})
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

	tool := invoicetools.NewUpdateInvoiceDraftTool(&fakeInvoiceLifecycle{})
	err := tool.(agentports.ToolValidator).Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		common.ParamInvoiceID: pulid.MustNew("inv_").String(),
	}))
	require.ErrorIs(t, err, invoicetools.ErrNothingToUpdate)

	err = tool.(agentports.ToolValidator).Validate(t.Context(), agenttoolstest.ExecuteParams(map[string]any{
		common.ParamInvoiceID: pulid.MustNew("inv_").String(),
		"emailTo":             []any{"not an address"},
	}))
	require.Error(t, err)
}

func TestGenerateInvoicePDF_RendersButRefusesWhenItWouldEmailTheCustomer(t *testing.T) {
	t.Parallel()

	inv := lifecycleDraft("INV-5002")
	fake := &fakeInvoiceLifecycle{
		guard:   &agenttoolstest.WriteGuard{},
		pdfPlan: &invoiceports.InvoicePDFGenerationPlan{Invoice: inv},
	}
	tool := invoicetools.NewGenerateInvoicePDFTool(fake)
	params := agenttoolstest.ExecuteParams(map[string]any{common.ParamInvoiceID: inv.ID.String()})

	preview := agenttoolstest.PreviewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would render the PDF of Invoice INV-5002")
	assert.Empty(t, preview.Warnings)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, fake.generated)
	assert.Equal(t, inv.ID, fake.generated.InvoiceID)

	fake.pdfPlan.AutoSends = true
	fake.generated = nil
	err := tool.(agentports.ToolValidator).Validate(t.Context(), params)
	require.ErrorIs(t, err, invoicetools.ErrPDFWouldSend)
	preview, err = tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
	require.ErrorIs(t, tool.Execute(t.Context(), params), invoicetools.ErrPDFWouldSend)
	assert.Nil(t, fake.generated)
}

func TestCreateInvoice_PreviewsEachDraftAndOnlyAPersonCreatesThem(t *testing.T) {
	t.Parallel()

	shipper := lifecycleDraft("")
	consignee := lifecycleDraft("")
	consignee.BillToName = "Beta Farms"
	consignee.IsSplitBill = true
	fake := &fakeInvoiceLifecycle{guard: &agenttoolstest.WriteGuard{}, createPlan: &invoiceports.CreateInvoicesPlan{
		Drafts:       []*invoice.Invoice{shipper, consignee},
		PrimaryIndex: 1,
	}}
	tool := invoicetools.NewCreateInvoiceTool(fake)
	shipmentID := pulid.MustNew("shp_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamShipmentIDs:        []any{shipmentID.String()},
		invoicetools.ParamOffCycleNote: "Customer asked for this load on its own",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would create 2 draft invoices")
	assert.Contains(t, preview.Summary, "Acme Foods")
	assert.Contains(t, preview.Summary, "Beta Farms")
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, agent.PreviewOperationCreate, agenttoolstest.PreviewChange(t, preview, 0).Operation)
	require.NotNil(t, agenttoolstest.PreviewChange(t, preview, 1).Money)
	require.NotNil(t, fake.planned)
	assert.Equal(t, []pulid.ID{shipmentID}, fake.planned.ShipmentIDs)
	assert.Equal(t, "Customer asked for this load on its own", fake.planned.OffCycleReason)

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
	result, err := tool.(agentports.ToolResultReporter).
		ExecuteWithResult(t.Context(), agenttoolstest.Approved(params))
	require.NoError(t, err)
	require.NotNil(t, fake.fromShips)
	assert.Nil(t, fake.fromOrder)
	require.NotNil(t, result.Record)
	assert.Equal(t, common.InvoiceRecordEntity, result.Record.EntityType)

	orderID := pulid.MustNew("ord_")
	byOrder := agenttoolstest.Approved(agenttoolstest.ExecuteParams(map[string]any{common.ParamOrderID: orderID.String()}))
	_, err = tool.(agentports.ToolResultReporter).ExecuteWithResult(t.Context(), byOrder)
	require.NoError(t, err)
	require.NotNil(t, fake.fromOrder)
	assert.Equal(t, orderID, fake.fromOrder.OrderID)

	both := agenttoolstest.ExecuteParams(map[string]any{})
	err = tool.(agentports.ToolValidator).Validate(t.Context(), both)
	require.ErrorIs(t, err, invoicetools.ErrOrderOrShipments)
}

func TestCreateInvoiceMemo_PreviewsTheMemoAndNeverPostsIt(t *testing.T) {
	t.Parallel()

	memo := lifecycleDraft("")
	memo.BillType = billingqueue.BillTypeCreditMemo
	memo.Scope = invoice.ScopeMemo
	memo.TotalAmount = decimal.RequireFromString("-75.50")
	memo.TotalAmountMinor = -7550
	fake := &fakeInvoiceLifecycle{guard: &agenttoolstest.WriteGuard{}, memoPlan: memo}
	tool := invoicetools.NewCreateInvoiceMemoTool(fake)
	customerID := pulid.MustNew("cus_")
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamCustomerID: customerID.String(),
		"billType":             string(billingqueue.BillTypeCreditMemo),
		common.ParamReason:     "Goodwill credit after a late delivery",
		"invoiceDate":          "2026-09-25",
		"lines": []any{
			map[string]any{"description": "Late delivery goodwill", common.ParamAmount: "50.00"},
			map[string]any{"description": "Detention waived", common.ParamAmount: "25.50", "quantity": "1"},
		},
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would raise a draft CreditMemo to Acme Foods for -75.50 USD")
	assert.Contains(t, preview.Summary, "invoice_post")
	require.Len(t, preview.Changes, 1)

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
	result, err := tool.(agentports.ToolResultReporter).
		ExecuteWithResult(t.Context(), agenttoolstest.Approved(params))
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

	bad := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamCustomerID: customerID.String(),
		"billType":             string(billingqueue.BillTypeInvoice),
		common.ParamReason:     "x",
		"lines":                []any{map[string]any{"description": "y", common.ParamAmount: "1"}},
	})
	require.Error(t, tool.(agentports.ToolValidator).Validate(t.Context(), bad))
}

func TestVoidInvoice_PreviewsADraftVoidAndAPostedReversal(t *testing.T) {
	t.Parallel()

	before := lifecycleDraft("INV-5003")
	after := *before
	after.Status = invoice.StatusVoided
	after.VoidReason = "Billed the wrong customer"
	after.VoidDisposition = invoice.VoidDispositionRebill
	fake := &fakeInvoiceLifecycle{guard: &agenttoolstest.WriteGuard{}, voidPlan: &invoiceports.InvoiceVoidPreview{
		Before: before, After: &after,
	}}
	tool := invoicetools.NewVoidInvoiceTool(fake)
	params := agenttoolstest.ExecuteParams(map[string]any{
		common.ParamInvoiceID:             before.ID.String(),
		common.ParamReason:                "Billed the wrong customer",
		invoicetools.ParamVoidDisposition: string(invoice.VoidDispositionRebill),
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would void draft Invoice INV-5003")
	assert.Contains(t, preview.Summary, "back to the billing queue")
	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Voided", agenttoolstest.FieldByPath(t, change, common.FieldStatus).After)

	posted := agenttoolstest.ReceivableInvoice("INV-5004", 100000, 0)
	postedAfter := *posted
	postedAfter.VoidDisposition = invoice.VoidDispositionDoNotRebill
	fake.voidPlan = &invoiceports.InvoiceVoidPreview{
		Before: posted, After: &postedAfter, Posted: true,
		Reversal: &invoiceadjustmentports.InvoiceAdjustmentPreview{
			InvoiceID:         posted.ID,
			InvoiceNumber:     "INV-5004",
			Kind:              invoiceadjustment.KindFullReversal,
			CreditTotalAmount: decimal.NewFromInt(-1000),
			RequiresApproval:  true,
			CurrencyCode:      "USD",
		},
	}
	preview, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "full reversal crediting 1000.00")
	assert.Contains(t, preview.Summary, "waits for approval")
	assert.Contains(t, preview.Summary, "canceled")
	require.NotNil(t, agenttoolstest.PreviewChange(t, preview, 0).Money)

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
	_, err = tool.(agentports.ToolResultReporter).ExecuteWithResult(t.Context(), agenttoolstest.Approved(params))
	require.NoError(t, err)
	require.NotNil(t, fake.voided)
	assert.Equal(t, invoice.VoidDispositionRebill, fake.voided.Disposition)

	policy := tool.Policy()
	assert.Equal(t, permission.OpCancel, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
}

func TestSendInvoiceEDI_PreviewsThePartnerAndOnlyAPersonSends(t *testing.T) {
	t.Parallel()

	inv := agenttoolstest.ReceivableInvoice("INV-5005", 100000, 0)
	fake := &fakeInvoiceLifecycle{guard: &agenttoolstest.WriteGuard{}, ediPlan: &invoiceports.InvoiceEDISendPreview{
		Invoice: inv,
		Plan: &invoiceports.InvoiceEDISendPlan{
			Enabled:             true,
			PartnerName:         "Acme EDI",
			CommunicationMethod: "SFTP",
		},
	}}
	tool := invoicetools.NewSendInvoiceEDITool(fake)
	params := agenttoolstest.ExecuteParams(map[string]any{common.ParamInvoiceID: inv.ID.String()})

	preview := agenttoolstest.PreviewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
		return tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "Would send Invoice INV-5005 to Acme EDI as an EDI 210")
	change := agenttoolstest.PreviewChange(t, preview, 0)
	require.NotNil(t, change.Message)
	assert.Equal(t, agent.MessageChannelEDI, change.Message.Channel)
	assert.Equal(t, []string{"Acme EDI"}, change.Message.To)

	require.ErrorIs(t, tool.Execute(t.Context(), params), common.ErrNeedsAPersonsApproval)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.Approved(params)))
	require.NotNil(t, fake.ediSent)
	assert.False(t, fake.ediSent.Force)

	fake.refusal = errortypes.NewValidationError("invoiceId", errortypes.ErrInvalidOperation,
		"This invoice has already been sent by EDI; resend it with force")
	preview, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)

	policy := tool.Policy()
	assert.Equal(t, permission.OpSubmit, policy.Operation)
	assert.Equal(t, []agent.EgressClass{agent.EgressExternalRecipient}, policy.Egress)
}

func TestReceivableTools_PinTheRecordTheyChange(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("x_")
	cases := []struct {
		tool     agentports.AgentTool
		key      string
		resource permission.Resource
	}{
		{invoicetools.NewResolveInvoiceDisputeTool(nil), common.ParamDisputeID, permission.ResourceInvoiceDispute},
		{invoicetools.NewWithdrawInvoiceDisputeTool(nil), common.ParamDisputeID, permission.ResourceInvoiceDispute},
		{
			receivabletools.NewApplyCustomerPaymentTool(nil),
			receivabletools.ParamCustomerPaymentID,
			permission.ResourceCustomerPayment,
		},
		{
			receivabletools.NewReverseCustomerPaymentTool(nil),
			receivabletools.ParamCustomerPaymentID,
			permission.ResourceCustomerPayment,
		},
		{
			receivabletools.NewUnapplyCreditMemoTool(nil),
			receivabletools.ParamCreditMemoApplicationID,
			agentports.RecordCreditMemoApplication,
		},
		{
			invoicetools.NewApproveInvoiceAdjustmentTool(nil),
			invoicetools.ParamAdjustmentID,
			agentports.RecordInvoiceAdjustment,
		},
		{
			invoicetools.NewRejectInvoiceAdjustmentTool(nil),
			invoicetools.ParamAdjustmentID,
			agentports.RecordInvoiceAdjustment,
		},
		{
			invoicetools.NewSubmitInvoiceAdjustmentTool(nil),
			invoicetools.ParamDraftAdjustmentID,
			agentports.RecordInvoiceAdjustment,
		},
		{invoicetools.NewCommitInvoiceRunTool(nil), invoicetools.ParamInvoiceRunID, permission.ResourceInvoiceRun},
		{invoicetools.NewCancelInvoiceRunTool(nil), invoicetools.ParamInvoiceRunID, permission.ResourceInvoiceRun},
		{invoicetools.NewUpdateInvoiceDraftTool(nil), common.ParamInvoiceID, permission.ResourceInvoice},
		{invoicetools.NewVoidInvoiceTool(nil), common.ParamInvoiceID, permission.ResourceInvoice},
		{invoicetools.NewSendInvoiceEDITool(nil), common.ParamInvoiceID, permission.ResourceInvoice},
	}
	for _, tc := range cases {
		t.Run(tc.tool.Name(), func(t *testing.T) {
			t.Parallel()
			target, ok := tc.tool.(agentports.TargetedTool).
				Target(map[string]any{tc.key: id.String()})
			require.True(t, ok)
			assert.Equal(t, agentports.ToolTarget{Resource: tc.resource, ID: id}, target)
		})
	}
}
