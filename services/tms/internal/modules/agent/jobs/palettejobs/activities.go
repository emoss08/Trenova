package palettejobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/app/paletteanswer"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/palettejobs/palettecontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/runstream"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/pkg/errortypes"
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
	Logger    *zap.Logger
}

type Activities struct {
	answers   *paletteanswer.Service
	questions paletteports.QuestionRepository
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		answers:   p.Answers,
		questions: p.Questions,
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

	events := &streamEvents{topic: topic}
	events.publish(assistantstreamports.AssistantEventAccepted, assistantstreamports.AssistantAcceptedEvent{
		Content: input.Question.Content,
	}, true)

	answer, err := a.answers.Answer(ctx, &input.Question, input.Actor(), events)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		message := failureMessage(err)
		a.l.Warn("a palette question was not answered",
			zap.String("askId", input.AskID.String()), zap.Error(err))
		events.publish(assistantstreamports.AssistantEventError, map[string]string{"message": message}, true)

		return &paletteports.Answer{Question: input.Question, Answer: answer, Failure: message}, nil
	}

	events.publish(assistantstreamports.AssistantEventDone, paletteports.AnsweredEvent{AskID: input.AskID}, true)

	return &paletteports.Answer{Question: input.Question, Answer: answer}, nil
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
