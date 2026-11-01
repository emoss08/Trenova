package paletteresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (r *MutationResolver) AskPalette(ctx context.Context, input gqlmodel.AskPaletteInput) (*paletteports.Ask, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAssistant, permission.OpRead)
	if err != nil {
		return nil, err
	}

	question, err := questionFromInput(&input)
	if err != nil {
		return nil, err
	}

	return r.PaletteService.Ask(ctx, &paletteports.AskRequest{
		Question:   question,
		TenantInfo: resolvercore.TenantInfo(authCtx),
	}, actorutil.FromAuthContext(authCtx))
}

func (r *MutationResolver) StopPaletteAsk(ctx context.Context, askID string) (bool, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAssistant, permission.OpRead)
	if err != nil {
		return false, err
	}

	ref, err := askRef(authCtx, askID)
	if err != nil {
		return false, err
	}
	if err = r.PaletteService.Stop(ctx, ref); err != nil {
		return false, err
	}

	return true, nil
}

func (r *MutationResolver) KeepPaletteAsk(ctx context.Context, askID string) (*gqlmodel.KeptPaletteAsk, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAssistant, permission.OpCreate)
	if err != nil {
		return nil, err
	}

	ref, err := askRef(authCtx, askID)
	if err != nil {
		return nil, err
	}

	kept, err := r.PaletteService.Keep(ctx, ref, actorutil.FromAuthContext(authCtx))
	if err != nil {
		return nil, err
	}

	return keptToModel(kept), nil
}
