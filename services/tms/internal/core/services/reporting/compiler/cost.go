package compiler

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/uptrace/bun"
)

type CostLimits struct {
	MaxEstimatedCost float64
	MaxEstimatedRows float64
}

type explainPlanNode struct {
	TotalCost float64 `json:"Total Cost"`
	PlanRows  float64 `json:"Plan Rows"`
}

type explainEntry struct {
	Plan explainPlanNode `json:"Plan"`
}

func PreflightCost(
	ctx context.Context,
	db bun.IDB,
	compiled *services.CompiledReportQuery,
	costLimits CostLimits,
) error {
	explainSQL := "EXPLAIN (FORMAT JSON) " + compiled.SQL

	var raw []byte
	err := db.NewRaw(explainSQL, compiled.Args...).Scan(ctx, &raw)
	if err != nil {
		return fmt.Errorf("explain report query: %w", err)
	}

	var entries []explainEntry
	if err = sonic.Unmarshal(raw, &entries); err != nil {
		return fmt.Errorf("parse explain output: %w", err)
	}
	if len(entries) == 0 {
		return errors.New("explain returned no plan")
	}

	plan := entries[0].Plan
	if costLimits.MaxEstimatedCost > 0 && plan.TotalCost > costLimits.MaxEstimatedCost {
		return fmt.Errorf(
			"report query is too expensive to run (estimated cost %.0f exceeds the limit of %.0f) — narrow your filters or date range",
			plan.TotalCost,
			costLimits.MaxEstimatedCost,
		)
	}
	if costLimits.MaxEstimatedRows > 0 && plan.PlanRows > costLimits.MaxEstimatedRows {
		return fmt.Errorf(
			"report query would scan too many rows (estimated %.0f exceeds the limit of %.0f) — narrow your filters or date range",
			plan.PlanRows,
			costLimits.MaxEstimatedRows,
		)
	}

	return nil
}
