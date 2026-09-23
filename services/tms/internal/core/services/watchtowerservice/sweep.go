package watchtowerservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

// Backfill fills the feed from what every source reports open right now.
// It is what switching the watchtower on runs, so the first look at it is
// not an empty page waiting for the next event.
func (s *Service) Backfill(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*services.WatchtowerSweepResult, error) {
	return s.sweep(ctx, tenant, false)
}

// Reconcile corrects the feed against the sources: what they report open is
// upserted, and open items they no longer report are resolved. A projection
// lost to a failed write or a code path that forgot to project is caught
// here, which is why the sources own a snapshot rather than the feed
// trusting itself.
func (s *Service) Reconcile(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*services.WatchtowerSweepResult, error) {
	return s.sweep(ctx, tenant, true)
}

func (s *Service) sweep(
	ctx context.Context,
	tenant pagination.TenantInfo,
	resolveMissing bool,
) (*services.WatchtowerSweepResult, error) {
	result := &services.WatchtowerSweepResult{Failed: []string{}}
	for _, kind := range watchtower.AllSourceKinds() {
		source, ok := s.sources[kind]
		if !ok {
			continue
		}
		snapshot, err := source.Snapshot(ctx, tenant)
		if err != nil {
			s.l.Warn("watchtower source could not be read",
				zap.String("kind", string(kind)),
				zap.String("organization", tenant.OrgID.String()),
				zap.Error(err),
			)
			result.Failed = append(result.Failed, string(kind))

			continue
		}

		open := make([]string, 0, len(snapshot))
		for _, input := range snapshot {
			input.TenantInfo = tenant
			input.SourceKind = kind
			if _, uErr := s.projector.upsert(ctx, input); uErr != nil {
				s.l.Warn("watchtower snapshot item rejected",
					zap.String("kind", string(kind)),
					zap.String("source", input.SourceID),
					zap.Error(uErr),
				)

				continue
			}
			open = append(open, input.SourceID)
			result.Upserted++
		}

		if !resolveMissing {
			continue
		}
		resolved, rErr := s.repo.ResolveMissing(
			ctx,
			repositories.ResolveMissingWatchtowerItemsRequest{
				TenantInfo:    tenant,
				SourceKind:    kind,
				OpenSourceIDs: open,
				ResolvedAt:    s.now(),
			},
		)
		if rErr != nil {
			s.l.Warn("watchtower reconcile could not resolve",
				zap.String("kind", string(kind)), zap.Error(rErr))
			result.Failed = append(result.Failed, string(kind))

			continue
		}
		result.Resolved += resolved
	}

	return result, nil
}

// RegisteredKinds is what the deployment can reconcile: the kinds with a
// source behind them.
func (s *Service) RegisteredKinds() []watchtower.SourceKind {
	kinds := make([]watchtower.SourceKind, 0, len(s.sources))
	for _, kind := range watchtower.AllSourceKinds() {
		if _, ok := s.sources[kind]; ok {
			kinds = append(kinds, kind)
		}
	}

	return kinds
}
