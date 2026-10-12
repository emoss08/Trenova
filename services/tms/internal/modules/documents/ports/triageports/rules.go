package triageports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrTooManyRules = errors.New("the organization has as many rules as it may")

type ListRulesRequest struct {
	TenantInfo  pagination.TenantInfo
	EnabledOnly bool
}

type GetRuleRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type FindRuleBySentenceRequest struct {
	TenantInfo pagination.TenantInfo
	Sentence   *triage.Rule
}

type CountRuleSettlementsRequest struct {
	TenantInfo pagination.TenantInfo
	RuleIDs    []pulid.ID
	Since      int64
}

type RecordRuleMatchRequest struct {
	TenantInfo pagination.TenantInfo
	RuleID     pulid.ID
	MessageID  pulid.ID
	At         int64
}

type TriageRuleRepository interface {
	List(ctx context.Context, req ListRulesRequest) ([]*triage.Rule, error)
	GetByID(ctx context.Context, req GetRuleRequest) (*triage.Rule, error)
	FindBySentence(ctx context.Context, req FindRuleBySentenceRequest) (*triage.Rule, error)
	Create(ctx context.Context, rule *triage.Rule, limit int) (*triage.Rule, error)
	Update(ctx context.Context, rule *triage.Rule) (*triage.Rule, error)
	SetEnabled(ctx context.Context, req SetRuleEnabledRequest) (*triage.Rule, error)
	Delete(ctx context.Context, req GetRuleRequest) error
	RecordMatch(ctx context.Context, req RecordRuleMatchRequest) error
	CountSettledSince(
		ctx context.Context,
		req CountRuleSettlementsRequest,
	) (map[pulid.ID]int, error)
}

type SetRuleEnabledRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Enabled    bool
}
