package invoicerunservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type runPlanFixture struct {
	tenantInfo pagination.TenantInfo
	actor      *servicesports.RequestActor
	run        *invoicerun.InvoiceRun
	first      *invoicerun.InvoiceRunGroup
	second     *invoicerun.InvoiceRunGroup
	svc        *Service
}

type readOnlyRunRepo struct {
	repositories.InvoiceRunRepository
	t          *testing.T
	run        *invoicerun.InvoiceRun
	tenantInfo pagination.TenantInfo
}

func (r *readOnlyRunRepo) GetByID(
	_ context.Context,
	req repositories.GetInvoiceRunByIDRequest,
) (*invoicerun.InvoiceRun, error) {
	require.Equal(r.t, r.run.ID, req.ID)
	require.Equal(r.t, r.tenantInfo, req.TenantInfo)
	require.True(r.t, req.IncludeGroups && req.IncludeItems)

	return r.run, nil
}

func runItem(groupID pulid.ID, amount string) *invoicerun.InvoiceRunGroupItem {
	return &invoicerun.InvoiceRunGroupItem{
		ID:                 pulid.MustNew("invrgi_"),
		GroupID:            groupID,
		BillingQueueItemID: pulid.MustNew("bqi_"),
		ShipmentID:         pulid.MustNew("shp_"),
		Amount:             decimal.RequireFromString(amount),
	}
}

func newRunPlanFixture(t *testing.T) *runPlanFixture {
	t.Helper()

	f := &runPlanFixture{
		tenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
	f.actor = testutil.NewSessionActor(pulid.MustNew("usr_"), f.tenantInfo.OrgID, f.tenantInfo.BuID)
	customerID := pulid.MustNew("cus_")
	f.first = &invoicerun.InvoiceRunGroup{
		ID:           pulid.MustNew("invrg_"),
		CustomerID:   customerID,
		GroupLabel:   "PO 1001",
		CurrencyCode: "USD",
		Status:       invoicerun.GroupStatusPending,
	}
	f.first.Items = []*invoicerun.InvoiceRunGroupItem{
		runItem(f.first.ID, "400.00"),
		runItem(f.first.ID, "600.00"),
	}
	f.second = &invoicerun.InvoiceRunGroup{
		ID:           pulid.MustNew("invrg_"),
		CustomerID:   customerID,
		GroupLabel:   "PO 1002",
		CurrencyCode: "USD",
		Status:       invoicerun.GroupStatusPending,
	}
	f.second.Items = []*invoicerun.InvoiceRunGroupItem{runItem(f.second.ID, "250.00")}
	f.run = &invoicerun.InvoiceRun{
		ID:             pulid.MustNew("invrun_"),
		OrganizationID: f.tenantInfo.OrgID,
		BusinessUnitID: f.tenantInfo.BuID,
		Number:         "RUN-7",
		Status:         invoicerun.StatusReady,
		Groups:         []*invoicerun.InvoiceRunGroup{f.first, f.second},
	}
	f.run.SyncTotals()
	f.svc = &Service{
		l:         zap.NewNop(),
		repo:      &readOnlyRunRepo{t: t, run: f.run, tenantInfo: f.tenantInfo},
		validator: NewValidator(ValidatorParams{}),
	}

	return f
}

func TestPreviewMembershipMovesAndExcludesWithoutWriting(t *testing.T) {
	t.Parallel()

	f := newRunPlanFixture(t)
	excluded := f.first.Items[0]
	moved := f.first.Items[1]

	preview, err := f.svc.PreviewMembership(t.Context(), &servicesports.AdjustInvoiceRunMembershipRequest{
		TenantInfo: f.tenantInfo,
		RunID:      f.run.ID,
		Exclude:    []servicesports.ItemExclusion{{ItemID: excluded.ID, Reason: "Short on the POD"}},
		Moves:      []servicesports.ItemMove{{ItemID: moved.ID, TargetGroupID: f.second.ID}},
	})
	require.NoError(t, err)

	assert.True(t, decimal.RequireFromString("1250").Equal(preview.Before.TotalAmount))
	assert.False(t, excluded.Excluded, "the loaded run is left as it was")
	require.Len(t, preview.After.Groups, 2)
	assert.Equal(t, 0, preview.After.Groups[0].ItemCount)
	assert.Equal(t, 2, preview.After.Groups[1].ItemCount)
	assert.True(t, decimal.RequireFromString("850").Equal(preview.After.Groups[1].TotalAmount))
	assert.True(t, decimal.RequireFromString("850").Equal(preview.After.TotalAmount))
	assert.Equal(t, 1, preview.After.ExcludedCount)
}

func TestPreviewMembershipRefusesAnUnknownItem(t *testing.T) {
	t.Parallel()

	f := newRunPlanFixture(t)
	_, err := f.svc.PreviewMembership(t.Context(), &servicesports.AdjustInvoiceRunMembershipRequest{
		TenantInfo: f.tenantInfo,
		RunID:      f.run.ID,
		Include:    []pulid.ID{pulid.MustNew("invrgi_")},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not on this run")
}

func TestPreviewCancelRefusesACommittedRun(t *testing.T) {
	t.Parallel()

	f := newRunPlanFixture(t)
	f.run.Status = invoicerun.StatusCommitted
	_, err := f.svc.PreviewCancel(t.Context(), &servicesports.CancelInvoiceRunRequest{
		TenantInfo: f.tenantInfo, RunID: f.run.ID, Reason: "Wrong period",
	}, f.actor)
	require.Error(t, err)

	ready := newRunPlanFixture(t)
	preview, err := ready.svc.PreviewCancel(t.Context(), &servicesports.CancelInvoiceRunRequest{
		TenantInfo: ready.tenantInfo, RunID: ready.run.ID, Reason: "Wrong period",
	}, ready.actor)
	require.NoError(t, err)
	assert.Equal(t, invoicerun.StatusCanceled, preview.After.Status)
	assert.Equal(t, "Wrong period", preview.After.FailureReason)
	assert.Equal(t, invoicerun.StatusReady, preview.Before.Status)
}

func TestPreviewCommitSaysWhichGroupsBillAndWhichSkip(t *testing.T) {
	t.Parallel()

	f := newRunPlanFixture(t)
	f.second.MinimumAmount = decimal.NewNullDecimal(decimal.RequireFromString("500"))
	billingQueueRepo := mocks.NewMockBillingQueueRepository(t)
	for _, item := range f.first.Items {
		billingQueueRepo.EXPECT().
			GetByID(mock.Anything, &repositories.GetBillingQueueItemByIDRequest{
				ItemID:     item.BillingQueueItemID,
				TenantInfo: f.tenantInfo,
			}).
			Return(&billingqueue.BillingQueueItem{
				ID:               item.BillingQueueItemID,
				BillToCustomerID: f.first.CustomerID,
				Status:           billingqueue.StatusApproved,
			}, nil).
			Once()
	}
	shipmentRepo := mocks.NewMockShipmentRepository(t)
	shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{ID: pulid.MustNew("shp_")}, nil).
		Times(2)
	f.svc.billingQueueRepo = billingQueueRepo
	f.svc.shipmentRepo = shipmentRepo

	plan, err := f.svc.PreviewCommit(t.Context(), &servicesports.CommitInvoiceRunRequest{
		TenantInfo: f.tenantInfo,
		RunID:      f.run.ID,
	})
	require.NoError(t, err)
	require.Len(t, plan.Groups, 2)
	assert.Equal(t, servicesports.InvoiceRunGroupBills, plan.Groups[0].Outcome)
	assert.True(t, decimal.RequireFromString("1000").Equal(plan.Groups[0].Total))
	assert.Equal(t, servicesports.InvoiceRunGroupSkips, plan.Groups[1].Outcome)
	assert.Contains(t, plan.Groups[1].Reason, "minimum")
}

func TestPreviewCommitOfACommittedRunChangesNothing(t *testing.T) {
	t.Parallel()

	f := newRunPlanFixture(t)
	f.run.Status = invoicerun.StatusCommitted
	plan, err := f.svc.PreviewCommit(t.Context(), &servicesports.CommitInvoiceRunRequest{
		TenantInfo: f.tenantInfo,
		RunID:      f.run.ID,
	})
	require.NoError(t, err)
	assert.True(t, plan.AlreadyCommitted)
	assert.Empty(t, plan.Groups)
}

func TestPreviewBuildGroupsTheCandidatesWithoutNumberingOrSaving(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	svc := serviceWithCandidates(t, []*repositories.ConsolidationCandidate{
		buildCandidate(customerID, "PRO-1", "100.00", nil),
		buildCandidate(customerID, "PRO-2", "150.00", nil),
	})
	svc.repo = &readOnlyRunRepo{t: t}
	svc.validator = NewValidator(ValidatorParams{})
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	run, err := svc.PreviewBuild(t.Context(), &servicesports.PreviewInvoiceRunRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: []pulid.ID{customerID},
		PeriodStart: 1_700_000_000,
		PeriodEnd:   1_702_592_000,
	}, testutil.NewSessionActor(pulid.MustNew("usr_"), tenantInfo.OrgID, tenantInfo.BuID))
	require.NoError(t, err)

	assert.Equal(t, unnumberedRun, run.Number)
	assert.Equal(t, invoicerun.StatusReady, run.Status)
	require.Len(t, run.Groups, 1)
	assert.Equal(t, 2, run.ItemCount)
	assert.True(t, decimal.RequireFromString("250").Equal(run.TotalAmount))
}

func TestPreviewBuildRefusesARunWithNoCustomers(t *testing.T) {
	t.Parallel()

	svc := &Service{validator: NewValidator(ValidatorParams{})}
	_, err := svc.PreviewBuild(t.Context(), &servicesports.PreviewInvoiceRunRequest{
		PeriodStart: 1, PeriodEnd: 2,
	}, testutil.NewSessionActor(pulid.MustNew("usr_"), pulid.MustNew("org_"), pulid.MustNew("bu_")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Select at least one customer")
}
