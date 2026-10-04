package billingqueueservice

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// memoryReview is the review repository held in memory, with the same rules
// the database one keeps: one issue per finding, open issues cleared when the
// finding goes, settled ones left alone.
type memoryReview struct {
	repositories.BillingQueueReviewRepository

	mu     sync.Mutex
	issues []*billingqueue.Issue
	events []*billingqueue.ItemEvent
	open   int
}

func (m *memoryReview) ListIssues(_ context.Context, _ pagination.TenantInfo, itemID pulid.ID) ([]*billingqueue.Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*billingqueue.Issue, 0, len(m.issues))
	for _, issue := range m.issues {
		if issue.ItemID == itemID {
			clone := *issue
			out = append(out, &clone)
		}
	}

	return out, nil
}

func (m *memoryReview) SyncIssues(
	ctx context.Context,
	req *repositories.SyncBillingQueueIssuesRequest,
) (*repositories.SyncBillingQueueIssuesResult, error) {
	m.mu.Lock()
	result := &repositories.SyncBillingQueueIssuesResult{}
	found := map[string]bool{}
	for _, finding := range req.Findings {
		found[finding.FindingKey()] = true
		var current *billingqueue.Issue
		for _, issue := range m.issues {
			if issue.ItemID == req.ItemID && issue.FindingKey() == finding.FindingKey() {
				current = issue
			}
		}
		switch {
		case current == nil:
			finding.ID = pulid.MustNew("bqis_")
			finding.ItemID = req.ItemID
			finding.CreatedAt = timeutils.NowUnix()
			m.issues = append(m.issues, finding)
			result.Raised = append(result.Raised, finding)
		case current.ResolutionKey != nil && *current.ResolutionKey == billingqueue.ResolutionCleared:
			current.ResolutionKey = nil
			current.ResolvedAt = nil
			result.Reopened = append(result.Reopened, current)
		}
	}
	for _, issue := range m.issues {
		if issue.ItemID == req.ItemID && issue.IsOpen() && !found[issue.FindingKey()] {
			cleared := billingqueue.ResolutionCleared
			now := timeutils.NowUnix()
			issue.ResolutionKey = &cleared
			issue.ResolvedAt = &now
			result.Cleared = append(result.Cleared, issue)
		}
	}
	m.mu.Unlock()

	issues, err := m.ListIssues(ctx, req.TenantInfo, req.ItemID)
	result.Issues = issues

	return result, err
}

func (m *memoryReview) GetIssue(_ context.Context, req *repositories.GetBillingQueueIssueRequest) (*billingqueue.Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, issue := range m.issues {
		if issue.ID == req.IssueID && issue.ItemID == req.ItemID {
			clone := *issue
			return &clone, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Billing queue issue not found")
}

func (m *memoryReview) UpdateIssue(_ context.Context, issue *billingqueue.Issue) (*billingqueue.Issue, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, stored := range m.issues {
		if stored.ID == issue.ID {
			clone := *issue
			m.issues[i] = &clone
			return issue, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Billing queue issue not found")
}

func (m *memoryReview) CountOpenIssues(_ context.Context, _ pagination.TenantInfo, itemID pulid.ID) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := m.open
	for _, issue := range m.issues {
		if issue.ItemID == itemID && issue.IsOpen() {
			count++
		}
	}

	return count, nil
}

func (m *memoryReview) CreateEvents(_ context.Context, events ...*billingqueue.ItemEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, events...)

	return nil
}

func (m *memoryReview) FindDuplicates(context.Context, *repositories.FindBillingQueueDuplicatesRequest) ([]*billingqueue.DuplicateRef, error) {
	return nil, nil
}

func (m *memoryReview) eventTexts() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.events))
	for _, event := range m.events {
		out = append(out, event.Text)
	}

	return out
}

// reviewWorld is one shipment with a linehaul the contract rated and a lumper
// fee added by hand, queued for its customer, with the repositories it is
// read and written through kept in step like a database would.
type reviewWorld struct {
	svc      *service
	review   *memoryReview
	tenant   pagination.TenantInfo
	actor    *services.RequestActor
	item     *billingqueue.BillingQueueItem
	shp      *shipment.Shipment
	lumperID pulid.ID

	mu sync.Mutex
}

func newReviewWorld(t *testing.T, status billingqueue.Status) *reviewWorld {
	t.Helper()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	cus := &customer.Customer{
		ID:             pulid.MustNew("cus_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           "Acme Manufacturing",
		Code:           "ACME",
		BillingProfile: &customer.CustomerBillingProfile{PaymentTerm: customer.PaymentTermNet30},
		EmailProfile:   &customer.CustomerEmailProfile{ToRecipients: "ap@acme.com"},
	}
	lumperID := pulid.MustNew("ac_")
	shp := &shipment.Shipment{
		ID:                  pulid.MustNew("shp_"),
		OrganizationID:      tenantInfo.OrgID,
		BusinessUnitID:      tenantInfo.BuID,
		ProNumber:           "S-1001",
		CustomerID:          cus.ID,
		Customer:            cus,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString("1945.65")),
		RatingDetail:        &shipment.RatingDetail{Result: 1945.65},
		AdditionalCharges: []*shipment.AdditionalCharge{{
			ID:                  lumperID,
			OrganizationID:      tenantInfo.OrgID,
			BusinessUnitID:      tenantInfo.BuID,
			AccessorialChargeID: pulid.MustNew("acc_"),
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.RequireFromString("110"),
			Unit:                1,
			AccessorialCharge:   &accessorialcharge.AccessorialCharge{Description: "Lumper fee"},
		}},
	}
	syncTotals(shp)

	biller := pulid.MustNew("usr_")
	item := &billingqueue.BillingQueueItem{
		ID:                   pulid.MustNew("bqi_"),
		OrganizationID:       tenantInfo.OrgID,
		BusinessUnitID:       tenantInfo.BuID,
		ShipmentID:           shp.ID,
		BillToCustomerID:     cus.ID,
		Status:               status,
		BillType:             billingqueue.BillTypeInvoice,
		Number:               "INV-24101",
		AssignedBillerID:     &biller,
		AssignedBiller:       &tenant.User{ID: biller, Name: "Avery Lane"},
		AllocatedTotalAmount: decimal.RequireFromString("2055.65"),
	}

	w := &reviewWorld{
		review:   &memoryReview{},
		tenant:   tenantInfo,
		actor:    userActor(tenantInfo),
		item:     item,
		shp:      shp,
		lumperID: lumperID,
	}

	repo := mocks.NewMockBillingQueueRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *repositories.GetBillingQueueItemByIDRequest) (*billingqueue.BillingQueueItem, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			clone := *w.item
			clone.Review = nil
			return &clone, nil
		}).Maybe()
	repo.EXPECT().Update(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *billingqueue.BillingQueueItem) (*billingqueue.BillingQueueItem, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			clone := *entity
			w.item = &clone
			return entity, nil
		}).Maybe()

	shipments := mocks.NewMockShipmentRepository(t)
	shipments.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, *repositories.GetShipmentByIDRequest) (*shipment.Shipment, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			clone := *w.shp
			clone.AdditionalCharges = make([]*shipment.AdditionalCharge, 0, len(w.shp.AdditionalCharges))
			for _, charge := range w.shp.AdditionalCharges {
				c := *charge
				clone.AdditionalCharges = append(clone.AdditionalCharges, &c)
			}
			return &clone, nil
		}).Maybe()
	shipments.EXPECT().UpdateDerivedState(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, shp *shipment.Shipment) (*shipment.Shipment, error) {
			w.mu.Lock()
			defer w.mu.Unlock()
			charges := make([]*shipment.AdditionalCharge, 0, len(shp.AdditionalCharges))
			for _, charge := range shp.AdditionalCharges {
				c := *charge
				if c.ID.IsNil() {
					c.ID = pulid.MustNew("ac_")
				}
				if c.AccessorialCharge == nil {
					c.AccessorialCharge = &accessorialcharge.AccessorialCharge{Description: "Lumper fee"}
				}
				charges = append(charges, &c)
			}
			w.shp.AdditionalCharges = charges
			syncTotals(w.shp)
			return w.shp, nil
		}).Maybe()

	customers := mocks.NewMockCustomerRepository(t)
	customers.EXPECT().GetByID(mock.Anything, mock.Anything).Return(cus, nil).Maybe()
	customers.EXPECT().GetByIDs(mock.Anything, mock.Anything).Return([]*customer.Customer{cus}, nil).Maybe()

	invoices := mocks.NewMockInvoiceRepository(t)
	invoices.EXPECT().ListByShipmentIDs(mock.Anything, mock.Anything).
		Return(map[pulid.ID][]*invoice.Invoice{}, nil).Maybe()

	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().PublishResourceInvalidation(mock.Anything, mock.Anything).Return(nil).Maybe()
	users := mocks.NewMockUserRepository(t)
	users.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tenant.User{ID: tenantInfo.UserID, Name: "Avery Lane"}, nil).Maybe()

	w.svc = &service{
		l:            zap.NewNop(),
		db:           passthroughDB{},
		repo:         repo,
		shipmentRepo: shipments,
		customerRepo: customers,
		invoiceRepo:  invoices,
		userRepo:     users,
		auditService: audit,
		realtime:     realtime,
		validator:    testValidator(),
		reviewRepo:   w.review,
	}

	return w
}

func syncTotals(shp *shipment.Shipment) {
	freight := shp.FreightChargeAmount.Decimal
	other := shipment.AdditionalChargesTotal(shp.AdditionalCharges, freight)
	shp.OtherChargeAmount = decimal.NewNullDecimal(other)
	shp.TotalChargeAmount = decimal.NewNullDecimal(freight.Add(other))
}

func (w *reviewWorld) read(t *testing.T) *billingqueue.BillingQueueItem {
	t.Helper()
	item, err := w.svc.GetByID(t.Context(), &repositories.GetBillingQueueItemByIDRequest{
		ItemID:                w.item.ID,
		TenantInfo:            w.tenant,
		ExpandShipmentDetails: true,
	})
	require.NoError(t, err)
	require.NotNil(t, item.Review)

	return item
}

func checkOf(item *billingqueue.BillingQueueItem, key billingqueue.CheckKey) *billingqueue.Check {
	for _, check := range item.Review.Checks {
		if check.Key == key {
			return check
		}
	}

	return nil
}

func lineOf(item *billingqueue.BillingQueueItem, label string) *billingqueue.ChargeLine {
	for _, line := range item.Review.Charges.Lines {
		if line.Label == label {
			return line
		}
	}

	return nil
}

// Reading an item raises what its checks find, once: the lumper fee nothing
// on the rate con prices is flagged, its line is marked, and the item cannot
// be approved until a person settles it.
func TestReviewRaisesTheLumperFeeOnceAndHoldsApproval(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	item := w.read(t)
	w.read(t)

	charges := checkOf(item, billingqueue.CheckCharges)
	require.Equal(t, billingqueue.CheckStateWarn, charges.State)
	assert.Equal(t, "$110.00 lumper fee isn't on the rate con", charges.Detail)
	assert.Equal(t, 1, item.Review.NeedsCount)
	assert.False(t, item.Review.Ready)
	assert.True(t, lineOf(item, "Lumper fee").Flagged)
	assert.True(t, decimal.RequireFromString("110").Equal(item.Review.Charges.Difference))

	require.Len(t, w.review.issues, 1, "a second read raises nothing new")
	assert.Equal(t, []string{"Flagged: $110.00 lumper fee isn't on the rate con"}, w.review.eventTexts())

	_, err := w.svc.UpdateStatus(t.Context(), &services.UpdateBillingQueueStatusRequest{
		ItemID:     w.item.ID,
		NewStatus:  billingqueue.StatusApproved,
		TenantInfo: w.tenant,
	}, w.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Settle the flagged check first")
}

func TestResolveKeepSettlesTheCheckAndUndoReopensIt(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	issueID := checkOf(w.read(t), billingqueue.CheckCharges).IssueID

	item, err := w.svc.ResolveIssue(t.Context(), &services.ResolveBillingQueueIssueRequest{
		ItemID: w.item.ID, IssueID: issueID, OptionKey: "keep", TenantInfo: w.tenant,
	}, w.actor)
	require.NoError(t, err)

	charges := checkOf(item, billingqueue.CheckCharges)
	assert.Equal(t, billingqueue.CheckStateOK, charges.State)
	assert.Equal(t, billingqueue.CheckCodeResolved, charges.Code)
	assert.Equal(t, "Lumper fee kept", charges.Detail)
	assert.True(t, item.Review.Ready)
	assert.False(t, lineOf(item, "Lumper fee").Flagged)

	item, err = w.svc.UndoIssue(t.Context(), &services.UndoBillingQueueIssueRequest{
		ItemID: w.item.ID, IssueID: issueID, TenantInfo: w.tenant,
	}, w.actor)
	require.NoError(t, err)
	assert.Equal(t, billingqueue.CheckStateWarn, checkOf(item, billingqueue.CheckCharges).State)
	assert.Contains(t, w.review.eventTexts(), "Lumper fee kept")
	assert.Contains(t, w.review.eventTexts(), "Undid: lumper fee kept")
}

// Removing the lumper fee takes it off the shipment through the charge edit
// path; the ledger keeps it, struck, and undo puts it back and reopens the
// same issue rather than raising a new one.
func TestResolveDropRemovesTheChargeAndUndoRestoresIt(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusReadyForReview)
	issueID := checkOf(w.read(t), billingqueue.CheckCharges).IssueID

	item, err := w.svc.ResolveIssue(t.Context(), &services.ResolveBillingQueueIssueRequest{
		ItemID: w.item.ID, IssueID: issueID, OptionKey: "drop", TenantInfo: w.tenant,
	}, w.actor)
	require.NoError(t, err)

	assert.Empty(t, w.shp.AdditionalCharges, "the charge left the shipment")
	removed := lineOf(item, "Lumper fee")
	require.NotNil(t, removed)
	assert.True(t, removed.Removed)
	assert.Equal(t, issueID, removed.IssueID)
	assert.True(t, decimal.RequireFromString("1945.65").Equal(item.Review.Charges.BilledTotal))
	assert.True(t, item.Review.Charges.Difference.IsZero())
	assert.Equal(t, "Lumper fee removed", checkOf(item, billingqueue.CheckCharges).Detail)

	item, err = w.svc.UndoIssue(t.Context(), &services.UndoBillingQueueIssueRequest{
		ItemID: w.item.ID, IssueID: issueID, TenantInfo: w.tenant,
	}, w.actor)
	require.NoError(t, err)

	require.Len(t, w.shp.AdditionalCharges, 1, "the charge is back")
	assert.True(t, decimal.RequireFromString("110").Equal(w.shp.AdditionalCharges[0].Amount))
	charges := checkOf(item, billingqueue.CheckCharges)
	assert.Equal(t, billingqueue.CheckStateWarn, charges.State)
	assert.Equal(t, issueID, charges.IssueID, "the same issue, following the restored charge")
	open := 0
	for _, issue := range w.review.issues {
		if issue.IsOpen() {
			open++
		}
	}
	assert.Equal(t, 1, open)
}

func TestResolveRefusesAnAgent(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	_, err := w.svc.ResolveIssue(t.Context(), &services.ResolveBillingQueueIssueRequest{
		ItemID: w.item.ID, IssueID: pulid.MustNew("bqis_"), OptionKey: "keep", TenantInfo: w.tenant,
	}, &services.RequestActor{PrincipalType: services.PrincipalTypeAgent, PrincipalID: pulid.MustNew("agt_")})

	require.Error(t, err)
}

// Approving an item a biller has but nobody started reviewing starts the
// review on the way: it used to be refused from the item.
func TestApproveFromReadyForReviewStartsTheReview(t *testing.T) {
	t.Parallel()

	svc, _, item, tenantInfo := statusChangeFixture(t, billingqueue.StatusReadyForReview)

	approved, err := svc.UpdateStatus(t.Context(), &services.UpdateBillingQueueStatusRequest{
		ItemID:     item.ID,
		NewStatus:  billingqueue.StatusApproved,
		TenantInfo: tenantInfo,
	}, userActor(tenantInfo))

	require.NoError(t, err)
	assert.Equal(t, billingqueue.StatusApproved, approved.Status)
	assert.NotNil(t, approved.ReviewStartedAt)
	assert.NotNil(t, approved.ReviewCompletedAt)
}

func TestHoldRecordsItsReasonAndReleaseReturnsWhereItWas(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	reason := billingqueue.HoldReasonCustomerDispute

	held, err := w.svc.UpdateStatus(t.Context(), &services.UpdateBillingQueueStatusRequest{
		ItemID:         w.item.ID,
		NewStatus:      billingqueue.StatusOnHold,
		HoldReasonCode: &reason,
		TenantInfo:     w.tenant,
	}, w.actor)
	require.NoError(t, err)
	require.NotNil(t, held.HoldReasonCode)
	assert.Equal(t, billingqueue.HoldReasonCustomerDispute, *held.HoldReasonCode)
	require.NotNil(t, held.StatusBeforeHold)
	assert.Equal(t, billingqueue.StatusInReview, *held.StatusBeforeHold)
	assert.NotNil(t, held.HeldAt)

	released, err := w.svc.Release(t.Context(), &services.BillingQueueItemRequest{
		ItemID: w.item.ID, TenantInfo: w.tenant,
	}, w.actor)
	require.NoError(t, err)
	assert.Equal(t, billingqueue.StatusInReview, released.Status)
	assert.Nil(t, released.HoldReasonCode)
	assert.Nil(t, released.StatusBeforeHold)
	assert.Nil(t, released.HeldAt)

	assert.Contains(t, w.review.eventTexts(), "Put on hold · customer dispute")
	assert.Contains(t, w.review.eventTexts(), "Released the hold")
}

func TestHoldRefusesAReasonItDoesNotKnow(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	reason := billingqueue.HoldReasonCode("Lunch")
	_, err := w.svc.UpdateStatus(t.Context(), &services.UpdateBillingQueueStatusRequest{
		ItemID: w.item.ID, NewStatus: billingqueue.StatusOnHold, HoldReasonCode: &reason, TenantInfo: w.tenant,
	}, w.actor)

	require.Error(t, err)
}

func TestReleaseRefusesAnItemNotOnHold(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	_, err := w.svc.Release(t.Context(), &services.BillingQueueItemRequest{
		ItemID: w.item.ID, TenantInfo: w.tenant,
	}, w.actor)

	require.Error(t, err)
}

func TestPostPostsTheItemsInvoiceAndSaysWhereItWent(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusApproved)
	w.shp.Customer.BillingProfile.AutoSendInvoiceOnGeneration = true
	w.shp.Customer.BillingProfile.EmailInvoiceEnabled = true

	draft := &invoice.Invoice{ID: pulid.MustNew("inv_"), Number: "INV-24101", Status: invoice.StatusDraft}
	invoices := mocks.NewMockInvoiceRepository(t)
	invoices.EXPECT().GetByBillingQueueItemID(mock.Anything, mock.Anything).Return(draft, nil)
	invoiceSvc := mocks.NewMockInvoiceService(t)
	invoiceSvc.EXPECT().Post(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *services.PostInvoiceRequest, _ *services.RequestActor) (*invoice.Invoice, error) {
			assert.Equal(t, draft.ID, req.InvoiceID)
			w.mu.Lock()
			w.item.Status = billingqueue.StatusPosted
			w.mu.Unlock()
			posted := *draft
			posted.Status = invoice.StatusPosted
			return &posted, nil
		})
	w.svc.invoiceRepo = invoices
	w.svc.invoiceSvc = invoiceSvc

	result, err := w.svc.Post(t.Context(), &services.BillingQueueItemRequest{
		ItemID: w.item.ID, TenantInfo: w.tenant,
	}, w.actor)

	require.NoError(t, err)
	assert.Equal(t, "INV-24101", result.InvoiceNumber)
	assert.Equal(t, "ap@acme.com", result.SentTo)
	assert.Equal(t, billingqueue.StatusPosted, result.Item.Status)
	assert.Contains(t, w.review.eventTexts(), "Posted as INV-24101 · emailed to ap@acme.com")
}

func TestPostRefusesAnItemNotYetApproved(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	w.svc.invoiceSvc = mocks.NewMockInvoiceService(t)

	_, err := w.svc.Post(t.Context(), &services.BillingQueueItemRequest{
		ItemID: w.item.ID, TenantInfo: w.tenant,
	}, w.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Approve the item before posting it")
}

func TestApproveIfReadyLeavesAnItemThatStillNeedsAPerson(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)

	result, err := w.svc.ApproveIfReady(t.Context(), &services.ApproveIfReadyRequest{
		ItemID: w.item.ID, TenantInfo: w.tenant,
	}, w.actor)

	require.NoError(t, err)
	assert.False(t, result.Approved)
	assert.Equal(t, billingqueue.ApprovalFailureNotReady, result.FailureCode)
	assert.Equal(t, "$110.00 lumper fee isn't on the rate con", result.Reason)
	assert.Equal(t, billingqueue.StatusInReview, w.item.Status)
}

func TestApproveIfReadySkipsAnItemSomebodyHeld(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusOnHold)

	result, err := w.svc.ApproveIfReady(t.Context(), &services.ApproveIfReadyRequest{
		ItemID: w.item.ID, TenantInfo: w.tenant,
	}, w.actor)

	require.NoError(t, err)
	assert.Equal(t, billingqueue.ApprovalFailureOnHold, result.FailureCode)
}

// unassigned leaves the world's item with nobody as its biller.
func (w *reviewWorld) unassigned() {
	w.item.AssignedBillerID = nil
	w.item.AssignedBiller = nil
}

func TestApproveIfReadyMakesTheApproverTheBillerWhenThatIsAllItWaitsOn(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	w.unassigned()
	w.shp.AdditionalCharges = nil
	syncTotals(w.shp)
	w.item.AllocatedTotalAmount = w.shp.TotalChargeAmount.Decimal
	invoices := mocks.NewMockInvoiceRepository(t)
	invoices.EXPECT().ListByShipmentIDs(mock.Anything, mock.Anything).
		Return(map[pulid.ID][]*invoice.Invoice{}, nil).Maybe()
	invoices.EXPECT().GetByBillingQueueItemID(mock.Anything, mock.Anything).
		Return(&invoice.Invoice{ID: pulid.MustNew("inv_"), Number: "INV-24101"}, nil)
	w.svc.invoiceRepo = invoices

	result, err := w.svc.ApproveIfReady(t.Context(), &services.ApproveIfReadyRequest{
		ItemID: w.item.ID, TenantInfo: w.tenant, AssignApprover: true,
	}, w.actor)

	require.NoError(t, err)
	assert.True(t, result.Approved)
	assert.Equal(t, billingqueue.StatusApproved, w.item.Status)
	require.NotNil(t, w.item.AssignedBillerID)
	assert.Equal(t, w.tenant.UserID, *w.item.AssignedBillerID)
	assert.Contains(t, w.review.eventTexts(), "Assigned Avery Lane as biller")
}

func TestApproveIfReadyDoesNotAssignAnItemThatAlsoNeedsSomethingElse(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	w.unassigned()

	result, err := w.svc.ApproveIfReady(t.Context(), &services.ApproveIfReadyRequest{
		ItemID: w.item.ID, TenantInfo: w.tenant, AssignApprover: true,
	}, w.actor)

	require.NoError(t, err)
	assert.False(t, result.Approved)
	assert.Nil(t, w.item.AssignedBillerID)
}

func TestApproveIfReadyLeavesAnUnassignedItemWhenNotAskedToAssign(t *testing.T) {
	t.Parallel()

	w := newReviewWorld(t, billingqueue.StatusInReview)
	w.unassigned()
	w.shp.AdditionalCharges = nil
	syncTotals(w.shp)
	w.item.AllocatedTotalAmount = w.shp.TotalChargeAmount.Decimal

	result, err := w.svc.ApproveIfReady(t.Context(), &services.ApproveIfReadyRequest{
		ItemID: w.item.ID, TenantInfo: w.tenant,
	}, w.actor)

	require.NoError(t, err)
	assert.False(t, result.Approved)
	assert.Equal(t, "Nobody is assigned as biller", result.Reason)
	assert.Nil(t, w.item.AssignedBillerID)
}
