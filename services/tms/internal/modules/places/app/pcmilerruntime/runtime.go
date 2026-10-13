package pcmilerruntime

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/places/domain/distancecontrol"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/places/ports/distanceprofileports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pcmiler"
)

type ProfileReader interface {
	GetByID(
		ctx context.Context,
		req distanceprofileports.GetDistanceProfileByIDRequest,
	) (*distanceprofile.DistanceProfile, error)
	EnsureDefault(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*distanceprofile.DistanceProfile, error)
}

type IntegrationReader interface {
	GetRuntimeConfig(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ integration.Type,
	) (*integrationservice.RuntimeConfig, error)
}

type Runtime struct {
	Client  *pcmiler.Client
	Options pcmiler.RouteOptions
	Profile *distanceprofile.DistanceProfile
	Ready   bool
}

type ResolveParams struct {
	Profiles     ProfileReader
	Integrations IntegrationReader
	TenantInfo   pagination.TenantInfo
	Control      *distancecontrol.DistanceControl
	Purpose      string
}

func Resolve(ctx context.Context, p *ResolveParams) Runtime {
	var profile *distanceprofile.DistanceProfile
	var err error
	if p.Control != nil {
		profile, err = p.Profiles.GetByID(ctx, distanceprofileports.GetDistanceProfileByIDRequest{
			ID:         p.Control.ProfileIDForPurpose(p.Purpose),
			TenantInfo: p.TenantInfo,
		})
	} else {
		profile, err = p.Profiles.EnsureDefault(ctx, p.TenantInfo)
	}
	if err != nil || profile == nil || profile.Status != distanceprofile.StatusActive {
		return Runtime{}
	}

	runtime := Runtime{Options: profile.RouteOptions(), Profile: profile}
	cfg, err := p.Integrations.GetRuntimeConfig(ctx, p.TenantInfo, integration.TypePCMiler)
	if err != nil || cfg == nil || !cfg.Ready {
		return runtime
	}

	client, err := pcmiler.New(pcmiler.Config{
		APIKey:  cfg.Config["apiKey"],
		BaseURL: cfg.Config["baseUrl"],
	})
	if err != nil {
		return runtime
	}

	runtime.Client = client
	runtime.Ready = true
	return runtime
}
