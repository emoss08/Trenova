package agentproposalnotifier

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRoles struct {
	users    []repositories.PermittedUser
	requests []repositories.ListUsersWithPermissionRequest
}

func (f *fakeRoles) ListUsersWithPermission(
	_ context.Context,
	req repositories.ListUsersWithPermissionRequest,
) ([]repositories.PermittedUser, error) {
	f.requests = append(f.requests, req)

	return f.users, nil
}

type fakeNotifications struct{ created []*notification.Notification }

func (f *fakeNotifications) Create(
	_ context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	f.created = append(f.created, entity)

	return entity, nil
}

type fakeMailer struct{ sent []*services.SendEmailRequest }

func (f *fakeMailer) Send(_ context.Context, req *services.SendEmailRequest) (*email.Message, error) {
	f.sent = append(f.sent, req)

	return &email.Message{}, nil
}

type fakeRenderer struct {
	requests []*services.RenderMessageRequest
}

func (f *fakeRenderer) RenderMessage(
	_ context.Context,
	req *services.RenderMessageRequest,
) (*services.RenderedMessage, error) {
	f.requests = append(f.requests, req)

	return &services.RenderedMessage{Subject: "reminder", HTML: "<p>x</p>", Text: "x"}, nil
}

type fakeShadow struct{ shadow bool }

func (f fakeShadow) Organization(context.Context, pagination.TenantInfo) (bool, error) {
	return f.shadow, nil
}

type fakeProposals struct {
	pending  []*agent.AgentProposal
	reminded []pulid.ID
	lastList repositories.ListPendingProposalsForReminderRequest
}

func (f *fakeProposals) ListPendingForReminder(
	_ context.Context,
	req repositories.ListPendingProposalsForReminderRequest,
) ([]*agent.AgentProposal, error) {
	f.lastList = req

	return f.pending, nil
}

func (f *fakeProposals) MarkReminded(
	_ context.Context,
	req repositories.MarkProposalsRemindedRequest,
) (int, error) {
	f.reminded = append(f.reminded, req.IDs...)

	return len(req.IDs), nil
}

type fakeRuns struct{ runs map[pulid.ID]*agent.AgentRun }

func (f fakeRuns) GetByID(_ context.Context, req repositories.GetAgentRunByIDRequest) (*agent.AgentRun, error) {
	return f.runs[req.ID], nil
}

type fakeDefinitions struct{ definition *agentdefinition.Definition }

func (f fakeDefinitions) GetByID(
	context.Context,
	repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	return f.definition, nil
}

type harness struct {
	svc           *Service
	roles         *fakeRoles
	notifications *fakeNotifications
	mailer        *fakeMailer
	renderer      *fakeRenderer
	proposals     *fakeProposals
	definition    *agentdefinition.Definition
	run           *agent.AgentRun
}

func newHarness(shadow bool) *harness {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	definition := &agentdefinition.Definition{
		ID:             pulid.MustNew("agdef_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Name:           "Dispatch coverage",
	}
	run := &agent.AgentRun{
		ID:                pulid.MustNew("arun_"),
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		AgentDefinitionID: definition.ID,
		Trigger:           agent.RunTriggerScheduled,
		StartedAt:         1_700_000_000,
	}

	h := &harness{
		roles: &fakeRoles{users: []repositories.PermittedUser{
			{UserID: pulid.MustNew("usr_"), Name: "Dana Ortiz", EmailAddress: "dana@example.com", Locale: "en"},
			{UserID: pulid.MustNew("usr_"), Name: "Lee Park", EmailAddress: "", Locale: "es"},
		}},
		notifications: &fakeNotifications{},
		mailer:        &fakeMailer{},
		renderer:      &fakeRenderer{},
		proposals:     &fakeProposals{},
		definition:    definition,
		run:           run,
	}
	h.svc = &Service{
		l:             zap.NewNop(),
		webBaseURL:    "https://app.example.com",
		roles:         h.roles,
		proposals:     h.proposals,
		runs:          fakeRuns{runs: map[pulid.ID]*agent.AgentRun{run.ID: run}},
		definitions:   fakeDefinitions{definition: definition},
		shadow:        fakeShadow{shadow: shadow},
		notifications: h.notifications,
		email:         h.mailer,
		templates:     h.renderer,
	}

	return h
}

func (h *harness) proposal(tool string, status agent.ProposalStatus, createdAt int64) *agent.AgentProposal {
	return &agent.AgentProposal{
		ID:             pulid.MustNew("ap_"),
		OrganizationID: h.run.OrganizationID,
		BusinessUnitID: h.run.BusinessUnitID,
		RunID:          h.run.ID,
		ToolName:       tool,
		Rationale:      "because",
		Status:         status,
		CreatedAt:      createdAt,
	}
}

func TestNotifyPending_TellsEveryDeciderOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(false)
	require.NoError(t, h.svc.NotifyPending(t.Context(), services.PendingProposalsNotice{
		Definition: h.definition,
		Run:        h.run,
		Proposals: []*agent.AgentProposal{
			h.proposal("assign_move", agent.ProposalStatusPending, 1),
			h.proposal("add_shipment_comment", agent.ProposalStatusPending, 2),
			h.proposal("notify_driver", agent.ProposalStatusExecuted, 3),
		},
	}))

	require.Len(t, h.roles.requests, 1)
	assert.Equal(t, permission.ResourceAgentProposal, h.roles.requests[0].Resource)
	assert.Equal(t, permission.OpUpdate, h.roles.requests[0].Operation)

	require.Len(t, h.notifications.created, 2, "one notice per decider")
	first := h.notifications.created[0]
	assert.Equal(t, EventProposalsPending, first.EventType)
	assert.Equal(t, notification.ChannelUser, first.Channel)
	assert.Equal(t, h.roles.users[0].UserID, *first.TargetUserID)
	assert.Equal(t, "Dispatch coverage has 2 changes waiting for a decision", first.Title)
	assert.Equal(t, "It proposed assign move and add shipment comment. Nothing runs until someone decides.", first.Message)
	assert.Equal(t, 2, first.Data["count"], "the executed proposal is not waiting on anyone")

	link, ok := first.Data["link"].(string)
	require.True(t, ok)
	parsed, err := url.Parse(link)
	require.NoError(t, err)
	assert.Equal(t, "/admin/agent-control", parsed.Path)
	assert.Equal(t, "proposals", parsed.Query().Get("activity"))
	assert.JSONEq(t,
		`[{"field":"runId","operator":"eq","value":"`+h.run.ID.String()+`"}]`,
		parsed.Query().Get("fieldFilters"),
		"the link lands on the proposals filtered to this run",
	)

	assert.Empty(t, h.mailer.sent, "the first notice is in-app only")
}

func TestNotifyPending_SkipsChatRunsAndShadow(t *testing.T) {
	t.Parallel()

	h := newHarness(false)
	chat := *h.run
	chat.Trigger = agent.RunTriggerChat
	require.NoError(t, h.svc.NotifyPending(t.Context(), services.PendingProposalsNotice{
		Definition: h.definition,
		Run:        &chat,
		Proposals:  []*agent.AgentProposal{h.proposal("assign_move", agent.ProposalStatusPending, 1)},
	}))
	assert.Empty(t, h.notifications.created, "the person who asked is looking at the answer")

	shadowed := newHarness(true)
	require.NoError(t, shadowed.svc.NotifyPending(t.Context(), services.PendingProposalsNotice{
		Definition: shadowed.definition,
		Run:        shadowed.run,
		Proposals:  []*agent.AgentProposal{shadowed.proposal("assign_move", agent.ProposalStatusPending, 1)},
	}))
	assert.Empty(t, shadowed.notifications.created, "hidden proposals are nothing to decide")
}

func TestRemindPending_NotifiesAtHighPriorityAndEmailsThoseWithAnAddress(t *testing.T) {
	t.Parallel()

	h := newHarness(false)
	now := int64(1_700_100_000)
	h.proposals.pending = []*agent.AgentProposal{
		h.proposal("assign_move", agent.ProposalStatusPending, now-6*3600),
		h.proposal("assign_move", agent.ProposalStatusPending, now-5*3600),
	}

	reminded, err := h.svc.RemindPending(t.Context(), services.RemindPendingProposalsRequest{
		Now:       now,
		OlderThan: 4 * time.Hour,
		Limit:     100,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, reminded)
	assert.Equal(t, now-4*3600, h.proposals.lastList.Before)
	assert.Len(t, h.proposals.reminded, 2, "each proposal is marked so the next sweep skips it")

	require.Len(t, h.notifications.created, 2)
	assert.Equal(t, EventProposalsReminder, h.notifications.created[0].EventType)
	assert.Equal(t, notification.PriorityHigh, h.notifications.created[0].Priority)
	assert.Equal(t, "Dispatch coverage still has 2 changes waiting after 6 hours", h.notifications.created[0].Title)

	require.Len(t, h.mailer.sent, 1, "only the decider with an address is emailed")
	assert.Equal(t, []string{"dana@example.com"}, h.mailer.sent[0].To)
	assert.Equal(t, email.PurposeNotifications, h.mailer.sent[0].Purpose)
	assert.Equal(t, "agent-proposal-reminder-"+h.run.ID.String()+"-"+h.roles.users[0].UserID.String(),
		h.mailer.sent[0].IdempotencyKey)

	require.Len(t, h.renderer.requests, 1)
	assert.Equal(t, "agent.proposal_reminder.email", string(h.renderer.requests[0].Kind))
}

func TestRemindPending_MarksShadowedProposalsWithoutTellingAnyone(t *testing.T) {
	t.Parallel()

	h := newHarness(true)
	now := int64(1_700_100_000)
	h.proposals.pending = []*agent.AgentProposal{h.proposal("assign_move", agent.ProposalStatusPending, now-9*3600)}

	reminded, err := h.svc.RemindPending(t.Context(), services.RemindPendingProposalsRequest{
		Now: now, OlderThan: 4 * time.Hour, Limit: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, reminded)
	assert.Len(t, h.proposals.reminded, 1, "a hidden proposal is not scanned again every sweep")
	assert.Empty(t, h.notifications.created)
	assert.Empty(t, h.mailer.sent)
}

func TestDescribeTools(t *testing.T) {
	t.Parallel()

	mk := func(names ...string) []*agent.AgentProposal {
		out := make([]*agent.AgentProposal, 0, len(names))
		for _, name := range names {
			out = append(out, &agent.AgentProposal{ToolName: name})
		}

		return out
	}

	assert.Equal(t, "assign move", describeTools(mk("assign_move")))
	assert.Equal(t, "assign move and notify driver", describeTools(mk("assign_move", "notify_driver", "assign_move")))
	assert.Equal(t, "a, b, c and 2 more", describeTools(mk("a", "b", "c", "d", "e")))
	assert.Equal(t, "a change", countChanges(1))
	assert.Equal(t, "3 changes", countChanges(3))
}
