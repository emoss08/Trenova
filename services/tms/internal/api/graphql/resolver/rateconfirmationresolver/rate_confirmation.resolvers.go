package rateconfirmationresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
)

func (r *MutationResolver) GenerateRateConfirmation(ctx context.Context, moveID string) (*rateconfirmation.RateConfirmation, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceRateConfirmation, permission.OpCreate)
	if err != nil {
		return nil, err
	}

	parsedMoveID, err := gqlid.ParseIDField("moveId", moveID)
	if err != nil {
		return nil, err
	}

	return r.RateConfirmationService.Generate(
		ctx,
		resolvercore.TenantInfo(authCtx),
		parsedMoveID,
		actorutil.FromAuthContext(authCtx),
	)
}

func (r *MutationResolver) SendRateConfirmation(ctx context.Context, id string) (*rateconfirmation.RateConfirmation, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceRateConfirmation, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	rateConfirmationID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	return r.RateConfirmationService.Send(ctx, resolvercore.TenantInfo(authCtx), rateConfirmationID)
}

func (r *MutationResolver) ConfirmRateConfirmation(ctx context.Context, id string, confirmedByName string) (*rateconfirmation.RateConfirmation, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceRateConfirmation, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	rateConfirmationID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	return r.RateConfirmationService.MarkConfirmed(
		ctx,
		resolvercore.TenantInfo(authCtx),
		rateConfirmationID,
		confirmedByName,
	)
}

func (r *MutationResolver) VoidRateConfirmation(ctx context.Context, id string, reason string) (*rateconfirmation.RateConfirmation, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceRateConfirmation, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	rateConfirmationID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	return r.RateConfirmationService.Void(
		ctx,
		resolvercore.TenantInfo(authCtx),
		rateConfirmationID,
		reason,
	)
}

func (r *QueryResolver) RateConfirmationsByMove(ctx context.Context, moveID string) ([]*rateconfirmation.RateConfirmation, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceRateConfirmation, permission.OpRead)
	if err != nil {
		return nil, err
	}

	parsedMoveID, err := gqlid.ParseIDField("moveId", moveID)
	if err != nil {
		return nil, err
	}

	return r.RateConfirmationService.ListByMove(
		ctx,
		&rateconfirmationports.ListRateConfirmationsByMoveRequest{
			TenantInfo:     resolvercore.TenantInfo(authCtx),
			ShipmentMoveID: parsedMoveID,
		},
	)
}

func (r *RateConfirmationResolver) GeneratedVia(ctx context.Context, obj *rateconfirmation.RateConfirmation) (string, error) {
	return obj.GeneratedVia.String(), nil
}

func (r *RateConfirmationResolver) ConfirmedVia(ctx context.Context, obj *rateconfirmation.RateConfirmation) (string, error) {
	return obj.ConfirmedVia.String(), nil
}
