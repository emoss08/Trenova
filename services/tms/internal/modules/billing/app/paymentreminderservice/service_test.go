package paymentreminderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/domain/paymentreminder"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/paymentreminderports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/invoicetemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
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
	"go.uber.org/zap"
)

const day = int64(86_400)

var now = 20_000 * day

type passthroughDB struct{}

func (passthroughDB) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	return fn(ctx)
}

type fakeRepo struct {
	overdue  []*paymentreminderports.OverdueInvoice
	inserted []*paymentreminder.Reminder
}

func (f *fakeRepo) ListOverdueInvoices(
	_ context.Context,
	req paymentreminderports.ListOverdueInvoicesRequest,
) ([]*paymentreminderports.OverdueInvoice, error) {
	if len(req.CustomerIDs) == 0 {
		return f.overdue, nil
	}
	out := make([]*paymentreminderports.OverdueInvoice, 0)
	for _, inv := range f.overdue {
		if inv.CustomerID == req.CustomerIDs[0] {
			out = append(out, inv)
		}
	}
	return out, nil
}

func (f *fakeRepo) InsertReminders(_ context.Context, reminders []*paymentreminder.Reminder) error {
	f.inserted = append(f.inserted, reminders...)
	return nil
}

type fakeCollections struct {
	collectionports.Repository
	awaiting   map[pulid.ID]bool
	activities []*collection.Activity
}

func (f *fakeCollections) ListAwaitingPromise(
	context.Context,
	collectionports.AwaitingPromiseRequest,
) (map[pulid.ID]bool, error) {
	return f.awaiting, nil
}

func (f *fakeCollections) InsertActivity(_ context.Context, activity *collection.Activity) error {
	f.activities = append(f.activities, activity)
	return nil
}

type fakeEmail struct {
	sent []*emailports.SendEmailRequest
}

func (f *fakeEmail) Send(
	_ context.Context,
	req *emailports.SendEmailRequest,
) (*email.Message, error) {
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
	collections *fakeCollections
	email       *fakeEmail
	contexts    []*invoicetemplate.PaymentReminderEmailContext
	control     *billingcontrol.BillingControl
	customers   map[pulid.ID]*customer.Customer
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		repo:        &fakeRepo{},
		collections: &fakeCollections{awaiting: map[pulid.ID]bool{}},
		email:       &fakeEmail{},
		customers:   map[pulid.ID]*customer.Customer{},
		control: &billingcontrol.BillingControl{
			OrganizationID:          pulid.MustNew("org_"),
			BusinessUnitID:          pulid.MustNew("bu_"),
			PaymentRemindersEnabled: true,
			PaymentReminderDays:     []int32{1, 15, 30},
			DefaultInvoiceTerms:     "Remit by ACH to 111000614",
		},
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
		}).
		Maybe()

	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tenant.Organization{Name: "Trenova Logistics", Timezone: "UTC"}, nil).Maybe()
	users := tenantportsmock.NewMockUserRepository(t)
	users.EXPECT().GetSystemUser(mock.Anything, mock.Anything).
		Return(&tenant.User{ID: pulid.MustNew("usr_")}, nil).Maybe()
	controls := tenantportsmock.NewMockBillingControlRepository(t)
	controls.EXPECT().GetByOrgID(mock.Anything, mock.Anything).Return(f.control, nil).Maybe()

	templates := documenttemplateportsmock.NewMockDocumentTemplateResolver(t)
	templates.EXPECT().RenderMessage(mock.Anything, mock.Anything).RunAndReturn(
		func(
			_ context.Context,
			req *documenttemplateports.RenderMessageRequest,
		) (*documenttemplateports.RenderedMessage, error) {
			data := req.Data.(*invoicetemplate.PaymentReminderEmailContext)
			f.contexts = append(f.contexts, data)
			return &documenttemplateports.RenderedMessage{
				Subject: "Reminder",
				HTML:    "<p/>",
				Text:    "text",
			}, nil
		}).Maybe()

	f.svc = &Service{
		db:              passthroughDB{},
		repo:            f.repo,
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

func (f *fixture) customer(to string) *customer.Customer {
	cus := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		Name:           "Halstead Grocery",
		BillingProfile: &customer.CustomerBillingProfile{SendPaymentReminders: true},
		EmailProfile:   &customer.CustomerEmailProfile{ToRecipients: to},
	}
	f.customers[cus.ID] = cus
	return cus
}

func overdue(
	cus *customer.Customer,
	number string,
	daysLate int64,
	openMinor int64,
	lastStep int,
) *paymentreminderports.OverdueInvoice {
	return &paymentreminderports.OverdueInvoice{
		InvoiceID:     pulid.MustNew("inv_"),
		InvoiceNumber: number,
		CustomerID:    cus.ID,
		CurrencyCode:  "USD",
		InvoiceDate:   now - (daysLate+30)*day,
		DueDate:       now - daysLate*day,
		OpenMinor:     openMinor,
		LastStep:      lastStep,
	}
}

func TestSendDue_SendsOneReminderListingEveryOverdueInvoice(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	cus := f.customer("ap@halstead.test, controller@halstead.test")
	newlyLate := overdue(cus, "INV-2", 16, 50_000, 1)
	alreadyReminded := overdue(cus, "INV-3", 3, 25_000, 1)
	f.repo.overdue = []*paymentreminderports.OverdueInvoice{alreadyReminded, newlyLate}

	result, err := f.svc.SendDue(t.Context(), f.control, now)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Sent)
	require.Len(t, f.email.sent, 1)
	sent := f.email.sent[0]
	assert.Equal(t, email.PurposeBilling, sent.Purpose)
	assert.Equal(t, []string{"ap@halstead.test", "controller@halstead.test"}, sent.To)

	require.Len(t, f.contexts, 1)
	ctx := f.contexts[0]
	assert.Equal(t, 2, ctx.ReminderNumber, "the furthest step reached")
	assert.False(t, ctx.IsFinalReminder)
	assert.Equal(t, "750.00 USD", ctx.TotalOverdue)
	require.Len(t, ctx.Invoices, 2)
	assert.Equal(t, "INV-2", ctx.Invoices[0].InvoiceNumber, "oldest first")
	assert.Equal(t, 16, ctx.Invoices[0].DaysPastDue)
	assert.Equal(t, "Remit by ACH to 111000614", ctx.RemittanceInstructions)

	require.Len(t, f.repo.inserted, 1, "only the invoice that reached a new step is recorded")
	assert.Equal(t, newlyLate.InvoiceID, f.repo.inserted[0].InvoiceID)
	assert.Equal(t, int16(2), f.repo.inserted[0].Step)
	assert.Equal(t, "ap@halstead.test, controller@halstead.test", f.repo.inserted[0].SentTo)

	require.Len(t, f.collections.activities, 1)
	assert.Equal(t, collection.ActivityEmail, f.collections.activities[0].Kind)
	assert.Equal(
		t,
		"Payment reminder 2 sent to ap@halstead.test, controller@halstead.test for INV-2, INV-3 (750.00 USD past due)",
		f.collections.activities[0].Summary,
	)
}

func TestSendDue_SendsNothingWhenNoInvoiceReachedANewStep(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	cus := f.customer("ap@halstead.test")
	f.repo.overdue = []*paymentreminderports.OverdueInvoice{overdue(cus, "INV-1", 10, 1_000, 1)}

	result, err := f.svc.SendDue(t.Context(), f.control, now)
	require.NoError(t, err)

	assert.Zero(t, result.Sent)
	assert.Empty(t, f.email.sent)
}

func TestSendDue_SkipsDisputedInvoicesOptedOutCustomersAndOpenPromises(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	disputed := f.customer("a@example.test")
	inv := overdue(disputed, "INV-D", 5, 1_000, 0)
	inv.Disputed = true
	optedOut := f.customer("b@example.test")
	optedOut.BillingProfile.SendPaymentReminders = false
	promised := f.customer("c@example.test")
	f.collections.awaiting[promised.ID] = true
	f.repo.overdue = []*paymentreminderports.OverdueInvoice{
		inv,
		overdue(optedOut, "INV-O", 5, 1_000, 0),
		overdue(promised, "INV-P", 5, 1_000, 0),
	}

	result, err := f.svc.SendDue(t.Context(), f.control, now)
	require.NoError(t, err)

	assert.Empty(t, f.email.sent)
	assert.Equal(t, 1, result.WaitedOnPromise)
}

func TestSendDue_ReportsCustomersWithNobodyToEmail(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	cus := f.customer("")
	f.repo.overdue = []*paymentreminderports.OverdueInvoice{overdue(cus, "INV-1", 2, 1_000, 0)}

	result, err := f.svc.SendDue(t.Context(), f.control, now)
	require.NoError(t, err)

	assert.Equal(t, []string{"Halstead Grocery"}, result.NoRecipients)
	assert.Empty(t, f.email.sent)
	assert.Empty(t, f.repo.inserted, "nothing is recorded for a reminder that was not sent")
}

func TestSendDue_MarksTheLastStepAsTheFinalReminder(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	cus := f.customer("ap@halstead.test")
	f.repo.overdue = []*paymentreminderports.OverdueInvoice{overdue(cus, "INV-1", 45, 1_000, 2)}

	_, err := f.svc.SendDue(t.Context(), f.control, now)
	require.NoError(t, err)

	require.Len(t, f.contexts, 1)
	assert.True(t, f.contexts[0].IsFinalReminder)
	assert.Equal(t, 3, f.contexts[0].ReminderNumber)
}

func TestSendDue_SendsTheSameReminderUnderTheSameKey(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	cus := f.customer("ap@halstead.test")
	f.repo.overdue = []*paymentreminderports.OverdueInvoice{overdue(cus, "INV-1", 2, 1_000, 0)}

	_, err := f.svc.SendDue(t.Context(), f.control, now)
	require.NoError(t, err)
	_, err = f.svc.SendDue(t.Context(), f.control, now)
	require.NoError(t, err)

	require.Len(t, f.email.sent, 2)
	assert.Equal(t, f.email.sent[0].IdempotencyKey, f.email.sent[1].IdempotencyKey,
		"a retried run whose records were lost replays the queued email")
}

func TestSendNow_SendsEverythingOverdueWithTheNote(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	cus := f.customer("ap@halstead.test")
	f.repo.overdue = []*paymentreminderports.OverdueInvoice{
		overdue(cus, "INV-1", 2, 1_000, 3),
		overdue(cus, "INV-2", 20, 4_000, 3),
	}
	sender := pulid.MustNew("usr_")

	results, err := f.svc.SendNow(t.Context(), &paymentreminderports.SendNowRequest{
		TenantInfo: tenantOf(f.control),
		CustomerID: cus.ID,
		Note:       "  Following up on our call this morning. ",
	}, &permissionports.RequestActor{UserID: sender})
	require.NoError(t, err)

	require.Len(t, results, 1)
	assert.Equal(t, []string{"ap@halstead.test"}, results[0].SentTo)
	assert.Equal(t, []string{"INV-2", "INV-1"}, results[0].InvoiceNumbers)
	assert.Equal(t, "50.00 USD", results[0].TotalOverdue)
	assert.Equal(t, "Following up on our call this morning.", f.contexts[0].Note)
	assert.Zero(t, f.contexts[0].ReminderNumber)
	require.Len(t, f.repo.inserted, 2)
	for _, row := range f.repo.inserted {
		assert.Equal(t, int16(paymentreminder.ManualStep), row.Step)
		assert.True(t, row.Manual)
		assert.Equal(t, sender, row.SentByID)
	}
	assert.Equal(t, sender, f.collections.activities[0].CreatedByID)
}

func TestSendNow_RefusesAnOptedOutCustomerAndOneWithNothingOverdue(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	optedOut := f.customer("ap@halstead.test")
	optedOut.BillingProfile.SendPaymentReminders = false
	current := f.customer("ap@peak.test")

	_, err := f.svc.SendNow(t.Context(), &paymentreminderports.SendNowRequest{
		TenantInfo: tenantOf(f.control), CustomerID: optedOut.ID,
	}, nil)
	assert.True(t, errortypes.IsBusinessError(err))

	_, err = f.svc.SendNow(t.Context(), &paymentreminderports.SendNowRequest{
		TenantInfo: tenantOf(f.control), CustomerID: current.ID,
	}, nil)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, f.email.sent)
}

func tenantOf(control *billingcontrol.BillingControl) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: control.OrganizationID, BuID: control.BusinessUnitID}
}
