package resolver

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"go.uber.org/zap"
)

// attentionSectionRunner returns a helper that executes one attention-summary
// section on its own goroutine. Sections the user cannot read are skipped, and
// section failures degrade to a missing count instead of failing the query.
func attentionSectionRunner(
	ctx context.Context,
	r *Resolver,
	authCtx *authctx.AuthContext,
	wg *sync.WaitGroup,
) func(resource permission.Resource, name string, fetch func(context.Context) error) {
	return func(resource permission.Resource, name string, fetch func(context.Context) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !r.hasPermission(ctx, authCtx, resource, permission.OpRead) {
				return
			}
			if err := fetch(ctx); err != nil {
				r.l.Warn("attention summary section failed",
					zap.String("section", name),
					zap.Error(err))
			}
		}()
	}
}
