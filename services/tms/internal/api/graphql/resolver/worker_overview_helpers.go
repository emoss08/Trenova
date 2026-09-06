package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/workeroverviewservice"
	"github.com/emoss08/trenova/pkg/authctx"
)

// overviewSections turns the signed-in user's permissions into the set of
// roll-ups the overview may compose. A section the user cannot read is left
// out entirely rather than returned empty, so a dispatcher without safety
// access sees a verdict drawn only from what they are allowed to see instead
// of one that quietly claims the safety record is clean.
func (r *Resolver) overviewSections(
	ctx context.Context,
	authCtx *authctx.AuthContext,
) workeroverviewservice.Sections {
	return workeroverviewservice.Sections{
		Credentials: r.hasPermission(ctx, authCtx, permission.ResourceWorkerCredential, permission.OpRead),
		Training:    r.hasPermission(ctx, authCtx, permission.ResourceWorkerTraining, permission.OpRead),
		Safety:      r.hasPermission(ctx, authCtx, permission.ResourceWorkerSafetyEvent, permission.OpRead),
		Checklists:  r.hasPermission(ctx, authCtx, permission.ResourceWorkerChecklist, permission.OpRead),
		PTO:         r.hasPermission(ctx, authCtx, permission.ResourceWorkerPTO, permission.OpRead),
		Reviews:     r.hasPermission(ctx, authCtx, permission.ResourcePerformanceReview, permission.OpRead),
	}
}
