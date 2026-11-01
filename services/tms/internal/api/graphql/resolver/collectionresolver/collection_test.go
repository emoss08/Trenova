package collectionresolver

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/paymentreminderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCollections struct {
	collectionports.Service

	worklist *collectionports.WorklistRequest
	logged   *collectionports.LogActivityRequest
	assigned *collectionports.AssignCollectorRequest
	account  collectionports.GetAccountRequest
	before   *collectionports.ActivityCursor
	view     *collectionports.AccountView
}

func (f *fakeCollections) Worklist(
	_ context.Context,
	req *collectionports.WorklistRequest,
) (*collectionports.WorklistPage, error) {
	f.worklist = req
	return &collectionports.WorklistPage{Total: 3}, nil
}

func (f *fakeCollections) GetAccount(
	_ context.Context,
	req collectionports.GetAccountRequest,
	before *collectionports.ActivityCursor,
) (*collectionports.AccountView, error) {
	f.account = req
	f.before = before
	return f.view, nil
}

func (f *fakeCollections) LogActivity(
	_ context.Context,
	req *collectionports.LogActivityRequest,
	_ *permissionports.RequestActor,
) (*collectionports.AccountView, error) {
	f.logged = req
	return f.view, nil
}

func (f *fakeCollections) AssignCollector(
	_ context.Context,
	req *collectionports.AssignCollectorRequest,
	_ *permissionports.RequestActor,
) (*collectionports.AccountView, error) {
	f.assigned = req
	return f.view, nil
}

type fakeReminders struct {
	paymentreminderports.Service

	sent *paymentreminderports.SendNowRequest
}

func (f *fakeReminders) SendNow(
	_ context.Context,
	req *paymentreminderports.SendNowRequest,
	_ *permissionports.RequestActor,
) ([]*paymentreminderports.SendNowResult, error) {
	f.sent = req
	return []*paymentreminderports.SendNowResult{{SentTo: []string{"ap@halstead.test"}}}, nil
}

func TestCollectionOperations_CheckTheAccountsReceivablePermission(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_").String()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"collectionWorklist", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).CollectionWorklist(
				ctx,
				gqlmodel.CollectionWorklistInput{},
			)
			return err
		}},
		{"collectionAccount", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).CollectionAccount(ctx, customerID, nil)
			return err
		}},
		{"logCollectionActivity", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).LogCollectionActivity(
				ctx,
				customerID,
				gqlmodel.LogCollectionActivityInput{Kind: collection.ActivityCall},
			)
			return err
		}},
		{
			"assignCollectionCollector",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).AssignCollectionCollector(ctx, customerID, nil)
				return err
			},
		},
		{
			"scheduleCollectionFollowUp",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ScheduleCollectionFollowUp(
					ctx,
					customerID,
					nil,
					"",
				)
				return err
			},
		},
		{"sendPaymentReminder", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).SendPaymentReminder(ctx, customerID, "")
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			resolvertest.RequireRefused(
				t,
				call.run(ctx, deps),
				engine,
				permission.ResourceAccountsReceivable,
				call.operation,
			)
		})
	}
}

func newDeps(
	collections *fakeCollections,
	reminders *fakeReminders,
) (*Deps, *resolvertest.RecordingPermissionEngine) {
	engine := &resolvertest.RecordingPermissionEngine{}
	return &Deps{
		Core:                   &resolvercore.Core{PermissionEngine: engine},
		CollectionService:      collections,
		PaymentReminderService: reminders,
	}, engine
}

func TestCollectionWorklist_MapsTheFiltersIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	collections := &fakeCollections{}
	deps, _ := newDeps(collections, nil)

	page, err := (&QueryResolver{deps}).CollectionWorklist(
		resolvertest.AuthedContext(t, authCtx),
		gqlmodel.CollectionWorklistInput{
			Scope:   collectionports.CollectorScopeMine,
			DueOnly: true,
			Search:  "halstead",
			Limit:   25,
			Offset:  50,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, 3, page.Total)
	assert.Equal(t, &collectionports.WorklistRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		Scope:      collectionports.CollectorScopeMine,
		DueOnly:    true,
		Search:     "halstead",
		Limit:      25,
		Offset:     50,
	}, collections.worklist)
}

func TestCollectionAccount_PassesTheHistoryCursor(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	activityID := pulid.MustNew("cact_")
	collections := &fakeCollections{view: &collectionports.AccountView{HasMore: true}}
	deps, _ := newDeps(collections, nil)

	view, err := (&QueryResolver{deps}).CollectionAccount(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		customerID.String(),
		&gqlmodel.CollectionActivityCursorInput{CreatedAt: 1792000000, ID: activityID.String()},
	)

	require.NoError(t, err)
	assert.True(t, view.HasMore)
	assert.Equal(t, customerID, collections.account.CustomerID)
	assert.Equal(
		t,
		&collectionports.ActivityCursor{CreatedAt: 1792000000, ID: activityID},
		collections.before,
	)
}

func TestLogCollectionActivity_MapsAPromiseToPay(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	customerID := pulid.MustNew("cus_")
	invoiceID := pulid.MustNew("inv_")
	invoiceRaw := invoiceID.String()
	amount := "1250.50"
	promisedOn := 1792500000
	collections := &fakeCollections{view: &collectionports.AccountView{}}
	deps, engine := newDeps(collections, nil)

	_, err := (&MutationResolver{deps}).LogCollectionActivity(
		resolvertest.AuthedContext(t, authCtx),
		customerID.String(),
		gqlmodel.LogCollectionActivityInput{
			Kind:           collection.ActivityPromiseToPay,
			Summary:        "Will pay Friday",
			ContactName:    "Dana",
			InvoiceID:      &invoiceRaw,
			PromisedAmount: &amount,
			PromisedOn:     &promisedOn,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, permission.OpUpdate, engine.Request.Operation)
	require.NotNil(t, collections.logged)
	assert.Equal(t, resolvercore.TenantInfo(authCtx), collections.logged.TenantInfo)
	assert.Equal(t, customerID, collections.logged.CustomerID)
	assert.Equal(t, collection.ActivityPromiseToPay, collections.logged.Kind)
	assert.Equal(t, invoiceID, collections.logged.InvoiceID)
	assert.True(t, collections.logged.PromisedAmount.Valid)
	assert.True(
		t,
		decimal.RequireFromString("1250.50").Equal(collections.logged.PromisedAmount.Decimal),
	)
	require.NotNil(t, collections.logged.PromisedOn)
	assert.Equal(t, int64(1792500000), *collections.logged.PromisedOn)
	assert.Nil(t, collections.logged.FollowUpOn)
}

func TestLogCollectionActivity_RefusesAnAmountThatIsNotANumber(t *testing.T) {
	t.Parallel()

	amount := "twelve"
	deps, _ := newDeps(&fakeCollections{}, nil)

	_, err := (&MutationResolver{deps}).LogCollectionActivity(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("cus_").String(),
		gqlmodel.LogCollectionActivityInput{
			Kind:           collection.ActivityPromiseToPay,
			PromisedAmount: &amount,
		},
	)

	var validationErr *errortypes.Error
	require.True(t, errors.As(err, &validationErr), "%v", err)
	assert.Equal(t, "promisedAmount", validationErr.Field)
}

func TestAssignCollectionCollector_LeavesTheAccountUnassignedOnNull(t *testing.T) {
	t.Parallel()

	collections := &fakeCollections{view: &collectionports.AccountView{}}
	deps, _ := newDeps(collections, nil)

	_, err := (&MutationResolver{deps}).AssignCollectionCollector(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("cus_").String(),
		nil,
	)

	require.NoError(t, err)
	require.NotNil(t, collections.assigned)
	assert.True(t, collections.assigned.CollectorID.IsNil())
}

func TestSendPaymentReminder_SendsTheNoteForTheCustomer(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	reminders := &fakeReminders{}
	deps, _ := newDeps(nil, reminders)

	sent, err := (&MutationResolver{deps}).SendPaymentReminder(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		customerID.String(),
		"As discussed",
	)

	require.NoError(t, err)
	require.Len(t, sent, 1)
	assert.Equal(t, customerID, reminders.sent.CustomerID)
	assert.Equal(t, "As discussed", reminders.sent.Note)
}

func TestCollectionWorklistRowLastActivityKind_IsNullWithoutActivity(t *testing.T) {
	t.Parallel()

	resolver := &CollectionWorklistRowResolver{}

	none, err := resolver.LastActivityKind(t.Context(), &collectionports.WorklistRow{})
	require.NoError(t, err)
	assert.Nil(t, none)

	kind, err := resolver.LastActivityKind(t.Context(), &collectionports.WorklistRow{
		LastActivityKind: collection.ActivityCall,
	})
	require.NoError(t, err)
	require.NotNil(t, kind)
	assert.Equal(t, collection.ActivityCall, *kind)
}
