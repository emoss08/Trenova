package aiproviderspendservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"go.uber.org/fx"
)

var Module = fx.Module("aiprovider-spend-service",
	fx.Provide(
		New,
		func(s *Service) aiproviderports.AIProviderSpendService { return s },
	),
)
