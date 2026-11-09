package factoringoverviewservice

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringwatchtowerports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	tenant  = pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")}
	factor  = pulid.ID("fac_lonestar")
	chicago = mustZone("America/Chicago")
	// 10:14 AM on Friday, November 6, 2026 in Chicago.
	now = time.Date(2026, 11, 6, 10, 14, 0, 0, chicago).Unix()
)

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func at(month time.Month, day, hour int) int64 {
	return time.Date(2026, month, day, hour, 0, 0, 0, chicago).Unix()
}

type fakeRepo struct {
	read      *factoringports.OverviewRead
	reqs      []*factoringports.GetOverviewRequest
	customers []*factoringports.ListFactorCustomersRequest
	activity  []*factoringports.GetAgreementActivityRequest
}

func (f *fakeRepo) Get(
	_ context.Context,
	req *factoringports.GetOverviewRequest,
) (*factoringports.OverviewRead, error) {
	f.reqs = append(f.reqs, req)
	return f.read, nil
}

func (f *fakeRepo) AgreementActivity(
	_ context.Context,
	req *factoringports.GetAgreementActivityRequest,
) (*factoringports.AgreementActivity, error) {
	f.activity = append(f.activity, req)
	return &factoringports.AgreementActivity{Schedules: 3, FundedSchedules: 2}, nil
}

func (f *fakeRepo) List(
	_ context.Context,
	req *factoringports.ListFactorCustomersRequest,
) (*factoringports.FactorCustomerPage, error) {
	f.customers = append(f.customers, req)
	return &factoringports.FactorCustomerPage{}, nil
}

func newService(repo *fakeRepo) *Service {
	return New(Params{Repo: repo, Customers: repo, Now: func() int64 { return now }})
}

func agreement() *factoring.Agreement {
	return &factoring.Agreement{
		ID:                   pulid.ID("facagr_v3"),
		FactoringCompanyID:   factor,
		Status:               factoring.AgreementStatusActive,
		AdvanceRateBP:        9000,
		ReserveRateBP:        700,
		FundingDays:          1,
		ChargebackWindowDays: 90,
		Recourse:             factoring.RecourseFull,
	}
}

func busyRead() *factoringports.OverviewRead {
	return &factoringports.OverviewRead{
		Factor: factoringports.OverviewFactor{
			Name:     "Lonestar Freight Funding",
			Status:   domaintypes.StatusActive,
			Timezone: "America/Chicago",
		},
		Summary: &factoringports.Summary{
			Ready:            factoringports.VerdictTotals{Count: 9, AmountMinor: 2_148_000},
			ReadyUnsubmitted: factoringports.VerdictTotals{Count: 3, AmountMinor: 640_000},
			Submitted:        factoringports.VerdictTotals{Count: 12, AmountMinor: 3_490_000},
			Funded:           factoringports.VerdictTotals{Count: 31, AmountMinor: 68_870_000},
			ReserveHeld:      factoringports.VerdictTotals{Count: 31, AmountMinor: 4_821_000},
			ChargebacksDue:   factoringports.VerdictTotals{Count: 4, AmountMinor: 1_240_000},
		},
		Released:    factoringports.VerdictTotals{Count: 44, AmountMinor: 6_239_940},
		ChargedBack: factoringports.VerdictTotals{Count: 1, AmountMinor: 185_000},
		SoonestChargeback: &factoringwatchtowerports.ChargebackDue{
			InvoiceID:     pulid.ID("inv_20431"),
			InvoiceNumber: "INV-20431",
			CustomerName:  "Hartwell Foods",
			DeadlineAt:    at(time.November, 12, 0),
		},
		Failed: []*factoringwatchtowerports.FailedSubmission{
			{SubmissionID: "facsub_1043", Number: "S-1043", InvoiceCount: 9, FaceMinor: 2_148_000,
				LastError: "host key mismatch", FailedAt: at(time.November, 6, 10)},
		},
		Overdue: []*factoringwatchtowerports.UnfundedSubmission{
			{SubmissionID: "facsub_1039", Number: "S-1039", ExpectedAdvanceMinor: 900_000,
				DueAt: at(time.November, 3, 10)},
		},
		Remittances: []*factoringports.OpenRemittance{
			{RemittanceID: "facrem_1105", FileName: "Lonestar_Remit_2026-11-05.csv",
				Status: factoringremittance.RemittanceStatusParsed, LineCount: 38, ExceptionCount: 2,
				TotalCashMinor: 4_120_618, CurrencyCode: "USD", CreatedAt: at(time.November, 5, 9)},
		},
		RemittanceCount: 1,
		Notices: factoringports.NoticeCounts{
			Assigned:     14,
			NeedsNotice:  2,
			Awaiting:     3,
			Acknowledged: 9,
		},
		Agreement: agreement(),
	}
}

func kinds(items []Attention) []AttentionKind {
	out := make([]AttentionKind, len(items))
	for i, item := range items {
		out[i] = item.Kind
	}
	return out
}

func text(segments []Segment) string {
	var b strings.Builder
	for _, segment := range segments {
		b.WriteString(segment.Text)
	}
	return b.String()
}

func TestBuildAsksForTheWindowsTheReadingsDependOn(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{read: busyRead()}
	_, err := newService(repo).Build(t.Context(), &Request{
		TenantInfo: tenant, FactoringCompanyID: factor, AmountsIncluded: true,
	})
	require.NoError(t, err)

	require.Len(t, repo.reqs, 1)
	req := repo.reqs[0]
	assert.Equal(t, factor, req.FactoringCompanyID)
	assert.Equal(t, now, req.At)
	assert.Equal(t, now-86400, req.ReadyBefore, "ready for more than a day")
	assert.Equal(t, now+7*86400, req.ChargebackBefore, "a week of chargeback warning")
	assert.Equal(t, now-30*86400, req.MovedSince)
	assert.Equal(t, PaceSchedules, req.PaceLimit)
}

func TestBuildListsWhatNeedsSomeoneWorstFirst(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{read: busyRead()}
	got, err := newService(repo).Build(t.Context(), &Request{
		TenantInfo: tenant, FactoringCompanyID: factor, AmountsIncluded: true,
	})
	require.NoError(t, err)

	assert.Equal(t, []AttentionKind{
		AttentionScheduleFailed,
		AttentionNoDelivery,
		AttentionChargebacksDue,
		AttentionFundingOverdue,
		AttentionRemittanceWaiting,
		AttentionNoticesMissing,
		AttentionNoticesAwaiting,
		AttentionReadyWaiting,
	}, kinds(got.Attention))

	failed := got.Attention[0]
	assert.Equal(t, ToneDanger, failed.Tone)
	assert.Equal(t, "S-1043", failed.Number)
	assert.Equal(t, "host key mismatch", failed.Detail)
	assert.Equal(t, 9, failed.LineCount)
	assert.Equal(t, int64(2_148_000), failed.AmountMinor)

	chargebacks := got.Attention[2]
	assert.Equal(t, 4, chargebacks.Count)
	assert.Equal(t, int64(1_240_000), chargebacks.AmountMinor)
	assert.Equal(t, "INV-20431", chargebacks.InvoiceNumber)
	assert.Equal(t, "Hartwell Foods", chargebacks.CustomerName)
	require.NotNil(t, chargebacks.At)
	assert.Equal(t, at(time.November, 12, 0), *chargebacks.At)

	remittance := got.Attention[4]
	assert.Equal(t, pulid.ID("facrem_1105"), remittance.RemittanceID)
	assert.Equal(t, 2, remittance.ExceptionCount)
}

func TestBuildNamesADeliveryTestFailureOnlyWhenNoScheduleFailed(t *testing.T) {
	t.Parallel()

	tested := at(time.November, 6, 9)
	read := busyRead()
	read.Failed = nil
	read.Transport = &factoringsubmission.Transport{
		Method:        factoringsubmission.TransportMethodSftp,
		LastTestedAt:  &tested,
		LastTestError: "permission denied (publickey)",
	}

	got, err := newService(&fakeRepo{read: read}).Build(t.Context(), &Request{
		TenantInfo: tenant, FactoringCompanyID: factor, AmountsIncluded: true,
	})
	require.NoError(t, err)

	assert.Equal(t, AttentionDeliveryFailing, got.Attention[0].Kind)
	assert.Equal(t, "permission denied (publickey)", got.Attention[0].Detail)
	assert.True(t, got.Delivery.Failing())
	assert.False(t, got.Setup.Delivery, "a delivery whose last test failed is not set up")
	assert.NotContains(t, kinds(got.Attention), AttentionNoDelivery)
}

func TestBuildCountsFundingPaceInBusinessDaysOldestFirst(t *testing.T) {
	t.Parallel()

	read := busyRead()
	read.Funded = []*factoringports.FundedSubmission{
		{SubmissionID: "facsub_1041", Number: "S-1041",
			SentAt: at(time.November, 4, 10), FundedAt: at(time.November, 5, 15)},
		{SubmissionID: "facsub_1040", Number: "S-1040",
			SentAt: at(time.October, 30, 10), FundedAt: at(time.November, 3, 11)},
		{SubmissionID: "facsub_1039", Number: "S-1039",
			SentAt: at(time.October, 29, 10), FundedAt: at(time.October, 30, 9)},
	}

	got, err := newService(&fakeRepo{read: read}).Build(t.Context(), &Request{
		TenantInfo: tenant, FactoringCompanyID: factor, AmountsIncluded: true,
	})
	require.NoError(t, err)

	require.Len(t, got.Pace, 3)
	assert.Equal(t, []string{"S-1039", "S-1040", "S-1041"},
		[]string{got.Pace[0].Number, got.Pace[1].Number, got.Pace[2].Number})
	assert.Equal(t, []int{1, 2, 1},
		[]int{got.Pace[0].BusinessDays, got.Pace[1].BusinessDays, got.Pace[2].BusinessDays},
		"Friday to Tuesday is two business days, not four calendar days")
	assert.Equal(t, []bool{false, true, false},
		[]bool{got.Pace[0].Late, got.Pace[1].Late, got.Pace[2].Late})
	assert.Equal(t, 1, got.PaceLate)
	require.NotNil(t, got.PaceMedianDays)
	assert.Equal(t, 1, *got.PaceMedianDays)
	assert.Contains(t, text(got.Summary), "1 of the last 3 schedules funded late.")
}

func TestBuildWithholdsEveryAmountWithoutReserveAccess(t *testing.T) {
	t.Parallel()

	got, err := newService(&fakeRepo{read: busyRead()}).Build(t.Context(), &Request{
		TenantInfo: tenant, FactoringCompanyID: factor, AmountsIncluded: false,
	})
	require.NoError(t, err)

	assert.False(t, got.AmountsIncluded)
	assert.Equal(t, factoringports.VerdictTotals{Count: 31}, got.Funded, "counts stay, amounts go")
	assert.Equal(t, int64(0), got.ReserveHeld.AmountMinor)
	assert.Equal(t, int64(0), got.ChargebacksDue.AmountMinor)
	assert.Equal(t, int64(0), got.Released.AmountMinor)
	for _, item := range got.Attention {
		assert.Equal(t, int64(0), item.AmountMinor, item.Kind)
	}
	summary := text(got.Summary)
	assert.NotContains(t, summary, "$", "the summary says no amount the reader can't see")
	assert.Contains(t, summary, "Lonestar Freight Funding has 31 funded invoices outstanding.")
}

func TestSummaryLeadsWithTheReserveAndOpensTheTabsItNames(t *testing.T) {
	t.Parallel()

	got, err := newService(&fakeRepo{read: busyRead()}).Build(t.Context(), &Request{
		TenantInfo: tenant, FactoringCompanyID: factor, AmountsIncluded: true,
	})
	require.NoError(t, err)

	summary := text(got.Summary)
	assert.True(t, strings.HasPrefix(summary,
		"Lonestar Freight Funding holds $48,210 of reserve on 31 funded invoices."), summary)
	assert.Contains(t, summary, "Schedule S-1043 didn't send, so its 9 invoices wait to go again.")
	assert.Contains(
		t,
		summary,
		"4 recourse invoices ($12,400) reach the chargeback window by Nov 12, 2026.",
	)
	assert.Contains(t, summary, "Schedule S-1039 is past its funding date.")
	assert.Contains(t, summary, "2 customers have no notice in force.")

	targets := map[Target]bool{}
	for _, segment := range got.Summary {
		if segment.Target != "" {
			targets[segment.Target] = true
		}
	}
	assert.Equal(t, map[Target]bool{
		TargetDelivery:  true,
		TargetLedger:    true,
		TargetCustomers: true,
	}, targets)
}

func TestANewFactorIsWalkedThroughItsSetup(t *testing.T) {
	t.Parallel()

	read := &factoringports.OverviewRead{
		Factor:  factoringports.OverviewFactor{Name: "Bluewater Receivables Finance"},
		Summary: &factoringports.Summary{},
	}
	got, err := newService(&fakeRepo{read: read}).Build(t.Context(), &Request{
		TenantInfo: tenant, FactoringCompanyID: factor, AmountsIncluded: true,
	})
	require.NoError(t, err)

	assert.Equal(t, Setup{}, got.Setup)
	assert.Equal(t, 0, got.Setup.Done())
	assert.Equal(
		t,
		[]AttentionKind{AttentionNoDelivery, AttentionNoAgreement},
		kinds(got.Attention),
	)
	assert.Equal(
		t,
		"Bluewater Receivables Finance can't take invoices yet: it needs an agreement and a way to send schedules.",
		text(got.Summary),
	)
	assert.Equal(t, "USD", got.CurrencyCode)
}

func TestCustomersRefusesAStageItDoesNotKnow(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	_, err := newService(repo).Customers(t.Context(), &CustomersRequest{
		Filter:             &pagination.QueryOptions{TenantInfo: tenant},
		FactoringCompanyID: factor,
		Stage:              "Signed",
	})
	var invalid *errortypes.Error
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "stage", invalid.Field)
	assert.Empty(t, repo.customers)

	filter := &pagination.QueryOptions{TenantInfo: tenant, Query: "acme"}
	_, err = newService(repo).Customers(t.Context(), &CustomersRequest{
		Filter:             filter,
		Cursor:             pagination.CursorInfo{After: "cursor-1", IncludeTotalCount: true},
		FactoringCompanyID: factor,
		Stage:              factoringports.CustomerNoticeNeeded,
	})
	require.NoError(t, err)
	require.Len(t, repo.customers, 1)
	assert.Equal(t, now, repo.customers[0].At, "a notice's expiry is judged against now")
	assert.Same(t, filter, repo.customers[0].Filter, "the table's search and filters reach the read")
	assert.Equal(t, "cursor-1", repo.customers[0].Cursor.After)
	assert.True(t, repo.customers[0].Cursor.IncludeTotalCount)
}

func TestAgreementActivityRefusesAMissingAgreement(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	_, err := newService(repo).AgreementActivity(t.Context(), &AgreementActivityRequest{
		TenantInfo: tenant,
	})
	var invalid *errortypes.Error
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "agreementId", invalid.Field)
	assert.Empty(t, repo.activity, "nothing is read without an agreement")
}

func TestAgreementActivityReadsTheAgreementInTheCallersTenant(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	got, err := newService(repo).AgreementActivity(t.Context(), &AgreementActivityRequest{
		TenantInfo: tenant, AgreementID: pulid.ID("fag_1"),
	})
	require.NoError(t, err)
	require.Len(t, repo.activity, 1)
	assert.Equal(t, tenant, repo.activity[0].TenantInfo)
	assert.Equal(t, pulid.ID("fag_1"), repo.activity[0].AgreementID)
	assert.Equal(t, 3, got.Schedules)
	assert.Equal(t, 2, got.FundedSchedules)
}
