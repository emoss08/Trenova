package deskthreadresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantturnservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationchapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantturnports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type threads struct {
	assistantports.AssistantService

	got     *conversationports.GetThreadRequest
	updated *assistantports.UpdateThreadRequest
	counted *conversationports.CountThreadShelvesRequest
}

func (s *threads) GetThread(
	_ context.Context,
	req *conversationports.GetThreadRequest,
) (*conversation.Thread, error) {
	s.got = req

	return nil, errortypes.NewNotFoundError("Thread not found")
}

func (s *threads) UpdateThread(
	_ context.Context,
	req *assistantports.UpdateThreadRequest,
	_ *permissionports.RequestActor,
) (*conversation.Thread, error) {
	s.updated = req

	return &conversation.Thread{ID: req.ThreadID}, nil
}

func (s *threads) CountThreadShelves(
	_ context.Context,
	req *conversationports.CountThreadShelvesRequest,
) (*conversationports.ThreadShelfCounts, error) {
	s.counted = req

	return &conversationports.ThreadShelfCounts{}, nil
}

type chapters struct {
	assistantchapterports.AssistantChapterService
}

func (chapters) List(
	context.Context,
	*assistantchapterports.ThreadChaptersRequest,
) ([]*conversationchapter.Chapter, error) {
	return nil, nil
}

type liveRecords struct {
	assistantturnports.AssistantTurnRepository

	live  []*assistantturnports.LiveAssistantTurn
	asked *assistantturnports.ListLiveAssistantTurnsRequest
}

func (r *liveRecords) ListLive(
	_ context.Context,
	req *assistantturnports.ListLiveAssistantTurnsRequest,
) ([]*assistantturnports.LiveAssistantTurn, error) {
	r.asked = req

	return r.live, nil
}

func core(engine permissionports.PermissionEngine) *resolvercore.Core {
	return &resolvercore.Core{L: zap.NewNop(), PermissionEngine: engine}
}

func grantedReadOnly() *resolvertest.GrantingPermissionEngine {
	return &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
		permission.ResourceAssistant.String() + "|" + string(permission.OpRead): true,
	}}
}

/*
A conversation is the person's own, read under their user id, so arranging it
needs no more than using the assistant; asking it something, handing it off or
compacting it writes to it and needs what sending a message needs.
*/
func TestDeskThreadOperationsCheckTheRESTGrants(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("athr_").String()
	cases := []struct {
		name      string
		operation permission.Operation
		call      func(ctx context.Context, d *Deps) error
	}{
		{"startAssistantThread", permission.OpCreate, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).StartAssistantThread(
				ctx,
				gqlmodel.StartAssistantThreadInput{AgentDefinitionID: id},
			)
			return err
		}},
		{"startAssistantTurn", permission.OpCreate, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).StartAssistantTurn(
				ctx,
				id,
				gqlmodel.AssistantMessageInput{Content: "hi"},
			)
			return err
		}},
		{"compactAssistantThread", permission.OpCreate, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).CompactAssistantThread(ctx, id)
			return err
		}},
		{"handOffAssistantThread", permission.OpCreate, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).HandOffAssistantThread(
				ctx,
				id,
				gqlmodel.HandOffAssistantThreadInput{AgentDefinitionID: id},
			)
			return err
		}},
		{"updateAssistantThread", permission.OpRead, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).UpdateAssistantThread(
				ctx,
				id,
				gqlmodel.UpdateAssistantThreadInput{},
			)
			return err
		}},
		{"deleteAssistantThread", permission.OpRead, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).DeleteAssistantThread(ctx, id)
			return err
		}},
		{"stopAssistantTurn", permission.OpRead, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).StopAssistantTurn(ctx, id)
			return err
		}},
		{"saveAssistantProposalEdits", permission.OpRead, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).SaveAssistantProposalEdits(ctx, id, id, nil)
			return err
		}},
		{"openAssistantChapter", permission.OpRead, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).OpenAssistantChapter(ctx, id, id)
			return err
		}},
		{"assistantThreads", permission.OpRead, func(ctx context.Context, d *Deps) error {
			_, err := (&QueryResolver{d}).AssistantThreads(ctx, nil, nil, nil, nil, nil, nil)
			return err
		}},
		{"assistantLiveTurns", permission.OpRead, func(ctx context.Context, d *Deps) error {
			_, err := (&QueryResolver{d}).AssistantLiveTurns(ctx)
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := tc.call(ctx, &Deps{Core: core(engine)})

			resolvertest.RequireRefused(t, err, engine, permission.ResourceAssistant, tc.operation)
		})
	}
}

/*
A conversation has one live reply at a time. Recording the turn before
checking whose conversation it was let anyone who knew a thread's id take that
slot. The turn service is absent here: reaching it would panic.
*/
func TestStartAssistantTurnChecksTheConversationIsThePersonsBeforeTakingIt(t *testing.T) {
	t.Parallel()

	auth := resolvertest.UserAuthContext()
	service := &threads{}
	resolver := &MutationResolver{
		&Deps{Core: core(grantedReadOnlyAndCreate()), AssistantService: service},
	}
	threadID := pulid.MustNew("athr_")

	turn, err := resolver.StartAssistantTurn(
		resolvertest.AuthedContext(t, auth),
		threadID.String(),
		gqlmodel.AssistantMessageInput{Content: "hello"},
	)

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Nil(t, turn)
	require.NotNil(t, service.got)
	assert.Equal(t, threadID, service.got.ID)
	assert.Equal(t, auth.UserID, service.got.UserID, "the conversation is read as the caller's own")
}

func grantedReadOnlyAndCreate() *resolvertest.GrantingPermissionEngine {
	engine := grantedReadOnly()
	engine.Granted[permission.ResourceAssistant.String()+"|"+string(permission.OpCreate)] = true

	return engine
}

func TestUpdateAssistantThreadLeavesWhatIsNotSentAndClearsAnEmptyFactList(t *testing.T) {
	t.Parallel()

	auth := resolvertest.UserAuthContext()
	service := &threads{}
	resolver := &MutationResolver{&Deps{Core: core(grantedReadOnly()), AssistantService: service}}
	threadID := pulid.MustNew("athr_")
	title := "Late loads"

	_, err := resolver.UpdateAssistantThread(
		resolvertest.AuthedContext(t, auth),
		threadID.String(),
		gqlmodel.UpdateAssistantThreadInput{Title: &title},
	)
	require.NoError(t, err)
	require.NotNil(t, service.updated)
	assert.Equal(t, threadID, service.updated.ThreadID)
	assert.Equal(t, auth.OrganizationID, service.updated.TenantInfo.OrgID)
	assert.Equal(t, &title, service.updated.Title)
	assert.Nil(t, service.updated.Pinned)
	assert.Nil(t, service.updated.PinnedFacts, "facts that are not sent are left as they are")

	_, err = resolver.UpdateAssistantThread(
		resolvertest.AuthedContext(t, auth),
		threadID.String(),
		gqlmodel.UpdateAssistantThreadInput{PinnedFacts: []string{}},
	)
	require.NoError(t, err)
	require.NotNil(t, service.updated.PinnedFacts, "an empty list unpins every fact")
	assert.Empty(t, *service.updated.PinnedFacts)
}

func TestAssistantThreadShelvesReadsEachEdgeInOrderForTheCaller(t *testing.T) {
	t.Parallel()

	auth := resolvertest.UserAuthContext()
	service := &threads{}
	resolver := &QueryResolver{&Deps{Core: core(grantedReadOnly()), AssistantService: service}}

	_, err := resolver.AssistantThreadShelves(
		resolvertest.AuthedContext(t, auth),
		[]int{1_700_086_400, 1_700_000_000},
	)

	require.NoError(t, err)
	require.NotNil(t, service.counted)
	assert.Equal(t, []int64{1_700_086_400, 1_700_000_000}, service.counted.Edges)
	assert.Equal(t, auth.UserID, service.counted.UserID)
}

func TestAssistantLiveTurnsListsTheCallersRepliesInProgress(t *testing.T) {
	t.Parallel()

	auth := resolvertest.UserAuthContext()
	turn := conversationturn.AssistantTurn{
		ID:        pulid.MustNew("atrn_"),
		ThreadID:  pulid.MustNew("athr_"),
		UserID:    auth.UserID,
		Origin:    conversationturn.AssistantTurnOriginDecisionFollowUp,
		Status:    conversation.AssistantTurnStatusRunning,
		StartedAt: 1_700_000_000,
	}
	records := &liveRecords{live: []*assistantturnports.LiveAssistantTurn{{
		AssistantTurn: turn,
		ThreadTitle:   "Late loads this week",
	}}}
	resolver := &QueryResolver{&Deps{
		Core: core(grantedReadOnly()),
		AssistantTurnService: assistantturnservice.New(assistantturnservice.Params{
			Logger: zap.NewNop(),
			Turns:  records,
		}),
	}}

	live, err := resolver.AssistantLiveTurns(resolvertest.AuthedContext(t, auth))

	require.NoError(t, err)
	require.NotNil(t, records.asked)
	assert.Equal(t, auth.UserID, records.asked.UserID, "only the caller's own replies are listed")
	assert.ElementsMatch(
		t,
		[]conversation.ThreadOrigin{
			conversation.ThreadOriginImport,
			conversation.ThreadOriginFormula,
		},
		records.asked.ExcludeOrigins,
		"a reply on an import or formula page is read there, not listed",
	)
	assert.Equal(t, []*gqlmodel.AssistantLiveTurn{{
		TurnID:      turn.ID.String(),
		ThreadID:    turn.ThreadID.String(),
		ThreadTitle: "Late loads this week",
		Origin:      "DecisionFollowUp",
		StartedAt:   1_700_000_000,
	}}, live)
}

func TestAssistantChaptersIsAnEmptyListWhenThereAreNone(t *testing.T) {
	t.Parallel()

	resolver := &QueryResolver{&Deps{
		Core:                    core(grantedReadOnly()),
		AssistantChapterService: chapters{},
	}}

	got, err := resolver.AssistantChapters(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("athr_").String(),
	)

	require.NoError(t, err)
	listed, ok := got.(chapterItems)
	require.True(t, ok)
	assert.NotNil(t, listed.Items)
	assert.Empty(t, listed.Items)
}
