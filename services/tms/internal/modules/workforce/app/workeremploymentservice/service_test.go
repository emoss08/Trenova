package workeremploymentservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/ptopolicyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeremploymentservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/ptopolicyports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workeremploymentports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fleetcodeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/workerportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeEventRepo struct {
	events map[pulid.ID]*employment.WorkerEmploymentEvent
	order  []pulid.ID
}

func newFakeEventRepo() *fakeEventRepo {
	return &fakeEventRepo{events: map[pulid.ID]*employment.WorkerEmploymentEvent{}}
}

func (f *fakeEventRepo) List(
	_ context.Context,
	req *workeremploymentports.ListWorkerEmploymentEventsRequest,
) ([]*employment.WorkerEmploymentEvent, error) {
	out := make([]*employment.WorkerEmploymentEvent, 0, len(f.order))
	for _, id := range f.order {
		event := f.events[id]
		if event.WorkerID == req.WorkerID {
			out = append(out, event)
		}
	}
	return out, nil
}

func (f *fakeEventRepo) GetByID(
	_ context.Context,
	req *workeremploymentports.GetWorkerEmploymentEventByIDRequest,
) (*employment.WorkerEmploymentEvent, error) {
	event, ok := f.events[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("event")
	}
	copied := *event
	return &copied, nil
}

func (f *fakeEventRepo) Create(
	_ context.Context,
	entity *employment.WorkerEmploymentEvent,
) (*employment.WorkerEmploymentEvent, error) {
	entity.ID = pulid.MustNew("wee_")
	f.events[entity.ID] = entity
	f.order = append(f.order, entity.ID)
	return entity, nil
}

func (f *fakeEventRepo) Update(
	_ context.Context,
	entity *employment.WorkerEmploymentEvent,
) (*employment.WorkerEmploymentEvent, error) {
	entity.Version++
	f.events[entity.ID] = entity
	return entity, nil
}

type fakeWorkers struct {
	updated []*worker.Worker
}

func (f *fakeWorkers) UpdateFromEmployment(
	_ context.Context,
	entity *worker.Worker,
	_ *permissionports.RequestActor,
) (*worker.Worker, error) {
	f.updated = append(f.updated, entity)
	return entity, nil
}

type fakePolicies struct {
	open           *pto.WorkerPTOPolicyAssignment
	ended          []*ptopolicyservice.EndAssignmentRequest
	defaultApplied []pulid.ID
}

func (f *fakePolicies) ListAssignments(
	context.Context,
	*ptopolicyports.ListPTOAssignmentsRequest,
) ([]*pto.WorkerPTOPolicyAssignment, error) {
	if f.open == nil {
		return nil, nil
	}
	return []*pto.WorkerPTOPolicyAssignment{f.open}, nil
}

func (f *fakePolicies) EndAssignment(
	_ context.Context,
	req *ptopolicyservice.EndAssignmentRequest,
) (*pto.WorkerPTOPolicyAssignment, error) {
	f.ended = append(f.ended, req)
	return f.open, nil
}

func (f *fakePolicies) AssignDefaultPolicy(
	_ context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	_ int64,
	_ pulid.ID,
) error {
	f.defaultApplied = append(f.defaultApplied, workerID)
	return nil
}

type fakePTO struct {
	workerptoports.WorkerPTOService
	upcoming  []*worker.WorkerPTO
	cancelled []*workerptoports.UpdatePTOStatusRequest
}

func (f *fakePTO) List(
	_ context.Context,
	req *workerptoports.ListPTORequest,
) (*pagination.CursorListResult[*worker.WorkerPTO], error) {
	items := make([]*worker.WorkerPTO, 0, len(f.upcoming))
	for _, pto := range f.upcoming {
		if pto.Status.String() == req.Status && pto.StartDate >= req.StartDateFrom {
			items = append(items, pto)
		}
	}
	return &pagination.CursorListResult[*worker.WorkerPTO]{Items: items}, nil
}

func (f *fakePTO) Cancel(
	_ context.Context,
	req *workerptoports.UpdatePTOStatusRequest,
) (*worker.WorkerPTO, error) {
	f.cancelled = append(f.cancelled, req)
	return &worker.WorkerPTO{ID: req.ID}, nil
}

type fakePayRepo struct {
	driverpayports.WorkerPayAssignmentRepository
	assignment *payassignment.WorkerPayAssignment
	err        error
}

func (f *fakePayRepo) GetEffectiveForWorker(
	context.Context,
	driverpayports.GetWorkerPayAssignmentRequest,
) (*payassignment.WorkerPayAssignment, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.assignment, nil
}

type fakePay struct {
	ended []pulid.ID
}

func (f *fakePay) EndAssignment(
	_ context.Context,
	_ pagination.TenantInfo,
	assignmentID pulid.ID,
	_ int64,
	_ *permissionports.RequestActor,
) (*payassignment.WorkerPayAssignment, error) {
	f.ended = append(f.ended, assignmentID)
	return &payassignment.WorkerPayAssignment{ID: assignmentID}, nil
}

type fakeLedger struct {
	settled    []int64
	settlement *ptoledgerservice.TerminationSettlement
}

func (f *fakeLedger) SettleOnTermination(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	effectiveAt int64,
	_ ptoledgerservice.Actor,
) (*ptoledgerservice.TerminationSettlement, error) {
	f.settled = append(f.settled, effectiveAt)
	if f.settlement == nil {
		return &ptoledgerservice.TerminationSettlement{}, nil
	}
	return f.settlement, nil
}

type fakePortal struct {
	hadAccess bool
	err       error
	revoked   []pulid.ID
}

func (f *fakePortal) RevokeOnTermination(
	_ context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	_ *permissionports.RequestActor,
) (bool, error) {
	f.revoked = append(f.revoked, workerID)
	if f.err != nil {
		return false, f.err
	}
	return f.hadAccess, nil
}

type harness struct {
	svc      *workeremploymentservice.Service
	ledger   *fakeLedger
	events   *fakeEventRepo
	workers  *fakeWorkers
	policies *fakePolicies
	pto      *fakePTO
	pay      *fakePay
	payRepo  *fakePayRepo
	portal   *fakePortal
	fleet    *fleetcodeportsmock.MockFleetCodeRepository
	lists    *fakeChecklists
	tenant   pagination.TenantInfo
	wrk      *worker.Worker
	userID   pulid.ID
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	wrk := &worker.Worker{
		ID:                   pulid.MustNew("wrk_"),
		OrganizationID:       tenant.OrgID,
		BusinessUnitID:       tenant.BuID,
		Status:               domaintypes.StatusActive,
		CanBeAssigned:        true,
		AvailableForDispatch: true,
		DriverType:           worker.DriverTypeLocal,
		Type:                 worker.WorkerTypeEmployee,
		FleetCodeID:          pulid.MustNew("fc_"),
		FleetCode:            &fleetcode.FleetCode{Code: "SOUTH"},
		Profile:              &worker.WorkerProfile{HireDate: 1_600_000_000},
	}

	workerRepo := workerportsmock.NewMockWorkerRepository(t)
	workerRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(wrk, nil).Maybe()
	docRepo := documentportsmock.NewMockDocumentRepository(t)
	payRepo := &fakePayRepo{err: errortypes.NewNotFoundError("none")}
	fleet := fleetcodeportsmock.NewMockFleetCodeRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Maybe()

	h := &harness{
		events:   newFakeEventRepo(),
		workers:  &fakeWorkers{},
		policies: &fakePolicies{},
		pto:      &fakePTO{},
		ledger:   &fakeLedger{},
		pay:      &fakePay{},
		payRepo:  payRepo,
		portal:   &fakePortal{hadAccess: true},
		lists:    &fakeChecklists{},
		fleet:    fleet,
		tenant:   tenant,
		wrk:      wrk,
		userID:   pulid.MustNew("usr_"),
	}
	h.svc = workeremploymentservice.NewWithDeps(workeremploymentservice.Deps{
		Repo:          h.events,
		WorkerRepo:    workerRepo,
		FleetCodeRepo: fleet,
		DocumentRepo:  docRepo,
		PayAssignRepo: payRepo,
		Workers:       h.workers,
		PTOPolicies:   h.policies,
		PTO:           h.pto,
		PTOLedger:     h.ledger,
		DriverPay:     h.pay,
		AuditService:  audit,
		Portal:        h.portal,
		Checklists:    h.lists,
	})
	return h
}

// fakeChecklists records what the employment service asked of the checklist
// port, so a test can see the order and the event each call carried.
type fakeChecklists struct {
	closed  []employment.EmploymentEventKind
	spawned []employment.EmploymentEventKind
}

func (f *fakeChecklists) SpawnForEvent(
	_ context.Context,
	event *employment.WorkerEmploymentEvent,
	_ *worker.Worker,
	_ pulid.ID,
) (*workerchecklist.WorkerChecklist, error) {
	f.spawned = append(f.spawned, event.Kind)
	if _, ok := workerchecklist.ChecklistTriggerForEvent(event.Kind); !ok {
		return nil, nil //nolint:nilnil // nothing to spawn
	}
	return &workerchecklist.WorkerChecklist{ID: pulid.MustNew("wcl_"), SourceEventID: event.ID}, nil
}

func (f *fakeChecklists) CloseForEvent(
	_ context.Context,
	event *employment.WorkerEmploymentEvent,
	_ *worker.Worker,
	_ pulid.ID,
) (int, error) {
	f.closed = append(f.closed, event.Kind)
	if _, ok := workerchecklist.ChecklistKindClosedByEvent(event.Kind); ok {
		return 1, nil
	}
	return 0, nil
}

func (h *harness) record(
	kind employment.EmploymentEventKind,
	effective int64,
	mutate func(*workeremploymentservice.RecordRequest),
) (*workeremploymentservice.RecordResult, error) {
	req := &workeremploymentservice.RecordRequest{
		TenantInfo:  h.tenant,
		WorkerID:    h.wrk.ID,
		Kind:        kind,
		EffectiveAt: effective,
		Reason:      "because",
		UserID:      h.userID,
	}
	if mutate != nil {
		mutate(req)
	}
	return h.svc.Record(context.Background(), req)
}

// A terminated worker must lose their Dash sign-in as part of the termination
// itself. Leaving it to the offboarding checklist means a departed driver keeps
// a working portal until somebody remembers to close the item.
func TestRecord_TerminationRevokesPortalAccess(t *testing.T) {
	h := newHarness(t)
	h.portal.hadAccess = true

	result, err := h.record(employment.EmploymentEventTerminated, 1_800_000_000, nil)
	require.NoError(t, err)

	require.Equal(t, []pulid.ID{h.wrk.ID}, h.portal.revoked)
	assert.True(t, result.Cascade.PortalAccessRevoked)
	assert.Empty(t, result.Cascade.PortalRevocationError)
}

// A worker who was never invited to Dash has nothing to revoke, and the
// summary must not claim otherwise.
func TestRecord_TerminationWithoutPortalAccessClaimsNothing(t *testing.T) {
	h := newHarness(t)
	h.portal.hadAccess = false

	result, err := h.record(employment.EmploymentEventTerminated, 1_800_000_000, nil)
	require.NoError(t, err)

	require.Len(t, h.portal.revoked, 1)
	assert.False(t, result.Cascade.PortalAccessRevoked)
	assert.Empty(t, result.Cascade.PortalRevocationError)
}

// The worker row is already Inactive by the time the cascade runs, so a portal
// failure must not abort the record and lose the timeline event. It is
// reported instead, so the office knows to shut the account off by hand.
func TestRecord_PortalRevocationFailureIsReportedNotFatal(t *testing.T) {
	h := newHarness(t)
	h.portal.err = errors.New("portal is unreachable")

	result, err := h.record(employment.EmploymentEventTerminated, 1_800_000_000, nil)
	require.NoError(t, err)
	require.NotNil(t, result.Event)

	assert.False(t, result.Cascade.PortalAccessRevoked)
	assert.Contains(t, result.Cascade.PortalRevocationError, "portal is unreachable")
}

// Suspension and leave take a worker off the board but do not end employment,
// so they keep their sign-in. Only termination revokes.
func TestRecord_NonTerminationLeavesPortalAlone(t *testing.T) {
	for _, kind := range []employment.EmploymentEventKind{
		employment.EmploymentEventSuspended,
		employment.EmploymentEventLeaveStarted,
	} {
		t.Run(kind.String(), func(t *testing.T) {
			h := newHarness(t)
			_, err := h.record(
				kind,
				1_800_000_000,
				func(req *workeremploymentservice.RecordRequest) {
					leave := worker.LeaveTypeMedical
					req.LeaveType = &leave
				},
			)
			require.NoError(t, err)
			assert.Empty(t, h.portal.revoked)
		})
	}
}

func TestRecord_TerminationCascades(t *testing.T) {
	h := newHarness(t)
	effective := int64(1_800_000_000)
	h.policies.open = &pto.WorkerPTOPolicyAssignment{
		ID:            pulid.MustNew("wppa_"),
		EffectiveFrom: 1_700_000_000,
		Version:       3,
	}
	payAssignment := &payassignment.WorkerPayAssignment{
		ID:            pulid.MustNew("wpa_"),
		EffectiveFrom: 1_700_000_000,
	}
	h.payRepo.assignment = payAssignment
	h.payRepo.err = nil
	h.pto.upcoming = []*worker.WorkerPTO{
		{
			ID:        pulid.MustNew("wrkpto_"),
			Status:    worker.PTOStatusApproved,
			StartDate: effective + 86400,
			Version:   1,
		},
		{
			ID:        pulid.MustNew("wrkpto_"),
			Status:    worker.PTOStatusRequested,
			StartDate: effective + 5*86400,
			Version:   2,
		},
		{
			ID:        pulid.MustNew("wrkpto_"),
			Status:    worker.PTOStatusApproved,
			StartDate: effective - 86400,
			Version:   1,
		},
	}
	h.ledger.settlement = &ptoledgerservice.TerminationSettlement{
		PaidOutDays:   decimal.NewFromFloat(6.5),
		ForfeitedDays: decimal.NewFromInt(2),
	}

	result, err := h.record(employment.EmploymentEventTerminated, effective, nil)
	require.NoError(t, err)

	assert.Equal(t, domaintypes.StatusInactive, h.wrk.Status)
	assert.False(t, h.wrk.CanBeAssigned)
	require.NotNil(t, h.wrk.Profile.TerminationDate)
	assert.Equal(t, effective, *h.wrk.Profile.TerminationDate)
	require.Len(t, h.workers.updated, 1)

	assert.True(t, result.Cascade.PTOAssignmentEnded)
	require.Len(t, h.policies.ended, 1)
	assert.Equal(t, effective, h.policies.ended[0].EffectiveTo)
	assert.Equal(t, int64(3), h.policies.ended[0].Version)

	assert.True(t, result.Cascade.PayAssignmentEnded)
	assert.Equal(t, []pulid.ID{payAssignment.ID}, h.pay.ended)

	assert.Equal(
		t,
		2,
		result.Cascade.UpcomingPTOCancelled,
		"only time off starting on or after the termination is cancelled",
	)
	require.Len(t, h.pto.cancelled, 2)
	assert.Equal(t, "Employment ended", h.pto.cancelled[0].Reason)
	assert.Equal(t, worker.PTOStatusCancelled, h.pto.cancelled[0].Status)

	assert.Equal(
		t,
		[]int64{effective},
		h.ledger.settled,
		"balances are closed as of the termination",
	)
	assert.True(t, result.Cascade.PTOPaidOutDays.Equal(decimal.NewFromFloat(6.5)))
	assert.True(t, result.Cascade.PTOForfeitedDays.Equal(decimal.NewFromInt(2)))

	event := result.Event
	assert.Equal(t, "Active", event.FromValues[employment.EmploymentValueStatus])
	assert.Equal(t, "Inactive", event.ToValues[employment.EmploymentValueStatus])
	assert.Equal(t, h.userID, event.RecordedByID)

	_, err = h.record(employment.EmploymentEventTerminated, effective+86400, nil)
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr, "a second termination is refused")
	assert.Equal(t, "kind", verr.Field)
}

func TestRecord_TerminationNeedsReason(t *testing.T) {
	h := newHarness(t)
	_, err := h.record(
		employment.EmploymentEventTerminated,
		1_800_000_000,
		func(req *workeremploymentservice.RecordRequest) {
			req.Reason = "  "
		},
	)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "reason", multiErr.Errors[0].Field)
	assert.Empty(t, h.workers.updated, "validation runs before any effect")
}

func TestRecord_RehireReopensAndReenrols(t *testing.T) {
	h := newHarness(t)
	h.wrk.Status = domaintypes.StatusInactive
	h.wrk.CanBeAssigned = false
	term := int64(1_750_000_000)
	h.wrk.Profile.TerminationDate = &term

	result, err := h.record(employment.EmploymentEventRehired, 1_790_000_000, nil)
	require.NoError(t, err)
	assert.Equal(t, domaintypes.StatusActive, h.wrk.Status)
	assert.True(t, h.wrk.CanBeAssigned)
	assert.Nil(t, h.wrk.Profile.TerminationDate)
	assert.Equal(t, int64(1_790_000_000), h.wrk.Profile.HireDate)
	assert.True(t, result.Cascade.DefaultPolicyApplied)
	assert.Equal(t, []pulid.ID{h.wrk.ID}, h.policies.defaultApplied)
	assert.Equal(t, "1600000000", result.Event.FromValues[employment.EmploymentValueHireDate])
}

func TestRecord_TransferValidatesFleetAndRecordsBothSides(t *testing.T) {
	h := newHarness(t)
	target := pulid.MustNew("fc_")
	h.fleet.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&fleetcode.FleetCode{ID: target, Code: "NORTH"}, nil).
		Once()

	result, err := h.record(
		employment.EmploymentEventTransferred,
		1_800_000_000,
		func(req *workeremploymentservice.RecordRequest) {
			req.FleetCodeID = &target
		},
	)
	require.NoError(t, err)
	assert.Equal(t, target, h.wrk.FleetCodeID)
	assert.Equal(t, "SOUTH", result.Event.FromValues[employment.EmploymentValueFleetCode])
	assert.Equal(t, "NORTH", result.Event.ToValues[employment.EmploymentValueFleetCode])
	require.Len(t, h.workers.updated, 1)

	same := h.wrk.FleetCodeID
	_, err = h.record(
		employment.EmploymentEventTransferred,
		1_800_100_000,
		func(req *workeremploymentservice.RecordRequest) {
			req.FleetCodeID = &same
		},
	)
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "fleetCodeId", verr.Field)
}

func TestRecord_LeaveAndSuspensionPairing(t *testing.T) {
	h := newHarness(t)

	_, err := h.record(employment.EmploymentEventLeaveEnded, 1_800_000_000, nil)
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr, "cannot end a leave that never started")

	_, err = h.record(employment.EmploymentEventLeaveStarted, 1_800_000_000, nil)
	require.ErrorAs(t, err, &verr, "a leave must say what kind it is")
	assert.Equal(t, "leaveType", verr.Field)

	fmla := worker.LeaveTypeFMLA
	started, err := h.record(
		employment.EmploymentEventLeaveStarted,
		1_800_000_000,
		func(req *workeremploymentservice.RecordRequest) {
			req.LeaveType = &fmla
		},
	)
	require.NoError(t, err)
	assert.False(t, h.wrk.CanBeAssigned)
	assert.Equal(t, domaintypes.StatusActive, h.wrk.Status, "leave keeps the worker employed")
	assert.Equal(t, worker.LeaveTypeFMLA, h.wrk.LeaveType)
	assert.Equal(t, "FMLA", started.Event.ToValues[employment.EmploymentValueLeaveType])

	_, err = h.record(
		employment.EmploymentEventLeaveStarted,
		1_800_100_000,
		func(req *workeremploymentservice.RecordRequest) {
			req.LeaveType = &fmla
		},
	)
	require.ErrorAs(t, err, &verr, "leave cannot stack")

	ended, err := h.record(employment.EmploymentEventLeaveEnded, 1_800_200_000, nil)
	require.NoError(t, err)
	assert.True(t, h.wrk.CanBeAssigned)
	assert.Empty(t, h.wrk.LeaveType, "returning clears the leave")
	assert.Equal(t, "FMLA", ended.Event.FromValues[employment.EmploymentValueLeaveType])
}

func TestRecord_PromotionRequiresAChange(t *testing.T) {
	h := newHarness(t)
	local := worker.DriverTypeLocal
	_, err := h.record(
		employment.EmploymentEventPromoted,
		1_800_000_000,
		func(req *workeremploymentservice.RecordRequest) {
			req.DriverType = &local
		},
	)
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "driverType", verr.Field)

	otr := worker.DriverTypeOTR
	result, err := h.record(
		employment.EmploymentEventPromoted,
		1_800_000_000,
		func(req *workeremploymentservice.RecordRequest) {
			req.DriverType = &otr
		},
	)
	require.NoError(t, err)
	assert.Equal(t, worker.DriverTypeOTR, h.wrk.DriverType)
	assert.Equal(t, "Local", result.Event.FromValues[employment.EmploymentValueDriverType])
	assert.Equal(t, "OTR", result.Event.ToValues[employment.EmploymentValueDriverType])
}

func TestRecord_RateChangeIsInformational(t *testing.T) {
	h := newHarness(t)
	result, err := h.record(
		employment.EmploymentEventRateChanged,
		1_800_000_000,
		func(req *workeremploymentservice.RecordRequest) {
			req.Rate = "0.65"
			req.RateUnit = "per mile"
		},
	)
	require.NoError(t, err)
	assert.Empty(t, h.workers.updated, "no worker change for a rate note")
	assert.Equal(t, "0.65", result.Event.ToValues[employment.EmploymentValueRate])
}

func TestAmend_CorrectsNarrativeWithoutReplayingEffects(t *testing.T) {
	h := newHarness(t)
	result, err := h.record(employment.EmploymentEventTerminated, 1_800_000_000, nil)
	require.NoError(t, err)
	updatesBefore := len(h.workers.updated)

	_, err = h.svc.Amend(context.Background(), &workeremploymentservice.AmendRequest{
		ID:         result.Event.ID,
		TenantInfo: h.tenant,
		Version:    result.Event.Version,
		UserID:     h.userID,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr, "an amendment must say why")
	assert.Equal(t, "amendmentNote", verr.Field)

	newEffective := int64(1_800_086_400)
	reason := "Resigned"
	amended, err := h.svc.Amend(context.Background(), &workeremploymentservice.AmendRequest{
		ID:            result.Event.ID,
		TenantInfo:    h.tenant,
		EffectiveAt:   &newEffective,
		Reason:        &reason,
		AmendmentNote: "Wrong date entered",
		Version:       result.Event.Version,
		UserID:        h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, newEffective, amended.EffectiveAt)
	assert.Equal(t, "Resigned", amended.Reason)
	assert.True(t, amended.IsAmended())
	assert.Equal(
		t,
		int64(1_800_000_000),
		*h.wrk.Profile.TerminationDate,
		"effects are not replayed",
	)
	assert.Len(t, h.workers.updated, updatesBefore)

	_, err = h.svc.Amend(context.Background(), &workeremploymentservice.AmendRequest{
		ID:            result.Event.ID,
		TenantInfo:    h.tenant,
		AmendmentNote: "stale",
		Version:       amended.Version + 5,
		UserID:        h.userID,
	})
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "version", verr.Field)
}

// A termination closes the onboarding it interrupts before it opens the
// offboarding, and the office is told both happened.
func TestRecord_TerminatedClosesStaleChecklistsThenSpawns(t *testing.T) {
	h := newHarness(t)

	result, err := h.record(
		employment.EmploymentEventTerminated,
		1_700_500_000,
		func(req *workeremploymentservice.RecordRequest) {
			req.Reason = "Resigned"
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Cascade.ChecklistsClosed)
	assert.True(t, result.Cascade.ChecklistStarted)
	assert.Equal(
		t,
		[]employment.EmploymentEventKind{employment.EmploymentEventTerminated},
		h.lists.closed,
	)
	assert.Equal(
		t,
		[]employment.EmploymentEventKind{employment.EmploymentEventTerminated},
		h.lists.spawned,
	)
}
