package controlplane

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
)

type CloudEntitlementProviderParams struct {
	fx.In

	Config   *config.Config
	Client   Client
	Registry *platformcatalog.Registry
}

type CloudEntitlementProvider struct {
	cfg      *config.Config
	client   Client
	registry *platformcatalog.Registry
}

func NewCloudEntitlementProvider(p CloudEntitlementProviderParams) *CloudEntitlementProvider {
	return &CloudEntitlementProvider{
		cfg:      p.Config,
		client:   p.Client,
		registry: p.Registry,
	}
}

func (p *CloudEntitlementProvider) CheckFeature(
	ctx context.Context,
	req *services.FeatureCheckRequest,
) (*services.FeatureCheckResult, error) {
	result, err := p.client.CheckFeature(ctx, req)
	if err == nil {
		return result, nil
	}
	if !failOpenAllowed(p.cfg) {
		return nil, err
	}

	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = nowUnix()
	}

	return &services.FeatureCheckResult{
		FeatureKey: req.FeatureKey,
		Allowed:    true,
		Reason:     fmt.Sprintf("fail_open:%s", err.Error()),
		CheckedAt:  checkedAt,
		FailOpen:   true,
	}, nil
}

func (p *CloudEntitlementProvider) ListEntitlements(
	ctx context.Context,
	req *services.EntitlementsRequest,
) (*services.EntitlementsResult, error) {
	features := p.registry.ListFeatures()
	results := make([]services.FeatureCheckResult, 0, len(features))
	for i := range features {
		feature := features[i]
		checkResult, err := p.CheckFeature(ctx, &services.FeatureCheckRequest{
			OrganizationID: req.OrganizationID,
			BusinessUnitID: req.BusinessUnitID,
			PrincipalType:  req.PrincipalType,
			PrincipalID:    req.PrincipalID,
			UserID:         req.UserID,
			APIKeyID:       req.APIKeyID,
			FeatureKey:     feature.Key,
			CheckedAt:      req.CheckedAt,
		})
		if err != nil {
			return nil, err
		}
		results = append(results, *checkResult)
	}

	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = nowUnix()
	}

	return &services.EntitlementsResult{
		Features:  results,
		CheckedAt: checkedAt,
	}, nil
}
