package latechargeservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/latecharge"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
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

const day = int64(86400)

var (
	rate    = decimal.RequireFromString("1.5")
	dueDate = int64(1_700_000_000)
)

func candidate(customerID pulid.ID, name string, number string, openMinor int64, assessed ...int) *repositories.LateChargeCandidate {
	return &repositories.LateChargeCandidate{
		CustomerID:       customerID,
		CustomerName:     name,
		InvoiceID:        pulid.MustNew("inv_"),
		InvoiceNumber:    number,
		CurrencyCode:     "USD",
		DueDate:          dueDate,
		GracePeriodDays:  5,
		RatePercent:      rate,
		OpenBalanceMinor: openMinor,
		AssessedPeriods:  assessed,
	}
}

func TestBuildPlansGroupsByCustomerAndChargesEachPendingPeriod(t *testing.T) {
	t.Parallel()

	intel := pulid.MustNew("cus_")
	amd := pulid.MustNew("cus_")
	// Grace ends at due + 5 days; as-of is 61 days after that, so periods 1-3 have begun.
	asOf := dueDate + 5*day + 61*day
	actor := testutil.NewSessionActor(pulid.MustNew("usr_"), pulid.MustNew("org_"), pulid.MustNew("bu_"))

	plans := buildPlans([]*repositories.LateChargeCandidate{
		candidate(intel, "Intel", "INV-1", 120_000, 1),
		candidate(amd, "AMD", "INV-2", 50_000),
		candidate(intel, "Intel", "INV-3", 10_000, 1, 2, 3),
	}, asOf, 0, actor)

	require.Len(t, plans, 2)
	intelPlan, amdPlan := plans[0], plans[1]
	assert.Equal(t, "Intel", intelPlan.result.CustomerName)
	assert.Equal(t, "AMD", amdPlan.result.CustomerName)

	// INV-1 owes periods 2 and 3 at 1.5% of 1,200.00 each; INV-3 is fully assessed.
	require.Len(t, intelPlan.result.Lines, 2)
	assert.Equal(t, []int{2, 3}, []int{intelPlan.result.Lines[0].PeriodIndex, intelPlan.result.Lines[1].PeriodIndex})
	assert.Equal(t, int64(1800), intelPlan.result.Lines[0].ChargeMinor)
	assert.Equal(t, int64(3600), intelPlan.result.TotalChargeMinor)
	assert.False(t, intelPlan.result.Skipped)
	require.Len(t, intelPlan.assessments, 2)
	assert.Equal(t, actor.UserID, intelPlan.assessments[0].CreatedByID)
	assert.Equal(t, asOf, intelPlan.assessments[0].AsOfDate)
	start, end := latecharge.PeriodBounds(dueDate+5*day, 2)
	assert.Equal(t, start, intelPlan.assessments[0].PeriodStart)
	assert.Equal(t, end, intelPlan.assessments[0].PeriodEnd)

	require.Len(t, amdPlan.result.Lines, 3)
	assert.Equal(t, int64(750*3), amdPlan.result.TotalChargeMinor)
}

func TestBuildPlansSkipsFullyAssessedAndBelowMinimum(t *testing.T) {
	t.Parallel()

	asOf := dueDate + 5*day + 1
	actor := testutil.NewSessionActor(pulid.MustNew("usr_"), pulid.MustNew("org_"), pulid.MustNew("bu_"))

	plans := buildPlans([]*repositories.LateChargeCandidate{
		candidate(pulid.MustNew("cus_"), "Done", "INV-1", 120_000, 1),
		candidate(pulid.MustNew("cus_"), "Tiny", "INV-2", 1_000),
		candidate(pulid.MustNew("cus_"), "Big", "INV-3", 200_000),
	}, asOf, 500, actor)

	require.Len(t, plans, 3)
	assert.True(t, plans[0].result.Skipped)
	assert.Contains(t, plans[0].result.SkipReason, "already been assessed")
	assert.True(t, plans[1].result.Skipped, "15 minor units is below the 5.00 minimum")
	assert.Contains(t, plans[1].result.SkipReason, "below the organization minimum")
	assert.False(t, plans[2].result.Skipped)
	assert.Equal(t, int64(3000), plans[2].result.TotalChargeMinor)
}

type lateChargeFixture struct {
	tenantInfo     pagination.TenantInfo
	actor          *servicesports.RequestActor
	repo           *mocks.MockLateChargeRepository
	billingRepo    *mocks.MockBillingControlRepository
	invoiceService *mocks.MockInvoiceService
	svc            *Service
}

func newLateChargeFixture(t *testing.T, control *tenant.BillingControl) *lateChargeFixture {
	t.Helper()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	f := &lateChargeFixture{
		tenantInfo:     pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		actor:          testutil.NewSessionActor(userID, orgID, buID),
		repo:           mocks.NewMockLateChargeRepository(t),
		billingRepo:    mocks.NewMockBillingControlRepository(t),
		invoiceService: mocks.NewMockInvoiceService(t),
	}
	f.billingRepo.EXPECT().GetByOrgID(mock.Anything, orgID).Return(control, nil).Once()
	f.svc = &Service{
		l:              zap.NewNop(),
		repo:           f.repo,
		billingRepo:    f.billingRepo,
		invoiceService: f.invoiceService,
		auditService:   &mocks.NoopAuditService{},
	}

	return f
}

func (f *lateChargeFixture) expectCandidates(asOf int64, candidates ...*repositories.LateChargeCandidate) {
	f.repo.EXPECT().
		ListCandidates(mock.Anything, &repositories.ListLateChargeCandidatesRequest{TenantInfo: f.tenantInfo, AsOfDate: asOf}).
		Return(candidates, nil).
		Once()
}

func TestAssessPreviewWritesNothing(t *testing.T) {
	t.Parallel()

	f := newLateChargeFixture(t, &tenant.BillingControl{LateChargeAssessmentMode: tenant.LateChargeAssessmentModeDisabled})
	asOf := dueDate + 5*day + 1
	f.expectCandidates(asOf, candidate(pulid.MustNew("cus_"), "AMD", "INV-2", 50_000))

	result, err := f.svc.Assess(t.Context(), &servicesports.LateChargeAssessmentRequest{
		TenantInfo: f.tenantInfo, AsOfDate: asOf, Preview: true,
	}, f.actor)

	require.NoError(t, err)
	assert.True(t, result.Preview)
	assert.Equal(t, tenant.LateChargeAssessmentModeDisabled, result.Mode)
	assert.Equal(t, 0, result.MemosCreated)
	assert.Equal(t, int64(750), result.TotalChargeMinor)
	require.Len(t, result.Customers, 1)
	assert.False(t, result.Customers[0].Skipped)
	f.repo.AssertNotCalled(t, "InsertAssessments", mock.Anything, mock.Anything)
	f.invoiceService.AssertNotCalled(t, "CreateMemo", mock.Anything, mock.Anything, mock.Anything)
}

func TestAssessRefusesARealRunWhenDisabled(t *testing.T) {
	t.Parallel()

	f := newLateChargeFixture(t, &tenant.BillingControl{LateChargeAssessmentMode: tenant.LateChargeAssessmentModeDisabled})

	_, err := f.svc.Assess(t.Context(), &servicesports.LateChargeAssessmentRequest{TenantInfo: f.tenantInfo, AsOfDate: dueDate}, f.actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
	f.repo.AssertNotCalled(t, "ListCandidates", mock.Anything, mock.Anything)
}

func TestAssessRaisesOneLateChargeMemoPerCustomer(t *testing.T) {
	t.Parallel()

	f := newLateChargeFixture(t, &tenant.BillingControl{
		LateChargeAssessmentMode: tenant.LateChargeAssessmentModeAutomatic,
		InvoicePostingMode:       tenant.InvoicePostingModeAutomaticWhenNoBlockingExceptions,
	})
	asOf := dueDate + 5*day + 31*day
	amd := candidate(pulid.MustNew("cus_"), "AMD", "INV-2", 50_000)
	f.expectCandidates(asOf, amd)

	f.repo.EXPECT().
		InsertAssessments(mock.Anything, mock.MatchedBy(func(rows []*latecharge.LateChargeAssessment) bool {
			return len(rows) == 2 &&
				rows[0].OrganizationID == f.tenantInfo.OrgID &&
				rows[0].RunKey != "" && rows[0].RunKey == rows[1].RunKey &&
				rows[0].DebitMemoInvoiceID.IsNotNil() &&
				rows[0].DebitMemoInvoiceID == rows[1].DebitMemoInvoiceID
		})).
		RunAndReturn(func(_ context.Context, rows []*latecharge.LateChargeAssessment) ([]*latecharge.LateChargeAssessment, error) {
			// Another run already charged period 1: only period 2 is this run's.
			return []*latecharge.LateChargeAssessment{rows[1]}, nil
		}).
		Once()

	memoID := pulid.Nil
	memoLine := &invoice.InvoiceLine{ID: pulid.MustNew("invl_")}
	f.invoiceService.EXPECT().
		CreateMemo(mock.Anything, mock.MatchedBy(func(req *servicesports.CreateMemoRequest) bool {
			memoID = req.ID
			return req.CustomerID == amd.CustomerID &&
				req.BillType == billingqueue.BillTypeDebitMemo &&
				req.MemoKind == invoice.MemoKindLateCharge &&
				req.AutoPost &&
				req.InvoiceDate == asOf &&
				len(req.Lines) == 1 &&
				req.Lines[0].Amount.Equal(decimal.RequireFromString("7.5")) &&
				req.Lines[0].Description != ""
		}), f.actor).
		RunAndReturn(func(_ context.Context, req *servicesports.CreateMemoRequest, _ *servicesports.RequestActor) (*invoice.Invoice, error) {
			return &invoice.Invoice{ID: req.ID, Number: "DM-5", Status: invoice.StatusPosted, Lines: []*invoice.InvoiceLine{memoLine}}, nil
		}).
		Once()
	f.repo.EXPECT().
		SetDebitMemoLines(mock.Anything, mock.MatchedBy(func(rows []*latecharge.LateChargeAssessment) bool {
			return len(rows) == 1 && rows[0].DebitMemoLineID == memoLine.ID && rows[0].DebitMemoInvoiceID == memoID
		})).
		Return(nil).
		Once()

	result, err := f.svc.Assess(t.Context(), &servicesports.LateChargeAssessmentRequest{TenantInfo: f.tenantInfo, AsOfDate: asOf}, f.actor)

	require.NoError(t, err)
	assert.Equal(t, 1, result.MemosCreated)
	assert.Equal(t, 1, result.MemosPosted)
	assert.Equal(t, int64(750), result.TotalChargeMinor, "only the period this run inserted is billed")
	require.Len(t, result.Customers, 1)
	assert.Equal(t, "DM-5", result.Customers[0].DebitMemoNumber)
	assert.True(t, result.Customers[0].Posted)
	require.Len(t, result.Customers[0].Lines, 1)
	assert.Equal(t, 2, result.Customers[0].Lines[0].PeriodIndex)
}

func TestAssessDoesNotAutoPostUnderManualReview(t *testing.T) {
	t.Parallel()

	f := newLateChargeFixture(t, &tenant.BillingControl{
		LateChargeAssessmentMode: tenant.LateChargeAssessmentModeAutomatic,
		InvoicePostingMode:       tenant.InvoicePostingModeManualReviewRequired,
	})
	asOf := dueDate + 5*day + 1
	f.expectCandidates(asOf, candidate(pulid.MustNew("cus_"), "AMD", "INV-2", 50_000))
	f.repo.EXPECT().InsertAssessments(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, rows []*latecharge.LateChargeAssessment) ([]*latecharge.LateChargeAssessment, error) {
			return rows, nil
		}).
		Once()
	f.invoiceService.EXPECT().
		CreateMemo(mock.Anything, mock.MatchedBy(func(req *servicesports.CreateMemoRequest) bool { return !req.AutoPost }), f.actor).
		RunAndReturn(func(_ context.Context, req *servicesports.CreateMemoRequest, _ *servicesports.RequestActor) (*invoice.Invoice, error) {
			return &invoice.Invoice{ID: req.ID, Number: "DM-6", Status: invoice.StatusDraft, Lines: []*invoice.InvoiceLine{{ID: pulid.MustNew("invl_")}}}, nil
		}).
		Once()
	f.repo.EXPECT().SetDebitMemoLines(mock.Anything, mock.Anything).Return(nil).Once()

	result, err := f.svc.Assess(t.Context(), &servicesports.LateChargeAssessmentRequest{TenantInfo: f.tenantInfo, AsOfDate: asOf}, f.actor)

	require.NoError(t, err)
	assert.Equal(t, 1, result.MemosCreated)
	assert.Equal(t, 0, result.MemosPosted)
}

func TestAssessRollsBackAssessmentsWhenTheMemoFails(t *testing.T) {
	t.Parallel()

	f := newLateChargeFixture(t, &tenant.BillingControl{LateChargeAssessmentMode: tenant.LateChargeAssessmentModeAutomatic})
	asOf := dueDate + 5*day + 1
	f.expectCandidates(asOf, candidate(pulid.MustNew("cus_"), "AMD", "INV-2", 50_000))
	var runKey string
	f.repo.EXPECT().InsertAssessments(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, rows []*latecharge.LateChargeAssessment) ([]*latecharge.LateChargeAssessment, error) {
			runKey = rows[0].RunKey
			return rows, nil
		}).
		Once()
	f.invoiceService.EXPECT().CreateMemo(mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("sequence exhausted")).
		Once()
	f.repo.EXPECT().
		DeleteByRunKey(mock.Anything, f.tenantInfo, mock.MatchedBy(func(key string) bool { return key == runKey })).
		Return(1, nil).
		Once()

	result, err := f.svc.Assess(t.Context(), &servicesports.LateChargeAssessmentRequest{TenantInfo: f.tenantInfo, AsOfDate: asOf}, f.actor)

	require.NoError(t, err, "one customer failing does not fail the run")
	assert.Equal(t, 0, result.MemosCreated)
	assert.Equal(t, 1, result.CustomersSkipped)
	require.Len(t, result.Customers, 1)
	assert.True(t, result.Customers[0].Skipped)
	assert.Contains(t, result.Customers[0].SkipReason, "sequence exhausted")
}

func TestAssessSkipsACustomerAnotherRunGotToFirst(t *testing.T) {
	t.Parallel()

	f := newLateChargeFixture(t, &tenant.BillingControl{LateChargeAssessmentMode: tenant.LateChargeAssessmentModeAutomatic})
	asOf := dueDate + 5*day + 1
	f.expectCandidates(asOf, candidate(pulid.MustNew("cus_"), "AMD", "INV-2", 50_000))
	f.repo.EXPECT().InsertAssessments(mock.Anything, mock.Anything).
		Return([]*latecharge.LateChargeAssessment{}, nil).
		Once()

	result, err := f.svc.Assess(t.Context(), &servicesports.LateChargeAssessmentRequest{TenantInfo: f.tenantInfo, AsOfDate: asOf}, f.actor)

	require.NoError(t, err)
	assert.Equal(t, 0, result.MemosCreated)
	assert.Equal(t, 1, result.CustomersSkipped)
	assert.Equal(t, int64(0), result.TotalChargeMinor)
	assert.Contains(t, result.Customers[0].SkipReason, "Another run")
	f.invoiceService.AssertNotCalled(t, "CreateMemo", mock.Anything, mock.Anything, mock.Anything)
}
