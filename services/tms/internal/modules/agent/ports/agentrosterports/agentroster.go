package agentrosterports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentroster"
	"github.com/emoss08/trenova/pkg/pagination"
)

type AgentRosterService interface {
	Roster(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*agentroster.Stat, error)
}
