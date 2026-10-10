package iftacontract

import (
	"github.com/emoss08/trenova/pkg/pagination"
)

const BackfillJurisdictionMilesWorkflowName = "BackfillJurisdictionMilesWorkflow"

const DefaultBackfillMaxMoves = 2000

type BackfillInput struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Start      int64                 `json:"start"`
	End        int64                 `json:"end"`
	MaxMoves   int                   `json:"maxMoves"`
}

func (i BackfillInput) EffectiveMaxMoves() int {
	if i.MaxMoves <= 0 {
		return DefaultBackfillMaxMoves
	}
	return i.MaxMoves
}
