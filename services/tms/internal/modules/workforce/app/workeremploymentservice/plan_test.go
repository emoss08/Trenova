package workeremploymentservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeremploymentservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type comparableEffect struct {
	Kind    employment.EffectKind
	Name    string
	Count   int
	Days    string
	PTOType worker.PTOType
}

func comparableEffects(effects []*employment.Effect) []comparableEffect {
	out := make([]comparableEffect, 0, len(effects))
	for _, effect := range effects {
		c := comparableEffect{Kind: effect.Kind}
		if effect.Name != nil {
			c.Name = *effect.Name
		}
		if effect.Count != nil {
			c.Count = *effect.Count
		}
		if effect.Days != nil {
			c.Days = effect.Days.String()
		}
		if effect.PTOType != nil {
			c.PTOType = *effect.PTOType
		}
		out = append(out, c)
	}
	return out
}

func (h *harness) seedTermination() {
	h.ledger.assignment = &pto.WorkerPTOPolicyAssignment{
		ID:            pulid.MustNew("wppa_"),
		EffectiveFrom: 1_700_000_000,
	}
	h.ledger.policy = &pto.PTOPolicy{
		ID:   pulid.MustNew("ptop_"),
		Name: "Drivers",
		Rules: []*pto.PTOPolicyRule{
			{PTOType: worker.PTOTypeVacation, OnTermination: pto.PTOTerminationPayOut},
		},
	}
	h.ledger.balances = []*pto.WorkerPTOBalance{
		{PTOType: worker.PTOTypeVacation, BalanceDays: decimal.NewFromInt(5)},
	}
	h.pto.upcoming = []*worker.WorkerPTO{{
		ID:        pulid.MustNew("wrkpto_"),
		Status:    worker.PTOStatusRequested,
		Type:      worker.PTOTypeVacation,
		StartDate: 1_800_086_400,
		EndDate:   1_800_172_800,
		Days:      decimal.NewFromInt(2),
	}}
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
}

// The preview runs the plan the write runs: what it lists is exactly what
// recording the event then carries out, kind by kind.
func TestPreview_EqualsWhatRecordThenDoes(t *testing.T) {
	leave := worker.LeaveTypeMedical
	otr := worker.DriverTypeOTR
	cases := []struct {
		name   string
		kind   employment.EmploymentEventKind
		setup  func(h *harness)
		mutate func(req *workeremploymentservice.RecordRequest)
	}{
		{
			name:  "terminated",
			kind:  employment.EmploymentEventTerminated,
			setup: (*harness).seedTermination,
		},
		{
			name: "rehired",
			kind: employment.EmploymentEventRehired,
			setup: func(h *harness) {
				h.wrk.Status = domaintypes.StatusInactive
				h.wrk.CanBeAssigned = false
				h.policies.defaultPolicy = &pto.PTOPolicy{
					ID:   pulid.MustNew("ptop_"),
					Name: "Standard",
				}
				h.training.courses = map[worker.DriverType][]*workertraining.TrainingCourse{
					worker.DriverTypeLocal: {{ID: pulid.MustNew("wtc_"), Name: "Orientation"}},
				}
				h.lists.template = &workerchecklist.WorkerChecklistTemplate{
					ID:   pulid.MustNew("wclt_"),
					Name: "Rehire",
				}
			},
		},
		{
			name:   "leave started",
			kind:   employment.EmploymentEventLeaveStarted,
			mutate: func(req *workeremploymentservice.RecordRequest) { req.LeaveType = &leave },
		},
		{
			name: "promoted",
			kind: employment.EmploymentEventPromoted,
			setup: func(h *harness) {
				h.training.courses = map[worker.DriverType][]*workertraining.TrainingCourse{
					worker.DriverTypeOTR: {{ID: pulid.MustNew("wtc_"), Name: "Mountain driving"}},
				}
			},
			mutate: func(req *workeremploymentservice.RecordRequest) { req.DriverType = &otr },
		},
		{name: "probation ended", kind: employment.EmploymentEventProbationEnded},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			if tc.setup != nil {
				tc.setup(h)
			}
			req := h.request(tc.kind, 1_800_000_000, tc.mutate)

			preview, err := h.svc.Preview(t.Context(), req)
			require.NoError(t, err)
			assert.Empty(t, h.events.order, "a preview writes nothing")
			assert.Empty(t, h.workers.updated)
			assert.Empty(t, h.pto.cancelled)
			assert.Empty(t, h.ledger.settled)
			assert.Empty(t, h.portal.revoked)
			assert.Empty(t, h.lists.spawned)
			assert.Empty(t, h.training.assigned)
			assert.Empty(t, h.policies.hired)

			result, err := h.svc.Record(t.Context(), req)
			require.NoError(t, err)

			assert.Equal(t, comparableEffects(preview.Effects), comparableEffects(result.Effects))
			assert.Equal(t, preview.FromValues, result.Event.FromValues)
			assert.Equal(t, preview.ToValues, result.Event.ToValues)
			for _, effect := range preview.Effects {
				assert.False(t, effect.Failed, "a preview never reports a failure")
			}
		})
	}
}

func TestPreview_TerminationListsEveryEffectInOrder(t *testing.T) {
	h := newHarness(t)
	h.seedTermination()

	preview, err := h.svc.Preview(
		t.Context(),
		h.request(employment.EmploymentEventTerminated, 1_800_000_000, nil),
	)
	require.NoError(t, err)

	assert.Equal(t, []comparableEffect{
		{Kind: employment.EffectTimeOffCancelled, Days: "2", PTOType: worker.PTOTypeVacation},
		{Kind: employment.EffectPTOPaidOut, Days: "5", PTOType: worker.PTOTypeVacation},
		{Kind: employment.EffectPTOPolicyEnded, Name: "Drivers"},
		{Kind: employment.EffectPortalAccessRevoked},
		{Kind: employment.EffectChecklistClosed, Name: "Driver onboarding", Count: 1},
		{Kind: employment.EffectChecklistStarted, Name: "Driver offboarding"},
	}, comparableEffects(preview.Effects))
	assert.Equal(t, employment.EmploymentEventTerminated, preview.Kind)
	assert.Equal(t, int64(1_800_000_000), preview.EffectiveAt)
}

// A rehire in a tenant with no default policy applies none, so it must not
// say it did.
func TestRecord_RehireWithoutADefaultPolicyAppliesNone(t *testing.T) {
	h := newHarness(t)
	h.wrk.Status = domaintypes.StatusInactive
	h.wrk.CanBeAssigned = false

	result, err := h.record(employment.EmploymentEventRehired, 1_790_000_000, nil)
	require.NoError(t, err)

	assert.Empty(t, effectsOf(result.Effects, employment.EffectDefaultPTOPolicyApplied))
	assert.Empty(t, h.policies.hired)
}

func TestRecord_DefaultPolicyThatFailsToApplyIsReportedAsFailed(t *testing.T) {
	h := newHarness(t)
	h.wrk.Status = domaintypes.StatusInactive
	h.policies.defaultPolicy = &pto.PTOPolicy{ID: pulid.MustNew("ptop_"), Name: "Standard"}
	h.policies.hireErr = errortypes.NewBusinessError("policy archived")

	result, err := h.record(employment.EmploymentEventRehired, 1_790_000_000, nil)
	require.NoError(t, err)

	applied := effectsOf(result.Effects, employment.EffectDefaultPTOPolicyApplied)
	require.Len(t, applied, 1)
	assert.True(t, applied[0].Failed)
}

// Promotion to a driver type that needs courses plans them against the worker
// the event makes, not the stored one, so an OTR course is assigned when a
// local driver becomes OTR.
func TestRecord_PromotionPlansTrainingAgainstTheProjectedWorker(t *testing.T) {
	h := newHarness(t)
	mountain := &workertraining.TrainingCourse{ID: pulid.MustNew("wtc_"), Name: "Mountain driving"}
	h.training.courses = map[worker.DriverType][]*workertraining.TrainingCourse{
		worker.DriverTypeOTR: {mountain},
	}
	otr := worker.DriverTypeOTR

	result, err := h.record(
		employment.EmploymentEventPromoted,
		1_800_000_000,
		func(req *workeremploymentservice.RecordRequest) { req.DriverType = &otr },
	)
	require.NoError(t, err)

	assert.Equal(t, []worker.DriverType{worker.DriverTypeOTR}, h.training.plannedOn)
	assigned := effectsOf(result.Effects, employment.EffectTrainingAssigned)
	require.Len(t, assigned, 1)
	assert.Equal(t, 1, *assigned[0].Count)
	assert.Equal(t, [][]*workertraining.TrainingCourse{{mountain}}, h.training.assigned)
}

func TestRecord_TrainingThatCannotBeAssignedIsReportedAsFailed(t *testing.T) {
	h := newHarness(t)
	h.training.courses = map[worker.DriverType][]*workertraining.TrainingCourse{
		worker.DriverTypeOTR: {{ID: pulid.MustNew("wtc_"), Name: "Mountain driving"}},
	}
	h.training.failAll = true
	otr := worker.DriverTypeOTR

	result, err := h.record(
		employment.EmploymentEventPromoted,
		1_800_000_000,
		func(req *workeremploymentservice.RecordRequest) { req.DriverType = &otr },
	)
	require.NoError(t, err)

	assigned := effectsOf(result.Effects, employment.EffectTrainingAssigned)
	require.Len(t, assigned, 1)
	assert.True(t, assigned[0].Failed)
	assert.Equal(t, 0, *assigned[0].Count)
}

func (h *harness) member(user *tenant.User) {
	h.users.EXPECT().
		GetTenantMember(mock.Anything, mock.MatchedBy(func(req tenantports.GetTenantMemberRequest) bool {
			return req.UserID == user.ID && req.TenantInfo == h.tenant
		})).
		Return(user, nil).
		Maybe()
}

// A transfer to a new manager moves the worker's team, which is who may see
// and act for them: it is validated, written to both sides of the event by
// name, and recorded as an access change.
func TestRecord_TransferToANewManagerIsAnAuditedAccessChange(t *testing.T) {
	h := newHarness(t)
	previous := &tenant.User{ID: pulid.MustNew("usr_"), Name: "Dana Ortiz"}
	h.wrk.ManagerID = previous.ID
	h.wrk.Manager = previous
	next := &tenant.User{
		ID:     pulid.MustNew("usr_"),
		Name:   "Sam Reyes",
		Status: domaintypes.StatusActive,
	}
	h.member(next)

	result, err := h.record(
		employment.EmploymentEventTransferred,
		1_800_000_000,
		func(req *workeremploymentservice.RecordRequest) { req.ManagerID = &next.ID },
	)
	require.NoError(t, err)

	require.Len(t, h.workers.updated, 1)
	assert.Equal(t, next.ID, h.workers.updated[0].ManagerID)
	assert.Equal(
		t,
		previous.ID.String(),
		result.Event.FromValues[employment.EmploymentValueManagerID],
	)
	assert.Equal(t, "Dana Ortiz", result.Event.FromValues[employment.EmploymentValueManager])
	assert.Equal(t, next.ID.String(), result.Event.ToValues[employment.EmploymentValueManagerID])
	assert.Equal(t, "Sam Reyes", result.Event.ToValues[employment.EmploymentValueManager])

	require.Len(t, h.security.changes, 1)
	change := h.security.changes[0]
	assert.Equal(t, permission.ResourceWorker, change.Resource)
	assert.Equal(t, h.wrk.ID.String(), change.ResourceID)
	assert.Equal(t, "worker.team_changed", change.Action)
	assert.Equal(t, h.userID, change.Actor.UserID)
	assert.Equal(t, result.Event.ID.String(), change.Metadata["employmentEventId"])
}

func TestRecord_TransferRefusesAManagerWhoIsNotAnActiveMember(t *testing.T) {
	h := newHarness(t)
	inactive := &tenant.User{
		ID:     pulid.MustNew("usr_"),
		Name:   "Gone",
		Status: domaintypes.StatusInactive,
	}
	h.member(inactive)
	stranger := pulid.MustNew("usr_")
	h.users.EXPECT().
		GetTenantMember(mock.Anything, mock.MatchedBy(func(req tenantports.GetTenantMemberRequest) bool {
			return req.UserID == stranger
		})).
		Return(nil, errortypes.NewNotFoundError("user")).
		Maybe()

	for _, managerID := range []pulid.ID{inactive.ID, stranger} {
		_, err := h.record(
			employment.EmploymentEventTransferred,
			1_800_000_000,
			func(req *workeremploymentservice.RecordRequest) { req.ManagerID = &managerID },
		)
		var verr *errortypes.Error
		require.ErrorAs(t, err, &verr)
		assert.Equal(t, "managerId", verr.Field())
	}
	assert.Empty(t, h.workers.updated)
	assert.Empty(t, h.events.order)
	assert.Empty(t, h.security.changes)
}

func TestRecord_TransferToTheSameManagerIsRefused(t *testing.T) {
	h := newHarness(t)
	current := pulid.MustNew("usr_")
	h.wrk.ManagerID = current

	_, err := h.record(
		employment.EmploymentEventTransferred,
		1_800_000_000,
		func(req *workeremploymentservice.RecordRequest) { req.ManagerID = &current },
	)
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "managerId", verr.Field())
}

func TestRecord_TransferNeedsAFleetOrAManager(t *testing.T) {
	h := newHarness(t)

	_, err := h.record(employment.EmploymentEventTransferred, 1_800_000_000, nil)
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "fleetCodeId", verr.Field())
	assert.Equal(t, errortypes.ErrRequired, verr.Code)
}

// Creating a worker opens their timeline the way a recorded hire does: the
// Hired event is audited, the hire checklist starts and the required courses
// are assigned.
func TestRecordHired_AuditsAndAssignsRequiredTraining(t *testing.T) {
	h := newHarness(t)
	orientation := &workertraining.TrainingCourse{ID: pulid.MustNew("wtc_"), Name: "Orientation"}
	h.training.courses = map[worker.DriverType][]*workertraining.TrainingCourse{
		worker.DriverTypeLocal: {orientation},
	}
	h.lists.template = &workerchecklist.WorkerChecklistTemplate{
		ID:   pulid.MustNew("wclt_"),
		Name: "Driver onboarding",
	}

	err := h.svc.RecordHired(t.Context(), h.wrk, h.userID)
	require.NoError(t, err)

	require.Len(t, h.events.order, 1)
	hired := h.events.events[h.events.order[0]]
	assert.Equal(t, employment.EmploymentEventHired, hired.Kind)
	assert.Equal(t, [][]*workertraining.TrainingCourse{{orientation}}, h.training.assigned)
	assert.Equal(
		t,
		[]employment.EmploymentEventKind{employment.EmploymentEventHired},
		h.lists.spawned,
	)
}

func TestRecordHired_WithoutAHireDateRecordsNothing(t *testing.T) {
	h := newHarness(t)
	h.wrk.Profile.HireDate = 0

	require.NoError(t, h.svc.RecordHired(t.Context(), h.wrk, h.userID))
	assert.Empty(t, h.events.order)
	assert.Empty(t, h.training.assigned)
}

func TestOptions_ComeFromTheDomainRule(t *testing.T) {
	h := newHarness(t)

	options, err := h.svc.Options(t.Context(), h.tenant, h.wrk.ID)
	require.NoError(t, err)

	assert.Equal(t, employment.RecordableKinds(employment.EmploymentState{
		Status:        domaintypes.StatusActive,
		CanBeAssigned: true,
		HireDate:      1_600_000_000,
	}), options.RecordableKinds)
	assert.NotContains(t, options.RecordableKinds, employment.EmploymentEventHired)
	assert.Equal(t, employment.ReasonRequiredKinds(), options.ReasonRequiredKinds)
}
