package interceptors

import (
	"context"
	"time"

	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type LoggingInterceptor struct {
	interceptor.InterceptorBase
	logger   *zap.Logger
	logLevel string
}

func NewLoggingInterceptor(logger *zap.Logger, logLevel string) *LoggingInterceptor {
	return &LoggingInterceptor{
		logger:   logger.Named("temporal"),
		logLevel: logLevel,
	}
}

func (l *LoggingInterceptor) InterceptActivity(
	_ context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &loggingActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		logger:                         l.logger,
		logLevel:                       l.logLevel,
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
	}
}

type loggingActivityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	logger   *zap.Logger
	logLevel string
}

func (l *loggingActivityInbound) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (any, error) {
	start := time.Now()

	log := l.logger.With(
		zap.String("type", "activity"),
	)

	if l.logLevel == "debug" { //nolint:goconst // no need to check this.
		log.Debug("activity started")
	}

	result, err := l.Next.ExecuteActivity(ctx, in)

	duration := time.Since(start)
	fields := []zap.Field{
		zap.Duration("duration", duration),
	}

	switch {
	case err != nil:
		log.Error("activity failed", append(fields, zap.Error(err))...)
	case l.logLevel == "debug":
		log.Debug("activity completed", fields...)
	default:
		log.Info("activity completed", fields...)
	}

	return result, err
}

type loggingWorkflowInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	logger   *zap.Logger
	logLevel string
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

	if l.logLevel == "debug" {
		log.Debug("workflow started")
	} else {
		log.Info("workflow started")
	}

	result, err := l.Next.ExecuteWorkflow(ctx, in)

	switch {
	case err != nil:
		log.Error("workflow failed", zap.Error(err))
	case l.logLevel == "debug":
		log.Debug("workflow completed")
	default:
		log.Info("workflow completed")
	}

	return result, err
}
