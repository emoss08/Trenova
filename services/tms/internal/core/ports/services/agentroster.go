package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentroster"
	"github.com/emoss08/trenova/pkg/pagination"
)

type AgentRosterService interface {
	Roster(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*agentroster.Stat, error)
}
