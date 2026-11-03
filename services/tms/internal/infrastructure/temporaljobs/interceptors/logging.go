package interceptors

import (
	"context"
	"errors"
	"slices"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

// LoggingInterceptor reports activity and workflow outcomes. Failures are
// always reported, once per attempt, with the activity, the workflow and
// whether a retry follows: a Warn while the retry policy will try again, an
// Error when the failure is final. The SDK's own per-attempt line is lowered to
// Debug (temporaljobs.sdkLogger), so a failed attempt is one line, not two.
// Routine started and completed lines are written only when logging is enabled.
type LoggingInterceptor struct {
	interceptor.InterceptorBase
	logger   *zap.Logger
	logLevel string
	quiet    bool
}

func NewLoggingInterceptor(logger *zap.Logger, logLevel string) *LoggingInterceptor {
	return &LoggingInterceptor{
		logger:   logger.Named("temporal"),
		logLevel: logLevel,
	}
}

// NewFailureLoggingInterceptor reports failures only, for a worker whose
// routine Temporal logging is switched off.
func NewFailureLoggingInterceptor(logger *zap.Logger) *LoggingInterceptor {
	return &LoggingInterceptor{logger: logger.Named("temporal"), quiet: true}
}

func (l *LoggingInterceptor) InterceptActivity(
	_ context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &loggingActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		logger:                         l.logger,
		logLevel:                       l.logLevel,
		quiet:                          l.quiet,
	}
}

func (l *LoggingInterceptor) InterceptWorkflow(
	_ workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	return &loggingWorkflowInbound{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
		logger:                         l.logger,
		logLevel:                       l.logLevel,
		quiet:                          l.quiet,
	}
}

type loggingActivityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	logger   *zap.Logger
	logLevel string
	quiet    bool
}

func (l *loggingActivityInbound) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (any, error) {
	start := time.Now()

	log := l.logger.With(activityFields(ctx)...)

	if !l.quiet && l.logLevel == "debug" { //nolint:goconst // no need to check this.
		log.Debug("activity started")
	}

	result, err := l.Next.ExecuteActivity(ctx, in)

	duration := zap.Duration("duration", time.Since(start))

	switch {
	case err != nil && benign(err):
		log.Debug("activity answered with an expected failure", duration, zap.Error(err))
	case err != nil && willRetry(ctx, err):
		log.Warn("activity attempt failed; it will be retried", duration, zap.Error(err))
	case err != nil:
		log.Error("activity failed", duration, zap.Error(err))
	case l.quiet:
	case l.logLevel == "debug":
		log.Debug("activity completed", duration)
	default:
		log.Info("activity completed", duration)
	}

	return result, err
}

func activityFields(ctx context.Context) []zap.Field {
	fields := []zap.Field{zap.String("type", "activity")}
	if !activity.IsActivity(ctx) {
		return fields
	}

	info := activity.GetInfo(ctx)
	fields = append(fields,
		zap.String("activityType", info.ActivityType.Name),
		zap.String("workflowID", info.WorkflowExecution.ID),
		zap.String("runID", info.WorkflowExecution.RunID),
		zap.Int32("attempt", info.Attempt),
	)
	if info.WorkflowType != nil {
		fields = append(fields, zap.String("workflowType", info.WorkflowType.Name))
	}

	return fields
}

// benign reports an error the activity marked as expected in normal operation,
// such as a conversation still busy, which is not worth a warning.
func benign(err error) bool {
	appErr, ok := errors.AsType[*temporal.ApplicationError](err)

	return ok && appErr.Category() == temporal.ApplicationErrorCategoryBenign
}

// willRetry reports whether the retry policy will run the activity again after
// err (retryFollows). The server sends every activity its retry policy; an
// activity without one retries without end.
func willRetry(ctx context.Context, err error) bool {
	if !activity.IsActivity(ctx) {
		return false
	}
	info := activity.GetInfo(ctx)

	return retryFollows(err, info.Attempt, info.RetryPolicy)
}

// retryFollows reports whether attempt is followed by another after err: not
// for a cancellation, a non-retryable error or one whose type the policy names,
// nor once the policy's attempts are spent.
func retryFollows(err error, attempt int32, policy *temporal.RetryPolicy) bool {
	if errors.Is(err, context.Canceled) || temporal.IsCanceledError(err) {
		return false
	}
	appErr, isAppErr := errors.AsType[*temporal.ApplicationError](err)
	if isAppErr && appErr.NonRetryable() {
		return false
	}
	if policy == nil {
		return true
	}
	if isAppErr && slices.Contains(policy.NonRetryableErrorTypes, appErr.Type()) {
		return false
	}

	return policy.MaximumAttempts <= 0 || attempt < policy.MaximumAttempts
}

type loggingWorkflowInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	logger   *zap.Logger
	logLevel string
	quiet    bool
}

func (l *loggingWorkflowInbound) ExecuteWorkflow(
	ctx workflow.Context,
	in *interceptor.ExecuteWorkflowInput,
) (any, error) {
	info := workflow.GetInfo(ctx)

	log := l.logger.With(
		zap.String("type", "workflow"),
		zap.String("workflowType", info.WorkflowType.Name),
		zap.String("workflowID", info.WorkflowExecution.ID),
		zap.String("runID", info.WorkflowExecution.RunID),
		zap.String("taskQueue", info.TaskQueueName),
	)

	// A replay re-runs the workflow from its first event; what it logged then
	// was logged the first time.
	replaying := workflow.IsReplaying(ctx)

	switch {
	case l.quiet || replaying:
	case l.logLevel == "debug":
		log.Debug("workflow started")
	default:
		log.Info("workflow started")
	}

	result, err := l.Next.ExecuteWorkflow(ctx, in)

	switch {
	case workflow.IsReplaying(ctx):
	case err != nil:
		log.Error("workflow failed", zap.Error(err))
	case l.quiet:
	case l.logLevel == "debug":
		log.Debug("workflow completed")
	default:
		log.Info("workflow completed")
	}

	return result, err
}
