package workeremploymentservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/driverportalinvitationservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/ptopolicyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeremploymentservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/portalinvitation"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerchecklistports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workeremploymentports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fleetcodeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/workerportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
	stored  *worker.Worker
	updated []*worker.Worker
	err     error
}

func (f *fakeWorkers) UpdateFromEmployment(
	_ context.Context,
	entity *worker.Worker,
	_ *permissionports.RequestActor,
) (*worker.Worker, error) {
	if f.err != nil {
		return nil, f.err
	}
	saved := *entity
	f.updated = append(f.updated, &saved)
	*f.stored = saved
	return &saved, nil
}

type fakePolicies struct {
	open          *pto.WorkerPTOPolicyAssignment
	defaultPolicy *pto.PTOPolicy
	ended         []*ptopolicyservice.EndAssignmentRequest
	hired         []*ptopolicyservice.HireAssignment
	hireErr       error
}

func (f *fakePolicies) ActiveAssignment(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*pto.WorkerPTOPolicyAssignment, error) {
	return f.open, nil
}

func (f *fakePolicies) EndAssignment(
	_ context.Context,
	req *ptopolicyservice.EndAssignmentRequest,
) (*pto.WorkerPTOPolicyAssignment, error) {
	f.ended = append(f.ended, req)
	return f.open, nil
}

func (f *fakePolicies) DefaultPolicy(
	context.Context,
	pagination.TenantInfo,
) (*pto.PTOPolicy, error) {
	return f.defaultPolicy, nil
}

func (f *fakePolicies) AssignAtHire(
	_ context.Context,
	req *ptopolicyservice.HireAssignment,
) error {
	f.hired = append(f.hired, req)
	return f.hireErr
}

type fakePTO struct {
	workerptoports.WorkerPTOService
	upcoming  []*worker.WorkerPTO
	cancelled []*workerptoports.UpdatePTOStatusRequest
	cancelErr map[pulid.ID]error
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
	if err := f.cancelErr[req.ID]; err != nil {
		return nil, err
	}
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
	err   error
}

func (f *fakePay) EndAssignment(
	_ context.Context,
	_ pagination.TenantInfo,
	assignmentID pulid.ID,
	_ int64,
	_ *permissionports.RequestActor,
) (*payassignment.WorkerPayAssignment, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.ended = append(f.ended, assignmentID)
	return &payassignment.WorkerPayAssignment{ID: assignmentID}, nil
}

type fakeLedger struct {
	assignment *pto.WorkerPTOPolicyAssignment
	policy     *pto.PTOPolicy
	balances   []*pto.WorkerPTOBalance
	released   [][]ptoledgerservice.ReleasedUsage
	settled    []*ptoledgerservice.TerminationPlan
}

func (f *fakeLedger) PreviewTerminationSettlement(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	_ int64,
	released []ptoledgerservice.ReleasedUsage,
) (*ptoledgerservice.TerminationPlan, error) {
	f.released = append(f.released, released)
	if f.policy == nil {
		return &ptoledgerservice.TerminationPlan{}, nil
	}
	byType := map[worker.PTOType]*pto.WorkerPTOBalance{}
	balances := make([]*pto.WorkerPTOBalance, 0, len(f.balances)+len(released))
	for _, bal := range f.balances {
		projected := *bal
		byType[projected.PTOType] = &projected
		balances = append(balances, &projected)
	}
	for _, usage := range released {
		bal, ok := byType[usage.PTOType]
		if !ok {
			bal = &pto.WorkerPTOBalance{PTOType: usage.PTOType}
			byType[usage.PTOType] = bal
			balances = append(balances, bal)
		}
		bal.BalanceDays = bal.BalanceDays.Add(usage.Days)
	}
	return &ptoledgerservice.TerminationPlan{
		Resolved: &ptoledgerservice.ResolvedPolicy{
			Assignment: f.assignment,
			Policy:     f.policy,
		},
		Settlements: ptoledgerservice.PlanTerminationSettlement(balances, f.policy),
	}, nil
}

func (f *fakeLedger) SettleOnTermination(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	_ int64,
	plan *ptoledgerservice.TerminationPlan,
	_ ptoledgerservice.Actor,
) (*ptoledgerservice.TerminationSettlement, error) {
	f.settled = append(f.settled, plan)
	result := &ptoledgerservice.TerminationSettlement{}
	for _, planned := range plan.Settlements {
		result.Entries = append(result.Entries, &pto.WorkerPTOLedgerEntry{
			PTOType:    planned.PTOType,
			EntryType:  planned.EntryType,
			AmountDays: planned.Days.Neg(),
		})
	}
	return result, nil
}

type fakePortal struct {
	status  driverportalinvitationservice.PortalStatus
	err     error
	revoked []pulid.ID
}

func (f *fakePortal) GetPortalStatus(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*driverportalinvitationservice.PortalStatus, error) {
	status := f.status
	return &status, nil
}

func (f *fakePortal) RevokeAccess(
	_ context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	_ *permissionports.RequestActor,
) error {
	f.revoked = append(f.revoked, workerID)
	return f.err
}

type fakeChecklists struct {
	template *workerchecklist.WorkerChecklistTemplate
	open     []*workerchecklist.WorkerChecklist
	planned  []employment.EmploymentEventKind
	closed   []employment.EmploymentEventKind
	spawned  []employment.EmploymentEventKind
}

func (f *fakeChecklists) PlanForEvent(
	_ context.Context,
	kind employment.EmploymentEventKind,
	wrk *worker.Worker,
) (*workerchecklistports.EventPlan, error) {
	f.planned = append(f.planned, kind)
	plan := &workerchecklistports.EventPlan{Worker: wrk}
	if closes, ok := workerchecklist.ChecklistKindClosedByEvent(kind); ok {
		for _, checklist := range f.open {
			if checklist.Kind == closes && checklist.IsOpen() {
				plan.Close = append(plan.Close, checklist)
			}
		}
	}
	if _, ok := workerchecklist.ChecklistTriggerForEvent(kind); ok {
		plan.Template = f.template
	}
	return plan, nil
}

func (f *fakeChecklists) SpawnForEvent(
	_ context.Context,
	event *employment.WorkerEmploymentEvent,
	plan *workerchecklistports.EventPlan,
	_ pulid.ID,
) (*workerchecklist.WorkerChecklist, error) {
	if plan.Template == nil {
		return nil, nil //nolint:nilnil // nothing to spawn
	}
	f.spawned = append(f.spawned, event.Kind)
	return &workerchecklist.WorkerChecklist{ID: pulid.MustNew("wcl_"), SourceEventID: event.ID}, nil
}

func (f *fakeChecklists) CloseForEvent(
	_ context.Context,
	event *employment.WorkerEmploymentEvent,
	plan *workerchecklistports.EventPlan,
	_ pulid.ID,
) (int, error) {
	f.closed = append(f.closed, event.Kind)
	for _, checklist := range plan.Close {
		checklist.Status = workerchecklist.ChecklistStatusCancelled
	}
	return len(plan.Close), nil
}

type fakeTraining struct {
	courses   map[worker.DriverType][]*workertraining.TrainingCourse
	plannedOn []worker.DriverType
	assigned  [][]*workertraining.TrainingCourse
	failAll   bool
}

func (f *fakeTraining) PlanAssignRequiredFor(
	_ context.Context,
	_ pagination.TenantInfo,
	wrk *worker.Worker,
) ([]*workertraining.TrainingCourse, error) {
	f.plannedOn = append(f.plannedOn, wrk.DriverType)
	return f.courses[wrk.DriverType], nil
}

func (f *fakeTraining) AssignPlanned(
	_ context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	courses []*workertraining.TrainingCourse,
	_ pulid.ID,
) []*workertraining.WorkerTrainingRecord {
	f.assigned = append(f.assigned, courses)
	if f.failAll {
		return nil
	}
	out := make([]*workertraining.WorkerTrainingRecord, 0, len(courses))
	for _, course := range courses {
		out = append(out, &workertraining.WorkerTrainingRecord{
			ID:       pulid.MustNew("wtr_"),
			WorkerID: workerID,
			CourseID: course.ID,
		})
	}
	return out
}

type fakeSecurity struct {
	changes []*permissionports.SecurityChange
}

func (f *fakeSecurity) RecordChange(_ context.Context, change *permissionports.SecurityChange) {
	f.changes = append(f.changes, change)
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
	users    *tenantportsmock.MockUserRepository
	lists    *fakeChecklists
	training *fakeTraining
	security *fakeSecurity
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
	workerRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(context.Context, workerports.GetWorkerByIDRequest) (*worker.Worker, error) {
			copied := *wrk
			profile := *wrk.Profile
			copied.Profile = &profile
			return &copied, nil
		}).
		Maybe()
	docRepo := documentportsmock.NewMockDocumentRepository(t)
	payRepo := &fakePayRepo{err: errortypes.NewNotFoundError("none")}
	fleet := fleetcodeportsmock.NewMockFleetCodeRepository(t)
	users := tenantportsmock.NewMockUserRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	h := &harness{
		events:   newFakeEventRepo(),
		workers:  &fakeWorkers{stored: wrk},
		policies: &fakePolicies{},
		pto:      &fakePTO{},
		ledger:   &fakeLedger{},
		pay:      &fakePay{},
		payRepo:  payRepo,
		portal:   &fakePortal{status: driverportalinvitationservice.PortalStatus{Linked: true}},
		lists:    &fakeChecklists{},
		training: &fakeTraining{},
		security: &fakeSecurity{},
		fleet:    fleet,
		users:    users,
		tenant:   tenant,
		wrk:      wrk,
		userID:   pulid.MustNew("usr_"),
	}
	h.svc = workeremploymentservice.New(workeremploymentservice.Params{
		Logger:        zap.NewNop(),
		DB:            dbtest.NopConnection{},
		Repo:          h.events,
		WorkerRepo:    workerRepo,
		FleetCodeRepo: fleet,
		DocumentRepo:  docRepo,
		PayAssignRepo: payRepo,
		Users:         users,
		PTO:           h.pto,
		AuditService:  audit,
		Security:      h.security,
		Realtime:      &mocks.NoopRealtimeService{},
		Checklists:    h.lists,
		Training:      h.training,
	})
	h.svc.SetCollaborators(h.workers, h.policies, h.ledger, h.pay)
	h.svc.SetPortalRevoker(h.portal)
	return h
}

func (h *harness) request(
	kind employment.EmploymentEventKind,
	effective int64,
	mutate func(*workeremploymentservice.RecordRequest),
) *workeremploymentservice.RecordRequest {
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
	return req
}

func (h *harness) record(
	kind employment.EmploymentEventKind,
	effective int64,
	mutate func(*workeremploymentservice.RecordRequest),
) (*workeremploymentservice.RecordResult, error) {
	return h.svc.Record(context.Background(), h.request(kind, effective, mutate))
}

func effectsOf(
	effects []*employment.Effect,
	kind employment.EffectKind,
) []*employment.Effect {
	out := make([]*employment.Effect, 0)
	for _, effect := range effects {
		if effect.Kind == kind {
			out = append(out, effect)
		}
	}
	return out
}

func kindsOf(effects []*employment.Effect) []employment.EffectKind {
	out := make([]employment.EffectKind, 0, len(effects))
	for _, effect := range effects {
		out = append(out, effect.Kind)
	}
	return out
}

// A terminated worker must lose their Dash sign-in as part of the termination
// itself, and the office is told it happened.
func TestRecord_TerminationRevokesPortalAccess(t *testing.T) {
	h := newHarness(t)

	result, err := h.record(employment.EmploymentEventTerminated, 1_800_000_000, nil)
	require.NoError(t, err)

	require.Equal(t, []pulid.ID{h.wrk.ID}, h.portal.revoked)
	revoked := effectsOf(result.Effects, employment.EffectPortalAccessRevoked)
	require.Len(t, revoked, 1)
	assert.False(t, revoked[0].Failed)
}

// An open invitation is access too: it is revoked so a departed driver cannot
// accept it later.
func TestRecord_TerminationRevokesAPendingInvitation(t *testing.T) {
	h := newHarness(t)
	h.portal.status = driverportalinvitationservice.PortalStatus{
		PendingInvitation: &portalinvitation.PortalInvitation{ID: pulid.MustNew("pinv_")},
	}

	result, err := h.record(employment.EmploymentEventTerminated, 1_800_000_000, nil)
	require.NoError(t, err)

	assert.Equal(t, []pulid.ID{h.wrk.ID}, h.portal.revoked)
	assert.Len(t, effectsOf(result.Effects, employment.EffectPortalAccessRevoked), 1)
}

// A worker who was never invited to Dash has nothing to revoke, and neither the
// plan nor the write claims otherwise.
func TestRecord_TerminationWithoutPortalAccessClaimsNothing(t *testing.T) {
	h := newHarness(t)
	h.portal.status = driverportalinvitationservice.PortalStatus{}

	result, err := h.record(employment.EmploymentEventTerminated, 1_800_000_000, nil)
	require.NoError(t, err)

	assert.Empty(t, h.portal.revoked)
	assert.Empty(t, effectsOf(result.Effects, employment.EffectPortalAccessRevoked))
}

// The termination has committed by the time the portal is shut off, so a
// portal failure is reported on the effect rather than undoing the event.
func TestRecord_PortalRevocationFailureIsReportedNotFatal(t *testing.T) {
	h := newHarness(t)
	h.portal.err = errors.New("portal is unreachable")

	result, err := h.record(employment.EmploymentEventTerminated, 1_800_000_000, nil)
	require.NoError(t, err)
	require.NotNil(t, result.Event)
	assert.False(t, result.Event.ID.IsNil())

	revoked := effectsOf(result.Effects, employment.EffectPortalAccessRevoked)
	require.Len(t, revoked, 1)
	assert.True(t, revoked[0].Failed)
}

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

func TestRecord_TerminationCarriesOutEveryPlannedEffect(t *testing.T) {
	h := newHarness(t)
	effective := int64(1_800_000_000)
	assignment := &pto.WorkerPTOPolicyAssignment{
		ID:            pulid.MustNew("wppa_"),
		EffectiveFrom: 1_700_000_000,
		Version:       3,
	}
	h.ledger.assignment = assignment
	h.ledger.policy = &pto.PTOPolicy{
		ID:   pulid.MustNew("ptop_"),
		Name: "Drivers",
		Rules: []*pto.PTOPolicyRule{
			{PTOType: worker.PTOTypeVacation, OnTermination: pto.PTOTerminationPayOut},
			{PTOType: worker.PTOTypeSick},
		},
	}
	h.ledger.balances = []*pto.WorkerPTOBalance{
		{PTOType: worker.PTOTypeVacation, BalanceDays: decimal.NewFromFloat(3.5)},
		{PTOType: worker.PTOTypeSick, BalanceDays: decimal.NewFromInt(2)},
	}
	payAssignment := &payassignment.WorkerPayAssignment{
		ID:            pulid.MustNew("wpa_"),
		EffectiveFrom: 1_700_000_000,
		PayProfile:    &driverpay.PayProfile{Name: "Linehaul mileage"},
	}
	h.payRepo.assignment = payAssignment
	h.payRepo.err = nil
	booked := &worker.WorkerPTO{
		ID:               pulid.MustNew("wrkpto_"),
		Status:           worker.PTOStatusApproved,
		Type:             worker.PTOTypeVacation,
		StartDate:        effective + 86400,
		EndDate:          effective + 3*86400,
		Days:             decimal.NewFromInt(3),
		BalanceAfterDays: decimal.NewNullDecimal(decimal.NewFromFloat(0.5)),
		Version:          1,
	}
	requested := &worker.WorkerPTO{
		ID:        pulid.MustNew("wrkpto_"),
		Status:    worker.PTOStatusRequested,
		Type:      worker.PTOTypeSick,
		StartDate: effective + 5*86400,
		EndDate:   effective + 5*86400,
		Days:      decimal.NewFromInt(1),
		Version:   2,
	}
	past := &worker.WorkerPTO{
		ID:        pulid.MustNew("wrkpto_"),
		Status:    worker.PTOStatusApproved,
		StartDate: effective - 86400,
		Version:   1,
	}
	h.pto.upcoming = []*worker.WorkerPTO{booked, requested, past}

	result, err := h.record(employment.EmploymentEventTerminated, effective, nil)
	require.NoError(t, err)

	require.Len(t, h.workers.updated, 1)
	saved := h.workers.updated[0]
	assert.Equal(t, domaintypes.StatusInactive, saved.Status)
	assert.False(t, saved.CanBeAssigned)
	require.NotNil(t, saved.Profile.TerminationDate)
	assert.Equal(t, effective, *saved.Profile.TerminationDate)

	assert.Equal(t, []employment.EffectKind{
		employment.EffectTimeOffCancelled,
		employment.EffectTimeOffCancelled,
		employment.EffectPTOForfeited,
		employment.EffectPTOPaidOut,
		employment.EffectPTOPolicyEnded,
		employment.EffectPayAssignmentEnded,
		employment.EffectPortalAccessRevoked,
	}, kindsOf(result.Effects))

	require.Len(t, h.pto.cancelled, 2, "only time off starting on or after the termination")
	assert.Equal(t, booked.ID, h.pto.cancelled[0].ID)
	assert.Equal(t, "Employment ended", h.pto.cancelled[0].Reason)
	assert.Equal(t, worker.PTOStatusCancelled, h.pto.cancelled[0].Status)
	cancelled := effectsOf(result.Effects, employment.EffectTimeOffCancelled)
	assert.Equal(t, worker.PTOTypeVacation, *cancelled[0].PTOType)
	assert.Equal(t, booked.StartDate, *cancelled[0].StartDate)
	assert.True(t, cancelled[0].Days.Equal(decimal.NewFromInt(3)))

	require.Len(t, h.ledger.released, 1)
	assert.Equal(t, []ptoledgerservice.ReleasedUsage{
		{PTOType: worker.PTOTypeVacation, Days: decimal.NewFromInt(3)},
	}, h.ledger.released[0], "only approved, booked time off hands days back")
	paid := effectsOf(result.Effects, employment.EffectPTOPaidOut)
	require.Len(t, paid, 1)
	assert.True(
		t,
		paid[0].Days.Equal(decimal.NewFromFloat(6.5)),
		"the released days are paid out with the balance",
	)
	forfeited := effectsOf(result.Effects, employment.EffectPTOForfeited)
	require.Len(t, forfeited, 1)
	assert.True(t, forfeited[0].Days.Equal(decimal.NewFromInt(2)))

	require.Len(t, h.policies.ended, 1)
	assert.Equal(t, assignment.ID, h.policies.ended[0].AssignmentID)
	assert.Equal(t, effective, h.policies.ended[0].EffectiveTo)
	assert.Equal(t, int64(3), h.policies.ended[0].Version)
	assert.Equal(t, "Drivers", *effectsOf(result.Effects, employment.EffectPTOPolicyEnded)[0].Name)

	assert.Equal(t, []pulid.ID{payAssignment.ID}, h.pay.ended)
	assert.Equal(
		t,
		"Linehaul mileage",
		*effectsOf(result.Effects, employment.EffectPayAssignmentEnded)[0].Name,
	)

	event := result.Event
	assert.Equal(t, "Active", event.FromValues[employment.EmploymentValueStatus])
	assert.Equal(t, "Inactive", event.ToValues[employment.EmploymentValueStatus])
	assert.Equal(t, h.userID, event.RecordedByID)

	_, err = h.record(employment.EmploymentEventTerminated, effective+86400, nil)
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr, "a second termination is refused")
	assert.Equal(t, "kind", verr.Field())
}

// A failed write inside the transaction fails the record, and nothing that
// runs after commit is started for an event that never committed.
func TestRecord_AnAtomicFailureFailsTheRecordAndRunsNothingAfterIt(t *testing.T) {
	h := newHarness(t)
	h.payRepo.assignment = &payassignment.WorkerPayAssignment{
		ID:            pulid.MustNew("wpa_"),
		EffectiveFrom: 1_700_000_000,
	}
	h.payRepo.err = nil
	h.pay.err = errors.New("pay assignment is locked")

	_, err := h.record(employment.EmploymentEventTerminated, 1_800_000_000, nil)
	require.Error(t, err)
	assert.Empty(t, h.portal.revoked, "nothing after commit runs when the commit failed")
	assert.Empty(t, h.lists.spawned)
}

func TestRecord_TimeOffThatCannotBeCancelledIsReportedAsFailed(t *testing.T) {
	h := newHarness(t)
	stuck := &worker.WorkerPTO{
		ID:        pulid.MustNew("wrkpto_"),
		Status:    worker.PTOStatusRequested,
		Type:      worker.PTOTypePersonal,
		StartDate: 1_800_086_400,
		Days:      decimal.NewFromInt(1),
	}
	h.pto.upcoming = []*worker.WorkerPTO{stuck}
	h.pto.cancelErr = map[pulid.ID]error{stuck.ID: errors.New("version moved")}

	result, err := h.record(employment.EmploymentEventTerminated, 1_800_000_000, nil)
	require.NoError(t, err)

	cancelled := effectsOf(result.Effects, employment.EffectTimeOffCancelled)
	require.Len(t, cancelled, 1)
	assert.True(t, cancelled[0].Failed)
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
	assert.Equal(t, "reason", multiErr.Errors[0].Field())
	assert.Empty(t, h.workers.updated, "validation runs before any effect")
}

func TestRecord_RehireReopensAndReenrols(t *testing.T) {
	h := newHarness(t)
	h.wrk.Status = domaintypes.StatusInactive
	h.wrk.CanBeAssigned = false
	term := int64(1_750_000_000)
	h.wrk.Profile.TerminationDate = &term
	h.policies.defaultPolicy = &pto.PTOPolicy{ID: pulid.MustNew("ptop_"), Name: "Standard"}

	result, err := h.record(employment.EmploymentEventRehired, 1_790_000_000, nil)
	require.NoError(t, err)
	saved := h.workers.updated[0]
	assert.Equal(t, domaintypes.StatusActive, saved.Status)
	assert.True(t, saved.CanBeAssigned)
	assert.Nil(t, saved.Profile.TerminationDate)
	assert.Equal(t, int64(1_790_000_000), saved.Profile.HireDate)

	applied := effectsOf(result.Effects, employment.EffectDefaultPTOPolicyApplied)
	require.Len(t, applied, 1)
	assert.Equal(t, "Standard", *applied[0].Name)
	require.Len(t, h.policies.hired, 1)
	assert.Equal(t, h.policies.defaultPolicy.ID, h.policies.hired[0].PolicyID)
	assert.Equal(t, int64(1_790_000_000), h.policies.hired[0].EffectiveFrom)
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
	require.Len(t, h.workers.updated, 1)
	assert.Equal(t, target, h.workers.updated[0].FleetCodeID)
	assert.Equal(t, "SOUTH", result.Event.FromValues[employment.EmploymentValueFleetCode])
	assert.Equal(t, "NORTH", result.Event.ToValues[employment.EmploymentValueFleetCode])

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
	assert.Equal(t, "fleetCodeId", verr.Field())
}

func TestRecord_LeaveAndSuspensionPairing(t *testing.T) {
	h := newHarness(t)

	_, err := h.record(employment.EmploymentEventLeaveEnded, 1_800_000_000, nil)
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr, "cannot end a leave that never started")

	_, err = h.record(employment.EmploymentEventLeaveStarted, 1_800_000_000, nil)
	require.ErrorAs(t, err, &verr, "a leave must say what kind it is")
	assert.Equal(t, "leaveType", verr.Field())

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
	assert.Equal(
		t,
		[]employment.EffectKind{employment.EffectDispatchRemoved},
		kindsOf(started.Effects),
	)

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
	assert.Equal(
		t,
		[]employment.EffectKind{employment.EffectDispatchRestored},
		kindsOf(ended.Effects),
	)
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
	assert.Equal(t, "driverType", verr.Field())

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
	assert.Empty(t, result.Effects)
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
	assert.Equal(t, "amendmentNote", verr.Field())

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
	assert.Equal(t, "version", verr.Field())
}

// A termination closes the onboarding it interrupts before it opens the
// offboarding, and the office is told both happened, with their names.
func TestRecord_TerminatedClosesStaleChecklistsThenSpawns(t *testing.T) {
	h := newHarness(t)
	h.lists.open = []*workerchecklist.WorkerChecklist{{
		ID:     pulid.MustNew("wcl_"),
		Name:   "Driver onboarding",
		Kind:   workerchecklist.ChecklistKindOnboarding,
		Status: workerchecklist.ChecklistStatusOpen,
	}}
	h.lists.template = &workerchecklist.WorkerChecklistTemplate{
		ID:   pulid.MustNew("wclt_"),
		Name: "Driver offboarding",
	}

	result, err := h.record(
		employment.EmploymentEventTerminated,
		1_700_500_000,
		func(req *workeremploymentservice.RecordRequest) {
			req.Reason = "Resigned"
		},
	)
	require.NoError(t, err)

	closed := effectsOf(result.Effects, employment.EffectChecklistClosed)
	require.Len(t, closed, 1)
	assert.Equal(t, 1, *closed[0].Count)
	assert.Equal(t, "Driver onboarding", *closed[0].Name)
	started := effectsOf(result.Effects, employment.EffectChecklistStarted)
	require.Len(t, started, 1)
	assert.Equal(t, "Driver offboarding", *started[0].Name)
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
