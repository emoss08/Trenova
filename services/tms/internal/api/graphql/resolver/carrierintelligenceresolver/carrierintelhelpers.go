package carrierintelligenceresolver

import (
	"context"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/pkg/pagination"
)

func providerInfoToModel(
	provider string,
	fallback *string,
	descriptor carrierintel.ProviderDescriptor,
	configured bool,
) *gqlmodel.CarrierIntelProviderInfo {
	info := &gqlmodel.CarrierIntelProviderInfo{
		Configured:       configured,
		FallbackProvider: fallback,
		Capabilities:     descriptor.Capabilities.Names(),
		Sections:         slices.Clone(descriptor.Sections),
	}
	if info.Sections == nil {
		info.Sections = []carrierintel.Section{}
	}
	if provider != "" {
		info.Provider = &provider
	}
	return info
}

func (r *Deps) CarrierIntelNow() int64 {
	return time.Now().Unix()
}

func (r *Deps) CarrierIntelProviderInfo(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*gqlmodel.CarrierIntelProviderInfo, error) {
	control, err := r.CarrierIntelService.Control(ctx, tenant)
	if err != nil {
		return nil, err
	}
	var fallback *string
	if provider, ok := control.FallbackType(); ok {
		value := provider.String()
		fallback = &value
	}
	provider, descriptor, configured, err := r.CarrierIntelService.ProviderDescriptor(ctx, tenant)
	if err != nil {
		return nil, err
	}
	return providerInfoToModel(provider.String(), fallback, descriptor, configured), nil
}
