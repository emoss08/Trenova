package agent

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	ToolRuleOverrideIDPrefix = "atro_"
	MaxToolRuleReasonLength  = 500
)

type ToolRuleOverride struct {
	bun.BaseModel `bun:"table:agent_tool_rule_overrides,alias:atro" json:"-"`

	ID             pulid.ID     `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID     `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID     `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ToolName       string       `json:"toolName"       bun:"tool_name,type:VARCHAR(100),notnull"`
	MaxTier        AutonomyTier `json:"maxTier"        bun:"max_tier,type:VARCHAR(30),nullzero"`
	ReadsExternal  ExternalRead `json:"readsExternal"  bun:"reads_external,type:VARCHAR(20),nullzero"`
	Reason         string       `json:"reason"         bun:"reason,type:TEXT,notnull"`
	UpdatedByID    pulid.ID     `json:"updatedById"    bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	Version        int64        `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64        `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64        `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (o *ToolRuleOverride) GetID() pulid.ID { return o.ID }

func (o *ToolRuleOverride) GetTableName() string { return "agent_tool_rule_overrides" }

func (o *ToolRuleOverride) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if o.ID.IsNil() {
			o.ID = pulid.MustNew(ToolRuleOverrideIDPrefix)
		}
		o.CreatedAt = now
		o.UpdatedAt = now
	case *bun.UpdateQuery:
		o.UpdatedAt = now
	}

	return nil
}

func (o *ToolRuleOverride) Empty() bool {
	return o == nil || (o.MaxTier == "" && o.ReadsExternal == "")
}

func ExternalReadRank(read ExternalRead) int {
	switch read {
	case ExternalReadMarked:
		return 1
	case ExternalReadAlways:
		return 2
	default:
		return 0
	}
}

type ToolRuleBounds struct {
	MaxTier       AutonomyTier
	ReadsExternal ExternalRead
	Changes       bool
}

func (o *ToolRuleOverride) Validate(bounds ToolRuleBounds, multiErr *errortypes.MultiError) {
	if o.MaxTier != "" {
		switch {
		case !o.MaxTier.IsValid():
			multiErr.Add("maxTier", errortypes.ErrInvalid, "Unknown tier")
		case !bounds.Changes:
			multiErr.Add("maxTier", errortypes.ErrInvalid, "Reads never change records, so they always run")
		case o.MaxTier.Above(bounds.MaxTier):
			multiErr.Add("maxTier", errortypes.ErrInvalid,
				"The tool's own rule allows at most {0}; a rule here can only hold it lower", bounds.MaxTier.Label())
		}
	}
	if o.ReadsExternal != "" {
		switch {
		case !o.ReadsExternal.IsValid():
			multiErr.Add("readsExternal", errortypes.ErrInvalid, "Unknown outside text setting")
		case ExternalReadRank(o.ReadsExternal) < ExternalReadRank(bounds.ReadsExternal):
			multiErr.Add("readsExternal", errortypes.ErrInvalid,
				"The tool's own rule already treats more of what it returns as outside text; a rule here can only treat more")
		}
	}
	if len(strings.TrimSpace(o.Reason)) > MaxToolRuleReasonLength {
		multiErr.Add("reason", errortypes.ErrInvalid, "Keep the reason to 500 characters")
	}
}
