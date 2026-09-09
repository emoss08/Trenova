package iftajobs

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	BackfillJurisdictionMilesWorkflowName = "BackfillJurisdictionMilesWorkflow"

	DefaultBackfillMaxMoves = 2000
	ListMovesPageSize       = 200
	AttributeBatchSize      = 20
)

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

type ListMovesMissingBreakdownInput struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Start      int64                 `json:"start"`
	End        int64                 `json:"end"`
	Limit      int                   `json:"limit"`
	Offset     int                   `json:"offset"`
}

type ListMovesMissingBreakdownResult struct {
	MoveIDs    []pulid.ID      `json:"moveIds"`
	TotalMoves int             `json:"totalMoves"`
	TotalMiles decimal.Decimal `json:"totalMiles"`
}

type AttributeMovesInput struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	MoveIDs    []pulid.ID            `json:"moveIds"`
}

type AttributeMovesResult struct {
	Processed int             `json:"processed"`
	Failed    int             `json:"failed"`
	Miles     decimal.Decimal `json:"miles"`
}

type BackfillResult struct {
	Processed int             `json:"processed"`
	Failed    int             `json:"failed"`
	Miles     decimal.Decimal `json:"miles"`
}
