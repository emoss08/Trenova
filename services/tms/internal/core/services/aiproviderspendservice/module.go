package aiproviderspendservice

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

var Module = fx.Module("aiprovider-spend-service",
	fx.Provide(
		New,
		func(s *Service) services.AIProviderSpendService { return s },
	),
)
