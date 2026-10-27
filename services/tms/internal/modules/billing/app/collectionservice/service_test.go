package collectionservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	day = int64(secondsPerDay)
	now = 1000 * day
)

type passthroughDB struct{}

func (passthroughDB) Transact(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context) error,
) error {
	return fn(ctx)
}

type fakeRepo struct {
	rows       []*collectionports.WorklistRow
	account    *collection.Account
	activities []*collection.Activity
	saved      []collection.Account
}

func (f *fakeRepo) GetAccountForUpdate(
	_ context.Context,
	req collectionports.GetAccountRequest,
) (*collection.Account, error) {
	if f.account == nil {
		return &collection.Account{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			CustomerID:     req.CustomerID,
		}, nil
	}
	copied := *f.account
	return &copied, nil
}

func (f *fakeRepo) SaveAccount(_ context.Context, account *collection.Account) (*collection.Account, error) {
	if account.ID.IsNil() {
		account.ID = pulid.MustNew("cacc_")
	}
	f.saved = append(f.saved, *account)
	f.account = account
	return account, nil
}

func (f *fakeRepo) InsertActivity(_ context.Context, activity *collection.Activity) error {
	activity.ID = pulid.MustNew("cact_")
	f.activities = append(f.activities, activity)
	return nil
}

func (f *fakeRepo) ListActivities(
	context.Context,
	collectionports.ListActivitiesRequest,
) ([]*collectionports.ActivityView, error) {
	views := make([]*collectionports.ActivityView, 0, len(f.activities))
	for i := len(f.activities) - 1; i >= 0; i-- {
		views = append(views, &collectionports.ActivityView{Activity: f.activities[i]})
	}
	return views, nil
}

func (f *fakeRepo) ListWorklist(
	context.Context,
	*collectionports.WorklistRequest,
) (*collectionports.WorklistPage, error) {
	return &collectionports.WorklistPage{Rows: f.rows, Total: len(f.rows)}, nil
}

func (f *fakeRepo) ListAwaitingPromise(
	context.Context,
	collectionports.AwaitingPromiseRequest,
) (map[pulid.ID]bool, error) {
	return map[pulid.ID]bool{}, nil
}

func (f *fakeRepo) GetWorklistRow(
	_ context.Context,
	req collectionports.GetAccountRequest,
	_ int64,
) (*collectionports.WorklistRow, error) {
	row := &collectionports.WorklistRow{
		CustomerID:       req.CustomerID,
		CustomerName:     "Acme Logistics",
		OverdueMinor:     250_000,
		OpenInvoiceCount: 3,
	}
	if f.account != nil {
		row.CollectorID = f.account.CollectorID
		row.NextFollowUpOn = f.account.NextFollowUpOn
	}
	return row, nil
}

type fakeMembers struct {
	members map[pulid.ID]string
}

func (f fakeMembers) GetTenantMember(
	_ context.Context,
	req tenantports.GetTenantMemberRequest,
) (*tenant.User, error) {
	name, ok := f.members[req.UserID]
	if !ok {
		return nil, errortypes.NewNotFoundError("User not found")
	}
	return &tenant.User{ID: req.UserID, Name: name}, nil
}

type fakeNotifier struct {
	sent []*notification.Notification
}

func (f *fakeNotifier) Create(_ context.Context, entity *notification.Notification) (*notification.Notification, error) {
	f.sent = append(f.sent, entity)
	return entity, nil
}

var (
	tenantInfo = pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	clerk      = pulid.MustNew("usr_")
)

func actor() *permissionports.RequestActor {
	return &permissionports.RequestActor{UserID: clerk, PrincipalID: clerk}
}

func newService(t *testing.T, repo *fakeRepo, notifier *fakeNotifier, members fakeMembers) *Service {
	t.Helper()
	customers := customerportsmock.NewMockCustomerRepository(t)
	customers.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&customer.Customer{
		BillingProfile: &customer.CustomerBillingProfile{BillingCurrency: "CAD"},
	}, nil).Maybe()
	return &Service{
		db:        passthroughDB{},
		repo:      repo,
		customers: customers,
		users:     members,
		notifier:  notifier,
		now:       func() int64 { return now },
		l:         zap.NewNop(),
	}
}

func ptr(v int64) *int64 { return &v }

func TestWorklist_PutsBrokenPromisesAndTheLongestOverdueFirst(t *testing.T) {
	t.Parallel()

	promised := &collectionports.WorklistRow{
		CustomerName: "Open promise", OverdueMinor: 900_000, OldestDaysPastDue: 45, CurrencyCode: "CAD",
		Promise: &collectionports.Promise{AmountMinor: 100_000, PromisedOn: now + 3*day},
	}
	broken := &collectionports.WorklistRow{
		CustomerName: "Broken promise", OverdueMinor: 10_000, CurrencyCode: "USD", OldestDaysPastDue: 12,
		Promise:     &collectionports.Promise{AmountMinor: 100_000, PromisedOn: now - 2*day, PaidSinceMinor: 5_000},
		CollectorID: clerk,
	}
	old := &collectionports.WorklistRow{CustomerName: "Ninety days", OverdueMinor: 20_000, CurrencyCode: "USD", OldestDaysPastDue: 90}
	recent := &collectionports.WorklistRow{
		CustomerName: "Recent", OverdueMinor: 500_000, OldestDaysPastDue: 5, CurrencyCode: "USD",
	}
	followUp := &collectionports.WorklistRow{
		CustomerName: "Follow-up today", OverdueMinor: 30_000, CurrencyCode: "USD", OldestDaysPastDue: 10,
		NextFollowUpOn: ptr(now - 3600),
	}

	repo := &fakeRepo{rows: []*collectionports.WorklistRow{promised, broken, old, recent, followUp}}
	page, err := newService(t, repo, &fakeNotifier{}, fakeMembers{}).
		Worklist(t.Context(), &collectionports.WorklistRequest{TenantInfo: tenantInfo})
	require.NoError(t, err)

	names := make([]string, 0, len(page.Rows))
	for _, row := range page.Rows {
		names = append(names, row.CustomerName)
	}
	assert.Equal(t, []string{"Ninety days", "Broken promise", "Follow-up today", "Recent", "Open promise"}, names,
		"critical by amount first, then due follow-ups, then the rest, an open promise last")
	assert.Equal(t, collection.PromiseBroken, broken.Promise.Status)
	assert.Equal(t, collection.PromiseOpen, promised.Promise.Status)
	assert.Equal(t, collectionports.WorklistSummary{
		Accounts: 5,
		Overdue: []collectionports.CurrencyTotal{
			{CurrencyCode: "CAD", AmountMinor: 900_000},
			{CurrencyCode: "USD", AmountMinor: 560_000},
		},
		FollowUpsDue:   1,
		BrokenPromises: 1,
		Unassigned:     4,
	}, page.Summary)
}

func TestWorklist_DueOnlyLeavesOutAccountsWaitingOnAPromiseOrAFollowUp(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{rows: []*collectionports.WorklistRow{
		{CustomerName: "Nobody scheduled", OverdueMinor: 10, OldestDaysPastDue: 3},
		{CustomerName: "Follow-up next week", OverdueMinor: 10, NextFollowUpOn: ptr(now + 7*day)},
		{
			CustomerName: "Promise open", OverdueMinor: 10,
			Promise: &collectionports.Promise{AmountMinor: 100, PromisedOn: now + day},
		},
		{CustomerName: "Follow-up due", OverdueMinor: 10, NextFollowUpOn: ptr(now)},
	}}
	page, err := newService(t, repo, &fakeNotifier{}, fakeMembers{}).Worklist(t.Context(),
		&collectionports.WorklistRequest{TenantInfo: tenantInfo, DueOnly: true, Limit: 1, Offset: 1})
	require.NoError(t, err)

	assert.Equal(t, 2, page.Total)
	require.Len(t, page.Rows, 1, "paged after ranking")
	assert.Equal(t, "Nobody scheduled", page.Rows[0].CustomerName)
	assert.Equal(t, 2, page.Summary.Accounts)
}

func TestLogActivity_APromiseBecomesTheStandingPromiseAndSetsTheFollowUp(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	customerID := pulid.MustNew("cus_")
	_, err := newService(t, repo, &fakeNotifier{}, fakeMembers{}).LogActivity(t.Context(),
		&collectionports.LogActivityRequest{
			TenantInfo:     tenantInfo,
			CustomerID:     customerID,
			Kind:           collection.ActivityPromiseToPay,
			ContactName:    "Dana in payables",
			PromisedAmount: decimal.NewNullDecimal(decimal.RequireFromString("1250.50")),
			PromisedOn:     ptr(now + 5*day),
		}, actor())
	require.NoError(t, err)

	require.Len(t, repo.activities, 1)
	promise := repo.activities[0]
	assert.Equal(t, int64(125_050), *promise.PromisedAmountMinor)
	assert.Equal(t, "CAD", promise.CurrencyCode, "a promise is in the customer's billing currency")
	assert.Equal(t, clerk, promise.CreatedByID)
	require.Len(t, repo.saved, 1)
	assert.Equal(t, promise.ID, repo.saved[0].PromiseActivityID)
	assert.Equal(t, now+6*day, *repo.saved[0].NextFollowUpOn, "follow up the day after the promised date")
}

func TestLogActivity_RefusesAPromiseWithoutAnAmountOrDate(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	_, err := newService(t, repo, &fakeNotifier{}, fakeMembers{}).LogActivity(t.Context(),
		&collectionports.LogActivityRequest{
			TenantInfo: tenantInfo,
			CustomerID: pulid.MustNew("cus_"),
			Kind:       collection.ActivityPromiseToPay,
			FollowUpOn: ptr(now - 3*day),
		}, actor())

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t, []string{"promisedAmount", "promisedOn", "followUpOn"}, fields)
	assert.Empty(t, repo.activities)
}

func TestLogActivity_OnlyTakesContactKinds(t *testing.T) {
	t.Parallel()

	_, err := newService(t, &fakeRepo{}, &fakeNotifier{}, fakeMembers{}).LogActivity(t.Context(),
		&collectionports.LogActivityRequest{
			TenantInfo: tenantInfo,
			CustomerID: pulid.MustNew("cus_"),
			Kind:       collection.ActivityCollectorAssigned,
			Summary:    "assigned myself",
		}, actor())

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "kind", fieldErr.Field)
}

func TestAssignCollector_RecordsTheChangeAndTellsTheNewCollector(t *testing.T) {
	t.Parallel()

	collector := pulid.MustNew("usr_")
	repo := &fakeRepo{}
	notifier := &fakeNotifier{}
	svc := newService(t, repo, notifier, fakeMembers{members: map[pulid.ID]string{collector: "Jordan Lee"}})

	view, err := svc.AssignCollector(t.Context(), &collectionports.AssignCollectorRequest{
		TenantInfo: tenantInfo, CustomerID: pulid.MustNew("cus_"), CollectorID: collector,
	}, actor())
	require.NoError(t, err)

	assert.Equal(t, collector, view.Row.CollectorID)
	require.Len(t, repo.activities, 1)
	assert.Equal(t, collection.ActivityCollectorAssigned, repo.activities[0].Kind)
	assert.Equal(t, "Assigned to Jordan Lee", repo.activities[0].Summary)
	require.Len(t, notifier.sent, 1)
	assert.Equal(t, collector, *notifier.sent[0].TargetUserID)
	assert.Equal(t, "You're collecting from Acme Logistics", notifier.sent[0].Title)
	assert.Equal(t, "2500.00 CAD is overdue across 3 open invoices.", notifier.sent[0].Message)

	_, err = svc.AssignCollector(t.Context(), &collectionports.AssignCollectorRequest{
		TenantInfo: tenantInfo, CustomerID: view.Row.CustomerID, CollectorID: collector,
	}, actor())
	require.NoError(t, err)
	assert.Len(t, repo.activities, 1, "assigning the same collector again records nothing")
	assert.Len(t, notifier.sent, 1)
}

func TestAssignCollector_RefusesSomeoneOutsideTheOrganization(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	_, err := newService(t, repo, &fakeNotifier{}, fakeMembers{}).AssignCollector(t.Context(),
		&collectionports.AssignCollectorRequest{
			TenantInfo: tenantInfo, CustomerID: pulid.MustNew("cus_"), CollectorID: pulid.MustNew("usr_"),
		}, actor())

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "collectorId", fieldErr.Field)
	assert.Empty(t, repo.saved)
}

func TestAssigningYourselfSendsNoNotice(t *testing.T) {
	t.Parallel()

	notifier := &fakeNotifier{}
	_, err := newService(t, &fakeRepo{}, notifier, fakeMembers{members: map[pulid.ID]string{clerk: "Me"}}).
		AssignCollector(t.Context(), &collectionports.AssignCollectorRequest{
			TenantInfo: tenantInfo, CustomerID: pulid.MustNew("cus_"), CollectorID: clerk,
		}, actor())
	require.NoError(t, err)
	assert.Empty(t, notifier.sent)
}

func TestScheduleFollowUp_MovesTheFollowUpAndRecordsWhy(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{account: &collection.Account{ID: pulid.MustNew("cacc_"), NextFollowUpOn: ptr(now)}}
	svc := newService(t, repo, &fakeNotifier{}, fakeMembers{})

	_, err := svc.ScheduleFollowUp(t.Context(), &collectionports.ScheduleFollowUpRequest{
		TenantInfo: tenantInfo, CustomerID: pulid.MustNew("cus_"), FollowUpOn: ptr(now + 4*day),
		Note: "  Controller back from leave on Monday ",
	}, actor())
	require.NoError(t, err)
	assert.Equal(t, now+4*day, *repo.account.NextFollowUpOn)
	assert.Equal(t, "Controller back from leave on Monday", repo.activities[0].Summary)

	_, err = svc.ScheduleFollowUp(t.Context(), &collectionports.ScheduleFollowUpRequest{
		TenantInfo: tenantInfo, CustomerID: pulid.MustNew("cus_"),
	}, actor())
	require.NoError(t, err)
	assert.Nil(t, repo.account.NextFollowUpOn)
	assert.Equal(t, "Cleared the follow-up", repo.activities[1].Summary)
}
