package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/core/services/workersafetyservice"
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

type wfSafetyReader interface {
	ListEvents(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) ([]*worker.WorkerSafetyEvent, error)
	ListViolations(
		ctx context.Context,
		req *repositories.ListWorkerSafetyViolationsRequest,
	) ([]*worker.WorkerSafetyViolation, error)
	ListRecognitions(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		visibleOnly bool,
	) ([]*worker.WorkerRecognition, error)
}

type wfDrugAlcoholReader interface {
	ListTests(
		ctx context.Context,
		req *repositories.ListWorkerDOTTestsRequest,
	) ([]*worker.WorkerDOTTest, error)
	ListPools(
		ctx context.Context,
		req *repositories.ListDOTRandomPoolsRequest,
	) (*pagination.CursorListResult[*worker.DOTRandomPool], error)
	ListDraws(
		ctx context.Context,
		req *repositories.ListDOTRandomDrawsRequest,
	) ([]*worker.DOTRandomDraw, error)
	GetDraw(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.DOTRandomDraw, error)
	ListDrawEntries(
		ctx context.Context,
		req *repositories.ListDOTRandomDrawEntriesRequest,
	) ([]*worker.DOTRandomDrawEntry, error)
}

var (
	_ wfSafetyReader      = (*workersafetyservice.Service)(nil)
	_ wfDrugAlcoholReader = (*workerdrugalcoholservice.Service)(nil)
)

type safetyViolationRow struct {
	ID             string `json:"id"`
	Basic          string `json:"basic"`
	Code           string `json:"code"`
	Description    string `json:"description,omitempty"`
	SeverityWeight int16  `json:"severityWeight"`
	OutOfService   bool   `json:"outOfService"`
}

type safetyEventRow struct {
	ID               string               `json:"id"`
	Kind             string               `json:"kind"`
	Severity         string               `json:"severity"`
	Status           string               `json:"status"`
	OccurredAt       optionalDate         `json:"occurredAt"`
	Location         string               `json:"location,omitempty"`
	Description      string               `json:"description,omitempty"`
	Preventable      bool                 `json:"preventable"`
	Points           int32                `json:"points"`
	PointsExpireAt   optionalDate         `json:"pointsExpireAt"`
	InspectionResult string               `json:"inspectionResult,omitempty"`
	OutOfService     bool                 `json:"outOfService"`
	ShipmentID       string               `json:"shipmentId,omitempty"`
	Resolution       string               `json:"resolution,omitempty"`
	ClosedAt         optionalDate         `json:"closedAt"`
	Violations       []safetyViolationRow `json:"violations,omitempty"`
}

type recognitionRow struct {
	ID              string       `json:"id"`
	Kind            string       `json:"kind"`
	Title           string       `json:"title"`
	OccurredAt      optionalDate `json:"occurredAt"`
	VisibleToWorker bool         `json:"visibleToWorker"`
}

type workerSafetyRecord struct {
	WorkerID     string           `json:"workerId"`
	Events       []safetyEventRow `json:"events"`
	Recognitions []recognitionRow `json:"recognitions,omitempty"`
	Withheld     []string         `json:"withheldByAccess,omitempty"`
}

func newListWorkerSafetyEventsTool(
	safety wfSafetyReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	access := newFieldAccess(permissions)
	return &workforceRead{
		name: "list_worker_safety_events",
		description: "List one worker's safety events, roadside violations and recognitions. " +
			"Events are accidents, inspections, citations, complaints and near misses with " +
			"their points and status, each with its violations. It yields the " +
			"safetyEventId, violationId and recognitionId the safety tools take.",
		resource: permission.ResourceWorkerSafetyEvent,
		properties: map[string]any{
			paramWorkerID: wfWorkerProperty("The worker"),
		},
		required: []string{paramWorkerID},
		access:   access,
		run: func(
			ctx context.Context,
			params *serviceports.QueryToolParams,
			gate *fieldGate,
		) (any, error) {
			workerID, err := requirePulid(params.Params, paramWorkerID)
			if err != nil {
				return nil, err
			}
			tenant := tenantOf(params)
			events, err := safety.ListEvents(ctx, tenant, workerID)
			if err != nil {
				return nil, err
			}
			violations, err := safety.ListViolations(ctx,
				&repositories.ListWorkerSafetyViolationsRequest{
					TenantInfo: tenant,
					WorkerID:   workerID,
				})
			if err != nil {
				return nil, err
			}
			byEvent := make(map[pulid.ID][]safetyViolationRow, len(violations))
			for _, violation := range violations {
				byEvent[violation.SafetyEventID] = append(byEvent[violation.SafetyEventID],
					safetyViolationRow{
						ID:             violation.ID.String(),
						Basic:          string(violation.Basic),
						Code:           violation.Code,
						Description:    gatedText(gate, wfFieldDescription, violation.Description),
						SeverityWeight: violation.SeverityWeight,
						OutOfService:   violation.OutOfService,
					})
			}
			record := &workerSafetyRecord{
				WorkerID: workerID.String(),
				Events:   make([]safetyEventRow, 0, len(events)),
			}
			for _, event := range events {
				record.Events = append(record.Events, safetyEventRow{
					ID:          event.ID.String(),
					Kind:        string(event.Kind),
					Severity:    string(event.Severity),
					Status:      string(event.Status),
					OccurredAt:  recordedDate(event.OccurredAt),
					Location:    event.Location,
					Description: gatedText(gate, wfFieldDescription, event.Description),
					Preventable: event.Preventable,
					Points:      event.Points,
					PointsExpireAt: expectedDate(
						typeutils.ValueOrZero(event.PointsExpireAt),
						absentNotExpiring,
					),
					InspectionResult: string(event.InspectionResult),
					OutOfService:     event.OutOfService,
					ShipmentID:       pulidString(event.ShipmentID),
					Resolution:       gatedText(gate, wfFieldResolution, event.Resolution),
					ClosedAt:         expectedDate(typeutils.ValueOrZero(event.ClosedAt), absentNotClosed),
					Violations:       byEvent[event.ID],
				})
			}
			if access.mayRead(ctx, params, permission.ResourceWorkerRecognition) {
				recognitions, recErr := safety.ListRecognitions(ctx, tenant, workerID, false)
				if recErr != nil {
					return nil, recErr
				}
				record.Recognitions = make([]recognitionRow, 0, len(recognitions))
				for _, recognition := range recognitions {
					record.Recognitions = append(record.Recognitions, recognitionRow{
						ID:              recognition.ID.String(),
						Kind:            string(recognition.Kind),
						Title:           recognition.Title,
						OccurredAt:      recordedDate(recognition.OccurredAt),
						VisibleToWorker: recognition.VisibleToWorker,
					})
				}
			}
			record.Withheld = gate.Withheld()
			return record, nil
		},
	}
}

type dotTestRow struct {
	ID             string       `json:"id"`
	WorkerID       string       `json:"workerId"`
	Worker         string       `json:"worker,omitempty"`
	TestType       string       `json:"testType"`
	Substance      string       `json:"substance"`
	Status         string       `json:"status"`
	Result         string       `json:"result"`
	IsDOT          bool         `json:"isDot"`
	ScheduledAt    optionalDate `json:"scheduledAt"`
	CollectedAt    optionalDate `json:"collectedAt"`
	CollectionSite string       `json:"collectionSite,omitempty"`
	Reason         string       `json:"reason,omitempty"`
	SafetyEventID  string       `json:"safetyEventId,omitempty"`
	DrawEntryID    string       `json:"drawEntryId,omitempty"`
}

func newListDOTTestsTool(
	tests wfDrugAlcoholReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &workforceRead{
		name: "list_dot_tests",
		description: "List drug and alcohol tests, newest first: one worker's, or every " +
			"collection still open across the organization. Each row has the test type, " +
			"substance, status and result. It yields the testId cancel_dot_test takes.",
		resource: permission.ResourceWorkerDOTTest,
		properties: map[string]any{
			paramWorkerID: wfWorkerProperty("Narrow to one worker"),
			wfParamOpenOnly: wfBoolProperty("Only collections not yet completed or " +
				"cancelled."),
			paramLimit: limitProperty(),
		},
		access: newFieldAccess(permissions),
		run: func(
			ctx context.Context,
			params *serviceports.QueryToolParams,
			gate *fieldGate,
		) (any, error) {
			workerID, err := optionalWorker(params.Params)
			if err != nil {
				return nil, err
			}
			found, err := tests.ListTests(ctx, &repositories.ListWorkerDOTTestsRequest{
				TenantInfo:    tenantOf(params),
				WorkerID:      workerID,
				OpenOnly:      optionalBool(params.Params, wfParamOpenOnly),
				IncludeWorker: true,
				Limit:         receivableLimit(params.Params),
			})
			if err != nil {
				return nil, err
			}
			rows := make([]dotTestRow, 0, len(found))
			for _, test := range found {
				rows = append(rows, dotTestRow{
					ID:             test.ID.String(),
					WorkerID:       test.WorkerID.String(),
					Worker:         workerName(test.Worker),
					TestType:       string(test.TestType),
					Substance:      string(test.Substance),
					Status:         string(test.Status),
					Result:         string(test.Result),
					IsDOT:          test.IsDOT,
					ScheduledAt:    expectedDate(typeutils.ValueOrZero(test.ScheduledAt), absentNotScheduled),
					CollectedAt:    pointerDate(test.CollectedAt),
					CollectionSite: test.CollectionSite,
					Reason:         gatedText(gate, wfFieldReason, test.Reason),
					SafetyEventID:  pulidString(test.SafetyEventID),
					DrawEntryID:    pulidString(test.DrawEntryID),
				})
			}
			return newReceivableList(rows, gate), nil
		},
	}
}

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
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &workforceRead{
		name: "list_dot_random_draws",
		description: "List the random drug and alcohol testing pools and the rounds drawn " +
			"from them. Pools carry their annual rates; rounds, newest first, are draft, " +
			"final or cancelled, with how many drivers each substance needed and got. It yields the " +
			"poolId and drawId the random testing tools take; get_dot_random_draw names " +
			"who was picked.",
		resource:   permission.ResourceDOTRandomPool,
		properties: map[string]any{},
		access:     newFieldAccess(permissions),
		run: func(
			ctx context.Context,
			params *serviceports.QueryToolParams,
			_ *fieldGate,
		) (any, error) {
			tenant := tenantOf(params)
			pools, err := draws.ListPools(ctx, &repositories.ListDOTRandomPoolsRequest{
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
			found, err := draws.ListDraws(ctx, &repositories.ListDOTRandomDrawsRequest{
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
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &workforceRead{
		name:        "get_dot_random_draw",
		searchTerms: []string{"who was selected", "selected drivers", "draw results"},
		description: "Retrieve one random testing round with every driver it selected. " +
			"Each selection has its substance, rank and where it stands: selected, " +
			"notified, completed with its test, excused or missed. It yields the drawEntryId " +
			"schedule_dot_test and update_dot_random_selection take.",
		resource: permission.ResourceDOTRandomPool,
		properties: map[string]any{
			paramDrawID: agenttoolschema.KindID(
				"The round, from list_dot_random_draws. Never guess one.",
				permission.KindDOTRandomDraw,
			),
		},
		required: []string{paramDrawID},
		access:   newFieldAccess(permissions),
		run: func(
			ctx context.Context,
			params *serviceports.QueryToolParams,
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
				&repositories.ListDOTRandomDrawEntriesRequest{
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
					ID:           entry.ID.String(),
					WorkerID:     entry.WorkerID.String(),
					Worker:       workerName(entry.Worker),
					Substance:    string(entry.Substance),
					Rank:         entry.Rank,
					Status:       string(entry.Status),
					NotifiedAt:   pointerDate(entry.NotifiedAt),
					CompletedAt:  expectedDate(typeutils.ValueOrZero(entry.CompletedAt), absentNotCompleted),
					TestID:       pulidString(entry.TestID),
					ExcuseReason: entry.ExcuseReason,
				})
			}
			return detail, nil
		},
	}
}

func provideListWorkerSafetyEventsTool(
	safety *workersafetyservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListWorkerSafetyEventsTool(safety, permissions)
}

func provideListDOTTestsTool(
	tests *workerdrugalcoholservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListDOTTestsTool(tests, permissions)
}

func provideListDOTRandomDrawsTool(
	draws *workerdrugalcoholservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListDOTRandomDrawsTool(draws, permissions)
}

func provideGetDOTRandomDrawTool(
	draws *workerdrugalcoholservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetDOTRandomDrawTool(draws, permissions)
}
