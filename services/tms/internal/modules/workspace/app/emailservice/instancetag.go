package emailservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ConfiguredInstance struct {
	ID string
}

var _ emailports.InstanceTags = ConfiguredInstance{}

func NewConfiguredInstance(cfg *managedconfig.Config) emailports.InstanceTags {
	return ConfiguredInstance{ID: cfg.InstanceID}
}

func (c ConfiguredInstance) InstanceTag(context.Context, pagination.TenantInfo) (string, error) {
	return c.ID, nil
}
