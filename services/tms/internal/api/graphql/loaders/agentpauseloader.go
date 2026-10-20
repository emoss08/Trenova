package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type AgentsPausedLoaderFactoryParams struct {
	fx.In

	Controls tenantports.AgentControlRepository
}

type AgentsPausedLoaderFactory struct {
	controls tenantports.AgentControlRepository
}

func NewAgentsPausedLoaderFactory(p AgentsPausedLoaderFactoryParams) *AgentsPausedLoaderFactory {
	return &AgentsPausedLoaderFactory{controls: p.Controls}
}

func newAgentsPausedLoader(f *AgentsPausedLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, bool] {
	return newLoader(batchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]bool, error) {
			paused := make(map[pulid.ID]bool, len(ids))
			if f.controls == nil {
				return paused, nil
			}
			control, err := f.controls.GetOrCreate(ctx, tenantInfo)
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				paused[id] = id == tenantInfo.OrgID && control.ShadowMode
			}

			return paused, nil
		},
	))
}
