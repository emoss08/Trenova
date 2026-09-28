package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/internal/core/services/workerdrugalcoholservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramDOTTestID     = "testId"
	paramDOTTestType   = "testType"
	paramSubstance     = "substance"
	paramScheduledAt   = "scheduledAt"
	paramCollection    = "collectionSite"
	paramDrawEntryID   = "drawEntryId"
	paramIsDOT         = "isDot"
	paramPoolID        = "poolId"
	paramDrawID        = "drawId"
	paramDrawDay       = "drawDate"
	paramSelectionMove = "status"
	paramExcuseReason  = "excuseReason"
	kindDOTTest        = "DOT test"
	kindRandomRound    = "random testing round"
	kindRandomPick     = "random selection"
)

var (
	dotTestTypes      = agenttoolschema.Source("worker.dotTestType", worker.DOTTestTypeValues())
	dotTestSubstances = agenttoolschema.Source(
		"worker.dotTestSubstance",
		worker.DOTTestSubstanceValues(),
	)
	randomSelectionMoves = agenttoolschema.Source(
		"dotRandomSelection.agentMove",
		[]worker.RandomEntryStatus{
			worker.RandomEntryNotified,
			worker.RandomEntryExcused,
			worker.RandomEntryMissed,
		},
	)
	dotTestFields = []string{
		wfFieldWorkerID, paramDOTTestType, paramSubstance, fieldStatus, "result", paramIsDOT,
		fieldReason, paramScheduledAt, paramCollection, paramSafetyEventID, paramDrawEntryID,
		wfFieldNotes, "orderedById",
	}
	drawFields = []string{
		"periodKey", fieldPeriodStart, fieldPeriodEnd, fieldStatus, "poolSize", "drugTarget",
		"alcoholTarget", "drugSelected", "alcoholSelected", "drawnAt", "finalizedAt",
		wfFieldNotes,
	}
)

type drugAlcoholKeeper interface {
	PlanRecordTest(
		ctx context.Context,
		entity *worker.WorkerDOTTest,
		userID pulid.ID,
	) (*worker.WorkerDOTTest, error)
	RecordTest(
		ctx context.Context,
		entity *worker.WorkerDOTTest,
		userID pulid.ID,
	) (*worker.WorkerDOTTest, error)
	PlanCancelTest(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		reason string,
	) (*workerdrugalcoholservice.TestChange, error)
	CancelTest(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		reason string,
		userID pulid.ID,
	) (*worker.WorkerDOTTest, error)
	PlanRunDraw(
		ctx context.Context,
		req *workerdrugalcoholservice.RunDrawRequest,
	) (*worker.DOTRandomDraw, error)
	RunDraw(
		ctx context.Context,
		req *workerdrugalcoholservice.RunDrawRequest,
	) (*worker.DOTRandomDraw, error)
	PlanFinalizeDraw(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*workerdrugalcoholservice.DrawChange, error)
	FinalizeDraw(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) (*worker.DOTRandomDraw, error)
	PlanCancelDraw(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		reason string,
	) (*workerdrugalcoholservice.DrawChange, error)
	CancelDraw(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		reason string,
		userID pulid.ID,
	) (*worker.DOTRandomDraw, error)
	PlanUpdateDrawEntry(
		ctx context.Context,
		req *workerdrugalcoholservice.UpdateEntryRequest,
	) (*workerdrugalcoholservice.EntryChange, error)
	UpdateDrawEntry(
		ctx context.Context,
		req *workerdrugalcoholservice.UpdateEntryRequest,
	) (*worker.DOTRandomDrawEntry, error)
}

var _ drugAlcoholKeeper = (*workerdrugalcoholservice.Service)(nil)

func drugAlcoholToolProviders() []any {
	return []any{
		provideScheduleDOTTestTool,
		provideCancelDOTTestTool,
		provideRunDOTRandomDrawTool,
		provideFinalizeDOTRandomDrawTool,
		provideCancelDOTRandomDrawTool,
		provideUpdateDOTRandomSelectionTool,
	}
}

func dotTestRecord(test *worker.WorkerDOTTest) toolpreview.Record {
	return wfRecord(permission.ResourceWorkerDOTTest, test.ID,
		fmt.Sprintf("%s %s test", test.TestType.Label(), strings.ToLower(string(test.Substance))),
		test.Version)
}

func drawRecord(draw *worker.DOTRandomDraw) toolpreview.Record {
	return wfRecord(permission.ResourceDOTRandomPool, draw.ID,
		"Random testing round "+draw.PeriodKey, draw.Version)
}

func scheduledTestFrom(params *serviceports.ToolExecuteParams) (*worker.WorkerDOTTest, error) {
	workerID, err := requirePulid(params.Params, paramWorkerID)
	if err != nil {
		return nil, err
	}
	testType, err := requireEnum(params.Params, paramDOTTestType, dotTestTypes.Values)
	if err != nil {
		return nil, err
	}
	substance, err := requireEnum(params.Params, paramSubstance, dotTestSubstances.Values)
	if err != nil {
		return nil, err
	}
	scheduledAt, err := optionalDateTime(params.Params, paramScheduledAt)
	if err != nil {
		return nil, err
	}
	reason, err := boundedText(params.Params, fieldReason, wfNoteChars)
	if err != nil {
		return nil, err
	}
	site, err := boundedText(params.Params, paramCollection, wfShortChars)
	if err != nil {
		return nil, err
	}
	notes, err := boundedText(params.Params, fieldNotes, wfNoteChars)
	if err != nil {
		return nil, err
	}
	isDOT, err := optionalBoolParam(params.Params, paramIsDOT, true)
	if err != nil {
		return nil, err
	}
	test := &worker.WorkerDOTTest{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		WorkerID:       workerID,
		TestType:       testType,
		Substance:      substance,
		Status:         worker.DOTTestStatusScheduled,
		Result:         worker.DOTResultPending,
		IsDOT:          isDOT,
		Reason:         reason,
		ScheduledAt:    scheduledAt,
		CollectionSite: site,
		Notes:          notes,
	}
	if test.SafetyEventID, err = optionalID(params.Params, paramSafetyEventID); err != nil {
		return nil, err
	}
	if test.DrawEntryID, err = optionalID(params.Params, paramDrawEntryID); err != nil {
		return nil, err
	}
	return test, nil
}

func newScheduleDOTTestTool(tests drugAlcoholKeeper) serviceports.AgentTool {
	spec := withSchema(wfSpec(
		"schedule_dot_test",
		"Schedule a drug or alcohol test for a driver. It covers a random selection's "+
			"collection, a post-accident or reasonable-suspicion test, pre-employment, "+
			"return-to-duty or follow-up. A collection covering both substances is two calls. It records the "+
			"order only; results are entered by the person who receives them from the lab or "+
			"the medical review officer.",
		"Files a scheduled collection inside Trenova. The driver is sent nothing, no result is "+
			"recorded, and cancel_dot_test withdraws it.",
		permission.ResourceWorkerDOTTest,
		permission.OpCreate,
	), map[string]any{
		paramWorkerID:    workerProperty(),
		paramDOTTestType: agenttoolschema.Enum("Why the driver is tested.", dotTestTypes),
		paramSubstance:   agenttoolschema.Enum("What is tested.", dotTestSubstances),
		paramScheduledAt: dateTimeProperty("When the collection is booked."),
		fieldReason: wfNoteProperty("What prompted it. Required for a post-accident or " +
			"reasonable-suspicion test; write what was observed or what happened."),
		paramCollection: stringProperty("The collection site.", wfShortChars),
		paramSafetyEventID: idProperty("For a post-accident test: the accident, from " +
			"list_worker_safety_events."),
		paramDrawEntryID: idProperty("For a random test: the selection it answers, from " +
			"get_dot_random_draw."),
		paramIsDOT: booleanProperty("Whether it is a DOT-regulated test. Defaults to true; " +
			"a company-policy test is false."),
		fieldNotes: wfNoteProperty("Anything the collector should know."),
	}, paramWorkerID, paramDOTTestType, paramSubstance)
	spec.searchTerms = []string{"drug test", "alcohol test", "random test", "collection"}

	return newReportingReceivableTool(spec,
		receivablePlan[*worker.WorkerDOTTest, *worker.WorkerDOTTest]{
			request: scheduledTestFrom,
			plan: func(
				ctx context.Context,
				test *worker.WorkerDOTTest,
				params *serviceports.ToolExecuteParams,
			) (*worker.WorkerDOTTest, error) {
				return tests.PlanRecordTest(ctx, test, params.Actor.UserID)
			},
			refused: func(*worker.WorkerDOTTest) string {
				return "Would schedule a drug or alcohol test."
			},
			render: func(
				_ *worker.WorkerDOTTest,
				planned *worker.WorkerDOTTest,
			) (*agent.ToolPreview, error) {
				change, err := toolpreview.Create(
					wfRecord(permission.ResourceWorkerDOTTest, pulid.Nil, "New test", 0),
					planned, wfOptions(dotTestFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would schedule a %s %s test. No result is recorded.",
					strings.ToLower(planned.TestType.Label()),
					strings.ToLower(string(planned.Substance)),
				), change), nil
			},
			run: func(
				ctx context.Context,
				test *worker.WorkerDOTTest,
				params *serviceports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				created, err := tests.RecordTest(ctx, test, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return wfResult("scheduled", kindDOTTest, paramDOTTestID, created.ID,
					created.WorkerID), nil
			},
		})
}

type reasonedRecord struct {
	id     pulid.ID
	reason string
	tenant pagination.TenantInfo
	userID pulid.ID
}

func reasonedRecordFrom(
	idKey, reasonKey string,
) func(*serviceports.ToolExecuteParams) (*reasonedRecord, error) {
	return func(params *serviceports.ToolExecuteParams) (*reasonedRecord, error) {
		id, err := requirePulid(params.Params, idKey)
		if err != nil {
			return nil, err
		}
		reason, err := requireBoundedText(params.Params, reasonKey, wfNoteChars)
		if err != nil {
			return nil, err
		}
		return &reasonedRecord{
			id:     id,
			reason: reason,
			tenant: tenantFrom(*params),
			userID: params.Actor.UserID,
		}, nil
	}
}

func newCancelDOTTestTool(tests drugAlcoholKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"cancel_dot_test",
		"Cancel a drug or alcohol test that did not happen, with the reason kept on the "+
			"record. A test with a result cannot be cancelled; a correction is a new test.",
		"Voids a scheduled collection; the row and its reason stay on the record an auditor "+
			"reads, and it cannot be undone.",
		permission.ResourceWorkerDOTTest,
		permission.OpCancel,
	), map[string]any{
		paramDOTTestID: idProperty("The test, from list_dot_tests. Never guess one."),
		fieldReason:    wfNoteProperty("Why the collection did not happen."),
	}, paramDOTTestID, fieldReason), paramDOTTestID, permission.ResourceWorkerDOTTest)
	spec.maxTier = agent.TierPropose
	spec.reversible = false

	return newReportingReceivableTool(spec,
		receivablePlan[*reasonedRecord, *workerdrugalcoholservice.TestChange]{
			request: reasonedRecordFrom(paramDOTTestID, fieldReason),
			plan: func(
				ctx context.Context,
				req *reasonedRecord,
				_ *serviceports.ToolExecuteParams,
			) (*workerdrugalcoholservice.TestChange, error) {
				return tests.PlanCancelTest(ctx, req.tenant, req.id, req.reason)
			},
			refused: func(*reasonedRecord) string { return "Would cancel a drug or alcohol test." },
			render: func(
				_ *reasonedRecord,
				change *workerdrugalcoholservice.TestChange,
			) (*agent.ToolPreview, error) {
				recorded, err := toolpreview.Changed(dotTestRecord(change.Before), change.Before,
					change.After, wfOptions(fieldStatus, "result", wfFieldNotes)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build("Would cancel the test and keep why on the record.",
					recorded), nil
			},
			run: func(
				ctx context.Context,
				req *reasonedRecord,
				_ *serviceports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				cancelled, err := tests.CancelTest(ctx, req.tenant, req.id, req.reason, req.userID)
				if err != nil {
					return nil, err
				}
				return wfResult("cancelled", kindDOTTest, paramDOTTestID, cancelled.ID,
					cancelled.WorkerID), nil
			},
		})
}

func drawRequestFrom(
	params *serviceports.ToolExecuteParams,
) (*workerdrugalcoholservice.RunDrawRequest, error) {
	poolID, err := optionalID(params.Params, paramPoolID)
	if err != nil {
		return nil, err
	}
	day, err := optionalScheduleDay(params.Params, paramDrawDay)
	if err != nil {
		return nil, err
	}
	notes, err := boundedText(params.Params, fieldNotes, wfNoteChars)
	if err != nil {
		return nil, err
	}
	req := &workerdrugalcoholservice.RunDrawRequest{
		TenantInfo: tenantFrom(*params),
		PoolID:     poolID,
		Notes:      notes,
		UserID:     params.Actor.UserID,
	}
	if day != nil {
		req.At = *day
	}
	return req, nil
}

func newRunDOTRandomDrawTool(draws drugAlcoholKeeper) serviceports.AgentTool {
	spec := withSchema(wfSpec(
		"run_dot_random_draw",
		"Draw a random testing round from a pool for the period a day falls in, today by "+
			"default. The pool's rates set how many drivers each substance needs; the names "+
			"are chosen only when the round is drawn, under a seed nobody sees first, so the "+
			"preview names no one. The round starts as a draft a person reviews and finalizes "+
			"with finalize_dot_random_draw, or cancels.",
		"Draws a draft round inside Trenova; nobody is told and nothing is final until a "+
			"person finalizes it, and cancel_dot_random_draw withdraws it.",
		permission.ResourceDOTRandomPool,
		permission.OpManage,
	), map[string]any{
		paramPoolID: idProperty("The pool, from list_dot_random_draws. Leave it out to draw " +
			"from the organization's default pool."),
		paramDrawDay: dayProperty("A day inside the period to draw for, for a period that was " +
			"missed. Defaults to today."),
		fieldNotes: wfNoteProperty("Anything the round's record should say."),
	})
	spec.artifact = ""
	spec.searchTerms = []string{"random drug test", "random selection", "testing pool"}

	return newReceivableTool(spec, receivablePlan[
		*workerdrugalcoholservice.RunDrawRequest, *worker.DOTRandomDraw,
	]{
		request: drawRequestFrom,
		plan: func(
			ctx context.Context,
			req *workerdrugalcoholservice.RunDrawRequest,
			_ *serviceports.ToolExecuteParams,
		) (*worker.DOTRandomDraw, error) {
			return draws.PlanRunDraw(ctx, req)
		},
		refused: func(*workerdrugalcoholservice.RunDrawRequest) string {
			return "Would draw a random testing round."
		},
		render: func(
			_ *workerdrugalcoholservice.RunDrawRequest,
			planned *worker.DOTRandomDraw,
		) (*agent.ToolPreview, error) {
			label := "Random testing round " + planned.PeriodKey
			if planned.Pool != nil {
				label = planned.Pool.Name + " " + planned.PeriodKey
			}
			change, err := toolpreview.Create(
				wfRecord(permission.ResourceDOTRandomPool, pulid.Nil, label, 0),
				planned, append(wfOptions(drawFields...), toolpreview.Volatile("drawnAt"))...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would draw %s from %d drivers: %d for drug testing and %d for alcohol, as a "+
					"draft. Who is picked is decided when it is drawn.",
				planned.PeriodKey, planned.PoolSize, planned.DrugSelected,
				planned.AlcoholSelected,
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *workerdrugalcoholservice.RunDrawRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := draws.RunDraw(ctx, req)
			return nil, err
		},
	})
}

type drawDecision struct {
	id     pulid.ID
	tenant pagination.TenantInfo
	userID pulid.ID
}

func drawIDProperty() map[string]any {
	return idProperty("The round, from list_dot_random_draws. Never guess one.")
}

func renderDrawChange(summary string) func(
	any,
	*workerdrugalcoholservice.DrawChange,
) (*agent.ToolPreview, error) {
	return func(_ any, change *workerdrugalcoholservice.DrawChange) (*agent.ToolPreview, error) {
		recorded, err := toolpreview.Changed(drawRecord(change.Before), change.Before,
			change.After, wfOptions(fieldStatus, "finalizedAt", wfFieldNotes)...)
		if err != nil {
			return nil, err
		}
		return toolpreview.Build(fmt.Sprintf(summary, change.Before.PeriodKey), recorded), nil
	}
}

func newFinalizeDOTRandomDrawTool(draws drugAlcoholKeeper) serviceports.AgentTool {
	spec := personOnly(targeting(withSchema(wfSpec(
		"finalize_dot_random_draw",
		"Finalize a draft random testing round so its selections stand as the evidence of "+
			"the period. A final round is never re-drawn; a mistake is corrected by cancelling "+
			"it. Only a person's approval runs it.",
		"Locks the round's selections as the record an auditor reads; it cannot be undone, "+
			"only cancelled, so a person approves it.",
		permission.ResourceDOTRandomPool,
		permission.OpManage,
	), map[string]any{paramDrawID: drawIDProperty()}, paramDrawID), paramDrawID,
		permission.ResourceDOTRandomPool))
	spec.reversible = false
	spec.artifact = ""
	render := renderDrawChange("Would finalize round %s; its selections become the record.")

	return newReceivableTool(
		spec,
		receivablePlan[*drawDecision, *workerdrugalcoholservice.DrawChange]{
			request: func(params *serviceports.ToolExecuteParams) (*drawDecision, error) {
				id, err := requirePulid(params.Params, paramDrawID)
				if err != nil {
					return nil, err
				}
				return &drawDecision{id: id, tenant: tenantFrom(*params),
					userID: params.Actor.UserID}, nil
			},
			plan: func(
				ctx context.Context,
				req *drawDecision,
				_ *serviceports.ToolExecuteParams,
			) (*workerdrugalcoholservice.DrawChange, error) {
				return draws.PlanFinalizeDraw(ctx, req.tenant, req.id)
			},
			refused: func(*drawDecision) string { return "Would finalize a random testing round." },
			render: func(
				req *drawDecision,
				change *workerdrugalcoholservice.DrawChange,
			) (*agent.ToolPreview, error) {
				return render(req, change)
			},
			run: func(
				ctx context.Context,
				req *drawDecision,
				_ *serviceports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				_, err := draws.FinalizeDraw(ctx, req.tenant, req.id, req.userID)
				return nil, err
			},
		},
	)
}

func newCancelDOTRandomDrawTool(draws drugAlcoholKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"cancel_dot_random_draw",
		"Cancel a random testing round, draft or final, so the period can be drawn again. "+
			"The round and the reason stay on the record.",
		"Voids a round so the period is drawn again; the cancelled round and its reason stay "+
			"on the record, and it cannot be undone.",
		permission.ResourceDOTRandomPool,
		permission.OpManage,
	), map[string]any{
		paramDrawID: drawIDProperty(),
		fieldReason: wfNoteProperty("Why the round is cancelled."),
	}, paramDrawID, fieldReason), paramDrawID, permission.ResourceDOTRandomPool)
	spec.maxTier = agent.TierPropose
	spec.reversible = false
	spec.artifact = ""
	render := renderDrawChange("Would cancel round %s so the period can be drawn again.")

	return newReceivableTool(spec, receivablePlan[
		*reasonedRecord, *workerdrugalcoholservice.DrawChange,
	]{
		request: reasonedRecordFrom(paramDrawID, fieldReason),
		plan: func(
			ctx context.Context,
			req *reasonedRecord,
			_ *serviceports.ToolExecuteParams,
		) (*workerdrugalcoholservice.DrawChange, error) {
			return draws.PlanCancelDraw(ctx, req.tenant, req.id, req.reason)
		},
		refused: func(*reasonedRecord) string { return "Would cancel a random testing round." },
		render: func(
			req *reasonedRecord,
			change *workerdrugalcoholservice.DrawChange,
		) (*agent.ToolPreview, error) {
			return render(req, change)
		},
		run: func(
			ctx context.Context,
			req *reasonedRecord,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := draws.CancelDraw(ctx, req.tenant, req.id, req.reason, req.userID)
			return nil, err
		},
	})
}

func newUpdateDOTRandomSelectionTool(draws drugAlcoholKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"update_dot_random_selection",
		"Record where one random testing selection stands. Notified once the driver has "+
			"been told to report, Excused with the reason when they cannot be tested this "+
			"period, or Missed. "+
			"A selection completes on its own when its test is recorded.",
		"Updates a selection on a round inside Trenova; nothing is sent to the driver, and "+
			"the selection is updated again the same way.",
		permission.ResourceDOTRandomPool,
		permission.OpManage,
	), map[string]any{
		paramDrawEntryID: idProperty("The selection, from get_dot_random_draw. Never guess one."),
		paramSelectionMove: agenttoolschema.Enum("Notified, Excused or Missed.",
			randomSelectionMoves),
		paramExcuseReason: stringProperty("For Excused: why, such as extended leave.",
			wfShortChars),
	}, paramDrawEntryID, paramSelectionMove), paramDrawEntryID, permission.ResourceDOTRandomPool)

	return newReportingReceivableTool(spec, receivablePlan[
		*workerdrugalcoholservice.UpdateEntryRequest, *workerdrugalcoholservice.EntryChange,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*workerdrugalcoholservice.UpdateEntryRequest, error) {
			id, err := requirePulid(params.Params, paramDrawEntryID)
			if err != nil {
				return nil, err
			}
			status, err := requireEnum(params.Params, paramSelectionMove,
				randomSelectionMoves.Values)
			if err != nil {
				return nil, err
			}
			excuse, err := boundedText(params.Params, paramExcuseReason, wfShortChars)
			if err != nil {
				return nil, err
			}
			return &workerdrugalcoholservice.UpdateEntryRequest{
				TenantInfo:   tenantFrom(*params),
				EntryID:      id,
				Status:       status,
				ExcuseReason: excuse,
				UserID:       params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *workerdrugalcoholservice.UpdateEntryRequest,
			_ *serviceports.ToolExecuteParams,
		) (*workerdrugalcoholservice.EntryChange, error) {
			return draws.PlanUpdateDrawEntry(ctx, req)
		},
		refused: func(req *workerdrugalcoholservice.UpdateEntryRequest) string {
			return fmt.Sprintf("Would mark a random selection %s.", req.Status)
		},
		render: func(
			_ *workerdrugalcoholservice.UpdateEntryRequest,
			change *workerdrugalcoholservice.EntryChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(
				wfRecord(permission.ResourceDOTRandomPool, change.Before.ID,
					fmt.Sprintf("%s selection #%d", change.Before.Substance, change.Before.Rank),
					change.Before.Version),
				change.Before, change.After,
				append(wfOptions(fieldStatus, "notifiedAt", paramExcuseReason),
					toolpreview.Volatile("notifiedAt"))...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would mark the selection %s.",
				change.After.Status), recorded), nil
		},
		run: func(
			ctx context.Context,
			req *workerdrugalcoholservice.UpdateEntryRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := draws.UpdateDrawEntry(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult(strings.ToLower(string(updated.Status)), kindRandomPick,
				paramDrawEntryID, updated.ID, updated.WorkerID), nil
		},
	})
}

func provideScheduleDOTTestTool(s *workerdrugalcoholservice.Service) serviceports.AgentTool {
	return newScheduleDOTTestTool(s)
}

func provideCancelDOTTestTool(s *workerdrugalcoholservice.Service) serviceports.AgentTool {
	return newCancelDOTTestTool(s)
}

func provideRunDOTRandomDrawTool(s *workerdrugalcoholservice.Service) serviceports.AgentTool {
	return newRunDOTRandomDrawTool(s)
}

func provideFinalizeDOTRandomDrawTool(
	s *workerdrugalcoholservice.Service,
) serviceports.AgentTool {
	return newFinalizeDOTRandomDrawTool(s)
}

func provideCancelDOTRandomDrawTool(s *workerdrugalcoholservice.Service) serviceports.AgentTool {
	return newCancelDOTRandomDrawTool(s)
}

func provideUpdateDOTRandomSelectionTool(
	s *workerdrugalcoholservice.Service,
) serviceports.AgentTool {
	return newUpdateDOTRandomSelectionTool(s)
}
