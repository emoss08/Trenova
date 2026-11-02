package triagenova

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

var ErrUnavailable = errortypes.NewBusinessError(
	"Nova cannot draft here: no AI provider is set up for inbound mail. Write it by hand, " +
		"or ask an administrator to connect one.",
)

type Params struct {
	fx.In

	Nova llmports.TaskRunner
}

// Caller runs the inbox's Nova tasks through the harness, which decides
// whether the call runs here or as a worker job.
type Caller struct {
	nova llmports.TaskRunner
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Caller {
	return &Caller{nova: p.Nova}
}

func (c *Caller) Available(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error) {
	return c.nova.Available(ctx, tenantInfo)
}

func (c *Caller) Run(
	ctx context.Context,
	task *llmports.NovaTask,
	out any,
) (*llmports.StructuredCompletionResult, error) {
	if task.Name == "" {
		task.Name = aitask.TaskInboundClassification
	}
	if task.Feature == "" {
		task.Feature = aiusage.FeatureInboundMail
	}

	result, err := c.nova.Run(ctx, task, out)
	var failed *llmports.ChatProvidersFailedError
	switch {
	case errors.Is(err, llmports.ErrNoProviderConfigured):
		return nil, ErrUnavailable
	case errors.As(err, &failed):
		return nil, errortypes.NewBusinessError(
			"Nova could not reach an AI provider just now. Try again in a moment.",
		).WithInternal(err)
	}

	return result, err
}

type Completer interface {
	Available(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error)
	Run(
		ctx context.Context,
		task *llmports.NovaTask,
		out any,
	) (*llmports.StructuredCompletionResult, error)
}

var _ Completer = (*Caller)(nil)
