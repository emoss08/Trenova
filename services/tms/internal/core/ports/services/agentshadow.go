package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentshadow"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AgentShadowReportRequest struct {
	TenantInfo pagination.TenantInfo
	AgentID    pulid.ID
	Days       int
}

// AgentShadowService reports how an agent in shadow compares with the people
// doing the same work.
type AgentShadowService interface {
	Report(ctx context.Context, req *AgentShadowReportRequest) (*agentshadow.Report, error)
}
