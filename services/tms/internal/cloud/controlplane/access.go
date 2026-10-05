package controlplane

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
)

type CloudAccessAuthorizerParams struct {
	fx.In

	Config *config.Config
	Client Client
}

type CloudAccessAuthorizer struct {
	cfg    *config.Config
	client Client
}

func NewCloudAccessAuthorizer(p CloudAccessAuthorizerParams) *CloudAccessAuthorizer {
	return &CloudAccessAuthorizer{
		cfg:    p.Config,
		client: p.Client,
	}
}

func (a *CloudAccessAuthorizer) AuthorizeAccess(
	ctx context.Context,
	req *services.AccessAuthorizeRequest,
) (*services.AccessAuthorizeResult, error) {
	result, err := a.client.AuthorizeAccess(ctx, req)
	if err == nil {
		return result, nil
	}
	if !failOpenAllowed(a.cfg) {
		return nil, err
	}

	checkedAt := req.CheckedAt
	if checkedAt == 0 {
		checkedAt = nowUnix()
	}

	return &services.AccessAuthorizeResult{
		FeatureKey: req.FeatureKey,
		Allowed:    true,
		Reason:     fmt.Sprintf("fail_open:%s", err.Error()),
		CheckedAt:  checkedAt,
		FailOpen:   true,
	}, nil
}
