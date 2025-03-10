package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ListHazmatSegregationRuleRequest struct {
	Filter                 *ports.LimitOffsetQueryOptions
	IncludeHazmatMaterials bool `query:"includeHazmatMaterials"`
}

type GetHazmatSegregationRuleByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type HazmatSegregationRuleRepository interface {
	List(ctx context.Context, req *ListHazmatSegregationRuleRequest) (*ports.ListResult[*hazmatsegregationrule.HazmatSegregationRule], error)
	GetByID(ctx context.Context, req *GetHazmatSegregationRuleByIDRequest) (*hazmatsegregationrule.HazmatSegregationRule, error)
	Create(ctx context.Context, hsr *hazmatsegregationrule.HazmatSegregationRule) (*hazmatsegregationrule.HazmatSegregationRule, error)
	Update(ctx context.Context, hsr *hazmatsegregationrule.HazmatSegregationRule) (*hazmatsegregationrule.HazmatSegregationRule, error)
}
