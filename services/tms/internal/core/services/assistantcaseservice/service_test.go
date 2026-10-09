package assistantcaseservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testNow = int64(1_800_000_000)

type fakeConversations struct {
	repositories.ConversationRepository

	thread  *conversation.Thread
	updates int
}

func (f *fakeConversations) GetThread(
	_ context.Context,
	req repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	if f.thread == nil || req.ID != f.thread.ID || req.UserID != f.thread.UserID {
		return nil, errortypes.NewNotFoundError("Thread not found")
	}
	copied := *f.thread

	return &copied, nil
}

func (f *fakeConversations) UpdateThread(
	_ context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	f.updates++
	stored := *thread
	f.thread = &stored

	return thread, nil
}

type fakeCases struct {
	repositories.AssistantCaseRepository

	records map[pulid.ID]*deskcase.Record
	parties []deskcase.Party
	queries int
}

func (f *fakeCases) ListRecords(
	_ context.Context,
	req *repositories.ListCaseRecordsRequest,
) (map[pulid.ID]*deskcase.Record, error) {
	f.queries++
	out := make(map[pulid.ID]*deskcase.Record, len(req.Refs))
	for _, ref := range req.Refs {
		if record, ok := f.records[ref.ID]; ok {
			out[ref.ID] = record
		}
	}

	return out, nil
}

func (f *fakeCases) Parties(
	context.Context,
	*repositories.ListCasePartiesRequest,
) ([]deskcase.Party, error) {
	return f.parties, nil
}

type fakeWaitRepo struct {
	repositories.AgentWaitRepository

	open map[pulid.ID][]*agentwait.Wait
}

func (f *fakeWaitRepo) ListOpenByThreads(
	_ context.Context,
	req *repositories.ListOpenWaitsByThreadsRequest,
) (map[pulid.ID][]*agentwait.Wait, error) {
	out := make(map[pulid.ID][]*agentwait.Wait, len(req.ThreadIDs))
	for _, id := range req.ThreadIDs {
		if waits, ok := f.open[id]; ok {
			out[id] = waits
		}
	}

	return out, nil
}

type fakeSubjects struct {
	repositories.AgentSubjectRepository

	exists bool
}

func (f *fakeSubjects) Exists(context.Context, repositories.AgentSubjectExistsRequest) (bool, error) {
	return f.exists, nil
}

type fakePermissions struct {
	serviceports.PermissionEngine

	allowed bool
}

func (f *fakePermissions) Check(
	context.Context,
	*serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	return &serviceports.PermissionCheckResult{Allowed: f.allowed}, nil
}

type fakeAgentWaits struct {
	serviceports.AgentWaitService

	registered *serviceports.RegisterWaitRequest
}

func (f *fakeAgentWaits) Register(
	_ context.Context,
	req *serviceports.RegisterWaitRequest,
) (*agentwait.Wait, error) {
	f.registered = req

	return &agentwait.Wait{ID: pulid.MustNew("awt_"), Kind: req.Kind}, nil
}

type fakeChecklists struct {
	repositories.CaseChecklistRepository

	template *deskcase.ChecklistTemplate
	asked    pulid.ID
	ticked   *deskcase.ChecklistTick
	unticked *repositories.UntickCaseChecklistItemRequest
}

func (f *fakeChecklists) TemplateFor(
	_ context.Context,
	req *repositories.GetCaseChecklistTemplateForRequest,
) (*deskcase.ChecklistTemplate, error) {
	f.asked = req.CustomerID

	return f.template, nil
}

func (f *fakeChecklists) Tick(_ context.Context, entity *deskcase.ChecklistTick) error {
	f.ticked = entity

	return nil
}

func (f *fakeChecklists) Untick(
	_ context.Context,
	req *repositories.UntickCaseChecklistItemRequest,
) error {
	f.unticked = req

	return nil
}

type harness struct {
	svc           *Service
	conversations *fakeConversations
	cases         *fakeCases
	waits         *fakeWaitRepo
	subjects      *fakeSubjects
	permissions   *fakePermissions
	agentWaits    *fakeAgentWaits
	checklists    *fakeChecklists
	thread        *conversation.Thread
	req           serviceports.CaseThreadRequest
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	userID := pulid.MustNew("usr_")
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	thread := &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		UserID:            userID,
		AgentDefinitionID: pulid.MustNew("agd_"),
		Status:            conversation.ThreadStatusActive,
		Origin:            conversation.ThreadOriginDesk,
	}
	h := &harness{
		conversations: &fakeConversations{thread: thread},
		cases:         &fakeCases{records: map[pulid.ID]*deskcase.Record{}},
		waits:         &fakeWaitRepo{open: map[pulid.ID][]*agentwait.Wait{}},
		subjects:      &fakeSubjects{exists: true},
		permissions:   &fakePermissions{allowed: true},
		agentWaits:    &fakeAgentWaits{},
		checklists:    &fakeChecklists{},
		thread:        thread,
	}
	states := &States{
		cases: h.cases,
		waits: h.waits,
		now:   func() int64 { return testNow },
		l:     zap.NewNop(),
	}
	h.svc = &Service{
		States:        states,
		conversations: h.conversations,
		subjects:      h.subjects,
		agentWaits:    h.agentWaits,
		checklists:    h.checklists,
		permissions:   h.permissions,
	}
	h.req = serviceports.CaseThreadRequest{
		ThreadID:   thread.ID,
		TenantInfo: tenant,
		Actor: &serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeUser,
			PrincipalID:    userID,
			UserID:         userID,
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
		},
	}

	return h
}

func (h *harness) bindShipment(t *testing.T) *deskcase.Record {
	t.Helper()

	record := &deskcase.Record{
		Type:       agent.SubjectShipment,
		ID:         pulid.MustNew("shp_"),
		Label:      "10293",
		Status:     "InTransit",
		CustomerID: pulid.MustNew("cus_"),
		CarrierIDs: []pulid.ID{pulid.MustNew("car_")},
	}
	h.cases.records[record.ID] = record
	h.conversations.thread.SubjectType = agent.SubjectShipment
	h.conversations.thread.SubjectID = record.ID

	return record
}

func TestBind_MakesTheConversationACase(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	shipmentID := pulid.MustNew("shp_")
	h.cases.records[shipmentID] = &deskcase.Record{Type: agent.SubjectShipment, ID: shipmentID}

	thread, err := h.svc.Bind(t.Context(), &serviceports.BindCaseRequest{
		CaseThreadRequest: h.req,
		SubjectType:       agent.SubjectShipment,
		SubjectID:         shipmentID,
	})

	require.NoError(t, err)
	assert.Equal(t, shipmentID, thread.SubjectID)
	require.NotNil(t, thread.Case)
	assert.Equal(t, deskcase.StateWorking, thread.Case.State)
	assert.Equal(t, 1, h.conversations.updates)
}

func TestBind_Refusals(t *testing.T) {
	t.Parallel()

	cases := map[string]func(h *harness) *serviceports.BindCaseRequest{
		"not a case subject": func(h *harness) *serviceports.BindCaseRequest {
			return &serviceports.BindCaseRequest{
				CaseThreadRequest: h.req,
				SubjectType:       agent.SubjectWorker,
				SubjectID:         pulid.MustNew("wrk_"),
			}
		},
		"an id of another kind": func(h *harness) *serviceports.BindCaseRequest {
			return &serviceports.BindCaseRequest{
				CaseThreadRequest: h.req,
				SubjectType:       agent.SubjectInvoice,
				SubjectID:         pulid.MustNew("shp_"),
			}
		},
		"a record the person may not read": func(h *harness) *serviceports.BindCaseRequest {
			h.permissions.allowed = false
			return &serviceports.BindCaseRequest{
				CaseThreadRequest: h.req,
				SubjectType:       agent.SubjectInvoice,
				SubjectID:         pulid.MustNew("inv_"),
			}
		},
		"a record that is not the organization's": func(h *harness) *serviceports.BindCaseRequest {
			h.subjects.exists = false
			return &serviceports.BindCaseRequest{
				CaseThreadRequest: h.req,
				SubjectType:       agent.SubjectInvoiceDispute,
				SubjectID:         pulid.MustNew("idsp_"),
			}
		},
		"a page-bound conversation": func(h *harness) *serviceports.BindCaseRequest {
			h.conversations.thread.Origin = conversation.ThreadOriginImport
			return &serviceports.BindCaseRequest{
				CaseThreadRequest: h.req,
				SubjectType:       agent.SubjectShipment,
				SubjectID:         pulid.MustNew("shp_"),
			}
		},
		"someone else's conversation": func(h *harness) *serviceports.BindCaseRequest {
			req := h.req
			req.Actor = &serviceports.RequestActor{UserID: pulid.MustNew("usr_")}
			return &serviceports.BindCaseRequest{
				CaseThreadRequest: req,
				SubjectType:       agent.SubjectShipment,
				SubjectID:         pulid.MustNew("shp_"),
			}
		},
	}

	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)

			_, err := h.svc.Bind(t.Context(), build(h))

			require.Error(t, err)
			assert.Zero(t, h.conversations.updates)
		})
	}
}

func TestBind_MovingTheCaseDropsTheSnooze(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.bindShipment(t)
	until := testNow + 3600
	h.conversations.thread.SnoozedUntil = &until
	h.conversations.thread.SnoozeAnchor = deskcase.SnoozeTime

	invoiceID := pulid.MustNew("inv_")
	thread, err := h.svc.Bind(t.Context(), &serviceports.BindCaseRequest{
		CaseThreadRequest: h.req,
		SubjectType:       agent.SubjectInvoice,
		SubjectID:         invoiceID,
	})

	require.NoError(t, err)
	assert.Nil(t, thread.SnoozedUntil)
	assert.Empty(t, thread.SnoozeAnchor)
	assert.True(t, thread.Case.Record.Gone(), "the fake has no such invoice to read")
}

func TestSnooze_Time(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.bindShipment(t)

	_, err := h.svc.Snooze(t.Context(), &serviceports.SnoozeCaseRequest{
		CaseThreadRequest: h.req, Anchor: deskcase.SnoozeTime, Until: testNow + 10,
	})
	require.Error(t, err, "a snooze ending before the Desk draws it is refused")

	thread, err := h.svc.Snooze(t.Context(), &serviceports.SnoozeCaseRequest{
		CaseThreadRequest: h.req, Anchor: deskcase.SnoozeTime, Until: testNow + 3600,
	})
	require.NoError(t, err)
	assert.Equal(t, deskcase.StateSnoozed, thread.Case.State)
	assert.Equal(t, testNow+3600, *thread.SnoozedUntil)
}

func TestSnooze_AppointmentFollowsTheNextStop(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	record := h.bindShipment(t)

	_, err := h.svc.Snooze(t.Context(), &serviceports.SnoozeCaseRequest{
		CaseThreadRequest: h.req, Anchor: deskcase.SnoozeAppointment,
	})
	require.Error(t, err, "no stop ahead")

	record.NextStopID = pulid.MustNew("stp_")
	appointment := testNow + 7200
	record.NextAppointmentAt = &appointment
	thread, err := h.svc.Snooze(t.Context(), &serviceports.SnoozeCaseRequest{
		CaseThreadRequest: h.req, Anchor: deskcase.SnoozeAppointment,
	})

	require.NoError(t, err)
	assert.Equal(t, record.NextStopID, thread.SnoozeStopID)
	assert.Equal(t, appointment, *thread.Case.SnoozedUntil)
}

func TestSnooze_RefusesASettledCase(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	record := h.bindShipment(t)
	record.Closed = true

	_, err := h.svc.Snooze(t.Context(), &serviceports.SnoozeCaseRequest{
		CaseThreadRequest: h.req, Anchor: deskcase.SnoozeTime, Until: testNow + 3600,
	})

	require.Error(t, err)
	assert.Zero(t, h.conversations.updates)
}

func TestAwaitReply_RegistersAReplyWaitOnTheRecordsOwnParty(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	record := h.bindShipment(t)
	h.cases.parties = []deskcase.Party{
		{Kind: deskcase.WaitingOnCustomer, ID: record.CustomerID, Name: "Acme"},
		{Kind: deskcase.WaitingOnCarrier, ID: record.CarrierIDs[0], Name: "Swift"},
	}

	_, err := h.svc.AwaitReply(t.Context(), &serviceports.AwaitCaseReplyRequest{
		CaseThreadRequest: h.req,
		Party:             deskcase.WaitingOnCarrier,
		PartyID:           pulid.MustNew("car_"),
	})
	require.Error(t, err, "a carrier not on the shipment is refused")
	require.Nil(t, h.agentWaits.registered)

	_, err = h.svc.AwaitReply(t.Context(), &serviceports.AwaitCaseReplyRequest{
		CaseThreadRequest: h.req,
		Party:             deskcase.WaitingOnCarrier,
		PartyID:           record.CarrierIDs[0],
		GiveUpAfterHours:  500,
	})
	require.NoError(t, err)

	registered := h.agentWaits.registered
	require.NotNil(t, registered)
	assert.Equal(t, agentwait.KindReply, registered.Kind)
	assert.Equal(t, record.CarrierIDs[0], registered.Condition.CarrierID)
	assert.True(t, registered.Condition.CustomerID.IsNil())
	assert.Equal(t, h.thread.ID, registered.ThreadID)
	assert.Equal(t, h.thread.AgentDefinitionID, registered.DefinitionID)
	assert.Equal(t, int64(maxReplyHours*secondsPerHour), registered.LifetimeSeconds)
	assert.Contains(t, registered.Description, "Swift")
}

func TestAttach_OneQueryForThePageAndWaitingNamesTheParty(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	record := h.bindShipment(t)
	caseThread := *h.conversations.thread
	plain := &conversation.Thread{ID: pulid.MustNew("athr_"), UserID: caseThread.UserID}
	h.waits.open[caseThread.ID] = []*agentwait.Wait{{
		Kind:      agentwait.KindReply,
		ThreadID:  caseThread.ID,
		Condition: &agentwait.Condition{CustomerID: record.CustomerID},
	}}

	err := h.svc.Attach(t.Context(), h.req.TenantInfo, caseThread.UserID,
		[]*conversation.Thread{&caseThread, plain})

	require.NoError(t, err)
	assert.Equal(t, 1, h.cases.queries)
	require.NotNil(t, caseThread.Case)
	assert.Equal(t, deskcase.StateWaiting, caseThread.Case.State)
	assert.Equal(t, deskcase.WaitingOnCustomer, caseThread.Case.WaitingOn)
	assert.Nil(t, plain.Case, "a conversation about nothing is no case")
}

func TestUnbind_RefusesAConversationThatIsNoCase(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, err := h.svc.Unbind(t.Context(), &h.req)

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestTick_OnlyAManualAddedStepOfTheTemplateThatApplies(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	record := h.bindShipment(t)
	record.BillToID = pulid.MustNew("cus_")
	manual := deskcase.ItemKey("custom:callshipper")
	h.checklists.template = &deskcase.ChecklistTemplate{
		Kind: deskcase.ChecklistReadyToBill,
		Items: append(deskcase.DefaultItems(deskcase.ChecklistReadyToBill), deskcase.TemplateItem{
			Key: manual, Mode: deskcase.ModeRequired,
			Custom: &deskcase.CustomItem{Label: "Shipper called", Check: deskcase.CheckManual},
		}),
	}

	_, err := h.svc.Tick(t.Context(), &serviceports.TickCaseItemRequest{
		CaseThreadRequest: h.req,
		ItemKey:           deskcase.ItemPOD,
		Ticked:            true,
	})
	require.Error(t, err, "a built-in step is ticked by the record, not a person")
	assert.Nil(t, h.checklists.ticked)
	assert.Equal(t, record.BillToID, h.checklists.asked, "the bill-to customer's template applies")

	_, err = h.svc.Tick(t.Context(), &serviceports.TickCaseItemRequest{
		CaseThreadRequest: h.req,
		ItemKey:           deskcase.ItemKey("custom:notonit"),
		Ticked:            true,
	})
	require.Error(t, err, "a step the template does not hold cannot be ticked")
	assert.Nil(t, h.checklists.ticked)
}

func TestTick_RefusesADisputesCase(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	disputeID := pulid.MustNew("idsp_")
	h.cases.records[disputeID] = &deskcase.Record{Type: agent.SubjectInvoiceDispute, ID: disputeID}
	h.conversations.thread.SubjectType = agent.SubjectInvoiceDispute
	h.conversations.thread.SubjectID = disputeID

	_, err := h.svc.Tick(t.Context(), &serviceports.TickCaseItemRequest{
		CaseThreadRequest: h.req,
		ItemKey:           deskcase.ItemKey("custom:callshipper"),
		Ticked:            true,
	})

	require.Error(t, err)
	assert.Nil(t, h.checklists.ticked)
}
