package paletteservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/turnlimits"
	"github.com/emoss08/trenova/internal/modules/agent/app/paletteanswer"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/palettequestion"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/palettejobs/palettecontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type questions struct {
	rows   map[pulid.ID]*palettequestion.Question
	asking int
}

func newQuestions() *questions {
	return &questions{rows: map[pulid.ID]*palettequestion.Question{}}
}

func (q *questions) Create(_ context.Context, question *palettequestion.Question) error {
	question.ID = pulid.MustNew(palettequestion.IDPrefix)
	q.rows[question.ID] = question

	return nil
}

func (q *questions) Get(_ context.Context, ref paletteports.AskRef) (*palettequestion.Question, error) {
	row, ok := q.rows[ref.AskID]
	if !ok || row.UserID != ref.TenantInfo.UserID {
		return nil, errortypes.NewNotFoundError("Question not found")
	}
	copied := *row

	return &copied, nil
}

func (q *questions) CountAsking(context.Context, pagination.TenantInfo, int64) (int, error) {
	return q.asking, nil
}

func (q *questions) CountUnkeptSince(context.Context, pagination.TenantInfo, int64) (int, error) {
	return 0, nil
}

func (q *questions) Finish(
	_ context.Context,
	ref paletteports.AskRef,
	status palettequestion.Status,
	_ int64,
) error {
	if row := q.rows[ref.AskID]; row.Status == palettequestion.StatusAsking {
		row.Status = status
	}

	return nil
}

func (q *questions) ClaimKeep(_ context.Context, ref paletteports.AskRef, now int64) (bool, error) {
	row := q.rows[ref.AskID]
	if row.KeptThreadID.IsNotNil() || row.KeepClaimedAt != nil {
		return false, nil
	}
	row.KeepClaimedAt = &now

	return true, nil
}

func (q *questions) MarkKept(_ context.Context, ref paletteports.AskRef, threadID pulid.ID) error {
	q.rows[ref.AskID].KeptThreadID = threadID

	return nil
}

func (q *questions) ReleaseKeep(_ context.Context, ref paletteports.AskRef) error {
	q.rows[ref.AskID].KeepClaimedAt = nil

	return nil
}

func (q *questions) DeleteBefore(context.Context, int64, int) (int, error) { return 0, nil }

type assistant struct {
	assistantports.AssistantService

	started []*assistantports.GeneralThreadRequest
	threads map[pulid.ID]*conversation.Thread
}

func (a *assistant) StartGeneralThread(
	_ context.Context,
	req *assistantports.GeneralThreadRequest,
	_ *permissionports.RequestActor,
) (*conversation.Thread, error) {
	a.started = append(a.started, req)
	thread := &conversation.Thread{ID: pulid.MustNew("athr_"), Title: req.Title}
	a.threads[thread.ID] = thread

	return thread, nil
}

func (a *assistant) GetThread(
	_ context.Context,
	req *conversationports.GetThreadRequest,
) (*conversation.Thread, error) {
	return a.threads[req.ID], nil
}

type conversations struct {
	conversationports.ConversationRepository

	appended []*conversationports.AppendTurnRequest
}

func (c *conversations) AppendTurn(
	_ context.Context,
	req *conversationports.AppendTurnRequest,
) ([]conversation.Message, error) {
	c.appended = append(c.appended, req)

	return req.Messages, nil
}

func (c *conversations) CountQuestionsSince(
	context.Context,
	conversationports.CountQuestionsSinceRequest,
) (int, error) {
	return 0, nil
}

type results struct{ answer *paletteports.Answer }

func (r results) WorkflowResult(_ context.Context, _ string, out any) error {
	*out.(*paletteports.Answer) = *r.answer

	return nil
}

type workflows struct {
	started []string
}

func (w *workflows) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	_ ...any,
) (client.WorkflowRun, error) {
	w.started = append(w.started, options.ID)

	return nil, nil
}

func (w *workflows) CancelWorkflow(context.Context, string, string) error { return nil }

func (w *workflows) SignalWorkflow(context.Context, string, string, string, any) error {
	return nil
}

func (w *workflows) Enabled() bool { return true }

type monthly struct {
	tenantports.AgentControlRepository

	limit int
}

func (m monthly) GetOrCreate(context.Context, pagination.TenantInfo) (*agentcontrol.AgentControl, error) {
	return &agentcontrol.AgentControl{PersonMonthlyMessages: m.limit}, nil
}

type anyTask struct{}

func (anyTask) Available(context.Context, aitask.Task, pagination.TenantInfo) (bool, error) {
	return true, nil
}

func (anyTask) Run(context.Context, *llmports.NovaTask, any) (*llmports.StructuredCompletionResult, error) {
	return &llmports.StructuredCompletionResult{}, nil
}

func (anyTask) Converse(
	context.Context,
	*llmports.NovaConversation,
	llmports.NovaEvents,
) (*llmports.NovaAnswer, error) {
	return &llmports.NovaAnswer{}, nil
}

type harness struct {
	svc       *Service
	questions *questions
	assistant *assistant
	turns     *conversations
	workflows *workflows
	actor     *permissionports.RequestActor
}

func newHarness(answer *paletteports.Answer) *harness {
	h := &harness{
		questions: newQuestions(),
		assistant: &assistant{threads: map[pulid.ID]*conversation.Thread{}},
		turns:     &conversations{},
		workflows: &workflows{},
		actor: permissionports.UserActor(pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		}),
	}
	h.svc = &Service{
		l: zap.NewNop(),
		answers: paletteanswer.New(paletteanswer.Params{
			Logger:    zap.NewNop(),
			Converser: anyTask{},
			Nova:      anyTask{},
		}),
		assistant:     h.assistant,
		conversations: h.turns,
		questions:     h.questions,
		workflows:     h.workflows,
		results:       results{answer: answer},
		limits: &turnlimits.Limits{
			Conversations:    h.turns,
			AgentControls:    monthly{},
			PaletteQuestions: h.questions,
		},
	}

	return h
}

func (h *harness) ask(t *testing.T) paletteports.AskRef {
	t.Helper()

	ask, err := h.svc.Ask(t.Context(), &paletteports.AskRequest{
		Question:   paletteports.Question{Content: "who is on S-104?"},
		TenantInfo: h.actor.TenantInfo(),
	}, h.actor)
	require.NoError(t, err)

	return paletteports.AskRef{AskID: ask.ID, TenantInfo: h.actor.TenantInfo()}
}

func answered() *paletteports.Answer {
	return &paletteports.Answer{
		Question: paletteports.Question{Content: "who is on S-104?"},
		Answer: &llmports.NovaAnswer{
			Text:    "Dana is driving S-104.",
			Records: []agentpagecontext.EntityRef{{Type: "shipment", ID: "shp_1", Label: "S-104"}},
		},
	}
}

func TestAsk_RecordsTheQuestionAndNamesItsWorkflowByIt(t *testing.T) {
	t.Parallel()

	h := newHarness(answered())

	ref := h.ask(t)

	row := h.questions.rows[ref.AskID]
	require.NotNil(t, row)
	assert.Equal(t, palettequestion.StatusAsking, row.Status)
	assert.Equal(t, h.actor.UserID, row.UserID)
	assert.Equal(t, []string{palettecontract.WorkflowID(ref)}, h.workflows.started)
}

func TestAsk_RefusesOnceTheMonthsAllowanceIsUsed(t *testing.T) {
	t.Parallel()

	h := newHarness(answered())
	h.svc.limits.AgentControls = monthly{limit: 1}
	h.svc.limits.Conversations = &countedConversations{count: 1}

	_, err := h.svc.Ask(t.Context(), &paletteports.AskRequest{
		Question:   paletteports.Question{Content: "who is on S-104?"},
		TenantInfo: h.actor.TenantInfo(),
	}, h.actor)

	require.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, h.questions.rows, "a refused question is not recorded")
	assert.Empty(t, h.workflows.started)
}

func TestAsk_RefusesWhileThePersonHasAsManyBeingAnsweredAsAllowed(t *testing.T) {
	t.Parallel()

	h := newHarness(answered())
	h.questions.asking = palettequestion.MaxAsking

	_, err := h.svc.Ask(t.Context(), &paletteports.AskRequest{
		Question:   paletteports.Question{Content: "who is on S-104?"},
		TenantInfo: h.actor.TenantInfo(),
	}, h.actor)

	require.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "still answering")
	assert.Empty(t, h.workflows.started)
}

func TestKeep_CarriesTheQuestionOnceAndReturnsTheSameConversationAfter(t *testing.T) {
	t.Parallel()

	h := newHarness(answered())
	ref := h.ask(t)

	first, err := h.svc.Keep(t.Context(), ref, h.actor)
	require.NoError(t, err)
	second, err := h.svc.Keep(t.Context(), ref, h.actor)
	require.NoError(t, err)

	assert.Equal(t, first.Thread.ID, second.Thread.ID)
	assert.Len(t, h.assistant.started, 1, "one question opens one conversation")
	require.Len(t, h.turns.appended, 1)
	assert.Len(t, h.turns.appended[0].Messages, 2)
	assert.Equal(t, first.Thread.ID, h.questions.rows[ref.AskID].KeptThreadID)
	assert.Empty(t, first.Unanswered)
}

func TestKeep_RefusesWhileAnotherRequestIsCarryingIt(t *testing.T) {
	t.Parallel()

	h := newHarness(answered())
	ref := h.ask(t)
	claimed := int64(1)
	h.questions.rows[ref.AskID].KeepClaimedAt = &claimed

	_, err := h.svc.Keep(t.Context(), ref, h.actor)

	require.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, h.assistant.started)
}

func TestKeep_OpensAnEmptyConversationForAQuestionWithNoAnswer(t *testing.T) {
	t.Parallel()

	h := newHarness(&paletteports.Answer{
		Question: paletteports.Question{Content: "who is on S-104?"},
		Failure:  "Nova couldn't answer that just now.",
	})
	ref := h.ask(t)

	kept, err := h.svc.Keep(t.Context(), ref, h.actor)
	require.NoError(t, err)

	assert.Equal(t, "who is on S-104?", kept.Unanswered)
	assert.Empty(t, h.turns.appended)
	assert.Equal(t, kept.Thread.ID, h.questions.rows[ref.AskID].KeptThreadID)
}

func TestKeep_ReachesOnlyThePersonsOwnQuestion(t *testing.T) {
	t.Parallel()

	h := newHarness(answered())
	ref := h.ask(t)
	someoneElse := ref
	someoneElse.TenantInfo.UserID = pulid.MustNew("usr_")

	_, err := h.svc.Keep(t.Context(), someoneElse, h.actor)

	require.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, h.assistant.started)
}

type countedConversations struct {
	conversationports.ConversationRepository

	count int
}

func (c *countedConversations) CountQuestionsSince(
	context.Context,
	conversationports.CountQuestionsSinceRequest,
) (int, error) {
	return c.count, nil
}

func TestStop_FreesThePersonsRoomAtOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(answered())
	ref := h.ask(t)

	require.NoError(t, h.svc.Stop(t.Context(), ref))

	assert.Equal(t, palettequestion.StatusStopped, h.questions.rows[ref.AskID].Status)
}
