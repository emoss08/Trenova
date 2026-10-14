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
	secondsPerHour    = 60 * 60
	secondsPerDay     = 24 * secondsPerHour
	RetentionDays     = 90
	BackfillDays      = 30
	OverlapSeconds    = 15 * 60
	ChunkSeconds      = secondsPerDay
	MaxChunksPerCycle = 14
)

func (s *Service) Refresh(
	ctx context.Context,
	tenant pagination.TenantInfo,
	progress aianalyticsports.RefreshProgress,
) (*aianalyticsports.RefreshResult, error) {
	now := s.clock().Unix()
	state, err := s.refresher.RefreshState(ctx, tenant)
	if err != nil {
		return nil, err
	}

	from := now - BackfillDays*secondsPerDay
	if state != nil {
		from = state.RefreshedThrough - OverlapSeconds
	}
	from = max(from, now-RetentionDays*secondsPerDay)

	result := &aianalyticsports.RefreshResult{RefreshedThrough: from}
	active, err := s.refresher.HasActivity(ctx, aianalyticsports.RefreshRequest{
		TenantInfo: tenant,
		From:       from,
		To:         now,
	})
	if err != nil {
		return nil, err
	}
	if active {
		err = s.refreshChunks(ctx, &chunkRun{
			tenant:   tenant,
			now:      now,
			result:   result,
			progress: progress,
		})
		s.invalidate(ctx, tenant, result)
		if err != nil {
			return nil, err
		}
	} else if err = s.markRefreshed(ctx, tenant, now, result); err != nil {
		return nil, err
	}
	result.CaughtUp = result.RefreshedThrough >= now

	if pruneDue(state, now) {
		if err = s.refresher.Prune(ctx, aianalyticsports.PruneRequest{
			TenantInfo: tenant,
			Before:     now - RetentionDays*secondsPerDay,
		}); err != nil {
			return nil, err
		}
	}

	s.l.Debug("ai analytics refreshed",
		zap.String("organization", tenant.OrgID.String()),
		zap.Bool("active", active),
		zap.Int("units", result.Units),
		zap.Int("hours", result.Hours),
		zap.Bool("caughtUp", result.CaughtUp),
	)

	return result, nil
}

func pruneDue(state *aianalytics.RefreshState, now int64) bool {
	return state == nil || now/secondsPerHour != state.RefreshedAt/secondsPerHour
}

func (s *Service) markRefreshed(
	ctx context.Context,
	tenant pagination.TenantInfo,
	now int64,
	result *aianalyticsports.RefreshResult,
) error {
	if err := s.refresher.SaveRefreshState(ctx, &aianalytics.RefreshState{
		OrganizationID:   tenant.OrgID,
		BusinessUnitID:   tenant.BuID,
		RefreshedThrough: now,
		RefreshedAt:      now,
	}); err != nil {
		return err
	}
	result.RefreshedThrough = now

	return nil
}

type chunkRun struct {
	tenant   pagination.TenantInfo
	now      int64
	result   *aianalyticsports.RefreshResult
	progress aianalyticsports.RefreshProgress
}

func (s *Service) refreshChunks(ctx context.Context, run *chunkRun) error {
	tenant, now, result := run.tenant, run.now, run.result
	from := result.RefreshedThrough
	lastStamp := int64(0)
	for chunk := 0; chunk < MaxChunksPerCycle && from < now; chunk++ {
		to := min(from+ChunkSeconds, now)
		stamp := max(s.clock().UnixMilli(), lastStamp+1)
		lastStamp = stamp

		req := aianalyticsports.RefreshRequest{TenantInfo: tenant, From: from, To: to, Stamp: stamp}
		units, err := s.refresher.RefreshUnits(ctx, req)
		if err != nil {
			return err
		}
		result.Units += units
		hours, err := s.refresher.RefreshProviderHours(ctx, req)
		if err != nil {
			return err
		}
		result.Hours += hours
		if err = s.keyModels(ctx, aianalyticsports.ModelKeyRequest{
			TenantInfo: tenant,
			Stamp:      stamp,
			FromHour:   from - from%secondsPerHour,
		}); err != nil {
			return err
		}
		if err = s.refresher.SaveRefreshState(ctx, &aianalytics.RefreshState{
			OrganizationID:   tenant.OrgID,
			BusinessUnitID:   tenant.BuID,
			RefreshedThrough: to,
			RefreshedAt:      s.clock().Unix(),
		}); err != nil {
			return err
		}

		result.RefreshedThrough = to
		from = to
		if run.progress != nil {
			run.progress(to)
		}
	}

	return nil
}

func (s *Service) keyModels(ctx context.Context, req aianalyticsports.ModelKeyRequest) error {
	models, err := s.refresher.UnkeyedModels(ctx, req)
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
	if err = s.refresher.SetModelKeys(ctx, req, keys); err != nil {
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
