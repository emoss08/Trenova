package systemconfig

import (
	"errors"

	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
)

var ErrProductionSystemUserPasswordRequired = errors.New(
	"production and staging require system.systemUserPassword that is not a placeholder",
)

type Config struct {
	SystemUserPassword string `mapstructure:"systemUserPassword" validate:"required,min=1,max=72"`
}

func (c *Config) ValidateProduction() error {
	if securityconfig.IsPlaceholder(c.SystemUserPassword) {
		return ErrProductionSystemUserPasswordRequired
	}

	return nil
}
