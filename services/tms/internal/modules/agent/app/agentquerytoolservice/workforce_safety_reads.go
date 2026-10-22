package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	paramDrawID     = "drawId"
	maxDrawsShown   = 24
	maxPoolsShown   = 20
	defaultDOTTests = 25
)

type wfDrugAlcoholReader interface {
	ListPools(
		ctx context.Context,
		req *workerports.ListDOTRandomPoolsRequest,
	) (*pagination.CursorListResult[*worker.DOTRandomPool], error)
	ListDraws(
		ctx context.Context,
		req *workerports.ListDOTRandomDrawsRequest,
	) ([]*worker.DOTRandomDraw, error)
	GetDraw(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.DOTRandomDraw, error)
	ListDrawEntries(
		ctx context.Context,
		req *workerports.ListDOTRandomDrawEntriesRequest,
	) ([]*worker.DOTRandomDrawEntry, error)
}

var _ wfDrugAlcoholReader = (*workerdrugalcoholservice.Service)(nil)

type randomPoolRow struct {
	ID                 string `json:"id"`
	Code               string `json:"code"`
	Name               string `json:"name"`
	Period             string `json:"period"`
	DrugRatePercent    int16  `json:"drugRatePercent"`
	AlcoholRatePercent int16  `json:"alcoholRatePercent"`
	IsDefault          bool   `json:"isDefault"`
}

type randomDrawRow struct {
	ID              string       `json:"id"`
	PoolID          string       `json:"poolId"`
	Pool            string       `json:"pool,omitempty"`
	PeriodKey       string       `json:"periodKey"`
	Status          string       `json:"status"`
	PoolSize        int32        `json:"poolSize"`
	DrugTarget      int32        `json:"drugTarget"`
	AlcoholTarget   int32        `json:"alcoholTarget"`
	DrugSelected    int32        `json:"drugSelected"`
	AlcoholSelected int32        `json:"alcoholSelected"`
	DrawnAt         optionalDate `json:"drawnAt"`
	FinalizedAt     optionalDate `json:"finalizedAt"`
}

type randomProgramme struct {
	Pools []randomPoolRow `json:"pools"`
	Draws []randomDrawRow `json:"draws"`
}

func toRandomDrawRow(draw *worker.DOTRandomDraw) randomDrawRow {
	row := randomDrawRow{
		ID:              draw.ID.String(),
		PoolID:          draw.PoolID.String(),
		PeriodKey:       draw.PeriodKey,
		Status:          string(draw.Status),
		PoolSize:        draw.PoolSize,
		DrugTarget:      draw.DrugTarget,
		AlcoholTarget:   draw.AlcoholTarget,
		DrugSelected:    draw.DrugSelected,
		AlcoholSelected: draw.AlcoholSelected,
		DrawnAt:         recordedDate(draw.DrawnAt),
		FinalizedAt:     expectedDate(typeutils.ValueOrZero(draw.FinalizedAt), absentNotFinal),
	}
	if draw.Pool != nil {
		row.Pool = draw.Pool.Name
	}

	return row
}

func newListDOTRandomDrawsTool(
	draws wfDrugAlcoholReader,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &workforceRead{
		name: "worker_list_dot_random_draws",
		description: "List the random drug and alcohol testing pools and the rounds drawn " +
			"from them. Pools carry their annual rates; rounds, newest first, are draft, " +
			"final or cancelled, with how many drivers each substance needed and got. It yields the " +
			"poolId and drawId the random testing tools take; worker_get_dot_random_draw names " +
			"who was picked.",
		resource:   permission.ResourceDOTRandomPool,
		properties: map[string]any{},
		access:     newFieldAccess(permissions),
		run: func(
			ctx context.Context,
			params *agentports.QueryToolParams,
			_ *fieldGate,
		) (any, error) {
			tenant := tenantOf(params)
			pools, err := draws.ListPools(ctx, &workerports.ListDOTRandomPoolsRequest{
				Filter: &pagination.QueryOptions{
					TenantInfo: tenant,
					Pagination: pagination.Info{Limit: maxPoolsShown},
				},
				Cursor: pagination.CursorInfo{Limit: maxPoolsShown},
				Status: activeStatus,
			})
			if err != nil {
				return nil, err
			}
			found, err := draws.ListDraws(ctx, &workerports.ListDOTRandomDrawsRequest{
				TenantInfo:  tenant,
				IncludePool: true,
				Limit:       maxDrawsShown,
			})
			if err != nil {
				return nil, err
			}
			result := &randomProgramme{
				Pools: make([]randomPoolRow, 0, len(pools.Items)),
				Draws: make([]randomDrawRow, 0, len(found)),
			}
			for _, pool := range pools.Items {
				result.Pools = append(result.Pools, randomPoolRow{
					ID:                 pool.ID.String(),
					Code:               pool.Code,
					Name:               pool.Name,
					Period:             string(pool.Period),
					DrugRatePercent:    pool.DrugRatePercent,
					AlcoholRatePercent: pool.AlcoholRatePercent,
					IsDefault:          pool.IsDefault,
				})
			}
			for _, draw := range found {
				result.Draws = append(result.Draws, toRandomDrawRow(draw))
			}
			return result, nil
		},
	}
}

type randomSelectionRow struct {
	ID           string       `json:"id"`
	WorkerID     string       `json:"workerId"`
	Worker       string       `json:"worker,omitempty"`
	Substance    string       `json:"substance"`
	Rank         int32        `json:"rank"`
	Status       string       `json:"status"`
	NotifiedAt   optionalDate `json:"notifiedAt"`
	CompletedAt  optionalDate `json:"completedAt"`
	TestID       string       `json:"testId,omitempty"`
	ExcuseReason string       `json:"excuseReason,omitempty"`
}

type randomDrawDetail struct {
	randomDrawRow

	Selections []randomSelectionRow `json:"selections"`
}

func newGetDOTRandomDrawTool(
	draws wfDrugAlcoholReader,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &workforceRead{
		name:        "worker_get_dot_random_draw",
		searchTerms: []string{"who was selected", "selected drivers", "draw results"},
		description: "Retrieve one random testing round with every driver it selected. " +
			"Each selection has its substance, rank and where it stands: selected, " +
			"notified, completed with its test, excused or missed. It yields the drawEntryId " +
			"worker_schedule_dot_test and worker_update_dot_random_selection take.",
		resource: permission.ResourceDOTRandomPool,
		properties: map[string]any{
			paramDrawID: agenttoolschema.KindID(
				"The round, from worker_list_dot_random_draws. Never guess one.",
				permission.KindDOTRandomDraw,
			),
		},
		required: []string{paramDrawID},
		access:   newFieldAccess(permissions),
		run: func(
			ctx context.Context,
			params *agentports.QueryToolParams,
			_ *fieldGate,
		) (any, error) {
			id, err := requirePulid(params.Params, paramDrawID)
			if err != nil {
				return nil, err
			}
			tenant := tenantOf(params)
			draw, err := draws.GetDraw(ctx, tenant, id)
			if err != nil {
				return nil, err
			}
			entries, err := draws.ListDrawEntries(
				ctx,
				&workerports.ListDOTRandomDrawEntriesRequest{
					TenantInfo:    tenant,
					DrawID:        id,
					IncludeWorker: true,
				},
			)
			if err != nil {
				return nil, err
			}
			detail := &randomDrawDetail{
				randomDrawRow: toRandomDrawRow(draw),
				Selections:    make([]randomSelectionRow, 0, len(entries)),
			}
			for _, entry := range entries {
				detail.Selections = append(detail.Selections, randomSelectionRow{
					ID:         entry.ID.String(),
					WorkerID:   entry.WorkerID.String(),
					Worker:     workerName(entry.Worker),
					Substance:  string(entry.Substance),
					Rank:       entry.Rank,
					Status:     string(entry.Status),
					NotifiedAt: pointerDate(entry.NotifiedAt),
					CompletedAt: expectedDate(
						typeutils.ValueOrZero(entry.CompletedAt),
						absentNotCompleted,
					),
					TestID:       pulidString(entry.TestID),
					ExcuseReason: entry.ExcuseReason,
				})
			}
			return detail, nil
		},
	}
}

func provideListDOTRandomDrawsTool(
	draws *workerdrugalcoholservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListDOTRandomDrawsTool(draws, permissions)
}

func provideGetDOTRandomDrawTool(
	draws *workerdrugalcoholservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newGetDOTRandomDrawTool(draws, permissions)
}
