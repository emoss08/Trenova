package aianalyticsservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	secondsPerDay     = 24 * 60 * 60
	RetentionDays     = 90
	BackfillDays      = 30
	OverlapSeconds    = 15 * 60
	ChunkSeconds      = secondsPerDay
	MaxChunksPerCycle = 14
)

func (s *Service) Refresh(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*aianalyticsports.RefreshResult, error) {
	now := s.clock().Unix()
	state, err := s.repo.RefreshState(ctx, tenant)
	if err != nil {
		return nil, err
	}

	from := now - BackfillDays*secondsPerDay
	if state != nil {
		from = state.RefreshedThrough - OverlapSeconds
	}
	from = max(from, now-RetentionDays*secondsPerDay)

	result := &aianalyticsports.RefreshResult{RefreshedThrough: from}
	lastStamp := int64(0)
	for chunk := 0; chunk < MaxChunksPerCycle && from < now; chunk++ {
		to := min(from+ChunkSeconds, now)
		stamp := max(s.clock().UnixMilli(), lastStamp+1)
		lastStamp = stamp

		req := aianalyticsports.RefreshRequest{TenantInfo: tenant, From: from, To: to, Stamp: stamp}
		units, unitErr := s.repo.RefreshUnits(ctx, req)
		if unitErr != nil {
			return nil, unitErr
		}
		hours, hourErr := s.repo.RefreshProviderHours(ctx, req)
		if hourErr != nil {
			return nil, hourErr
		}
		if keyErr := s.keyModels(ctx, aianalyticsports.ModelKeyRequest{
			TenantInfo: tenant,
			Stamp:      stamp,
			FromHour:   from - from%3600,
		}); keyErr != nil {
			return nil, keyErr
		}
		if saveErr := s.repo.SaveRefreshState(ctx, &aianalytics.RefreshState{
			OrganizationID:   tenant.OrgID,
			BusinessUnitID:   tenant.BuID,
			RefreshedThrough: to,
			RefreshedAt:      s.clock().Unix(),
		}); saveErr != nil {
			return nil, saveErr
		}

		result.Units += units
		result.Hours += hours
		result.RefreshedThrough = to
		from = to
	}
	result.CaughtUp = result.RefreshedThrough >= now
	s.invalidate(ctx, tenant, result)

	if err = s.repo.Prune(ctx, aianalyticsports.PruneRequest{
		TenantInfo: tenant,
		Before:     now - RetentionDays*secondsPerDay,
	}); err != nil {
		return nil, err
	}

	s.l.Debug("ai analytics refreshed",
		zap.String("organization", tenant.OrgID.String()),
		zap.Int("units", result.Units),
		zap.Int("hours", result.Hours),
		zap.Bool("caughtUp", result.CaughtUp),
	)

	return result, nil
}

func (s *Service) keyModels(ctx context.Context, req aianalyticsports.ModelKeyRequest) error {
	models, err := s.repo.UnkeyedModels(ctx, req)
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return nil
	}

	keys := make([]aianalyticsports.ModelKey, 0, len(models))
	for _, model := range models {
		key := aianalytics.CanonicalModel(model)
		if key == "" {
			key = model
		}
		keys = append(keys, aianalyticsports.ModelKey{Model: model, Key: key})
	}
	if err = s.repo.SetModelKeys(ctx, req, keys); err != nil {
		return fmt.Errorf("key ai analytics models: %w", err)
	}

	return nil
}

func (s *Service) invalidate(
	ctx context.Context,
	tenant pagination.TenantInfo,
	result *aianalyticsports.RefreshResult,
) {
	if s.cache == nil || (result.Units == 0 && result.Hours == 0) {
		return
	}
	if err := s.cache.BumpEpoch(ctx, tenant); err != nil {
		s.l.Warn("failed to invalidate cached ai analytics",
			zap.String("organization", tenant.OrgID.String()),
			zap.Error(err),
		)
	}
}
