package agenttooltrust

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	ToolRuleOverrideIDPrefix = "atro_"
	MaxToolRuleReasonLength  = 500
)

type ToolRuleOverride struct {
	keel.Table `keel:"agent_tool_rule_overrides"`

	ID             pulid.ID           `json:"id"             keel:"id,pk,autoid:atro_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	ToolName       string             `json:"toolName"       keel:"tool_name"`
	MaxTier        agent.AutonomyTier `json:"maxTier"        keel:"max_tier,nullzero"`
	ReadsExternal  ExternalRead       `json:"readsExternal"  keel:"reads_external,nullzero"`
	Reason         string             `json:"reason"         keel:"reason"`
	UpdatedByID    pulid.ID           `json:"updatedById"    keel:"updated_by_id,nullzero"`
	Version        int64              `json:"version"        keel:"version,version"`
	CreatedAt      int64              `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (o *ToolRuleOverride) GetID() pulid.ID { return o.ID }

func (o *ToolRuleOverride) GetTableName() string { return "agent_tool_rule_overrides" }

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
	MaxTier       agent.AutonomyTier
	ReadsExternal ExternalRead
	Changes       bool
}

func (o *ToolRuleOverride) Validate(bounds ToolRuleBounds, multiErr *errortypes.MultiError) {
	if o.MaxTier != "" {
		switch {
		case !o.MaxTier.IsValid():
			multiErr.Add("maxTier", errortypes.ErrInvalid, "Unknown tier")
		case !bounds.Changes:
			multiErr.Add(
				"maxTier",
				errortypes.ErrInvalid,
				"Reads never change records, so they always run",
			)
		case o.MaxTier.Above(bounds.MaxTier):
			multiErr.Add(
				"maxTier",
				errortypes.ErrInvalid,
				"The tool's own rule allows at most {0}; a rule here can only hold it lower",
				bounds.MaxTier.Label(),
			)
		}
	}
	if o.ReadsExternal != "" {
		switch {
		case !o.ReadsExternal.IsValid():
			multiErr.Add("readsExternal", errortypes.ErrInvalid, "Unknown outside text setting")
		case ExternalReadRank(o.ReadsExternal) < ExternalReadRank(bounds.ReadsExternal):
			multiErr.Add(
				"readsExternal",
				errortypes.ErrInvalid,
				"The tool's own rule already treats more of what it returns as outside text; a rule here can only treat more",
			)
		}
	}
	if len(strings.TrimSpace(o.Reason)) > MaxToolRuleReasonLength {
		multiErr.Add("reason", errortypes.ErrInvalid, "Keep the reason to 500 characters")
	}
}
