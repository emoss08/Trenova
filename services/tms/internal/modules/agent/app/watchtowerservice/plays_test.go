package watchtowerservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/requestwording"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// plannedPlaybook is a playbook whose plan a test sets: what the person is
// shown, what they must be allowed to do, and what was sent.
type plannedPlaybook struct {
	kind    watchtowerplay.Kind
	play    watchtowerplay.Play
	needs   []watchtowerports.PlayGrant
	planErr error
	sendErr error
	planned int
	sent    []*watchtowerports.PlaySending
}

func (p *plannedPlaybook) Kind() watchtowerplay.Kind { return p.kind }

func (p *plannedPlaybook) Plan(
	context.Context,
	*watchtowerports.PlanRequest,
) (*watchtowerports.PlannedPlay, error) {
	p.planned++
	if p.planErr != nil {
		return nil, p.planErr
	}
	play := p.play
	play.Kind = p.kind

	return &watchtowerports.PlannedPlay{
		Play:  play,
		Needs: p.needs,
		Asked: "Asked Laura Wright for the signed proof of delivery in Dash",
		Send: func(_ context.Context, sending *watchtowerports.PlaySending) error {
			if p.sendErr != nil {
				return p.sendErr
			}
			p.sent = append(p.sent, sending)

			return nil
		},
	}, nil
}

type namedUsers struct {
	tenantports.UserRepository
}

func (namedUsers) GetByID(
	context.Context,
	tenantports.GetUserByIDRequest,
) (*tenant.User, error) {
	return &tenant.User{Name: "Dana Reyes"}, nil
}

// novaWriter is Nova for a test: whether it can run, and the message it
// writes back.
type novaWriter struct {
	llmports.TaskRunner

	available bool
	writes    string
	asked     *llmports.NovaTask
}

func (n *novaWriter) Available(context.Context, pagination.TenantInfo) (bool, error) {
	return n.available, nil
}

func (n *novaWriter) Run(
	_ context.Context,
	task *llmports.NovaTask,
	out any,
) (*llmports.StructuredCompletionResult, error) {
	n.asked = task

	return &llmports.StructuredCompletionResult{}, llmports.DecodeAnswer(
		task.Name, `{"message":"`+n.writes+`"}`, out,
	)
}

type playRig struct {
	plays    *Plays
	repo     *stubRepo
	playbook *plannedPlaybook
	nova     *novaWriter
	actor    *permissionports.RequestActor
	item     *watchtower.Item
	owners   *agenttest.OpenCases
}

func newPlayRig(t *testing.T, perms *agenttest.StubPermissions) *playRig {
	t.Helper()

	repo := newStubRepo()
	actor := testActor()
	owners := &agenttest.OpenCases{}
	svc := newService(repo, perms)
	playbook := &plannedPlaybook{
		kind: watchtowerplay.AskForPOD,
		play: watchtowerplay.Play{
			Channel:      watchtowerplay.ChannelDash,
			Recipient:    "Laura Wright",
			Message:      "Hi Laura, please send the signed proof of delivery.",
			MessageLimit: 80,
		},
		needs: []watchtowerports.PlayGrant{
			{Resource: permission.ResourceDriverMessage, Operation: permission.OpCreate},
		},
	}
	nova := &novaWriter{available: true, writes: "Laura, third ask: send the signed POD today."}
	plays := NewPlays(PlaysParams{
		Logger:      zap.NewNop(),
		Exceptions:  svc.exceptions,
		Permissions: perms,
		Owners:      owners,
		Playbooks:   []watchtowerports.Playbook{playbook},
		Wording:     requestwording.New(nova),
		Users:       namedUsers{},
	})

	in := input(actor.TenantInfo(), watchtower.SourcePodMissing, "shp_1", watchtower.SeverityWarning, 100)
	in.SubjectType = agent.SubjectShipment
	in.SubjectID = pulid.MustNew("shp_")
	in.About = in.SubjectID
	require.NoError(t, svc.projector.upsert(t.Context(), in))

	return &playRig{
		plays:    plays,
		repo:     repo,
		playbook: playbook,
		nova:     nova,
		actor:    actor,
		item:     repo.items[repo.key(in.SourceKind, in.SourceID)],
		owners:   owners,
	}
}

func (r *playRig) target() watchtowerports.PlayTarget {
	return watchtowerports.PlayTarget{TenantInfo: r.actor.TenantInfo(), ItemID: r.item.ID}
}

// A person is shown the play their item's kind has, planned from the
// record, and whether Nova can reword it. Nothing is sent by looking.
func TestPlays_PreviewShowsThePlayAndSendsNothing(t *testing.T) {
	t.Parallel()

	r := newPlayRig(t, &agenttest.StubPermissions{})
	target := r.target()

	preview, err := r.plays.Preview(t.Context(), &target, r.actor)
	require.NoError(t, err)

	assert.Equal(t, r.item.ID, preview.ItemID)
	assert.Equal(t, watchtowerplay.AskForPOD, preview.Kind)
	assert.Equal(t, watchtowerplay.ChannelDash, preview.Channel)
	assert.Equal(t, "Laura Wright", preview.Recipient)
	assert.Equal(t, "Hi Laura, please send the signed proof of delivery.", preview.Message)
	assert.Equal(t, 80, preview.MessageLimit)
	assert.True(t, preview.CanReword)
	assert.Empty(t, r.playbook.sent)
	for _, event := range r.repo.events {
		assert.NotEqual(t, watchtower.EventRequested, event.Kind, "previewing notes nothing")
	}
	assert.Nil(t, r.item.LastRequest)

	r.nova.available = false
	preview, err = r.plays.Preview(t.Context(), &target, r.actor)
	require.NoError(t, err)
	assert.False(t, preview.CanReword, "rewording is offered only where Nova can write")
}

// Sending delivers the words the person approved, notes on the item who
// asked what of whom, records it in the item's history, and puts the item
// in progress. No agent is named anywhere on the item.
func TestPlays_SendDeliversThePersonsWordsAndNotesItOnTheItem(t *testing.T) {
	t.Parallel()

	r := newPlayRig(t, &agenttest.StubPermissions{})

	saved, err := r.plays.Send(t.Context(), &watchtowerports.SendPlayRequest{
		PlayTarget:     r.target(),
		Message:        "  Laura, please send the POD today.  ",
		IdempotencyKey: "key-1",
	}, r.actor)
	require.NoError(t, err)

	require.Len(t, r.playbook.sent, 1)
	assert.Equal(t, "Laura, please send the POD today.", r.playbook.sent[0].Message)
	assert.Equal(t, "key-1", r.playbook.sent[0].IdempotencyKey)

	require.NotNil(t, saved.LastRequest)
	assert.Equal(t, r.actor.UserID, saved.LastRequest.ByID)
	assert.Equal(t, "Dana Reyes", saved.LastRequest.ByName)
	assert.Equal(t,
		"Asked Laura Wright for the signed proof of delivery in Dash", saved.LastRequest.Summary)
	assert.Equal(t, watchtower.StatusInProgress, saved.Status)
	assert.True(t, saved.AgentDefinitionID.IsNil(), "a play is the person's, not an agent's")
	assert.True(t, saved.AgentRunID.IsNil())

	var requested *watchtower.Event
	for _, event := range r.repo.events {
		if event.Kind == watchtower.EventRequested {
			requested = event
		}
	}
	require.NotNil(t, requested)
	assert.Equal(t, r.actor.UserID, requested.ActorID)
	assert.Equal(t, "Asked Laura Wright for the signed proof of delivery in Dash", requested.Note)
}

// With nothing typed, the play's own words go out.
func TestPlays_SendWithNoWordsSendsThePlaysOwn(t *testing.T) {
	t.Parallel()

	r := newPlayRig(t, &agenttest.StubPermissions{})

	_, err := r.plays.Send(t.Context(), &watchtowerports.SendPlayRequest{PlayTarget: r.target()}, r.actor)
	require.NoError(t, err)
	require.Len(t, r.playbook.sent, 1)
	assert.Equal(t, "Hi Laura, please send the signed proof of delivery.", r.playbook.sent[0].Message)
}

// A message past the channel's bound is refused before anything is sent.
func TestPlays_SendRefusesAMessageTooLongForItsChannel(t *testing.T) {
	t.Parallel()

	r := newPlayRig(t, &agenttest.StubPermissions{})

	_, err := r.plays.Send(t.Context(), &watchtowerports.SendPlayRequest{
		PlayTarget: r.target(),
		Message:    strings.Repeat("x", 81),
	}, r.actor)
	require.Error(t, err)
	assert.Empty(t, r.playbook.sent)
	assert.Nil(t, r.item.LastRequest)
}

// A play is held to the permission the page for the same act asks. Without
// it the person is told which access is missing, and nothing is sent.
func TestPlays_RefuseAPersonWithoutThePermissionTheActNeeds(t *testing.T) {
	t.Parallel()

	r := newPlayRig(t, &agenttest.StubPermissions{
		Denied: map[string]bool{"driver_message:create": true},
	})
	target := r.target()

	_, err := r.plays.Preview(t.Context(), &target, r.actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Contains(t, err.Error(), "does not let you message drivers")

	_, err = r.plays.Send(t.Context(), &watchtowerports.SendPlayRequest{PlayTarget: target}, r.actor)
	require.Error(t, err)
	assert.Empty(t, r.playbook.sent)
}

// A case someone else owns is not this person's to act on: the play is
// refused before the record is even planned from.
func TestPlays_RefuseAnItemAboutSomeoneElsesCase(t *testing.T) {
	t.Parallel()

	r := newPlayRig(t, &agenttest.StubPermissions{})
	r.owners.Owned = map[pulid.ID]pulid.ID{r.item.AboutID: pulid.MustNew("usr_")}
	target := r.target()

	_, err := r.plays.Send(t.Context(), &watchtowerports.SendPlayRequest{PlayTarget: target}, r.actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Zero(t, r.playbook.planned)
	assert.Empty(t, r.playbook.sent)
}

// A kind with no play, and an item already resolved, have nothing to send.
func TestPlays_RefuseAKindWithNoPlayAndAResolvedItem(t *testing.T) {
	t.Parallel()

	r := newPlayRig(t, &agenttest.StubPermissions{})

	resolvedAt := int64(900)
	r.item.ResolvedAt = &resolvedAt
	target := r.target()
	_, err := r.plays.Preview(t.Context(), &target, r.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already resolved")
	r.item.ResolvedAt = nil

	r.item.SourceKind = watchtower.SourceServiceFailure
	_, err = r.plays.Preview(t.Context(), &target, r.actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Zero(t, r.playbook.planned)
}

// A play that cannot be planned says why, and one that fails to send notes
// nothing on the item: the case must not say a question is out when it is
// not.
func TestPlays_AFailedPlanOrSendNotesNothingOnTheItem(t *testing.T) {
	t.Parallel()

	r := newPlayRig(t, &agenttest.StubPermissions{})
	target := r.target()

	r.playbook.planErr = errortypes.NewBusinessError("Nobody is assigned to the delivering move")
	_, err := r.plays.Preview(t.Context(), &target, r.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Nobody is assigned")
	r.playbook.planErr = nil

	r.playbook.sendErr = errors.New("the mail server refused it")
	_, err = r.plays.Send(t.Context(), &watchtowerports.SendPlayRequest{PlayTarget: target}, r.actor)
	require.Error(t, err)
	assert.Nil(t, r.item.LastRequest)
	for _, event := range r.repo.events {
		assert.NotEqual(t, watchtower.EventRequested, event.Kind)
	}
}

// Rewording hands Nova the words in the box and the person's guidance, and
// nothing about who the play goes to; the result is held to the channel's
// bound. A play with no message has nothing to reword.
func TestPlays_RewordChangesOnlyTheWords(t *testing.T) {
	t.Parallel()

	r := newPlayRig(t, &agenttest.StubPermissions{})

	worded, err := r.plays.Reword(t.Context(), &watchtowerports.RewordPlayRequest{
		PlayTarget: r.target(),
		Message:    "Laura, send the POD.",
		Guidance:   "firmer, it is the third ask",
	}, r.actor)
	require.NoError(t, err)
	assert.Equal(t, "Laura, third ask: send the signed POD today.", worded)
	assert.LessOrEqual(t, len(worded), 80)
	assert.Empty(t, r.playbook.sent, "rewording sends nothing")

	require.NotNil(t, r.nova.asked)
	var sections []string
	for _, section := range r.nova.asked.Context.Sections {
		sections = append(sections, section.Title+"="+section.Content)
		assert.NotContains(t, section.Content, "Laura Wright", "the recipient is not Nova's to see")
	}
	assert.Contains(t, sections, "Standard message=Laura, send the POD.")
	assert.Contains(t, sections, "Sender's guidance=firmer, it is the third ask")
	assert.Equal(t, r.actor.UserID, r.nova.asked.UserID)

	r.playbook.play.Message = ""
	_, err = r.plays.Reword(t.Context(), &watchtowerports.RewordPlayRequest{
		PlayTarget: r.target(), Guidance: "shorter",
	}, r.actor)
	require.Error(t, err)
}
