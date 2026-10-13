package customerstatementservice

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerstatement"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerstatementports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documenttemplateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

var pdfBytes = []byte("%PDF-1.7 statement")

type passthroughDB struct{}

func (passthroughDB) DB() *bun.DB                          { return nil }
func (passthroughDB) DBForContext(context.Context) bun.IDB { return nil }
func (passthroughDB) HealthCheck(context.Context) error    { return nil }
func (passthroughDB) IsHealthy(context.Context) bool       { return true }
func (passthroughDB) Close() error                         { return nil }

func (passthroughDB) WithTx(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

type fakeRepo struct {
	candidates []pulid.ID
	sent       map[pulid.ID]bool
	inserted   []*customerstatement.Statement
	periods    []customerstatement.Period
}

func (f *fakeRepo) ListStatementCustomers(
	_ context.Context,
	req customerstatementports.ListStatementCustomersRequest,
) ([]pulid.ID, error) {
	f.periods = append(f.periods, req.Period)
	return f.candidates, nil
}

func (f *fakeRepo) ListMonthlySent(
	context.Context,
	customerstatementports.ListMonthlySentRequest,
) (map[pulid.ID]bool, error) {
	return f.sent, nil
}

func (f *fakeRepo) Insert(_ context.Context, statement *customerstatement.Statement) (bool, error) {
	statement.ID = pulid.MustNew("cst_")
	f.inserted = append(f.inserted, statement)
	return true, nil
}

func (f *fakeRepo) ListSent(
	context.Context,
	customerstatementports.ListSentRequest,
) ([]*customerstatementports.SentStatement, error) {
	return nil, nil
}

type fakeStatements struct {
	statement *customerledgerports.ARCustomerStatement
	requested []customerstatement.Period
}

func (f *fakeStatements) GetCustomerStatement(
	_ context.Context,
	_ pagination.TenantInfo,
	customerID pulid.ID,
	startDate, asOfDate int64,
) (*customerledgerports.ARCustomerStatement, error) {
	f.requested = append(f.requested, customerstatement.Period{Start: startDate, End: asOfDate})
	out := *f.statement
	out.CustomerID = customerID
	out.StartDate = startDate
	out.StatementDate = asOfDate
	return &out, nil
}

type fakeCollections struct {
	collectionports.Repository
	activities []*collection.Activity
}

func (f *fakeCollections) InsertActivity(_ context.Context, activity *collection.Activity) error {
	f.activities = append(f.activities, activity)
	return nil
}

type fakeEmail struct {
	sent []*emailports.SendEmailRequest
}

func (f *fakeEmail) Send(_ context.Context, req *emailports.SendEmailRequest) (*email.Message, error) {
	f.sent = append(f.sent, req)
	return &email.Message{}, nil
}

func (f *fakeEmail) SendPersisted(
	context.Context,
	*emailports.SendPersistedEmailRequest,
) (*email.Message, error) {
	return nil, nil
}

type fixture struct {
	svc         *Service
	repo        *fakeRepo
	statements  *fakeStatements
	collections *fakeCollections
	email       *fakeEmail
	pdfs        []*documenttemplate.CustomerStatementContext
	messages    []*documenttemplate.CustomerStatementEmailContext
	recorded    []*documenttemplateports.RecordGeneratedDocumentRequest
	control     *tenant.BillingControl
	customers   map[pulid.ID]*customer.Customer
	tenant      pagination.TenantInfo
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		repo:        &fakeRepo{sent: map[pulid.ID]bool{}},
		collections: &fakeCollections{},
		email:       &fakeEmail{},
		customers:   map[pulid.ID]*customer.Customer{},
		statements: &fakeStatements{statement: &customerledgerports.ARCustomerStatement{
			CustomerName:        "Halstead Grocery",
			OpeningBalanceMinor: 100_00,
			TotalChargesMinor:   500_00,
			TotalPaymentsMinor:  200_00,
			EndingBalanceMinor:  400_00,
			Aging: customerledgerports.ARAgingBucketTotals{
				CurrentMinor:    150_00,
				Days1To30Minor:  250_00,
				TotalOpenMinor:  400_00,
				Days31To60Minor: 0,
			},
			Transactions: []*customerledgerports.ARStatementTransaction{
				{
					TransactionDate:     time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC).Unix(),
					DocumentNumber:      "INV-1",
					Description:         "Invoice",
					ChargeMinor:         500_00,
					RunningBalanceMinor: 600_00,
				},
				{
					TransactionDate:     time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC).Unix(),
					DocumentNumber:      "PMT-9",
					Description:         "Payment",
					PaymentMinor:        200_00,
					RunningBalanceMinor: 400_00,
				},
			},
			OpenItems: []*customerledgerports.AROpenItem{
				{
					InvoiceNumber:    "INV-0",
					CurrencyCode:     "USD",
					InvoiceDate:      time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC).Unix(),
					DueDate:          time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC).Unix(),
					TotalAmountMinor: 300_00,
					OpenAmountMinor:  250_00,
					DaysPastDue:      12,
					DisputeStatus:    "Disputed",
				},
				{
					InvoiceNumber:    "INV-1",
					CurrencyCode:     "USD",
					TotalAmountMinor: 500_00,
					OpenAmountMinor:  150_00,
					DaysPastDue:      -10,
				},
			},
		}},
		control: &tenant.BillingControl{
			OrganizationID:           pulid.MustNew("org_"),
			BusinessUnitID:           pulid.MustNew("bu_"),
			MonthlyStatementsEnabled: true,
			MonthlyStatementDay:      3,
			DefaultInvoiceTerms:      "Remit by ACH to 111000614",
			DefaultInvoiceFooter:     "Thank you for your business.",
		},
	}
	f.tenant = pagination.TenantInfo{
		OrgID:  f.control.OrganizationID,
		BuID:   f.control.BusinessUnitID,
		UserID: pulid.MustNew("usr_"),
	}

	customers := customerportsmock.NewMockCustomerRepository(t)
	customers.EXPECT().GetByIDs(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, req customerports.GetCustomersByIDsRequest) ([]*customer.Customer, error) {
			out := make([]*customer.Customer, 0, len(req.CustomerIDs))
			for _, id := range req.CustomerIDs {
				if cus, ok := f.customers[id]; ok {
					out = append(out, cus)
				}
			}
			return out, nil
		}).Maybe()

	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&tenant.Organization{
		Name:         "Trenova Logistics",
		Timezone:     "UTC",
		AddressLine1: "2200 Commerce St",
		City:         "Dallas",
		PostalCode:   "75201",
	}, nil).Maybe()
	users := tenantportsmock.NewMockUserRepository(t)
	users.EXPECT().GetSystemUser(mock.Anything, mock.Anything).
		Return(&tenant.User{ID: pulid.MustNew("usr_")}, nil).Maybe()
	controls := tenantportsmock.NewMockBillingControlRepository(t)
	controls.EXPECT().GetByOrgID(mock.Anything, mock.Anything).Return(f.control, nil).Maybe()

	templates := documenttemplateportsmock.NewMockDocumentTemplateResolver(t)
	templates.EXPECT().RenderDocument(mock.Anything, mock.Anything).RunAndReturn(
		func(
			_ context.Context,
			req *documenttemplateports.RenderDocumentRequest,
		) (*documenttemplateports.RenderedDocument, error) {
			f.pdfs = append(f.pdfs, req.Data.(*documenttemplate.CustomerStatementContext))
			return &documenttemplateports.RenderedDocument{PDF: pdfBytes}, nil
		}).Maybe()
	templates.EXPECT().RenderMessage(mock.Anything, mock.Anything).RunAndReturn(
		func(
			_ context.Context,
			req *documenttemplateports.RenderMessageRequest,
		) (*documenttemplateports.RenderedMessage, error) {
			f.messages = append(f.messages, req.Data.(*documenttemplate.CustomerStatementEmailContext))
			return &documenttemplateports.RenderedMessage{Subject: "Statement", HTML: "<p/>", Text: "text"}, nil
		}).Maybe()
	templates.EXPECT().RecordGeneratedDocument(mock.Anything, mock.Anything).RunAndReturn(
		func(
			_ context.Context,
			req *documenttemplateports.RecordGeneratedDocumentRequest,
		) (*documenttemplate.GeneratedDocument, error) {
			f.recorded = append(f.recorded, req)
			return &documenttemplate.GeneratedDocument{}, nil
		}).Maybe()

	f.svc = &Service{
		db:              passthroughDB{},
		repo:            f.repo,
		statements:      f.statements,
		customers:       customers,
		collections:     f.collections,
		organizations:   orgs,
		users:           users,
		billingControls: controls,
		emailService:    f.email,
		templates:       templates,
		l:               zap.NewNop(),
	}
	return f
}

func (f *fixture) customer(code, to string) *customer.Customer {
	cus := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		Code:           code,
		Name:           "Halstead Grocery",
		AddressLine1:   "410 Market Ave",
		City:           "Fort Worth",
		PostalCode:     "76102",
		BillingProfile: &customer.CustomerBillingProfile{SendStatements: true, BillingCurrency: "USD"},
		EmailProfile:   &customer.CustomerEmailProfile{ToRecipients: to},
	}
	f.customers[cus.ID] = cus
	return cus
}

func (f *fixture) sendRequest(cus *customer.Customer, start, end int64, note string) *customerstatementports.SendRequest {
	return &customerstatementports.SendRequest{
		StatementRequest: customerstatementports.StatementRequest{
			TenantInfo: f.tenant,
			CustomerID: cus.ID,
			StartDate:  start,
			AsOfDate:   end,
		},
		Note: note,
	}
}

func TestSend_EmailsThePDFAndRecordsIt(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	cus := f.customer("HALSTEAD", "ap@halstead.test, controller@halstead.test")
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC).Unix()
	end := time.Date(2026, 8, 31, 23, 59, 59, 0, time.UTC).Unix()
	actor := &permissionports.RequestActor{UserID: f.tenant.UserID}

	result, err := f.svc.Send(t.Context(), f.sendRequest(cus, start, end, "  As requested  "), actor)
	require.NoError(t, err)

	assert.Equal(t, []string{"ap@halstead.test", "controller@halstead.test"}, result.SentTo)
	assert.Equal(t, "Statement-HALSTEAD-2026-08-31.pdf", result.FileName)
	assert.Equal(t, []customerstatement.Period{{Start: start, End: end}}, f.statements.requested)

	require.Len(t, f.email.sent, 1)
	sent := f.email.sent[0]
	assert.Equal(t, email.PurposeBilling, sent.Purpose)
	require.Len(t, sent.Attachments, 1)
	assert.Equal(t, "Statement-HALSTEAD-2026-08-31.pdf", sent.Attachments[0].FileName)
	assert.Equal(t, "application/pdf", sent.Attachments[0].ContentType)
	assert.Equal(t, pdfBytes, sent.Attachments[0].Content)
	assert.True(t, strings.HasPrefix(sent.IdempotencyKey, "customer-statement-"))

	require.Len(t, f.pdfs, 1)
	pdf := f.pdfs[0]
	assert.Equal(t, "2026-08-31", pdf.StatementDate)
	assert.Equal(t, "2026-08-01", pdf.PeriodStart)
	assert.Equal(t, "As requested", pdf.Note)
	assert.Equal(t, []string{"410 Market Ave", "Fort Worth, 76102"}, pdf.Customer.Lines)
	assert.Equal(t,
		[]string{"2200 Commerce St", "Dallas, 75201", "Remit by ACH to 111000614"},
		pdf.RemitTo.Lines,
	)
	require.Len(t, pdf.Transactions, 2)
	assert.Equal(t, "Invoice", pdf.Transactions[0].Description)
	assert.Empty(t, pdf.Transactions[0].Payment, "a charge leaves the payment cell blank")
	assert.NotEmpty(t, pdf.Transactions[1].Payment)
	require.Len(t, pdf.OpenItems, 2)
	assert.True(t, pdf.OpenItems[0].Disputed)
	assert.Equal(t, 0, pdf.OpenItems[1].DaysPastDue, "an invoice not yet due is not late")
	assert.NotEmpty(t, pdf.PastDue)
	assert.Equal(t, "Thank you for your business.", pdf.InvoiceFooter)

	require.Len(t, f.messages, 1)
	assert.False(t, f.messages[0].IsMonthly)
	assert.Equal(t, 2, f.messages[0].OpenItemCount)

	require.Len(t, f.repo.inserted, 1)
	record := f.repo.inserted[0]
	assert.False(t, record.Monthly)
	assert.Equal(t, int64(400_00), record.EndingBalanceMinor)
	assert.Equal(t, int64(250_00), record.PastDueMinor)
	assert.Equal(t, f.tenant.UserID, record.SentByID)

	require.Len(t, f.collections.activities, 1)
	activity := f.collections.activities[0]
	assert.Equal(t, collection.ActivityEmail, activity.Kind)
	assert.Contains(t, activity.Summary, "Statement as of 2026-08-31 sent to ap@halstead.test")
	assert.Contains(t, activity.Summary, "\nAs requested")

	require.Len(t, f.recorded, 1)
	assert.Equal(t, documenttemplate.KindCustomerStatementPDF, f.recorded[0].Kind)
	assert.Equal(t, record.ID, f.recorded[0].ReferenceID)
}

func TestSend_RefusesACustomerWithNobodyToEmail(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	cus := f.customer("HALSTEAD", "")

	_, err := f.svc.Send(t.Context(), f.sendRequest(cus, 0, 0, ""), nil)

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Empty(t, f.email.sent)
}

func TestSend_ChecksTheDates(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	cus := f.customer("HALSTEAD", "ap@halstead.test")
	future := time.Now().Add(72 * time.Hour).Unix()

	_, err := f.svc.Send(t.Context(), f.sendRequest(cus, 0, future, ""), nil)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "asOfDate", validation.Field)

	_, err = f.svc.Send(t.Context(), f.sendRequest(cus, 2_000, 1_000, ""), nil)
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "startDate", validation.Field)

	_, err = f.svc.Send(t.Context(), f.sendRequest(cus, 0, 0, strings.Repeat("x", maxNoteLength+1)), nil)
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "note", validation.Field)

	assert.Empty(t, f.email.sent)
}

func TestSend_GoesToACustomerOffMonthlyStatements(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	cus := f.customer("HALSTEAD", "ap@halstead.test")
	cus.BillingProfile.SendStatements = false

	_, err := f.svc.Send(t.Context(), f.sendRequest(cus, 0, 0, ""), nil)

	require.NoError(t, err)
	assert.Len(t, f.email.sent, 1)
}

func TestRender_ReturnsThePDF(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	cus := f.customer("", "ap@halstead.test")
	end := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC).Unix()

	rendered, err := f.svc.Render(t.Context(), &customerstatementports.StatementRequest{
		TenantInfo: f.tenant,
		CustomerID: cus.ID,
		AsOfDate:   end,
	})
	require.NoError(t, err)

	assert.Equal(t, pdfBytes, rendered.PDF)
	assert.Equal(t, "Statement-Halstead Grocery-2026-09-30.pdf", rendered.FileName,
		"a customer with no code is named in full")
	assert.Empty(t, f.pdfs[0].PeriodStart, "a statement from the first transaction has no start")
	assert.Empty(t, f.email.sent)
	assert.Empty(t, f.repo.inserted)
}

func TestSendMonthly_WaitsForTheStatementDay(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.repo.candidates = []pulid.ID{f.customer("HALSTEAD", "ap@halstead.test").ID}
	now := time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC).Unix()

	result, err := f.svc.SendMonthly(t.Context(), f.control, now)

	require.NoError(t, err)
	assert.Zero(t, result.Sent)
	assert.Empty(t, f.repo.periods, "nothing is read before the day")
}

func TestSendMonthly_SendsLastMonthOnceToEachCustomer(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	due := f.customer("HALSTEAD", "ap@halstead.test")
	alreadySent := f.customer("ACME", "ap@acme.test")
	optedOut := f.customer("NOSTMT", "ap@nostmt.test")
	optedOut.BillingProfile.SendStatements = false
	unreachable := f.customer("QUIET", "")
	unreachable.Name = "Quiet Freight"
	f.repo.candidates = []pulid.ID{due.ID, alreadySent.ID, optedOut.ID, unreachable.ID}
	f.repo.sent[alreadySent.ID] = true
	now := time.Date(2026, 9, 5, 15, 0, 0, 0, time.UTC).Unix()

	result, err := f.svc.SendMonthly(t.Context(), f.control, now)
	require.NoError(t, err)

	wantPeriod := customerstatement.Period{
		Start: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC).Unix(),
		End:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC).Unix() - 1,
	}
	assert.Equal(t, []customerstatement.Period{wantPeriod}, f.repo.periods)
	assert.Equal(t, 1, result.Sent)
	assert.Equal(t, 1, result.AlreadySent)
	assert.Equal(t, 2, result.Customers, "the opted-out customer is not counted")
	assert.Equal(t, []string{"Quiet Freight"}, result.NoRecipients)

	require.Len(t, f.email.sent, 1)
	assert.Equal(t, []string{"ap@halstead.test"}, f.email.sent[0].To)
	assert.Equal(t,
		"customer-statement-monthly-"+due.ID.String()+"-1788220799",
		f.email.sent[0].IdempotencyKey,
		"a retried month replays the queued email",
	)
	require.Len(t, f.messages, 1)
	assert.True(t, f.messages[0].IsMonthly)

	require.Len(t, f.repo.inserted, 1)
	assert.True(t, f.repo.inserted[0].Monthly)
	assert.Equal(t, wantPeriod.End, f.repo.inserted[0].PeriodEnd)
	assert.True(t, f.repo.inserted[0].SentByID.IsNil())
	require.Len(t, f.collections.activities, 1)
	assert.True(t, strings.HasPrefix(f.collections.activities[0].Summary, "Monthly statement as of 2026-08-31"))
}

func TestSendMonthly_DoesNothingWhileOff(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.control.MonthlyStatementsEnabled = false

	result, err := f.svc.SendMonthly(t.Context(), f.control, time.Now().Unix())

	require.NoError(t, err)
	assert.Zero(t, result.Customers)
	assert.Empty(t, f.repo.periods)
}
