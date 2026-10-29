package invoicesendservice

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicedocument"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceediservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/emailportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeInvoiceDocumentService struct {
	currentDocument *document.Document
}

func (f *fakeInvoiceDocumentService) Get(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	if f.currentDocument != nil {
		return f.currentDocument, nil
	}
	return &document.Document{ID: req.ID, LineageID: req.ID}, nil
}

func (f *fakeInvoiceDocumentService) GetDownloadContent(
	context.Context,
	documentports.GetDocumentByIDRequest,
) (*documentports.DocumentContent, error) {
	return nil, nil
}

func TestInvoiceDeliveryTemplates(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{
		Number:                 "INV-1001",
		InvoiceDate:            1_700_000_000,
		DueDate:                new(int64(1_700_086_400)),
		PaymentTerm:            invoice.PaymentTermNet30,
		CurrencyCode:           "USD",
		TotalAmount:            decimal.NewFromInt(1250),
		BillToName:             "Acme Logistics",
		BillToCode:             "ACME",
		ShipmentProNumber:      "PRO123",
		ShipmentBOL:            "BOL123",
		RemittanceInstructions: "ACH preferred",
	}
	context := invoiceTemplateContext(entity, &invoicedocument.DeliveryProfile{
		Organization: &tenant.Organization{Name: "Trenova Freight"},
		Shipment: &shipment.Shipment{
			ProNumber: "PRO123",
			BOL:       "BOL123",
		},
	})

	rendered := renderInvoiceTemplate(
		"Invoice #{number} for {customer} from {company}; {{invoice.number}} {{customer.name}} {{missing.value}}",
		context,
	)

	require.Equal(
		t,
		"Invoice #INV-1001 for Acme Logistics from Trenova Freight; INV-1001 Acme Logistics {{missing.value}}",
		rendered.Value,
	)
	require.Equal(t, []string{"missing.value"}, rendered.Unknown)
}

func TestResolveSubjectRendersDraftSnapshotBeforeCustomerTemplate(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{
		Number:               "INV-1001",
		BillToName:           "Acme Logistics",
		EmailSubjectSnapshot: "Draft invoice #{number} for {customer}",
	}
	profile := &customer.CustomerEmailProfile{
		Subject: "Customer {{invoice.number}}",
	}

	result, found := resolveSubject(entity, profile, invoiceTemplateContext(entity, nil))

	require.True(t, found)
	require.Equal(t, "Draft invoice #INV-1001 for Acme Logistics", result.Value)
	require.Empty(t, result.Unknown)
}

// With no draft and no customer profile text, the free-text tiers report nothing
// and the caller renders the organization's template. That is where the hardcoded
// "Invoice N" used to answer, and answering here again would mean an organization
// could never change the subject of the invoice emails it sends.
func TestResolveSubjectAndBodyDeferToTheTemplateWhenThereIsNoDraft(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{Number: "INV-1001", BillToName: "Acme Logistics"}
	context := invoiceTemplateContext(entity, nil)

	subject, subjectFound := resolveSubject(entity, nil, context)
	require.False(t, subjectFound)
	require.Empty(t, subject.Value)

	body, bodyFound := resolveBody(entity, nil, context)
	require.False(t, bodyFound)
	require.Empty(t, body.Value)

	// An email profile with only recipients configured is not text: it must not
	// pre-empt the template either.
	empty := &customer.CustomerEmailProfile{ToRecipients: "ap@example.com"}
	_, subjectFound = resolveSubject(entity, empty, context)
	require.False(t, subjectFound)
	_, bodyFound = resolveBody(entity, empty, context)
	require.False(t, bodyFound)
}

// The customer's own wording still beats the organization's template, which is
// the precedence customers already rely on.
func TestResolveSubjectPrefersTheCustomerProfileOverTheTemplate(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{Number: "INV-1001", BillToName: "Acme Logistics"}
	profile := &customer.CustomerEmailProfile{Subject: "Customer {{invoice.number}}"}

	result, found := resolveSubject(entity, profile, invoiceTemplateContext(entity, nil))

	require.True(t, found)
	require.Equal(t, "Customer INV-1001", result.Value)
}

func TestResolveBodyRendersDraftSnapshot(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{
		Number:            "INV-1001",
		BillToName:        "Acme Logistics",
		EmailBodySnapshot: "Please review {number}, {customer}.",
	}

	result, found := resolveBody(entity, nil, invoiceTemplateContext(entity, nil))

	require.True(t, found)
	require.Equal(t, "Please review INV-1001, Acme Logistics.", result.Value)
	require.Empty(t, result.Unknown)
}

// A template render must not become the invoice's stored draft. The snapshot
// columns are the first tier resolveSubject reads, so freezing a render there
// would mean a later edit to the template never reached a re-sent invoice — and
// the frozen copy would come back through the ad-hoc engine, losing the layout.
func TestSendSnapshotLeavesTheDeliveryOverridesAlone(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{
		Number:          "INV-1001",
		SendStatus:      invoice.SendStatusFailed,
		LastSendError:   "Resend refused to send",
		EmailToSnapshot: []string{"override@example.com"},
	}
	applySendSnapshot(entity, &invoiceports.InvoiceSendPlan{
		Subject:    "Invoice INV-1001",
		Body:       "Invoice INV-1001 is attached.",
		Recipients: invoiceports.InvoiceSendRecipients{To: []string{"ap@example.com"}},
		Warnings:   []string{"a warning"},
	})

	require.Equal(t, invoice.SendStatusSending, entity.SendStatus)
	require.Empty(t, entity.LastSendError)
	require.Equal(t, "a warning", entity.LastSendWarning)
	require.Empty(t, entity.EmailSubjectSnapshot)
	require.Empty(t, entity.EmailBodySnapshot)
	require.Equal(t, []string{"override@example.com"}, entity.EmailToSnapshot)
}

func TestInvoicePDFAttachmentNameSanitizesAndForcesPDF(t *testing.T) {
	t.Parallel()

	entity := &invoice.Invoice{Number: "INV/1001"}

	name := invoicePDFAttachmentName("../Invoice: {{bad}}.xlsx", entity)

	require.Equal(t, "Invoice- {{bad}}.pdf", name)
}

func TestPlanSendRendersTemplatesAndOrganizationAlias(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	invoiceID := pulid.MustNew("inv_")
	customerID := pulid.MustNew("cus_")
	documentID := pulid.MustNew("doc_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
	entity := &invoice.Invoice{
		ID:             invoiceID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CustomerID:     customerID,
		Number:         "INV-1001",
		Status:         invoice.StatusPosted,
		CurrencyCode:   "USD",
		BillToName:     "Snapshot Customer",
		PDFDocumentID:  documentID,
		PDFDocument: &document.Document{
			ID:           documentID,
			OriginalName: "invoice-INV-1001.pdf",
			FileType:     "application/pdf",
			FileSize:     512,
		},
	}

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{
			ID:         invoiceID,
			TenantInfo: tenantInfo,
		}).
		Return(entity, nil).
		Once()
	repo.EXPECT().
		ListAttachments(mock.Anything, invoiceports.ListInvoiceEmailAttemptsRequest{
			InvoiceID:  invoiceID,
			TenantInfo: tenantInfo,
		}).
		Return([]*invoice.Attachment{}, nil).
		Once()

	customerRepo := customerportsmock.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByID(mock.Anything, customerports.GetCustomerByIDRequest{
			ID:         customerID,
			TenantInfo: tenantInfo,
			CustomerFilterOptions: customerports.CustomerFilterOptions{
				IncludeEmailProfile:   true,
				IncludeBillingProfile: true,
			},
		}).
		Return(&customer.Customer{
			ID:   customerID,
			Name: "Acme Logistics",
			Code: "ACME",
			EmailProfile: &customer.CustomerEmailProfile{
				Subject:        "Invoice #{number} for {customer} from {company}",
				Comment:        "Invoice {{invoice.number}} for {{customer.name}} from {{organization.name}}",
				AttachmentName: "Invoice-{number}-{customer}-{company}.pdf",
				ToRecipients:   "ap@example.com",
			},
		}, nil).
		Once()

	organizationRepo := tenantportsmock.NewMockOrganizationRepository(t)
	organizationRepo.EXPECT().
		GetByID(mock.Anything, tenantports.GetOrganizationByIDRequest{TenantInfo: tenantInfo}).
		Return(&tenant.Organization{Name: "Trenova Freight"}, nil).
		Once()

	emailRepo := emailportsmock.NewMockEmailRepository(t)
	emailRepo.EXPECT().
		GetAssignedProfile(mock.Anything, tenantInfo, email.PurposeBilling).
		Return(&email.Profile{SenderEmail: "billing@example.com"}, nil).
		Once()

	billingRepo := tenantportsmock.NewMockBillingControlRepository(t)
	svc := &Service{
		l:                zap.NewNop(),
		repo:             repo,
		customerRepo:     customerRepo,
		organizationRepo: organizationRepo,
		emailRepo:        emailRepo,
		billingRepo:      billingRepo,
		edi:              invoiceediservice.New(invoiceediservice.Params{Logger: zap.NewNop()}),
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{Logger: zap.NewNop()},
		),
	}

	plan, err := svc.PlanSend(t.Context(), &invoiceports.InvoiceSendPlanRequest{
		InvoiceID:  invoiceID,
		TenantInfo: tenantInfo,
	})

	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Empty(t, plan.Errors)
	require.Empty(t, plan.Warnings)
	require.Len(t, plan.Parts, 1)
	require.Len(t, plan.Parts[0].Attachments, 1)
	assert.Equal(t, "Invoice #INV-1001 for Acme Logistics from Trenova Freight", plan.Subject)
	assert.Equal(t, "Invoice INV-1001 for Acme Logistics from Trenova Freight", plan.Body)
	assert.Equal(
		t,
		"Invoice-INV-1001-Acme Logistics-Trenova Freight.pdf",
		plan.Parts[0].Attachments[0].FileName,
	)
}

func TestAutoSendInvoiceAfterPDFGenerationSkipsDisabledAndAlreadySending(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		enabled  bool
		status   invoice.SendStatus
		customer *customer.Customer
	}{
		{
			name:    "disabled billing automation",
			enabled: false,
			status:  invoice.SendStatusNotSent,
		},
		{
			name:    "sending invoice",
			enabled: true,
			status:  invoice.SendStatusSending,
		},
		{
			name:    "sent invoice",
			enabled: true,
			status:  invoice.SendStatusSent,
		},
		{
			name:    "partially sent invoice",
			enabled: true,
			status:  invoice.SendStatusPartiallySent,
		},
		{
			name:    "missing billing profile",
			enabled: false,
			status:  invoice.SendStatusNotSent,
			customer: &customer.Customer{
				ID: pulid.MustNew("cus_"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orgID := pulid.MustNew("org_")
			buID := pulid.MustNew("bu_")
			userID := pulid.MustNew("usr_")
			invoiceID := pulid.MustNew("inv_")
			customerID := pulid.MustNew("cus_")
			tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
			entity := &invoice.Invoice{
				ID:             invoiceID,
				OrganizationID: orgID,
				BusinessUnitID: buID,
				CustomerID:     customerID,
				SendStatus:     tc.status,
			}

			repo := invoiceportsmock.NewMockInvoiceRepository(t)
			repo.EXPECT().
				GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{
					ID:         invoiceID,
					TenantInfo: tenantInfo,
				}).
				Return(entity, nil).
				Once()

			cus := tc.customer
			if cus == nil {
				cus = &customer.Customer{
					ID: customerID,
					BillingProfile: &customer.CustomerBillingProfile{
						AutoSendInvoiceOnGeneration: tc.enabled,
					},
				}
			} else {
				cus.ID = customerID
			}

			customerRepo := customerportsmock.NewMockCustomerRepository(t)
			customerRepo.EXPECT().
				GetByID(mock.Anything, customerports.GetCustomerByIDRequest{
					ID:         customerID,
					TenantInfo: tenantInfo,
					CustomerFilterOptions: customerports.CustomerFilterOptions{
						IncludeBillingProfile: true,
					},
				}).
				Return(cus, nil).
				Once()

			svc := &Service{
				l:            zap.NewNop(),
				repo:         repo,
				customerRepo: customerRepo,
				rec: invoiceaction.NewRecorder(
					invoiceaction.RecorderParams{Logger: zap.NewNop()},
				),
			}

			result, err := svc.AutoSendInvoiceAfterPDFGeneration(
				t.Context(),
				&invoiceports.AutoSendInvoiceAfterPDFGenerationRequest{
					InvoiceID:  invoiceID,
					TenantInfo: tenantInfo,
					BaseURL:    "https://billing.example.test",
				},
				testutil.NewSessionActor(userID, orgID, buID),
			)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Same(t, entity, result.Invoice)
			assert.Nil(t, result.Plan)
			assert.Empty(t, result.Attempts)
		})
	}
}

func TestAutoSendInvoiceAfterPDFGenerationRecordsSendFailure(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	invoiceID := pulid.MustNew("inv_")
	customerID := pulid.MustNew("cus_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}
	entity := &invoice.Invoice{
		ID:             invoiceID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CustomerID:     customerID,
		Status:         invoice.StatusPosted,
		SendStatus:     invoice.SendStatusNotSent,
	}

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{
			ID:         invoiceID,
			TenantInfo: tenantInfo,
		}).
		Return(entity, nil).
		Twice()
	repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(updated *invoice.Invoice) bool {
			return updated.ID == invoiceID &&
				updated.SendStatus == invoice.SendStatusFailed &&
				strings.Contains(
					updated.LastSendError,
					"Invoice email delivery is not configured",
				) &&
				updated.SentByID == userID
		})).
		Return(entity, nil).
		Once()

	customerRepo := customerportsmock.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByID(mock.Anything, customerports.GetCustomerByIDRequest{
			ID:         customerID,
			TenantInfo: tenantInfo,
			CustomerFilterOptions: customerports.CustomerFilterOptions{
				IncludeBillingProfile: true,
			},
		}).
		Return(&customer.Customer{
			ID: customerID,
			BillingProfile: &customer.CustomerBillingProfile{
				AutoSendInvoiceOnGeneration: true,
				EmailInvoiceEnabled:         true,
			},
		}, nil).
		Once()

	svc := &Service{
		l:            zap.NewNop(),
		repo:         repo,
		customerRepo: customerRepo,
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{
				Logger:       zap.NewNop(),
				AuditService: &mocks.NoopAuditService{},
			},
		),
	}

	result, err := svc.AutoSendInvoiceAfterPDFGeneration(
		t.Context(),
		&invoiceports.AutoSendInvoiceAfterPDFGenerationRequest{
			InvoiceID:  invoiceID,
			TenantInfo: tenantInfo,
			BaseURL:    "https://billing.example.test",
		},
		testutil.NewSessionActor(userID, orgID, buID),
	)

	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "Invoice email delivery is not configured")
}

func TestAppendShipmentDetailIncludesRouteAndCharges(t *testing.T) {
	t.Parallel()

	amount := decimal.NewFromInt(100)
	entity := &invoice.Invoice{
		ShipmentProNumber: "PRO123",
		ShipmentBOL:       "BOL123",
		CurrencyCode:      "USD",
		ServiceDate:       new(int64(1_700_000_000)),
		Lines: []*invoice.InvoiceLine{
			{
				Description: "Freight",
				Amount:      amount,
			},
		},
	}
	shp := &shipment.Shipment{
		ActualShipDate:     new(int64(1_699_900_000)),
		ActualDeliveryDate: new(int64(1_700_100_000)),
		Pieces:             new(int64(12)),
		Weight:             new(int64(24000)),
		Moves: []*shipment.ShipmentMove{
			{
				Stops: []*shipment.Stop{
					{
						Type:     shipmentvocab.StopTypePickup,
						Sequence: 1,
						Location: &location.Location{
							Name:       "Origin DC",
							City:       "Nashville",
							PostalCode: "37201",
						},
					},
					{
						Type:     shipmentvocab.StopTypeDelivery,
						Sequence: 2,
						Location: &location.Location{
							Name:       "Destination DC",
							City:       "Atlanta",
							PostalCode: "30301",
						},
					},
				},
			},
		},
	}

	body := appendShipmentDetail("Please pay.", entity, shp)

	require.Contains(t, body, "Shipment Detail")
	require.Contains(t, body, "Route: Origin DC Nashville 37201 -> Destination DC Atlanta 30301")
	require.Contains(t, body, "Pieces: 12")
	require.Contains(t, body, "- Freight: USD 100.00")
}

func TestRefuseSendInFlight(t *testing.T) {
	t.Parallel()

	now := int64(1_700_000_000)
	sending := &invoice.Invoice{
		Number:     "INV-1",
		SendStatus: invoice.SendStatusSending,
		UpdatedAt:  now - 60,
	}
	require.Error(t, refuseSendInFlight(sending, now))

	stale := *sending
	stale.UpdatedAt = now - int64(sendInFlightTimeout/time.Second)
	require.NoError(t, refuseSendInFlight(&stale, now))

	failed := *sending
	failed.SendStatus = invoice.SendStatusFailed
	require.NoError(t, refuseSendInFlight(&failed, now))
}

func TestSendEligibilityErrors(t *testing.T) {
	t.Parallel()

	posted := &invoice.Invoice{Status: invoice.StatusPosted}
	draft := &invoice.Invoice{Status: invoice.StatusDraft}
	emailOn := &customer.Customer{
		BillingProfile: &customer.CustomerBillingProfile{EmailInvoiceEnabled: true},
	}
	emailOff := &customer.Customer{BillingProfile: &customer.CustomerBillingProfile{}}

	require.Empty(t, sendEligibilityErrors(posted, emailOn))
	require.Empty(t, sendEligibilityErrors(posted, nil))
	require.Len(t, sendEligibilityErrors(draft, emailOn), 1)
	require.Contains(t, sendEligibilityErrors(draft, emailOn)[0], "Post the invoice")
	require.Len(t, sendEligibilityErrors(posted, emailOff), 1)
	require.Len(t, sendEligibilityErrors(draft, emailOff), 2)
}

func TestBillingSenderEmailIsTheBillingProfileSender(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		"mailbox@trenova.app",
		billingSenderEmail(&email.Profile{SenderEmail: " mailbox@trenova.app "}),
	)
	require.Empty(t, billingSenderEmail(nil))
}
