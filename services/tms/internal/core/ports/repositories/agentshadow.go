package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentshadow"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListShadowProposalsRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Since             int64
	Limit             int
}

type ListPersonChangesRequest struct {
	TenantInfo  pagination.TenantInfo
	ResourceIDs []string
	Since       int64
	Until       int64
}

// AgentShadowRepository reads what an agent's shadow report sets side by
// side: the writes it recorded in shadow, and what people did to the same
// records.
type AgentShadowRepository interface {
	ListShadowProposals(
		ctx context.Context,
		req *ListShadowProposalsRequest,
	) ([]agentshadow.Proposal, error)
	ListPersonChanges(
		ctx context.Context,
		req *ListPersonChangesRequest,
	) ([]agentshadow.PersonChange, error)
}
