package detentioncreditservice

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeInvoices struct {
	net      map[pulid.ID]decimal.Decimal
	invoices map[pulid.ID][]*invoice.Invoice
	memos    map[pulid.ID]*invoice.Invoice
}

func (f *fakeInvoices) NetBilledByCharge(
	_ context.Context,
	req *invoiceports.NetBilledByChargeRequest,
) (map[pulid.ID]decimal.Decimal, error) {
	out := make(map[pulid.ID]decimal.Decimal, len(req.ChargeIDs))
	for _, id := range req.ChargeIDs {
		if amount, ok := f.net[id]; ok {
			out[id] = amount
		}
	}

	return out, nil
}

func (f *fakeInvoices) ListByShipmentIDs(
	_ context.Context,
	_ invoiceports.ListInvoicesByShipmentIDsRequest,
) (map[pulid.ID][]*invoice.Invoice, error) {
	return f.invoices, nil
}

func (f *fakeInvoices) GetByID(
	_ context.Context,
	req invoiceports.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	memo, ok := f.memos[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Invoice not found")
	}

	return memo, nil
}

type fakeAdjustments struct {
	preview   *invoiceadjustmentports.InvoiceAdjustmentPreview
	result    *invoiceadjustment.InvoiceAdjustment
	submitErr error
	previewed []*invoiceadjustmentports.InvoiceAdjustmentRequest
	submitted []*invoiceadjustmentports.InvoiceAdjustmentRequest
}

func (f *fakeAdjustments) Preview(
	_ context.Context,
	req *invoiceadjustmentports.InvoiceAdjustmentRequest,
	_ *permissionports.RequestActor,
) (*invoiceadjustmentports.InvoiceAdjustmentPreview, error) {
	f.previewed = append(f.previewed, req)

	return f.preview, nil
}

func (f *fakeAdjustments) Submit(
	_ context.Context,
	req *invoiceadjustmentports.InvoiceAdjustmentRequest,
	_ *permissionports.RequestActor,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	f.submitted = append(f.submitted, req)
	if f.submitErr != nil {
		return nil, f.submitErr
	}

	return f.result, nil
}

type fakeFinder struct {
	byKey map[string]*invoiceadjustment.InvoiceAdjustment
	asked []string
}

func (f *fakeFinder) GetByIdempotencyKey(
	_ context.Context,
	req invoiceadjustmentports.GetInvoiceAdjustmentByIdempotencyRequest,
) (*invoiceadjustment.InvoiceAdjustment, error) {
	f.asked = append(f.asked, req.IdempotencyKey)
	if adjustment, ok := f.byKey[req.IdempotencyKey]; ok {
		return adjustment, nil
	}

	return nil, errortypes.NewNotFoundError("InvoiceAdjustment not found")
}

type fakePermissions struct {
	allowed bool
	checks  []*permissionports.PermissionCheckRequest
}

func (f *fakePermissions) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	f.checks = append(f.checks, req)

	return &permissionports.PermissionCheckResult{Allowed: f.allowed}, nil
}

type fixture struct {
	service     *Service
	invoices    *fakeInvoices
	adjustments *fakeAdjustments
	finder      *fakeFinder
	permissions *fakePermissions
	source      *invoice.Invoice
	line        *invoice.InvoiceLine
	request     *detentionports.CreditBilledDetentionRequest
}

func newFixture() *fixture {
	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	shipmentID := pulid.MustNew("shp_")
	chargeID := pulid.MustNew("sac_")
	line := &invoice.InvoiceLine{
		ID:                 pulid.MustNew("invl_"),
		Description:        "Detention",
		Amount:             decimal.RequireFromString("185"),
		AdditionalChargeID: chargeID,
	}
	source := &invoice.Invoice{
		ID:           pulid.MustNew("inv_"),
		Number:       "INV-20931",
		Status:       invoice.StatusPosted,
		BillType:     billingqueue.BillTypeInvoice,
		CurrencyCode: "USD",
		Lines: []*invoice.InvoiceLine{
			{ID: pulid.MustNew("invl_"), Amount: decimal.RequireFromString("1500")},
			line,
		},
	}
	memo := &invoice.Invoice{
		ID:       pulid.MustNew("inv_"),
		Number:   "CM-00012",
		BillType: billingqueue.BillTypeCreditMemo,
	}

	f := &fixture{
		invoices: &fakeInvoices{
			net:      map[pulid.ID]decimal.Decimal{chargeID: decimal.RequireFromString("185")},
			invoices: map[pulid.ID][]*invoice.Invoice{shipmentID: {source, memo}},
			memos:    map[pulid.ID]*invoice.Invoice{memo.ID: memo},
		},
		adjustments: &fakeAdjustments{
			preview: &invoiceadjustmentports.InvoiceAdjustmentPreview{
				CreditTotalAmount: decimal.RequireFromString("185"),
				Errors:            map[string][]string{},
				Lines: []*invoiceadjustmentports.InvoiceAdjustmentPreviewLine{{
					OriginalLineID:        line.ID,
					RequestedCreditAmount: decimal.RequireFromString("185"),
				}},
			},
			result: &invoiceadjustment.InvoiceAdjustment{
				ID:                  pulid.MustNew("iadj_"),
				Status:              invoiceadjustment.StatusExecuted,
				CreditTotalAmount:   decimal.RequireFromString("185"),
				CreditMemoInvoiceID: memo.ID,
			},
		},
		finder:      &fakeFinder{byKey: map[string]*invoiceadjustment.InvoiceAdjustment{}},
		permissions: &fakePermissions{allowed: true},
		source:      source,
		line:        line,
		request: &detentionports.CreditBilledDetentionRequest{
			TenantInfo: tenantInfo,
			Actor:      permissionports.UserActor(tenantInfo),
			ShipmentID: shipmentID,
			ChargeID:   chargeID,
			Reason:     "Detention waived: Decided in Triage",
			Waiver: detentionports.DetentionWaiver{
				OccurrenceID: pulid.MustNew("dto_"),
				Note:         "Decided in Triage",
				RequestedBy:  tenantInfo.UserID,
			},
		},
	}
	f.service = &Service{
		l:           zap.NewNop(),
		invoices:    f.invoices,
		adjustments: f.adjustments,
		finder:      f.finder,
		permissions: f.permissions,
	}

	return f
}

func refusedField(t *testing.T, err error) *errortypes.Error {
	t.Helper()

	fieldErr, ok := errors.AsType[*errortypes.Error](err)
	require.True(t, ok, "expected a field error, got %v", err)

	return fieldErr
}

func TestCreditBilledChargeLeavesAnUnbilledChargeToTheWaiver(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.invoices.net = map[pulid.ID]decimal.Decimal{}

	credit, err := f.service.CreditBilledCharge(t.Context(), f.request)
	require.NoError(t, err)
	assert.Nil(t, credit)
	assert.Empty(t, f.permissions.checks, "an unbilled waiver needs no invoice permission")
	assert.Empty(t, f.adjustments.submitted)
}

func TestCreditBilledChargeIssuesACreditForTheDetentionLine(t *testing.T) {
	t.Parallel()

	f := newFixture()

	credit, err := f.service.CreditBilledCharge(t.Context(), f.request)
	require.NoError(t, err)
	require.NotNil(t, credit)

	require.Len(t, f.permissions.checks, 1)
	assert.Equal(t, permission.ResourceInvoice.String(), f.permissions.checks[0].Resource)
	assert.Equal(t, permission.OpUpdate, f.permissions.checks[0].Operation)

	require.Len(t, f.adjustments.submitted, 1)
	submitted := f.adjustments.submitted[0]
	assert.Equal(t, f.source.ID, submitted.InvoiceID)
	assert.Equal(t, invoiceadjustment.KindCreditOnly, submitted.Kind)
	assert.Equal(t, f.request.Reason, submitted.Reason)
	require.Len(t, submitted.Lines, 1)
	assert.Equal(t, f.line.ID, submitted.Lines[0].OriginalLineID)
	assert.True(t, submitted.Lines[0].CreditAmount.IsZero(), "the remaining eligible amount is credited")
	assert.Equal(t, []detentionports.DetentionWaiver{f.request.Waiver}, submitted.DetentionWaivers)
	assert.Contains(t, submitted.IdempotencyKey, f.request.Waiver.OccurrenceID.String())

	assert.True(t, credit.Amount.Equal(decimal.RequireFromString("185")))
	assert.Equal(t, "USD", credit.Currency)
	assert.False(t, credit.Pending())
	assert.Equal(t, []string{"INV-20931"}, credit.InvoiceNumbers())
	assert.Equal(t, []string{"CM-00012"}, credit.CreditMemoNumbers())
}

func TestCreditBilledChargeWaitsForAnApprover(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.adjustments.result.Status = invoiceadjustment.StatusPendingApproval
	f.adjustments.result.CreditMemoInvoiceID = pulid.Nil

	credit, err := f.service.CreditBilledCharge(t.Context(), f.request)
	require.NoError(t, err)
	assert.True(t, credit.Pending())
	assert.Empty(t, credit.CreditMemoNumbers())
}

func TestCreditBilledChargeRefusesWithoutInvoicePermission(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.permissions.allowed = false

	_, err := f.service.CreditBilledCharge(t.Context(), f.request)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Empty(t, f.adjustments.submitted)
}

func TestCreditBilledChargeTurnsAPreviewRefusalIntoAFieldError(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.adjustments.preview.Errors = map[string][]string{
		"attachmentIds": {"Supporting documents are required by policy"},
	}

	_, err := f.service.CreditBilledCharge(t.Context(), f.request)
	require.Error(t, err)
	fieldErr := refusedField(t, err)
	assert.Equal(t, refusalField, fieldErr.Field)
	assert.Contains(t, fieldErr.Error(), "INV-20931")
	assert.Contains(t, fieldErr.Error(), "Supporting documents are required by policy")
	assert.Empty(t, f.adjustments.submitted)
}

func TestCreditBilledChargeTurnsASubmitRefusalIntoAFieldError(t *testing.T) {
	t.Parallel()

	f := newFixture()
	multiErr := errortypes.NewMultiError()
	multiErr.Add("invoiceId", errortypes.ErrInvalidOperation, "Only posted invoices may be adjusted")
	f.adjustments.submitErr = multiErr

	_, err := f.service.CreditBilledCharge(t.Context(), f.request)
	fieldErr := refusedField(t, err)
	assert.Equal(t, refusalField, fieldErr.Field)
	assert.Contains(t, fieldErr.Error(), "Only posted invoices may be adjusted")
}

func TestCreditBilledChargeRefusesAFailedExecution(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.adjustments.result.Status = invoiceadjustment.StatusExecutionFailed
	f.adjustments.result.ExecutionError = "The accounting period is closed"

	_, err := f.service.CreditBilledCharge(t.Context(), f.request)
	fieldErr := refusedField(t, err)
	assert.Contains(t, fieldErr.Error(), "The accounting period is closed")
}

func TestCreditBilledChargeRefusesWhileTheInvoiceIsADraft(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.source.Status = invoice.StatusDraft

	_, err := f.service.CreditBilledCharge(t.Context(), f.request)
	fieldErr := refusedField(t, err)
	assert.Equal(t, refusalField, fieldErr.Field)
	assert.Empty(t, f.adjustments.previewed)
}

func TestCreditBilledChargeReturnsTheCreditAlreadyWaitingForApproval(t *testing.T) {
	t.Parallel()

	f := newFixture()
	pending := &invoiceadjustment.InvoiceAdjustment{
		ID:                pulid.MustNew("iadj_"),
		Status:            invoiceadjustment.StatusPendingApproval,
		CreditTotalAmount: decimal.RequireFromString("185"),
	}
	f.finder.byKey[keyFor(f, 1)] = pending

	credit, err := f.service.CreditBilledCharge(t.Context(), f.request)
	require.NoError(t, err)
	assert.Empty(t, f.adjustments.submitted)
	require.Len(t, credit.Invoices, 1)
	assert.Equal(t, pending.ID, credit.Invoices[0].AdjustmentID)
	assert.True(t, credit.Pending())
}

func TestCreditBilledChargeTriesAgainAfterARejectedCredit(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.finder.byKey[keyFor(f, 1)] = &invoiceadjustment.InvoiceAdjustment{
		ID:     pulid.MustNew("iadj_"),
		Status: invoiceadjustment.StatusRejected,
	}

	_, err := f.service.CreditBilledCharge(t.Context(), f.request)
	require.NoError(t, err)
	require.Len(t, f.adjustments.submitted, 1)
	assert.Equal(t, keyFor(f, 2), f.adjustments.submitted[0].IdempotencyKey)
}

func keyFor(f *fixture, attempt int) string {
	return fmt.Sprintf(idempotencyKey, f.request.Waiver.OccurrenceID, f.source.ID, attempt)
}
