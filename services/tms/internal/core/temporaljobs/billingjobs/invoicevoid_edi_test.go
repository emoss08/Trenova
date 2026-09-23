package billingjobs

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

func TestInvoicePDFResourceAnchorsByWhatTheInvoiceHas(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	orderID := pulid.MustNew("ord_")
	customerID := pulid.MustNew("cus_")

	resource, err := invoicePDFResource(
		&invoice.Invoice{ShipmentID: shipmentID, OrderID: orderID, CustomerID: customerID},
	)
	require.NoError(t, err)
	assert.Equal(t, invoicePDFDocumentResource{Type: "shipment", ID: shipmentID.String()}, resource)

	resource, err = invoicePDFResource(&invoice.Invoice{OrderID: orderID, CustomerID: customerID})
	require.NoError(t, err)
	assert.Equal(t, invoicePDFDocumentResource{Type: "order", ID: orderID.String()}, resource)

	resource, err = invoicePDFResource(
		&invoice.Invoice{CustomerID: customerID, Scope: invoice.ScopeMemo},
	)
	require.NoError(t, err)
	assert.Equal(
		t,
		invoicePDFDocumentResource{Type: "customer", ID: customerID.String()},
		resource,
		"memos and statements file under the customer",
	)

	_, err = invoicePDFResource(
		&invoice.Invoice{ShipmentID: shipmentID, Status: invoice.StatusVoided},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Voided invoices do not generate documents")

	_, err = invoicePDFResource(&invoice.Invoice{})
	require.Error(t, err)
	_, err = invoicePDFResource(nil)
	require.Error(t, err)
}

func autoPostPayload(invoiceID pulid.ID, tenantInfo pagination.TenantInfo) *AutoPostInvoicePayload {
	return &AutoPostInvoicePayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			UserID:         tenantInfo.UserID,
		},
		InvoiceID: invoiceID,
	}
}

func TestAutoPostInvoiceActivityTreatsVoidedAsDone(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	invoiceID := pulid.MustNew("inv_")
	invoiceService := mocks.NewMockInvoiceService(t)
	invoiceService.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{ID: invoiceID, TenantInfo: tenantInfo}).
		Return(&invoice.Invoice{ID: invoiceID, Status: invoice.StatusVoided}, nil).
		Once()

	a := &Activities{invoiceService: invoiceService, logger: zap.NewNop()}
	result, err := a.AutoPostInvoiceActivity(t.Context(), autoPostPayload(invoiceID, tenantInfo))

	require.NoError(t, err)
	assert.True(t, result.AlreadyPosted)
	assert.True(t, result.Voided)
	assert.Equal(t, int64(0), result.PostedAt)
	invoiceService.AssertNotCalled(t, "Post", mock.Anything, mock.Anything, mock.Anything)
}

func TestAutoPostInvoiceActivityReportsAPostedInvoiceWithoutVoidFlag(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	invoiceID := pulid.MustNew("inv_")
	postedAt := int64(1_700_000_000)
	invoiceService := mocks.NewMockInvoiceService(t)
	invoiceService.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&invoice.Invoice{ID: invoiceID, Status: invoice.StatusPosted, PostedAt: &postedAt}, nil).
		Once()

	a := &Activities{invoiceService: invoiceService, logger: zap.NewNop()}
	result, err := a.AutoPostInvoiceActivity(t.Context(), autoPostPayload(invoiceID, tenantInfo))

	require.NoError(t, err)
	assert.True(t, result.AlreadyPosted)
	assert.False(t, result.Voided)
	assert.Equal(t, postedAt, result.PostedAt)
}

type ediActivityFixture struct {
	tenantInfo     pagination.TenantInfo
	inv            *invoice.Invoice
	invoiceService *mocks.MockInvoiceService
	invoiceRepo    *mocks.MockInvoiceRepository
	ediService     *mocks.MockEDIService
	activities     *Activities
}

func newEDIActivityFixture(t *testing.T) *ediActivityFixture {
	t.Helper()
	f := &ediActivityFixture{
		tenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		invoiceService: mocks.NewMockInvoiceService(t),
		invoiceRepo:    mocks.NewMockInvoiceRepository(t),
		ediService:     mocks.NewMockEDIService(t),
	}
	f.inv = &invoice.Invoice{
		ID:            pulid.MustNew("inv_"),
		Status:        invoice.StatusPosted,
		EDISendStatus: invoice.EDISendStatusQueued,
	}
	f.invoiceService.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{ID: f.inv.ID, TenantInfo: f.tenantInfo}).
		Return(f.inv, nil).
		Maybe()
	f.activities = &Activities{
		invoiceService: f.invoiceService,
		invoiceRepo:    f.invoiceRepo,
		ediService:     f.ediService,
		logger:         zap.NewNop(),
	}

	return f
}

func (f *ediActivityFixture) payload(force bool) *SendInvoiceEDIPayload {
	return &SendInvoiceEDIPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: f.tenantInfo.OrgID,
			BusinessUnitID: f.tenantInfo.BuID,
			UserID:         f.tenantInfo.UserID,
		},
		InvoiceID: f.inv.ID,
		Force:     force,
	}
}

func (f *ediActivityFixture) expectPlan(plan *services.InvoiceEDISendPlan) {
	f.invoiceService.EXPECT().
		ResolveEDISendPlans(mock.Anything, &services.ResolveInvoiceEDISendPlansRequest{
			TenantInfo: f.tenantInfo,
			Invoices:   []*invoice.Invoice{f.inv},
		}).
		Return(map[pulid.ID]*services.InvoiceEDISendPlan{f.inv.ID: plan}, nil).
		Once()
}

func readyPlan(method edi.ConnectionMethod) *services.InvoiceEDISendPlan {
	return &services.InvoiceEDISendPlan{
		Enabled:             true,
		PartnerID:           pulid.MustNew("edip_"),
		DocumentProfileID:   pulid.MustNew("epdp_"),
		CommunicationMethod: string(method),
		Blockers:            []string{},
	}
}

func TestSendInvoiceEDIActivityRefusesAnUnpostedInvoice(t *testing.T) {
	t.Parallel()

	f := newEDIActivityFixture(t)
	f.inv.Status = invoice.StatusDraft

	_, err := f.activities.SendInvoiceEDIActivity(t.Context(), f.payload(false))

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	assert.True(t, appErr.NonRetryable())
}

func TestSendInvoiceEDIActivitySkipsAnAlreadySentInvoiceUnlessForced(t *testing.T) {
	t.Parallel()

	f := newEDIActivityFixture(t)
	f.inv.EDISendStatus = invoice.EDISendStatusSent
	f.inv.LastEDIMessageID = pulid.MustNew("emsg_")

	result, err := f.activities.SendInvoiceEDIActivity(t.Context(), f.payload(false))

	require.NoError(t, err)
	assert.Equal(t, invoice.EDISendStatusSent, result.Status)
	assert.Equal(t, f.inv.LastEDIMessageID, result.MessageID)
	f.ediService.AssertNotCalled(t, "GenerateDocument", mock.Anything, mock.Anything)
}

func TestSendInvoiceEDIActivityMarksNotConfiguredWithTheFirstBlocker(t *testing.T) {
	t.Parallel()

	f := newEDIActivityFixture(t)
	f.expectPlan(
		&services.InvoiceEDISendPlan{Enabled: true, Blockers: []string{"no partner", "no profile"}},
	)
	f.invoiceRepo.EXPECT().
		UpdateEDISendStatus(mock.Anything, repositories.UpdateInvoiceEDISendStatusRequest{
			TenantInfo: f.tenantInfo,
			InvoiceID:  f.inv.ID,
			Status:     invoice.EDISendStatusNotConfigured,
			Error:      "no partner",
		}).
		Return(nil).
		Once()

	result, err := f.activities.SendInvoiceEDIActivity(t.Context(), f.payload(false))

	require.NoError(t, err)
	assert.Equal(t, invoice.EDISendStatusNotConfigured, result.Status)
	f.ediService.AssertNotCalled(t, "GenerateDocument", mock.Anything, mock.Anything)
}

func TestSendInvoiceEDIActivityRecordsGenerationFailures(t *testing.T) {
	t.Parallel()

	t.Run("a validation failure is not retried", func(t *testing.T) {
		t.Parallel()
		f := newEDIActivityFixture(t)
		f.expectPlan(readyPlan(edi.ConnectionMethodSFTP))
		multiErr := errortypes.NewMultiError()
		multiErr.Add("payload.billTo", errortypes.ErrRequired, "Bill-to is required")
		f.ediService.EXPECT().
			GenerateDocument(mock.Anything, mock.Anything).
			Return(nil, multiErr).
			Once()
		f.invoiceRepo.EXPECT().
			UpdateEDISendStatus(mock.Anything, mock.MatchedBy(func(req repositories.UpdateInvoiceEDISendStatusRequest) bool {
				return req.InvoiceID == f.inv.ID && req.Status == invoice.EDISendStatusFailed &&
					req.Error != ""
			})).
			Return(nil).
			Once()

		_, err := f.activities.SendInvoiceEDIActivity(t.Context(), f.payload(false))

		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		assert.True(t, appErr.NonRetryable())
	})

	t.Run("an outage is retried", func(t *testing.T) {
		t.Parallel()
		f := newEDIActivityFixture(t)
		f.expectPlan(readyPlan(edi.ConnectionMethodSFTP))
		f.ediService.EXPECT().
			GenerateDocument(mock.Anything, mock.Anything).
			Return(nil, errors.New("control number store down")).
			Once()
		f.invoiceRepo.EXPECT().UpdateEDISendStatus(mock.Anything, mock.Anything).Return(nil).Once()

		_, err := f.activities.SendInvoiceEDIActivity(t.Context(), f.payload(false))

		require.Error(t, err)
		var appErr *temporal.ApplicationError
		assert.False(t, errors.As(err, &appErr), "a plain error keeps the retry policy in charge")
	})
}

func TestSendInvoiceEDIActivityGeneratesForExternalAndSendsForInternalPartners(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     edi.ConnectionMethod
		wantStatus invoice.EDISendStatus
		wantSentAt bool
	}{
		{
			"external partner waits for delivery",
			edi.ConnectionMethodAS2,
			invoice.EDISendStatusGenerated,
			false,
		},
		{
			"internal partner is delivered by generation",
			edi.ConnectionMethodInternal,
			invoice.EDISendStatusSent,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newEDIActivityFixture(t)
			plan := readyPlan(tt.method)
			f.expectPlan(plan)
			messageID := pulid.MustNew("emsg_")
			f.ediService.EXPECT().
				GenerateDocument(mock.Anything, mock.MatchedBy(func(req *services.GenerateEDIDocumentRequest) bool {
					return req.InvoiceID == f.inv.ID &&
						req.EDIPartnerID == plan.PartnerID &&
						req.PartnerDocumentProfileID == plan.DocumentProfileID &&
						req.TransactionSet == edi.TransactionSet210 &&
						req.Direction == edi.DocumentDirectionOutbound &&
						req.GeneratedByID == f.tenantInfo.UserID
				})).
				Return(&edi.EDIMessage{ID: messageID}, nil).
				Once()
			f.invoiceRepo.EXPECT().
				UpdateEDISendStatus(mock.Anything, mock.MatchedBy(func(req repositories.UpdateInvoiceEDISendStatusRequest) bool {
					return req.InvoiceID == f.inv.ID &&
						req.MessageID == messageID &&
						req.Status == tt.wantStatus &&
						(req.SentAt != nil) == tt.wantSentAt
				})).
				Return(nil).
				Once()

			result, err := f.activities.SendInvoiceEDIActivity(t.Context(), f.payload(true))

			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, messageID, result.MessageID)
		})
	}
}
