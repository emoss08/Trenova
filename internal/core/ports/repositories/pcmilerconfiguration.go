package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/pcmilerconfiguration"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetPCMilerConfigurationOptions struct {
	OrgID pulid.ID
	BuID  pulid.ID
}

type PCMilerConfigurationRepository interface {
	GetPCMilerConfiguration(ctx context.Context, opts GetPCMilerConfigurationOptions) (*pcmilerconfiguration.PCMilerConfiguration, error)
}
