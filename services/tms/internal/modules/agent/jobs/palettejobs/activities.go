package palettejobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/app/paletteanswer"
	"github.com/emoss08/trenova/internal/modules/agent/domain/palettequestion"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/palettejobs/palettecontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/runstream"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const noProviderMessage = "No AI provider answers command palette questions. " +
	"An administrator can assign one in AI Control."

type ActivitiesParams struct {
	fx.In

	Answers   *paletteanswer.Service
	Questions paletteports.QuestionRepository
	Scopes    permissionports.ReadScopes
	Logger    *zap.Logger
}

type Activities struct {
	answers   *paletteanswer.Service
	questions paletteports.QuestionRepository
	scopes    permissionports.ReadScopes
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		answers:   p.Answers,
		questions: p.Questions,
		scopes:    p.Scopes,
		l:         p.Logger.Named("job.palette-answer"),
	}
}

var Module = fx.Module(
	"palette-jobs",
	fx.Provide(NewActivities),
)

// AnswerActivity answers the question, streaming each step and the answer to
// the person, and ends the stream with done or with what went wrong.
func (a *Activities) AnswerActivity(
	ctx context.Context,
	input *palettecontract.AnswerInput,
) (*paletteports.Answer, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	stream, topic, err := runstream.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer runstream.Flush(ctx, stream, a.l)

	// Every read is the person's, so it runs in their tenant and under their
	// data scopes, as a Desk tool call does (agentflow's tool activity) and a
	// request does (the read-scope middleware). Without the scope every
	// GraphQL-backed read is refused. Its reads are recorded in the same
	// tenant.
	actor := input.Actor()
	ctx = dbscope.WithTenant(ctx, actor.DBTenant())
	ctx = readscope.With(ctx, a.scopes.Resolver(actor.PermissionCheck("", "")))

	events := &streamEvents{topic: topic, record: a.recorder(ctx, input)}
	events.publish(
		assistantstreamports.AssistantEventAccepted,
		assistantstreamports.AssistantAcceptedEvent{
			Content: input.Question.Content,
		},
		true,
	)

	answer, err := a.answers.Answer(ctx, input.AskID, &input.Question, actor, events)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		message := failureMessage(err)
		a.l.Warn("a palette question was not answered",
			zap.String("askId", input.AskID.String()), zap.Error(err))
		events.publish(
			assistantstreamports.AssistantEventError,
			map[string]string{"message": message},
			true,
		)

		return &paletteports.Answer{Question: input.Question, Answer: answer, Failure: message}, nil
	}

	events.publish(
		assistantstreamports.AssistantEventDone,
		paletteports.AnsweredEvent{AskID: input.AskID},
		true,
	)

	return &paletteports.Answer{Question: input.Question, Answer: answer}, nil
}

// recorder keeps each read made for the question. A read that cannot be kept
// is logged and the answer goes on: the person is waiting on it, and the
// record is for looking back.
func (a *Activities) recorder(
	ctx context.Context,
	input *palettecontract.AnswerInput,
) func(*palettequestion.ToolCall) {
	return func(call *palettequestion.ToolCall) {
		call.OrganizationID = input.OrganizationID
		call.BusinessUnitID = input.BusinessUnitID
		call.QuestionID = input.AskID
		call.UserID = input.UserID
		if err := a.questions.RecordToolCall(ctx, call); err != nil {
			a.l.Error("a palette read was not recorded",
				zap.String("askId", input.AskID.String()),
				zap.String("tool", call.Tool),
				zap.String("callId", call.CallID),
				zap.Error(err))
		}
	}
}

func failureMessage(err error) string {
	if errors.Is(err, llmports.ErrNoProviderConfigured) {
		return noProviderMessage
	}
	var business *errortypes.BusinessError
	if errors.As(err, &business) {
		return business.Error()
	}

	return failedMessage
}

// FinishAnswerActivity records how the question ended, which frees the
// person's room to ask another.
func (a *Activities) FinishAnswerActivity(ctx context.Context, input *FinishInput) error {
	return a.questions.Finish(ctx, paletteports.AskRef{
		AskID:      input.AskID,
		TenantInfo: input.TenantInfo(),
	}, input.Status, timeutils.NowUnix())
}
