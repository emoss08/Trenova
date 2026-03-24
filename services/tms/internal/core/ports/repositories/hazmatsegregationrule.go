package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListHazmatSegregationRuleRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetHazmatSegregationRuleByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type HazmatSegregationRuleRepository interface {
	List(
		ctx context.Context,
		req *ListHazmatSegregationRuleRequest,
	) (*pagination.ListResult[*hazmatsegregationrule.HazmatSegregationRule], error)
	ListActiveByTenant(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*hazmatsegregationrule.HazmatSegregationRule, error)
	GetByID(
		ctx context.Context,
		req GetHazmatSegregationRuleByIDRequest,
	) (*hazmatsegregationrule.HazmatSegregationRule, error)
	Create(
		ctx context.Context,
		entity *hazmatsegregationrule.HazmatSegregationRule,
	) (*hazmatsegregationrule.HazmatSegregationRule, error)
	Update(
		ctx context.Context,
		entity *hazmatsegregationrule.HazmatSegregationRule,
	) (*hazmatsegregationrule.HazmatSegregationRule, error)
}
