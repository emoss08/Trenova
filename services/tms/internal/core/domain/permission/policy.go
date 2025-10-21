package permission

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/uptrace/bun"
)

type PolicyCondition struct {
	Type       PolicyConditionType `json:"type"`
	Field      string              `json:"field"`
	Operator   string              `json:"operator"`
	Value      any                 `json:"value"`
	Parameters map[string]any      `json:"parameters"`
}

type PolicyScope struct {
	BusinessUnitID  pulid.ID   `json:"businessUnitId"`
	OrganizationIDs []pulid.ID `json:"organizationIds"`
	Inheritable     bool       `json:"inheritable"`
}

type Subject struct {
	Type       SubjectType    `json:"type"`
	ID         pulid.ID       `json:"id"`
	Attributes map[string]any `json:"attributes"`
}

type ResourceRule struct {
	ResourceType string            `json:"resourceType"`
	Actions      ActionSet         `json:"actions"`
	Conditions   []PolicyCondition `json:"conditions"`
	DataScope    DataScope         `json:"dataScope"`
}

type ActionSet struct {
	StandardOps Operation `json:"standardOps"`
	ExtendedOps []string  `json:"extendedOps"`
}

func (a ActionSet) HasStandardOp(op Operation) bool {
	return (a.StandardOps & op) != 0
}

func (a ActionSet) HasExtendedOp(op string) bool {
	return slices.Contains(a.ExtendedOps, op)
}

var _ bun.BeforeAppendModelHook = (*Policy)(nil)

type Policy struct {
	bun.BaseModel `bun:"table:policies,alias:pol" json:"-"`

	ID          pulid.ID       `json:"id"          bun:"id,pk,type:VARCHAR(100)"`
	Name        string         `json:"name"        bun:"name,type:VARCHAR(255),notnull"`
	Description string         `json:"description" bun:"description,type:TEXT"`
	Scope       PolicyScope    `json:"scope"       bun:"scope,type:JSONB,notnull"`
	Resources   []ResourceRule `json:"resources"   bun:"resources,type:JSONB"`
	Subjects    []Subject      `json:"subjects"    bun:"subjects,type:JSONB"`
	Effect      Effect         `json:"effect"      bun:"effect,type:VARCHAR(10),notnull"`
	Priority    int            `json:"priority"    bun:"priority,default:0"`
	Tags        []string       `json:"tags"        bun:"tags,type:TEXT[]"`
	CreatedBy   pulid.ID       `json:"createdBy"   bun:"created_by,type:VARCHAR(100)"`
	CreatedAt   int64          `json:"createdAt"   bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt   int64          `json:"updatedAt"   bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (p *Policy) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := utils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("pol_")
		}
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
