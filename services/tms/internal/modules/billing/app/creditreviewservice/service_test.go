package creditreviewservice

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeLedger struct {
	customerledgerports.CustomerLedgerProjectionRepository
	balances map[pulid.ID]int64
	asked    [][]pulid.ID
	appended int
}

func (f *fakeLedger) SumCustomerBalances(
	_ context.Context,
	req customerledgerports.SumCustomerBalancesRequest,
) (map[pulid.ID]int64, error) {
	f.asked = append(f.asked, req.CustomerIDs)
	out := make(map[pulid.ID]int64, len(req.CustomerIDs))
	for _, id := range req.CustomerIDs {
		if balance, ok := f.balances[id]; ok {
			out[id] = balance
		}
	}
	return out, nil
}

func (f *fakeLedger) AppendEntries(context.Context, []*customerledger.CustomerLedgerEntry) error {
	f.appended++
	return nil
}

type fakeStandings struct {
	mu       sync.Mutex
	profiles map[pulid.ID]*customer.CustomerBillingProfile
	failFor  pulid.ID
	reviewed []*customerports.ReviewCustomerCreditRequest
	pages    [][]customerports.CreditReviewCustomer
	listed   []*customerports.ListCreditReviewCustomersRequest
}

func (f *fakeStandings) ReviewCredit(
	_ context.Context,
	req *customerports.ReviewCustomerCreditRequest,
) (*customerports.ReviewCustomerCreditResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reviewed = append(f.reviewed, req)
	if req.CustomerID == f.failFor {
		return nil, errors.New("database unavailable")
	}
	profile, ok := f.profiles[req.CustomerID]
	if !ok {
		return &customerports.ReviewCustomerCreditResult{}, nil
	}
	return &customerports.ReviewCustomerCreditResult{
		CustomerName: "Acme Logistics",
		Profile:      profile,
		Outcome:      profile.ReviewCredit(req.ExposureMinor, req.Now),
	}, nil
}

func (f *fakeStandings) ListCreditReviewCustomers(
	_ context.Context,
	req *customerports.ListCreditReviewCustomersRequest,
) ([]customerports.CreditReviewCustomer, error) {
	f.listed = append(f.listed, req)
	if len(f.pages) == 0 {
		return nil, nil
	}
	page := f.pages[0]
	f.pages = f.pages[1:]
	return page, nil
}

type fakeNotifier struct {
	sent []notificationservice.NotifyPermittedRequest
}

func (f *fakeNotifier) NotifyPermitted(
	_ context.Context,
	req notificationservice.NotifyPermittedRequest,
) (int, error) {
	f.sent = append(f.sent, req)
	return 1, nil
}

func managedProfile(limit string) *customer.CustomerBillingProfile {
	return &customer.CustomerBillingProfile{
		EnforceCreditLimit: true,
		AutoCreditHold:     true,
		CreditLimit:        decimal.NewNullDecimal(decimal.RequireFromString(limit)),
		CreditStatus:       customer.CreditStatusActive,
		BillingCurrency:    "USD",
	}
}

func newService(ledger *fakeLedger, standings *fakeStandings, notifier *fakeNotifier) *Service {
	return &Service{ledger: ledger, customers: standings, notifier: notifier, l: zap.NewNop()}
}

var tenant = pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

func TestReview_HoldsAnOverLimitCustomerAndTellsReceivables(t *testing.T) {
	t.Parallel()

	held := pulid.MustNew("cus_")
	fine := pulid.MustNew("cus_")
	ledger := &fakeLedger{balances: map[pulid.ID]int64{held: 1_500_000, fine: -20_000}}
	standings := &fakeStandings{profiles: map[pulid.ID]*customer.CustomerBillingProfile{
		held: managedProfile("10000"),
		fine: managedProfile("10000"),
	}}
	notifier := &fakeNotifier{}

	require.NoError(t, newService(ledger, standings, notifier).
		Review(t.Context(), tenant, []pulid.ID{held, fine, held}))

	assert.Equal(t, [][]pulid.ID{{held, fine}}, ledger.asked, "each customer is summed once")
	require.Len(t, standings.reviewed, 2)
	assert.Equal(t, int64(1_500_000), standings.reviewed[0].ExposureMinor)
	assert.Equal(t, int64(0), standings.reviewed[1].ExposureMinor,
		"credit on account is no exposure, not negative exposure")

	require.Len(t, notifier.sent, 1)
	sent := notifier.sent[0]
	assert.Equal(t, permission.ResourceCustomerPayment, sent.Resource)
	assert.Equal(t, permission.OpCreate, sent.Operation)
	assert.Equal(t, tenant, sent.Tenant)
	assert.Equal(t, "Acme Logistics is on credit hold", sent.Notification.Title)
	assert.Contains(t, sent.Notification.Message,
		"Open receivables of 15000.00 USD are over the credit limit of 10000.00 USD")
	assert.Equal(t, held.String()+":Held", *sent.Notification.CorrelationID)
}

func TestReview_TellsReceivablesWhenAHoldIsLiftedButNotAboutWarnings(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("cus_")
	profile := managedProfile("10000")
	profile.ReviewCredit(1_200_000, 1)
	ledger := &fakeLedger{balances: map[pulid.ID]int64{id: 950_000}}
	standings := &fakeStandings{profiles: map[pulid.ID]*customer.CustomerBillingProfile{id: profile}}
	notifier := &fakeNotifier{}
	svc := newService(ledger, standings, notifier)

	require.NoError(t, svc.Review(t.Context(), tenant, []pulid.ID{id}))
	require.Len(t, notifier.sent, 1)
	assert.Equal(t, "Acme Logistics is off credit hold", notifier.sent[0].Notification.Title)

	ledger.balances[id] = 0
	require.NoError(t, svc.Review(t.Context(), tenant, []pulid.ID{id}))
	assert.Len(t, notifier.sent, 1, "leaving the warning band pages nobody")
	assert.Equal(t, customer.CreditStatusActive, profile.CreditStatus)
}

func TestReview_ReviewsTheRestWhenOneCustomerFails(t *testing.T) {
	t.Parallel()

	broken := pulid.MustNew("cus_")
	other := pulid.MustNew("cus_")
	standings := &fakeStandings{
		failFor:  broken,
		profiles: map[pulid.ID]*customer.CustomerBillingProfile{other: managedProfile("100")},
	}

	err := newService(&fakeLedger{}, standings, &fakeNotifier{}).
		Review(t.Context(), tenant, []pulid.ID{broken, other})

	require.EqualError(t, err, "credit review failed for 1 of 2 customers")
	assert.Len(t, standings.reviewed, 2)
}

func TestSweep_ReviewsEveryPageUnderEachCustomersOwnTenant(t *testing.T) {
	t.Parallel()

	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	firstPage := make([]customerports.CreditReviewCustomer, 0, SweepPageSize)
	for i := range SweepPageSize {
		owner := tenant
		if i >= SweepPageSize/2 {
			owner = other
		}
		firstPage = append(firstPage, customerports.CreditReviewCustomer{
			TenantInfo: owner,
			CustomerID: pulid.MustNew("cus_"),
		})
	}
	last := customerports.CreditReviewCustomer{TenantInfo: other, CustomerID: pulid.MustNew("cus_")}
	ledger := &fakeLedger{}
	standings := &fakeStandings{pages: [][]customerports.CreditReviewCustomer{
		firstPage,
		{last},
	}}

	var beats []string
	result, err := newService(ledger, standings, &fakeNotifier{}).
		Sweep(t.Context(), func(detail string) { beats = append(beats, detail) })
	require.NoError(t, err)

	assert.Equal(t, SweepPageSize+1, result.Customers)
	assert.Zero(t, result.Failed)
	require.Len(t, standings.listed, 2, "a short page ends the sweep")
	assert.Nil(t, standings.listed[0].After)
	assert.Equal(t, firstPage[len(firstPage)-1], *standings.listed[1].After)
	assert.Len(t, ledger.asked, 3, "one balance query per tenant per page")
	assert.Equal(t, []string{tenant.OrgID.String(), other.OrgID.String(), other.OrgID.String()}, beats)
}

func TestReviewingLedger_ReviewsTheCustomersItPostedFor(t *testing.T) {
	t.Parallel()

	first := pulid.MustNew("cus_")
	second := pulid.MustNew("cus_")
	ledger := &fakeLedger{balances: map[pulid.ID]int64{first: 10}}
	standings := &fakeStandings{}
	wrapped := &reviewingLedger{
		CustomerLedgerProjectionRepository: ledger,
		reviews:                            newService(ledger, standings, &fakeNotifier{}),
		l:                                  zap.NewNop(),
	}

	require.NoError(t, wrapped.AppendEntries(t.Context(), []*customerledger.CustomerLedgerEntry{
		{OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID, CustomerID: first},
		{OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID, CustomerID: second},
		{OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID, CustomerID: first},
	}))

	assert.Equal(t, 1, ledger.appended)
	require.Len(t, standings.reviewed, 2)
	assert.Equal(t, first, standings.reviewed[0].CustomerID)
	assert.Equal(t, int64(10), standings.reviewed[0].ExposureMinor)
	assert.Equal(t, second, standings.reviewed[1].CustomerID)
}
