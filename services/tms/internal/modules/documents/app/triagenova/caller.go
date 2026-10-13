package triagenova

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
)

var ErrUnavailable = errortypes.NewBusinessError(
	"Nova cannot draft here: no AI provider is set up for inbound mail. Write it by hand, " +
		"or ask an administrator to connect one.",
)

type Mode int

const (
	Interactive Mode = iota
	Background
)

type Params struct {
	fx.In

	Interactive agentports.StructuredCompleter `optional:"true"`
	Background  agentports.CompletionService   `optional:"true"`
}

type Caller struct {
	interactive agentports.StructuredCompleter
	background  agentports.CompletionService
}

func New(p Params) *Caller {
	return &Caller{interactive: p.Interactive, background: p.Background}
}

func (c *Caller) Available(mode Mode) bool {
	return c.completer(mode) != nil
}

func (c *Caller) Complete(
	ctx context.Context,
	mode Mode,
	req *agentports.StructuredCompletionRequest,
) (*agentports.StructuredCompletionResult, error) {
	completer := c.completer(mode)
	if completer == nil {
		return nil, ErrUnavailable
	}
	if req.Task == "" {
		req.Task = aiprovider.TaskInboundClassification
	}

	result, err := completer.CompleteStructured(ctx, req)
	if err != nil {
		var failed *agentports.ChatProvidersFailedError
		if errors.As(err, &failed) {
			return nil, errortypes.NewBusinessError(
				"Nova could not reach an AI provider just now. Try again in a moment.",
			).WithInternal(err)
		}

		return nil, err
	}

	return result, nil
}

type structuredCompleter interface {
	CompleteStructured(
		ctx context.Context,
		req *agentports.StructuredCompletionRequest,
	) (*agentports.StructuredCompletionResult, error)
}

func (c *Caller) completer(mode Mode) structuredCompleter {
	if mode == Background && c.background != nil {
		return c.background
	}
	if c.interactive != nil {
		return c.interactive
	}
	if c.background != nil {
		return c.background
	}

	return nil
}

type Completer interface {
	Available(mode Mode) bool
	Complete(
		ctx context.Context,
		mode Mode,
		req *agentports.StructuredCompletionRequest,
	) (*agentports.StructuredCompletionResult, error)
}

var _ Completer = (*Caller)(nil)
