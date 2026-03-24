package customfield

import "github.com/emoss08/trenova/shared/pulid"

type DefinitionUsageStats struct {
	DefinitionID    pulid.ID           `json:"definitionId"`
	TotalValueCount int                `json:"totalValueCount"`
	ResourceCount   int                `json:"resourceCount"`
	OptionUsage     []OptionUsageStats `json:"optionUsage,omitempty"`
}

type OptionUsageStats struct {
	Value      string `json:"value"`
	Label      string `json:"label"`
	UsageCount int    `json:"usageCount"`
}

type BreakingChangeType string

const (
	BreakingChangeTypeBlocked BreakingChangeType = "BLOCKED"
	BreakingChangeTypeWarning BreakingChangeType = "WARNING"
)

type BreakingChange struct {
	Field      string             `json:"field"`
	ChangeType BreakingChangeType `json:"changeType"`
	Code       string             `json:"code"`
	Message    string             `json:"message"`
	Details    any                `json:"details,omitempty"`
}

type BreakingChangeResult struct {
	HasBlockingChanges bool             `json:"hasBlockingChanges"`
	Changes            []BreakingChange `json:"changes"`
}
