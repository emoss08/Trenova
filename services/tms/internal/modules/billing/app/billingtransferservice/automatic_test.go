package billingtransferservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingtransferports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentserviceportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRunRepo struct {
	billingtransferports.BillingTransferRunRepository
	lastQueuedAt int64
	active       *billingtransfer.BillingTransferRun
	created      []*billingtransfer.BillingTransferRun
	seeded       []pulid.ID
}

func (f *fakeRunRepo) LastQueuedAt(
	context.Context,
	*billingtransferports.LastBillingTransferRunQueuedAtRequest,
) (int64, error) {
	return f.lastQueuedAt, nil
}

func (f *fakeRunRepo) GetActive(
	context.Context,
	*billingtransferports.GetActiveBillingTransferRunRequest,
) (*billingtransfer.BillingTransferRun, error) {
	return f.active, nil
}

func (f *fakeRunRepo) Create(
	_ context.Context,
	run *billingtransfer.BillingTransferRun,
) (*billingtransfer.BillingTransferRun, error) {
	run.ID = pulid.MustNew("btr_")
	f.created = append(f.created, run)
	return run, nil
}

func (f *fakeRunRepo) SeedItems(
	_ context.Context,
	req *billingtransferports.SeedBillingTransferRunItemsRequest,
) (int, error) {
	f.seeded = append(f.seeded, req.ShipmentIDs...)
	return len(req.ShipmentIDs), nil
}

type automaticFixture struct {
	repo      *fakeRunRepo
	shipments *shipmentserviceportsmock.MockShipmentBillingTransferService
	workflows *kernelportsmock.MockWorkflowStarter
	svc       *Service
	control   *billingcontrol.BillingControl
	system    pulid.ID
}

func newAutomaticFixture(t *testing.T) *automaticFixture {
	t.Helper()

	f := &automaticFixture{
		repo:      &fakeRunRepo{},
		shipments: shipmentserviceportsmock.NewMockShipmentBillingTransferService(t),
		workflows: kernelportsmock.NewMockWorkflowStarter(t),
		system:    pulid.MustNew("usr_"),
		control: &billingcontrol.BillingControl{
			OrganizationID:                pulid.MustNew("org_"),
			BusinessUnitID:                pulid.MustNew("bu_"),
			BillingQueueTransferMode:      billingcontrol.BillingQueueTransferModeAutomaticWhenReady,
			BillingQueueTransferSchedule:  billingcontrol.TransferScheduleHourly,
			BillingQueueTransferBatchSize: 50,
		},
	}
	f.svc = &Service{
		runRepo:   f.repo,
		workflows: f.workflows,
		shipments: f.shipments,
		l:         zap.NewNop(),
	}

	return f
}

func (f *automaticFixture) start(
	t *testing.T,
	now int64,
) (*billingtransfer.BillingTransferRun, error) {
	t.Helper()

	return f.svc.StartAutomatic(t.Context(), &AutomaticRunRequest{
		Control:      f.control,
		SystemUserID: f.system,
		Now:          now,
	})
}

func TestStartAutomaticTransfersTheEligibleShipments(t *testing.T) {
	t.Parallel()

	f := newAutomaticFixture(t)
	ids := []pulid.ID{pulid.MustNew("shp_"), pulid.MustNew("shp_")}
	f.shipments.EXPECT().
		ListAutoTransferCandidateIDs(mock.Anything, mock.MatchedBy(func(req *shipmentbillingports.ListAutoTransferCandidateIDsRequest) bool {
			return req.Limit == 50 && req.ScanLimit == 250 &&
				req.TenantInfo.OrgID == f.control.OrganizationID
		})).
		Return(ids, nil).
		Once()
	f.workflows.EXPECT().Enabled().Return(true).Once()
	f.workflows.EXPECT().StartWorkflow(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()

	run, err := f.start(t, 10_000)

	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, billingtransfer.RunScopeAutomatic, run.Scope)
	assert.Equal(t, f.system, run.RequestedByID)
	assert.True(t, run.MarkCompletedReadyToInvoice)
	assert.Equal(t, 2, run.TotalCount)
	assert.Equal(t, ids, f.repo.seeded)
}

func TestStartAutomaticWaitsOutTheSchedule(t *testing.T) {
	t.Parallel()

	f := newAutomaticFixture(t)
	f.repo.lastQueuedAt = 10_000

	run, err := f.start(t, 10_000+3_599)

	require.NoError(t, err)
	assert.Nil(t, run)
	assert.Empty(t, f.repo.created)
}

func TestStartAutomaticLeavesARunningTransferAlone(t *testing.T) {
	t.Parallel()

	f := newAutomaticFixture(t)
	f.control.BillingQueueTransferSchedule = billingcontrol.TransferScheduleContinuous
	f.repo.active = &billingtransfer.BillingTransferRun{ID: pulid.MustNew("btr_")}

	run, err := f.start(t, 10_000)

	require.NoError(t, err)
	assert.Nil(t, run)
}

func TestStartAutomaticStartsNothingWhenNothingIsReady(t *testing.T) {
	t.Parallel()

	f := newAutomaticFixture(t)
	f.shipments.EXPECT().
		ListAutoTransferCandidateIDs(mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()

	run, err := f.start(t, 10_000)

	require.NoError(t, err)
	assert.Nil(t, run)
	assert.Empty(t, f.repo.created, "an empty sweep leaves no run in the report")
}

func TestStartAutomaticIgnoresManualOrganizations(t *testing.T) {
	t.Parallel()

	f := newAutomaticFixture(t)
	f.control.BillingQueueTransferMode = billingcontrol.BillingQueueTransferModeManualOnly

	run, err := f.start(t, 10_000)

	require.NoError(t, err)
	assert.Nil(t, run)
}

func TestAutomaticBatchSize(t *testing.T) {
	t.Parallel()

	control := &billingcontrol.BillingControl{BillingQueueTransferBatchSize: 200}
	assert.Equal(t, 200, AutomaticBatchSize(control))

	control.InvoiceDraftCreationMode = billingcontrol.InvoiceDraftCreationModeAutomaticWhenTransferred
	control.AutoInvoiceBatchSize = 40
	assert.Equal(
		t,
		40,
		AutomaticBatchSize(control),
		"no more drafts than the auto-invoice batch allows",
	)

	control.BillingQueueTransferBatchSize = billingtransfer.MaxRunShipments * 2
	control.AutoInvoiceBatchSize = 0
	assert.Equal(t, billingtransfer.MaxRunShipments, AutomaticBatchSize(control))
}

func TestAutomaticInterval(t *testing.T) {
	t.Parallel()

	assert.Zero(t, automaticInterval(billingcontrol.TransferScheduleContinuous))
	assert.Equal(t, int64(3600), automaticInterval(billingcontrol.TransferScheduleHourly))
	assert.Equal(t, int64(86_400), automaticInterval(billingcontrol.TransferScheduleDaily))
	assert.Equal(t, int64(604_800), automaticInterval(billingcontrol.TransferScheduleWeekly))
}
