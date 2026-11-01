package billingqueueworkspaceresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingapprovalrun"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuefilterpreset"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueuefilterpresetports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (r *BillingQueueBulkRunItemResolver) FailureCode(ctx context.Context, obj *billingapprovalrun.ApprovalRunItem) (string, error) {
	return string(obj.FailureCode), nil
}

func (r *BillingQueueLaneStatResolver) Total(ctx context.Context, obj *billingqueueports.BillingQueueLaneStat) (string, error) {
	return obj.Total.String(), nil
}

func (r *BillingQueueNovaGenerationResolver) RetryAt(ctx context.Context, obj *billingqueueports.NovaGeneration) (*int, error) {
	return positiveInt64(obj.RetryAt), nil
}

func (r *BillingQueuePulseResolver) DeliveryToInvoiceDays(ctx context.Context, obj *billingqueueports.BillingQueuePulse) (*string, error) {
	return gqldecimal.NullDecimalStringPtr(obj.DeliveryToInvoiceDays), nil
}

func (r *BillingQueuePulseResolver) PriorDeliveryToInvoiceDays(ctx context.Context, obj *billingqueueports.BillingQueuePulse) (*string, error) {
	return gqldecimal.NullDecimalStringPtr(obj.PriorDeliveryToInvoice), nil
}

func (r *BillingQueuePulseResolver) ApprovedTodayTotal(ctx context.Context, obj *billingqueueports.BillingQueuePulse) (string, error) {
	return obj.ApprovedTodayTotal.String(), nil
}

func (r *BillingQueueRecommendationResolver) Total(ctx context.Context, obj *billingqueueports.Recommendation) (string, error) {
	return obj.Total.String(), nil
}

func (r *BillingQueueSearchDatesResolver) LastDays(ctx context.Context, obj *billingqueueports.SearchDates) (*int, error) {
	if obj.LastDays <= 0 {
		return nil, nil //nolint:nilnil // a window with no day count maps to null
	}
	days := obj.LastDays

	return &days, nil
}

func (r *BillingQueueSearchDatesResolver) From(ctx context.Context, obj *billingqueueports.SearchDates) (*string, error) {
	return gqlscalar.EmptyToNil(obj.From), nil
}

func (r *BillingQueueSearchDatesResolver) To(ctx context.Context, obj *billingqueueports.SearchDates) (*string, error) {
	return gqlscalar.EmptyToNil(obj.To), nil
}

func (r *BillingQueueSearchIntentResolver) MinAmount(ctx context.Context, obj *billingqueueports.SearchIntent) (*string, error) {
	return gqldecimal.NullDecimalStringPtr(obj.MinAmount), nil
}

func (r *BillingQueueSearchIntentResolver) MaxAmount(ctx context.Context, obj *billingqueueports.SearchIntent) (*string, error) {
	return gqldecimal.NullDecimalStringPtr(obj.MaxAmount), nil
}

func (r *BillingQueueStatsResolver) OpenTotal(ctx context.Context, obj *billingqueueports.BillingQueueStats) (string, error) {
	return obj.OpenTotal.String(), nil
}

func (r *QueryResolver) BillingQueueStats(ctx context.Context, dayStart *int) (*billingqueueports.BillingQueueStats, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpRead)
	if err != nil {
		return nil, err
	}

	return r.BillingQueueService.GetStats(ctx, &billingqueueports.GetBillingQueueStatsRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		DayStart:   gqlscalar.Int64Value(dayStart),
	})
}

func (r *QueryResolver) BillingQueueRecommendations(ctx context.Context) ([]*billingqueueports.Recommendation, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpRead)
	if err != nil {
		return nil, err
	}

	recommendations, err := r.BillingQueueService.Recommendations(
		ctx,
		&billingqueueports.ListRecommendationsRequest{
			TenantInfo: resolvercore.TenantInfo(authCtx),
			UserID:     authCtx.UserID,
			CanPost:    r.HasPermission(ctx, authCtx, permission.ResourceInvoice, permission.OpPost),
			CanAssign: r.HasPermission(
				ctx,
				authCtx,
				permission.ResourceBillingQueue,
				permission.OpAssign,
			),
		},
	)
	if err != nil {
		return nil, err
	}
	if recommendations == nil {
		return []*billingqueueports.Recommendation{}, nil
	}

	return recommendations, nil
}

func (r *QueryResolver) BillingQueueAssist(ctx context.Context) (*billingqueueports.AssistAvailability, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpRead)
	if err != nil {
		return nil, err
	}

	return r.BillingQueueAssistService.Availability(ctx, resolvercore.TenantInfo(authCtx))
}

func (r *QueryResolver) BillingQueueBrief(ctx context.Context) (*billingqueueports.QueueBriefView, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpRead)
	if err != nil {
		return nil, err
	}

	return r.BillingQueueAssistService.GetBrief(ctx, resolvercore.TenantInfo(authCtx))
}

func (r *QueryResolver) BillingQueueNovaGeneration(ctx context.Context) (*billingqueueports.NovaGeneration, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpRead)
	if err != nil {
		return nil, err
	}

	return r.BillingQueueAssistService.NovaGeneration(ctx, resolvercore.TenantInfo(authCtx))
}

func (r *QueryResolver) BillingQueueSearchIntent(ctx context.Context, text string, today *string) (*billingqueueports.SearchIntent, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpRead)
	if err != nil {
		return nil, err
	}

	return r.BillingQueueAssistService.ComposeSearch(ctx, &billingqueueports.SearchIntentRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		UserID:     authCtx.UserID,
		Text:       text,
		Today:      gqlscalar.StringValue(today),
	})
}

func (r *QueryResolver) BillingQueueFilterPresets(ctx context.Context) ([]*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpRead)
	if err != nil {
		return nil, err
	}

	presets, err := r.BillingQueueFilterPresetRepo.ListByUserID(
		ctx,
		&billingqueuefilterpresetports.ListBillingQueueFilterPresetsRequest{
			UserID:     authCtx.UserID,
			TenantInfo: resolvercore.TenantInfo(authCtx),
		},
	)
	if err != nil {
		return nil, err
	}
	if presets == nil {
		return []*billingqueuefilterpreset.BillingQueueFilterPreset{}, nil
	}

	return presets, nil
}

func (r *QueryResolver) BillingQueueBulkRun(ctx context.Context, id string) (*billingapprovalrun.ApprovalRun, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpRead)
	if err != nil {
		return nil, err
	}

	req, err := runRequest(id, authCtx)
	if err != nil {
		return nil, err
	}

	return r.BillingQueueApprovalService.Get(ctx, req)
}

func (r *MutationResolver) CreateBillingQueueFilterPreset(ctx context.Context, input gqlmodel.BillingQueueFilterPresetInput) (*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpRead)
	if err != nil {
		return nil, err
	}

	entity, err := filterPresetFromInput(&input, authCtx)
	if err != nil {
		return nil, err
	}

	return r.BillingQueueFilterPresetRepo.Create(ctx, entity)
}

func (r *MutationResolver) UpdateBillingQueueFilterPreset(ctx context.Context, id string, input gqlmodel.BillingQueueFilterPresetInput) (*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpRead)
	if err != nil {
		return nil, err
	}

	presetID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}
	entity, err := filterPresetFromInput(&input, authCtx)
	if err != nil {
		return nil, err
	}
	entity.ID = presetID

	return r.BillingQueueFilterPresetRepo.Update(ctx, entity)
}

func (r *MutationResolver) DeleteBillingQueueFilterPreset(ctx context.Context, id string) (bool, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpRead)
	if err != nil {
		return false, err
	}

	presetID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return false, err
	}

	if err = r.BillingQueueFilterPresetRepo.Delete(
		ctx,
		&billingqueuefilterpresetports.DeleteBillingQueueFilterPresetRequest{
			PresetID:   presetID,
			UserID:     authCtx.UserID,
			TenantInfo: resolvercore.TenantInfo(authCtx),
		},
	); err != nil {
		return false, err
	}

	return true, nil
}

func (r *MutationResolver) StartBillingQueueBulkRun(ctx context.Context, input gqlmodel.StartBillingQueueBulkRunInput) (*billingapprovalrun.ApprovalRun, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	if len(input.ItemIds) > billingapprovalrun.MaxApprovalRunItems {
		return nil, errortypes.NewValidationError(
			"itemIds", errortypes.ErrInvalid, "Act on at most 500 items at a time",
		)
	}
	if err = r.requireBulkPermission(ctx, authCtx, input.Action); err != nil {
		return nil, err
	}

	req, err := startBulkRequest(&input, resolvercore.TenantInfo(authCtx))
	if err != nil {
		return nil, err
	}

	return r.BillingQueueApprovalService.Start(ctx, req)
}

func (r *MutationResolver) CancelBillingQueueBulkRun(ctx context.Context, id string) (*billingapprovalrun.ApprovalRun, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	req, err := runRequest(id, authCtx)
	if err != nil {
		return nil, err
	}

	return r.BillingQueueApprovalService.Undo(ctx, req)
}

func (r *MutationResolver) RedetectBillingQueuePodSignatures(ctx context.Context) (*billingqueueports.RedetectPODSignaturesResult, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceBillingQueue, permission.OpUpdate)
	if err != nil {
		return nil, err
	}
	if _, err = r.RequirePermission(ctx, permission.ResourceDocument, permission.OpUpdate); err != nil {
		return nil, err
	}

	return r.BillingQueueReviewService.RedetectPODSignatures(
		ctx,
		&billingqueueports.RedetectPODSignaturesRequest{
			TenantInfo: resolvercore.TenantInfo(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
}
