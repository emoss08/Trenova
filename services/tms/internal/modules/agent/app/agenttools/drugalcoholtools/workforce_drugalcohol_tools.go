package drugalcoholtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcekit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/dotrandom"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramDOTTestID = "testId"

const paramDOTTestType = "testType"

const paramSubstance = "substance"

const paramScheduledAt = "scheduledAt"

const paramCollection = "collectionSite"

const paramDrawEntryID = "drawEntryId"

const paramIsDOT = "isDot"

const paramPoolID = "poolId"

const paramDrawID = "drawId"

const paramDrawDay = "drawDate"

const paramSelectionMove = "status"

const paramExcuseReason = "excuseReason"

const kindDOTTest = "DOT test"

const kindRandomPick = "random selection"

var dotTestTypes = agenttoolschema.Source("worker.dotTestType", workerdot.DOTTestTypeValues())

var dotTestSubstances = agenttoolschema.Source(
	"worker.dotTestSubstance",
	workerdot.DOTTestSubstanceValues(),
)

var randomSelectionMoves = agenttoolschema.Source(
	"dotRandomSelection.agentMove",
	[]dotrandom.RandomEntryStatus{
		dotrandom.RandomEntryNotified,
		dotrandom.RandomEntryExcused,
		dotrandom.RandomEntryMissed,
	},
)

var dotTestFields = []string{
	workforcekit.FieldWorkerID, paramDOTTestType, paramSubstance, changepreview.FieldStatus, "result", paramIsDOT,
	changepreview.FieldReason, paramScheduledAt, paramCollection, workforcekit.ParamSafetyEventID, paramDrawEntryID,
	workforcekit.FieldNotes, "orderedById",
}

var drawFields = []string{
	"periodKey",
	changepreview.FieldPeriodStart,
	changepreview.FieldPeriodEnd,
	changepreview.FieldStatus,
	"poolSize",
	"drugTarget",
	"alcoholTarget",
	"drugSelected",
	"alcoholSelected",
	"drawnAt",
	"finalizedAt",
	workforcekit.FieldNotes,
}

type DrugAlcoholKeeper interface {
	PlanRecordTest(
		ctx context.Context,
		entity *workerdot.WorkerDOTTest,
		userID pulid.ID,
	) (*workerdot.WorkerDOTTest, error)
	RecordTest(
		ctx context.Context,
		entity *workerdot.WorkerDOTTest,
		userID pulid.ID,
	) (*workerdot.WorkerDOTTest, error)
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
	) (*workerdot.WorkerDOTTest, error)
	PlanRunDraw(
		ctx context.Context,
		req *workerdrugalcoholservice.RunDrawRequest,
	) (*dotrandom.DOTRandomDraw, error)
	RunDraw(
		ctx context.Context,
		req *workerdrugalcoholservice.RunDrawRequest,
	) (*dotrandom.DOTRandomDraw, error)
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
	) (*dotrandom.DOTRandomDraw, error)
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
	) (*dotrandom.DOTRandomDraw, error)
	PlanUpdateDrawEntry(
		ctx context.Context,
		req *workerdrugalcoholservice.UpdateEntryRequest,
	) (*workerdrugalcoholservice.EntryChange, error)
	UpdateDrawEntry(
		ctx context.Context,
		req *workerdrugalcoholservice.UpdateEntryRequest,
	) (*dotrandom.DOTRandomDrawEntry, error)
}

func DrugAlcoholToolProviders() []any {
	return []any{
		provideScheduleDOTTestTool,
		provideCancelDOTTestTool,
		provideRunDOTRandomDrawTool,
		provideFinalizeDOTRandomDrawTool,
		provideCancelDOTRandomDrawTool,
		provideUpdateDOTRandomSelectionTool,
	}
}

func dotTestRecord(test *workerdot.WorkerDOTTest) toolpreview.Record {
	return workforcekit.Record(permission.ResourceWorkerDOTTest, test.ID,
		fmt.Sprintf("%s %s test", test.TestType.Label(), strings.ToLower(string(test.Substance))),
		test.Version)
}

func drawRecord(draw *dotrandom.DOTRandomDraw) toolpreview.Record {
	return workforcekit.Record(permission.ResourceDOTRandomPool, draw.ID,
		"Random testing round "+draw.PeriodKey, draw.Version)
}

func scheduledTestFrom(params *agenttoolports.ToolExecuteParams) (*workerdot.WorkerDOTTest, error) {
	workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	testType, err := toolparams.RequireEnum(params.Params, paramDOTTestType, dotTestTypes.Values)
	if err != nil {
		return nil, err
	}
	substance, err := toolparams.RequireEnum(
		params.Params,
		paramSubstance,
		dotTestSubstances.Values,
	)
	if err != nil {
		return nil, err
	}
	scheduledAt, err := toolparams.OptionalDateTime(params.Params, paramScheduledAt)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.BoundedText(
		params.Params,
		changepreview.FieldReason,
		workforcekit.NoteChars,
	)
	if err != nil {
		return nil, err
	}
	site, err := toolparams.BoundedText(params.Params, paramCollection, workforcekit.ShortChars)
	if err != nil {
		return nil, err
	}
	notes, err := toolparams.BoundedText(
		params.Params,
		toolparams.ParamNote,
		workforcekit.NoteChars,
	)
	if err != nil {
		return nil, err
	}
	isDOT, err := workforcekit.OptionalBoolParam(params.Params, paramIsDOT, true)
	if err != nil {
		return nil, err
	}
	test := &workerdot.WorkerDOTTest{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		WorkerID:       workerID,
		TestType:       testType,
		Substance:      substance,
		Status:         workerdot.DOTTestStatusScheduled,
		Result:         workerdot.DOTResultPending,
		IsDOT:          isDOT,
		Reason:         reason,
		ScheduledAt:    scheduledAt,
		CollectionSite: site,
		Notes:          notes,
	}
	if test.SafetyEventID, err = toolparams.OptionalID(params.Params, workforcekit.ParamSafetyEventID); err != nil {
		return nil, err
	}
	if test.DrawEntryID, err = toolparams.OptionalID(params.Params, paramDrawEntryID); err != nil {
		return nil, err
	}
	return test, nil
}

func newScheduleDOTTestTool(tests DrugAlcoholKeeper) agenttoolports.AgentTool {
	spec := workforcekit.WithSchema(workforcekit.Spec(
		"worker_schedule_dot_test",
		"Schedule a drug or alcohol test for a driver. It covers a random selection's "+
			"collection, a post-accident or reasonable-suspicion test, pre-employment, "+
			"return-to-duty or follow-up. A collection covering both substances is two calls. It records the "+
			"order only; results are entered by the person who receives them from the lab or "+
			"the medical review officer.",
		"Files a scheduled collection inside Trenova. The driver is sent nothing, no result is "+
			"recorded, and worker_cancel_dot_test withdraws it.",
		permission.ResourceWorkerDOTTest,
		permission.OpCreate,
	), map[string]any{
		toolparams.ParamWorkerID: toolparams.WorkerProperty(),
		paramDOTTestType:         agenttoolschema.Enum("Why the driver is tested.", dotTestTypes),
		paramSubstance:           agenttoolschema.Enum("What is tested.", dotTestSubstances),
		paramScheduledAt:         agenttoolschema.DateTime("When the collection is booked."),
		changepreview.FieldReason: workforcekit.NoteProperty(
			"What prompted it. Required for a post-accident or " +
				"reasonable-suspicion test; write what was observed or what happened.",
		),
		paramCollection: toolparams.StringProperty("The collection site.", workforcekit.ShortChars),
		workforcekit.ParamSafetyEventID: agenttoolschema.RecordIDText(
			permission.ResourceWorkerSafetyEvent,
			"For a post-accident test: the accident, from "+
				"worker_list_safety_events.",
		),
		paramDrawEntryID: agenttoolschema.KindID(
			"For a random test: the selection it answers, from "+
				"worker_get_dot_random_draw.",
			permission.KindDOTRandomSelection,
		),
		paramIsDOT: toolparams.BooleanProperty(
			"Whether it is a DOT-regulated test. Defaults to true; " +
				"a company-policy test is false.",
		),
		toolparams.ParamNote: workforcekit.NoteProperty("Anything the collector should know."),
	}, toolparams.ParamWorkerID, paramDOTTestType, paramSubstance)
	spec.SearchTerms = []string{"drug test", "alcohol test", "random test", "collection"}

	return toolspec.NewReportingReceivableTool(spec,
		toolspec.ReceivablePlan[*workerdot.WorkerDOTTest, *workerdot.WorkerDOTTest]{
			Request: scheduledTestFrom,
			Plan: func(
				ctx context.Context,
				test *workerdot.WorkerDOTTest,
				params *agenttoolports.ToolExecuteParams,
			) (*workerdot.WorkerDOTTest, error) {
				return tests.PlanRecordTest(ctx, test, params.Actor.UserID)
			},
			Refused: func(*workerdot.WorkerDOTTest) string {
				return "Would schedule a drug or alcohol test."
			},
			Render: func(
				_ *workerdot.WorkerDOTTest,
				planned *workerdot.WorkerDOTTest,
			) (*agentpreview.ToolPreview, error) {
				change, err := toolpreview.Create(
					workforcekit.Record(permission.ResourceWorkerDOTTest, pulid.Nil, "New test", 0),
					planned, workforcekit.Options(dotTestFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would schedule a %s %s test. No result is recorded.",
					strings.ToLower(planned.TestType.Label()),
					strings.ToLower(string(planned.Substance)),
				), change), nil
			},
			Run: func(
				ctx context.Context,
				test *workerdot.WorkerDOTTest,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				created, err := tests.RecordTest(ctx, test, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return workforcekit.Result("scheduled", kindDOTTest, paramDOTTestID, created.ID,
					created.WorkerID), nil
			},
		})
}

func newCancelDOTTestTool(tests DrugAlcoholKeeper) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_cancel_dot_test",
		"Cancel a drug or alcohol test that did not happen, with the reason kept on the "+
			"record. A test with a result cannot be cancelled; a correction is a new test.",
		"Voids a scheduled collection; the row and its reason stay on the record an auditor "+
			"reads, and it cannot be undone.",
		permission.ResourceWorkerDOTTest,
		permission.OpCancel,
	), map[string]any{
		paramDOTTestID: agenttoolschema.RecordIDText(permission.ResourceWorkerDOTTest,
			"The test, from worker_list_dot_tests. Never guess one."),
		changepreview.FieldReason: workforcekit.NoteProperty("Why the collection did not happen."),
	}, paramDOTTestID, changepreview.FieldReason), paramDOTTestID, permission.ResourceWorkerDOTTest)
	spec.MaxTier = agent.TierPropose
	spec.Reversible = false

	return toolspec.NewReportingReceivableTool(spec,
		toolspec.ReceivablePlan[*workforcekit.ReasonedRecord, *workerdrugalcoholservice.TestChange]{
			Request: workforcekit.ReasonedRecordFrom(paramDOTTestID, changepreview.FieldReason),
			Plan: func(
				ctx context.Context,
				req *workforcekit.ReasonedRecord,
				_ *agenttoolports.ToolExecuteParams,
			) (*workerdrugalcoholservice.TestChange, error) {
				return tests.PlanCancelTest(ctx, req.Tenant, req.ID, req.Reason)
			},
			Refused: func(*workforcekit.ReasonedRecord) string { return "Would cancel a drug or alcohol test." },
			Render: func(
				_ *workforcekit.ReasonedRecord,
				change *workerdrugalcoholservice.TestChange,
			) (*agentpreview.ToolPreview, error) {
				recorded, err := toolpreview.Changed(
					dotTestRecord(change.Before),
					change.Before,
					change.After,
					workforcekit.Options(
						changepreview.FieldStatus,
						"result",
						workforcekit.FieldNotes,
					)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build("Would cancel the test and keep why on the record.",
					recorded), nil
			},
			Run: func(
				ctx context.Context,
				req *workforcekit.ReasonedRecord,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				cancelled, err := tests.CancelTest(ctx, req.Tenant, req.ID, req.Reason, req.UserID)
				if err != nil {
					return nil, err
				}
				return workforcekit.Result("cancelled", kindDOTTest, paramDOTTestID, cancelled.ID,
					cancelled.WorkerID), nil
			},
		})
}

func drawRequestFrom(
	params *agenttoolports.ToolExecuteParams,
) (*workerdrugalcoholservice.RunDrawRequest, error) {
	poolID, err := toolparams.OptionalID(params.Params, paramPoolID)
	if err != nil {
		return nil, err
	}
	day, err := toolparams.OptionalScheduleDay(params.Params, paramDrawDay)
	if err != nil {
		return nil, err
	}
	notes, err := toolparams.BoundedText(
		params.Params,
		toolparams.ParamNote,
		workforcekit.NoteChars,
	)
	if err != nil {
		return nil, err
	}
	req := &workerdrugalcoholservice.RunDrawRequest{
		TenantInfo: toolparams.TenantFrom(*params),
		PoolID:     poolID,
		Notes:      notes,
		UserID:     params.Actor.UserID,
	}
	if day != nil {
		req.At = *day
	}
	return req, nil
}

func newRunDOTRandomDrawTool(draws DrugAlcoholKeeper) agenttoolports.AgentTool {
	spec := workforcekit.WithSchema(workforcekit.Spec(
		"worker_run_dot_random_draw",
		"Draw a random testing round from a pool for the period a day falls in, today by "+
			"default. The pool's rates set how many drivers each substance needs; the names "+
			"are chosen only when the round is drawn, under a seed nobody sees first, so the "+
			"preview names no one. The round starts as a draft a person reviews and finalizes "+
			"with worker_finalize_dot_random_draw, or cancels.",
		"Draws a draft round inside Trenova; nobody is told and nothing is final until a "+
			"person finalizes it, and worker_cancel_dot_random_draw withdraws it.",
		permission.ResourceDOTRandomPool,
		permission.OpManage,
	), map[string]any{
		paramPoolID: agenttoolschema.KindID(
			"The pool, from worker_list_dot_random_draws. Leave it out to draw "+
				"from the organization's default pool.",
			permission.KindDOTRandomPool,
		),
		paramDrawDay: agenttoolschema.Date(
			"A day inside the period to draw for, for a period that was " +
				"missed. Defaults to today.",
		),
		toolparams.ParamNote: workforcekit.NoteProperty("Anything the round's record should say."),
	})
	spec.Artifact = ""
	spec.SearchTerms = []string{"random drug test", "random selection", "testing pool"}

	return toolspec.NewReceivableTool(spec, toolspec.ReceivablePlan[
		*workerdrugalcoholservice.RunDrawRequest, *dotrandom.DOTRandomDraw,
	]{
		Request: drawRequestFrom,
		Plan: func(
			ctx context.Context,
			req *workerdrugalcoholservice.RunDrawRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*dotrandom.DOTRandomDraw, error) {
			return draws.PlanRunDraw(ctx, req)
		},
		Refused: func(*workerdrugalcoholservice.RunDrawRequest) string {
			return "Would draw a random testing round."
		},
		Render: func(
			_ *workerdrugalcoholservice.RunDrawRequest,
			planned *dotrandom.DOTRandomDraw,
		) (*agentpreview.ToolPreview, error) {
			label := "Random testing round " + planned.PeriodKey
			if planned.Pool != nil {
				label = planned.Pool.Name + " " + planned.PeriodKey
			}
			change, err := toolpreview.Create(
				workforcekit.Record(permission.ResourceDOTRandomPool, pulid.Nil, label, 0),
				planned,
				append(workforcekit.Options(drawFields...), toolpreview.Volatile("drawnAt"))...)
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
		Run: func(
			ctx context.Context,
			req *workerdrugalcoholservice.RunDrawRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
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
	return agenttoolschema.KindID(
		agenttoolschema.IDDescription("The round", "worker_list_dot_random_draws"),
		permission.KindDOTRandomDraw,
	)
}

func renderDrawChange(summary string) func(
	any,
	*workerdrugalcoholservice.DrawChange,
) (*agentpreview.ToolPreview, error) {
	return func(_ any, change *workerdrugalcoholservice.DrawChange) (*agentpreview.ToolPreview, error) {
		recorded, err := toolpreview.Changed(
			drawRecord(change.Before),
			change.Before,
			change.After,
			workforcekit.Options(
				changepreview.FieldStatus,
				"finalizedAt",
				workforcekit.FieldNotes,
			)...)
		if err != nil {
			return nil, err
		}
		return toolpreview.Build(fmt.Sprintf(summary, change.Before.PeriodKey), recorded), nil
	}
}

func newFinalizeDOTRandomDrawTool(draws DrugAlcoholKeeper) agenttoolports.AgentTool {
	spec := workforcekit.PersonOnly(
		workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
			"worker_finalize_dot_random_draw",
			"Finalize a draft random testing round so its selections stand as the evidence of "+
				"the period. A final round is never re-drawn; a mistake is corrected by cancelling "+
				"it. Only a person's approval runs it.",
			"Locks the round's selections as the record an auditor reads; it cannot be undone, "+
				"only cancelled, so a person approves it.",
			permission.ResourceDOTRandomPool,
			permission.OpManage,
		), map[string]any{paramDrawID: drawIDProperty()}, paramDrawID), paramDrawID,
			permission.ResourceDOTRandomPool),
	)
	spec.Reversible = false
	spec.Artifact = ""
	spec.Recipe = []string{
		"worker_list_dot_random_draws",
		"worker_get_dot_random_draw",
		"worker_finalize_dot_random_draw",
	}
	render := renderDrawChange("Would finalize round %s; its selections become the record.")

	return toolspec.NewReceivableTool(
		spec,
		toolspec.ReceivablePlan[*drawDecision, *workerdrugalcoholservice.DrawChange]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*drawDecision, error) {
				id, err := toolparams.RequirePulid(params.Params, paramDrawID)
				if err != nil {
					return nil, err
				}
				return &drawDecision{id: id, tenant: toolparams.TenantFrom(*params),
					userID: params.Actor.UserID}, nil
			},
			Plan: func(
				ctx context.Context,
				req *drawDecision,
				_ *agenttoolports.ToolExecuteParams,
			) (*workerdrugalcoholservice.DrawChange, error) {
				return draws.PlanFinalizeDraw(ctx, req.tenant, req.id)
			},
			Refused: func(*drawDecision) string { return "Would finalize a random testing round." },
			Render: func(
				req *drawDecision,
				change *workerdrugalcoholservice.DrawChange,
			) (*agentpreview.ToolPreview, error) {
				return render(req, change)
			},
			Run: func(
				ctx context.Context,
				req *drawDecision,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				_, err := draws.FinalizeDraw(ctx, req.tenant, req.id, req.userID)
				return nil, err
			},
		},
	)
}

func newCancelDOTRandomDrawTool(draws DrugAlcoholKeeper) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_cancel_dot_random_draw",
		"Cancel a random testing round, draft or final, so the period can be drawn again. "+
			"The round and the reason stay on the record.",
		"Voids a round so the period is drawn again; the cancelled round and its reason stay "+
			"on the record, and it cannot be undone.",
		permission.ResourceDOTRandomPool,
		permission.OpManage,
	), map[string]any{
		paramDrawID:               drawIDProperty(),
		changepreview.FieldReason: workforcekit.NoteProperty("Why the round is cancelled."),
	}, paramDrawID, changepreview.FieldReason), paramDrawID, permission.ResourceDOTRandomPool)
	spec.SearchTerms = []string{"draw again", "redo draw", "cancel selection"}
	spec.MaxTier = agent.TierPropose
	spec.Reversible = false
	spec.Artifact = ""
	render := renderDrawChange("Would cancel round %s so the period can be drawn again.")

	return toolspec.NewReceivableTool(spec, toolspec.ReceivablePlan[
		*workforcekit.ReasonedRecord, *workerdrugalcoholservice.DrawChange,
	]{
		Request: workforcekit.ReasonedRecordFrom(paramDrawID, changepreview.FieldReason),
		Plan: func(
			ctx context.Context,
			req *workforcekit.ReasonedRecord,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerdrugalcoholservice.DrawChange, error) {
			return draws.PlanCancelDraw(ctx, req.Tenant, req.ID, req.Reason)
		},
		Refused: func(*workforcekit.ReasonedRecord) string { return "Would cancel a random testing round." },
		Render: func(
			req *workforcekit.ReasonedRecord,
			change *workerdrugalcoholservice.DrawChange,
		) (*agentpreview.ToolPreview, error) {
			return render(req, change)
		},
		Run: func(
			ctx context.Context,
			req *workforcekit.ReasonedRecord,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := draws.CancelDraw(ctx, req.Tenant, req.ID, req.Reason, req.UserID)
			return nil, err
		},
	})
}

func newUpdateDOTRandomSelectionTool(draws DrugAlcoholKeeper) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_update_dot_random_selection",
		"Record where one random testing selection stands. Notified once the driver has "+
			"been told to report, Excused with the reason when they cannot be tested this "+
			"period, or Missed. "+
			"A selection completes on its own when its test is recorded.",
		"Updates a selection on a round inside Trenova; nothing is sent to the driver, and "+
			"the selection is updated again the same way.",
		permission.ResourceDOTRandomPool,
		permission.OpManage,
	), map[string]any{
		paramDrawEntryID: agenttoolschema.KindID(
			"The selection, from worker_get_dot_random_draw. Never guess one.",
			permission.KindDOTRandomSelection,
		),
		paramSelectionMove: agenttoolschema.Enum("Notified, Excused or Missed.",
			randomSelectionMoves),
		paramExcuseReason: toolparams.StringProperty("For Excused: why, such as extended leave.",
			workforcekit.ShortChars),
	}, paramDrawEntryID, paramSelectionMove), paramDrawEntryID, permission.ResourceDOTRandomPool)

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*workerdrugalcoholservice.UpdateEntryRequest, *workerdrugalcoholservice.EntryChange,
	]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*workerdrugalcoholservice.UpdateEntryRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramDrawEntryID)
			if err != nil {
				return nil, err
			}
			status, err := toolparams.RequireEnum(params.Params, paramSelectionMove,
				randomSelectionMoves.Values)
			if err != nil {
				return nil, err
			}
			excuse, err := toolparams.BoundedText(
				params.Params,
				paramExcuseReason,
				workforcekit.ShortChars,
			)
			if err != nil {
				return nil, err
			}
			return &workerdrugalcoholservice.UpdateEntryRequest{
				TenantInfo:   toolparams.TenantFrom(*params),
				EntryID:      id,
				Status:       status,
				ExcuseReason: excuse,
				UserID:       params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *workerdrugalcoholservice.UpdateEntryRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerdrugalcoholservice.EntryChange, error) {
			return draws.PlanUpdateDrawEntry(ctx, req)
		},
		Refused: func(req *workerdrugalcoholservice.UpdateEntryRequest) string {
			return fmt.Sprintf("Would mark a random selection %s.", req.Status)
		},
		Render: func(
			_ *workerdrugalcoholservice.UpdateEntryRequest,
			change *workerdrugalcoholservice.EntryChange,
		) (*agentpreview.ToolPreview, error) {
			recorded, err := toolpreview.Changed(
				workforcekit.Record(permission.ResourceDOTRandomPool, change.Before.ID,
					fmt.Sprintf("%s selection #%d", change.Before.Substance, change.Before.Rank),
					change.Before.Version),
				change.Before,
				change.After,
				append(
					workforcekit.Options(
						changepreview.FieldStatus,
						"notifiedAt",
						paramExcuseReason,
					),
					toolpreview.Volatile("notifiedAt"),
				)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would mark the selection %s.",
				change.After.Status), recorded), nil
		},
		Run: func(
			ctx context.Context,
			req *workerdrugalcoholservice.UpdateEntryRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			updated, err := draws.UpdateDrawEntry(ctx, req)
			if err != nil {
				return nil, err
			}
			return workforcekit.Result(strings.ToLower(string(updated.Status)), kindRandomPick,
				paramDrawEntryID, updated.ID, updated.WorkerID), nil
		},
	})
}

func provideScheduleDOTTestTool(s *workerdrugalcoholservice.Service) agenttoolports.AgentTool {
	return newScheduleDOTTestTool(s)
}

func provideCancelDOTTestTool(s *workerdrugalcoholservice.Service) agenttoolports.AgentTool {
	return newCancelDOTTestTool(s)
}

func provideRunDOTRandomDrawTool(s *workerdrugalcoholservice.Service) agenttoolports.AgentTool {
	return newRunDOTRandomDrawTool(s)
}

func provideFinalizeDOTRandomDrawTool(
	s *workerdrugalcoholservice.Service,
) agenttoolports.AgentTool {
	return newFinalizeDOTRandomDrawTool(s)
}

func provideCancelDOTRandomDrawTool(s *workerdrugalcoholservice.Service) agenttoolports.AgentTool {
	return newCancelDOTRandomDrawTool(s)
}

func provideUpdateDOTRandomSelectionTool(
	s *workerdrugalcoholservice.Service,
) agenttoolports.AgentTool {
	return newUpdateDOTRandomSelectionTool(s)
}
