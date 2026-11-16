package commands

import (
	"context"

	"github.com/emoss08/trenova/operator/internal/redact"
)

func (r *Runner) redactor(ctx context.Context) (func(string) string, error) {
	red, err := redact.Load(ctx, r.kube, r.namespace)
	if err != nil {
		return nil, err
	}
	return red.Line, nil
}
