package aiproviderservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) providersChanged(
	ctx context.Context,
	tenant pagination.TenantInfo,
	touched ...*aiprovider.Provider,
) {
	s.analyticsChanged(ctx, tenant)
	if RoutesEmbedding(touched...) {
		s.indexer.Wake(ctx, tenant)
	}
}

func (s *Service) changesChanged(
	ctx context.Context,
	tenant pagination.TenantInfo,
	changes []change,
) {
	touched := make([]*aiprovider.Provider, 0, len(changes)*2)
	for idx := range changes {
		touched = append(touched, changes[idx].saved, changes[idx].previous)
	}
	s.providersChanged(ctx, tenant, touched...)
}

func RoutesEmbedding(providers ...*aiprovider.Provider) bool {
	return slices.ContainsFunc(providers, func(provider *aiprovider.Provider) bool {
		return provider != nil && provider.ServesTask(aitask.TaskEmbedding)
	})
}
