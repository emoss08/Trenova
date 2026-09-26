package journalrepairservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pagedRepo struct {
	payments  [][]*repositories.DriverPaymentRepair
	memoErr   error
	requests  []repositories.ListJournalRepairRequest
	memoCalls int
}

func (r *pagedRepo) ListUnjournaledAdjustmentMemos(
	context.Context,
	*repositories.ListJournalRepairRequest,
) ([]*repositories.AdjustmentMemoRepair, error) {
	r.memoCalls++
	return nil, r.memoErr
}

func (r *pagedRepo) ListUnjournaledDriverPayments(
	_ context.Context,
	req *repositories.ListJournalRepairRequest,
) ([]*repositories.DriverPaymentRepair, error) {
	r.requests = append(r.requests, *req)
	if len(r.payments) == 0 {
		return nil, nil
	}
	page := r.payments[0]
	r.payments = r.payments[1:]
	return page, nil
}

func (r *pagedRepo) SetDriverSettlementPaidBatch(
	context.Context,
	*repositories.SetDriverSettlementPaidBatchParams,
) error {
	return errors.New("dry run must not stamp a settlement")
}

type controlReader struct {
	control *tenant.AccountingControl
	calls   int
}

func (c *controlReader) GetByOrgID(context.Context, pulid.ID) (*tenant.AccountingControl, error) {
	c.calls++
	return c.control, nil
}

func paidWithoutSnapshot(orgID pulid.ID, number string) *repositories.DriverPaymentRepair {
	paidAt := int64(1_790_000_000)
	return &repositories.DriverPaymentRepair{Settlement: &driversettlement.Settlement{
		ID:               pulid.MustNew("dstl_"),
		OrganizationID:   orgID,
		BusinessUnitID:   pulid.MustNew("bu_"),
		SettlementNumber: number,
		Status:           driversettlement.StatusPaid,
		NetPayMinor:      50_000,
		PaidAt:           &paidAt,
	}}
}

func TestRepairRequiresAnActor(t *testing.T) {
	repo := &pagedRepo{}
	svc := New(&Deps{Repo: repo, Now: func() int64 { return 1 }})

	_, err := svc.Repair(t.Context(), &Request{DryRun: true})

	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Zero(t, repo.memoCalls)
}

func TestRepairAbortsOnAnUnexpectedRepositoryError(t *testing.T) {
	repo := &pagedRepo{memoErr: errors.New("connection reset")}
	svc := New(&Deps{Repo: repo, Now: func() int64 { return 1 }})

	_, err := svc.Repair(t.Context(), &Request{ActorID: pulid.MustNew("usr_"), DryRun: true})

	require.ErrorContains(t, err, "connection reset")
}

func TestRepairSkipsARecordItCannotJournalAndPagesOn(t *testing.T) {
	orgID := pulid.MustNew("org_")
	first := paidWithoutSnapshot(orgID, "STL-1")
	second := paidWithoutSnapshot(orgID, "STL-2")
	repo := &pagedRepo{payments: [][]*repositories.DriverPaymentRepair{{first}, {second}}}
	controls := &controlReader{control: &tenant.AccountingControl{
		OrganizationID:       orgID,
		DefaultCashAccountID: pulid.MustNew("gla_"),
	}}
	svc := New(&Deps{Repo: repo, Controls: controls, Now: func() int64 { return 1 }})

	report, err := svc.Repair(t.Context(), &Request{
		OrganizationID: orgID,
		ActorID:        pulid.MustNew("usr_"),
		DryRun:         true,
		BatchSize:      1,
	})

	require.NoError(t, err)
	assert.Zero(t, report.PaymentsJournaled)
	require.Len(t, report.Skipped, 2)
	assert.Equal(t, "STL-1", report.Skipped[0].Number)
	assert.Equal(t, KindDriverSettlement, report.Skipped[0].Kind)
	assert.Contains(t, report.Skipped[0].Reason, "no posted payable account")
	assert.Equal(t, "STL-2", report.Skipped[1].Number)

	require.Len(t, repo.requests, 3)
	assert.True(t, repo.requests[0].AfterID.IsNil())
	assert.Equal(t, first.Settlement.ID, repo.requests[1].AfterID)
	assert.Equal(t, second.Settlement.ID, repo.requests[2].AfterID)
	assert.Equal(t, orgID, repo.requests[1].OrganizationID)
	assert.Equal(t, 1, repo.requests[1].Limit)
	assert.Equal(t, 1, controls.calls)
}

func TestRepairLinksAnExistingPaymentJournalWithoutWritingInDryRun(t *testing.T) {
	orgID := pulid.MustNew("org_")
	linked := paidWithoutSnapshot(orgID, "STL-9")
	linked.JournalBatchID = pulid.MustNew("jb_")
	repo := &pagedRepo{payments: [][]*repositories.DriverPaymentRepair{{linked}}}
	svc := New(&Deps{Repo: repo, Now: func() int64 { return 1 }})

	report, err := svc.Repair(t.Context(), &Request{ActorID: pulid.MustNew("usr_"), DryRun: true})

	require.NoError(t, err)
	assert.Equal(t, 1, report.PaymentsLinked)
	assert.Zero(t, report.PaymentsJournaled)
	assert.Empty(t, report.Skipped)
}
