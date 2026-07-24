package integrationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	sharedsamsara "github.com/emoss08/trenova/shared/samsara"
)

func (s *Service) SamsaraClient(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*sharedsamsara.Client, error) {
	runtimeCfg, err := s.GetRuntimeConfig(ctx, tenantInfo, integration.TypeSamsara)
	if err != nil {
		return nil, err
	}

	token := runtimeCfg.Config["token"]
	if token == "" {
		return nil, errortypes.NewBusinessError("Samsara integration is not configured")
	}

	client, err := sharedsamsara.New(
		token,
		sharedsamsara.WithBaseURL(runtimeCfg.Config["baseUrl"]),
	)
	if err != nil {
		return nil, errortypes.NewBusinessError(
			"failed to initialize Samsara client",
		).WithInternal(err)
	}

	return client, nil
}
