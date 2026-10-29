package timeofftools

import (
	"context"
	"fmt"

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
	"github.com/emoss08/trenova/internal/modules/identity/app/orgstructureservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerleaveservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const paramLeaveCaseID = "leaveCaseId"

const paramLeaveEntryID = "leaveDayId"

const paramLeaveType = "leaveType"

const paramLeaveFrequency = "frequency"

const paramMilitaryCaregiver = "militaryCaregiver"

const paramStartsAt = "startsAt"

const paramEndsAt = "endsAt"

const paramRequestedAt = workforcekit.FieldRequestedAt

const paramEligibilityHours = "eligibilityHoursWorked"

const paramUsedOn = "usedOn"

const paramLeaveHours = "hours"

const paramCounts = "countsAgainstEntitlement"

const paramPTOLink = "ptoId"

const paramCertificationDue = "dueDate"

const kindLeaveCase = "leave case"

const kindLeaveDay = "day of leave"

const maxEligibilityHours = 10000

var leaveTypes = agenttoolschema.Source("worker.leaveType", worker.LeaveTypeValues())

var leaveFrequencies = agenttoolschema.Source(
	"worker.leaveFrequency",
	workerleave.LeaveFrequencyValues(),
)

var leaveCaseFields = []string{
	workforcekit.FieldWorkerID, paramLeaveType, changepreview.FieldStatus, paramLeaveFrequency, changepreview.FieldReason,
	paramMilitaryCaregiver, workforcekit.FieldRequestedAt, paramStartsAt, paramEndsAt, workforcekit.FieldClosedAt,
	"certificationStatus", workforcekit.FieldCertRequestedAt, "certificationDueAt",
	paramEligibilityHours, workforcekit.FieldDocument, workforcekit.FieldNotes,
}

var leaveDayFields = []string{
	workforcekit.FieldWorkerID, paramLeaveCaseID, paramUsedOn, paramLeaveHours, paramCounts, paramPTOLink,
	workforcekit.FieldNotes,
}

type LeaveKeeper interface {
	PlanOpenCase(
		ctx context.Context,
		entity *workerleave.WorkerLeaveCase,
		userID pulid.ID,
	) (*workerleave.WorkerLeaveCase, error)
	OpenCase(
		ctx context.Context,
		entity *workerleave.WorkerLeaveCase,
		userID pulid.ID,
	) (*workerleave.WorkerLeaveCase, error)
	PlanUpdateCase(
		ctx context.Context,
		req *workerleaveservice.UpdateCaseRequest,
	) (*workerleaveservice.CaseChange, error)
	UpdateCase(
		ctx context.Context,
		req *workerleaveservice.UpdateCaseRequest,
	) (*workerleave.WorkerLeaveCase, error)
	PlanCloseCase(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*workerleaveservice.CaseChange, error)
	CloseCase(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) (*workerleave.WorkerLeaveCase, error)
	PlanRequestCertification(
		ctx context.Context,
		req *workerleaveservice.RequestCertificationRequest,
	) (*workerleaveservice.CaseChange, error)
	RequestCertification(
		ctx context.Context,
		req *workerleaveservice.RequestCertificationRequest,
	) (*workerleave.WorkerLeaveCase, error)
	PlanRecordDay(
		ctx context.Context,
		req *workerleaveservice.RecordDayRequest,
	) (*workerleave.WorkerLeaveEntry, error)
	RecordDay(
		ctx context.Context,
		req *workerleaveservice.RecordDayRequest,
	) (*workerleave.WorkerLeaveEntry, error)
	PlanUpdateDay(
		ctx context.Context,
		req *workerleaveservice.UpdateDayRequest,
	) (*workerleaveservice.EntryChange, error)
	UpdateDay(
		ctx context.Context,
		req *workerleaveservice.UpdateDayRequest,
	) (*workerleave.WorkerLeaveEntry, error)
	PlanDeleteDay(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*workerleave.WorkerLeaveEntry, error)
	DeleteDay(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) error
}

func LeaveToolProviders() []any {
	return []any{
		provideOpenLeaveCaseTool,
		provideUpdateLeaveCaseTool,
		provideCloseLeaveCaseTool,
		provideRequestLeaveCertificationTool,
		provideRecordLeaveDayTool,
		provideUpdateLeaveDayTool,
		provideDeleteLeaveDayTool,
	}
}

func leaveCaseIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceWorkerLeave, "The leave case",
		"worker_list_leave_cases")
}

func leaveDayIDProperty() map[string]any {
	return agenttoolschema.KindID(
		agenttoolschema.IDDescription("The day of leave", "worker_list_leave_cases"),
		permission.KindLeaveDay,
	)
}

func leaveCaseRecord(entity *workerleave.WorkerLeaveCase) toolpreview.Record {
	return workforcekit.Record(permission.ResourceWorkerLeave, entity.ID,
		fmt.Sprintf("%s leave from %s", entity.LeaveType, changepreview.DayText(entity.StartsAt)),
		entity.Version)
}

func leaveDayRecord(entry *workerleave.WorkerLeaveEntry) toolpreview.Record {
	return workforcekit.Record(permission.ResourceWorkerLeave, entry.ID,
		"Leave on "+changepreview.DayText(entry.UsedOn), entry.Version)
}

func leaveCaseProperties() map[string]any {
	return map[string]any{
		paramLeaveType:      agenttoolschema.Enum("The kind of leave.", leaveTypes),
		paramLeaveFrequency: agenttoolschema.Enum("How the leave is taken.", leaveFrequencies),
		changepreview.FieldReason: toolparams.StringProperty(
			"Why, as the worker gave it. Keep medical detail out; "+
				"the certification carries that.",
			workforcekit.ShortChars,
		),
		paramMilitaryCaregiver: toolparams.BooleanProperty(
			"Whether it is leave to care for a covered " +
				"servicemember.",
		),
		paramEndsAt: agenttoolschema.Date("The last day of leave, when it is known."),
		paramEligibilityHours: toolparams.IntegerProperty(
			"Hours worked in the twelve months before, "+
				"for eligibility.",
			0,
			maxEligibilityHours,
		),
		workforcekit.ParamDocument: workforcekit.DocumentProperty(),
		toolparams.ParamNote:       workforcekit.NoteProperty("Anything the case should say."),
	}
}

func applyLeaveCaseOpen(entity *workerleave.WorkerLeaveCase, params map[string]any) error {
	var err error
	if kind, given, kindErr := toolparams.OptionalEnum(params, paramLeaveType,
		leaveTypes.Values); kindErr != nil {
		return kindErr
	} else if given {
		entity.LeaveType = kind
	}
	if frequency, given, freqErr := toolparams.OptionalEnum(params, paramLeaveFrequency,
		leaveFrequencies.Values); freqErr != nil {
		return freqErr
	} else if given {
		entity.Frequency = frequency
	}
	if entity.Reason, err = toolparams.BoundedText(
		params,
		changepreview.FieldReason,
		workforcekit.ShortChars,
	); err != nil {
		return err
	}
	if entity.MilitaryCaregiver, err = workforcekit.OptionalBoolParam(params, paramMilitaryCaregiver,
		false); err != nil {
		return err
	}
	if entity.EndsAt, err = toolparams.OptionalScheduleDay(params, paramEndsAt); err != nil {
		return err
	}
	if entity.DocumentID, err = toolparams.OptionalID(params, workforcekit.ParamDocument); err != nil {
		return err
	}
	if entity.Notes, err = toolparams.BoundedText(params, toolparams.ParamNote, workforcekit.NoteChars); err != nil {
		return err
	}
	hours, err := toolparams.OptionalIntInRange(
		params,
		paramEligibilityHours,
		0,
		maxEligibilityHours,
	)
	if err != nil {
		return err
	}
	if hours != nil {
		value := int32(*hours) //nolint:gosec // bounded by the schema's maximum
		entity.EligibilityHoursWorked = &value
	}
	return nil
}

func newOpenLeaveCaseTool(cases LeaveKeeper, scope LeaveScope) agenttoolports.AgentTool {
	properties := leaveCaseProperties()
	properties[toolparams.ParamWorkerID] = toolparams.WorkerProperty()
	properties[paramStartsAt] = agenttoolschema.Date("The first day of leave.")
	properties[paramRequestedAt] = agenttoolschema.Date(
		"The day the worker asked. Defaults to today.",
	)
	spec := workforcekit.WithSchema(workforcekit.Spec(
		"worker_open_leave_case",
		"Open a leave case for a worker who asked for FMLA, medical, military, parental or "+
			"personal leave, from the day it starts. The case waits for a decision a person "+
			"makes; the days taken are recorded against it with worker_record_leave_day.",
		"Files a pending leave request inside Trenova; nothing is decided or sent, and the "+
			"case is corrected with worker_update_leave_case.",
		permission.ResourceWorkerLeave,
		permission.OpCreate,
	), properties, toolparams.ParamWorkerID, paramStartsAt)
	spec.SearchTerms = []string{"FMLA", "leave of absence", "medical leave"}

	return toolspec.NewReportingReceivableTool(spec,
		toolspec.ReceivablePlan[*workerleave.WorkerLeaveCase, *workerleave.WorkerLeaveCase]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*workerleave.WorkerLeaveCase, error) {
				workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
				if err != nil {
					return nil, err
				}
				starts, err := toolparams.RequireScheduleDay(params.Params, paramStartsAt)
				if err != nil {
					return nil, err
				}
				requested, err := toolparams.OptionalScheduleDay(params.Params, paramRequestedAt)
				if err != nil {
					return nil, err
				}
				entity := &workerleave.WorkerLeaveCase{
					OrganizationID: params.OrganizationID,
					BusinessUnitID: params.BusinessUnitID,
					WorkerID:       workerID,
					StartsAt:       starts,
				}
				if requested != nil {
					entity.RequestedAt = *requested
				}
				return entity, applyLeaveCaseOpen(entity, params.Params)
			},
			Settle: func(
				ctx context.Context,
				entity *workerleave.WorkerLeaveCase,
				params *agenttoolports.ToolExecuteParams,
			) error {
				return scope.worker(ctx, params, permission.OpCreate, entity.WorkerID)
			},
			Plan: func(
				ctx context.Context,
				entity *workerleave.WorkerLeaveCase,
				params *agenttoolports.ToolExecuteParams,
			) (*workerleave.WorkerLeaveCase, error) {
				return cases.PlanOpenCase(ctx, entity, params.Actor.UserID)
			},
			Refused: func(*workerleave.WorkerLeaveCase) string { return "Would open a leave case." },
			Render: func(
				_ *workerleave.WorkerLeaveCase,
				planned *workerleave.WorkerLeaveCase,
			) (*agentpreview.ToolPreview, error) {
				change, err := toolpreview.Create(
					workforcekit.Record(
						permission.ResourceWorkerLeave,
						pulid.Nil,
						"New leave case",
						0,
					),
					planned,
					append(workforcekit.Options(leaveCaseFields...),
						toolpreview.Volatile(workforcekit.FieldRequestedAt))...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would open a %s leave case from %s, waiting for a decision.",
					planned.LeaveType, changepreview.DayText(planned.StartsAt)), change), nil
			},
			Run: func(
				ctx context.Context,
				entity *workerleave.WorkerLeaveCase,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				created, err := cases.OpenCase(ctx, entity, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return workforcekit.Result("opened", kindLeaveCase, paramLeaveCaseID, created.ID,
					created.WorkerID), nil
			},
		})
}

func leaveCaseUpdateFrom(
	params *agenttoolports.ToolExecuteParams,
) (*workerleaveservice.UpdateCaseRequest, error) {
	id, err := toolparams.RequirePulid(params.Params, paramLeaveCaseID)
	if err != nil {
		return nil, err
	}
	req := &workerleaveservice.UpdateCaseRequest{
		TenantInfo: toolparams.TenantFrom(*params),
		CaseID:     id,
		UserID:     params.Actor.UserID,
	}
	if kind, given, kindErr := toolparams.OptionalEnum(params.Params, paramLeaveType,
		leaveTypes.Values); kindErr != nil {
		return nil, kindErr
	} else if given {
		req.LeaveType = &kind
	}
	if frequency, given, freqErr := toolparams.OptionalEnum(params.Params, paramLeaveFrequency,
		leaveFrequencies.Values); freqErr != nil {
		return nil, freqErr
	} else if given {
		req.Frequency = &frequency
	}
	if req.Reason, err = toolparams.OptionalBoundedText(
		params.Params,
		changepreview.FieldReason,
		workforcekit.ShortChars,
	); err != nil {
		return nil, err
	}
	if req.Notes, err = toolparams.OptionalBoundedText(
		params.Params,
		toolparams.ParamNote,
		workforcekit.NoteChars,
	); err != nil {
		return nil, err
	}
	if req.MilitaryCaregiver, err = toolparams.OptionalBoolPointer(params.Params,
		paramMilitaryCaregiver); err != nil {
		return nil, err
	}
	if req.StartsAt, err = toolparams.OptionalScheduleDay(params.Params, paramStartsAt); err != nil {
		return nil, err
	}
	if req.EndsAt, err = toolparams.OptionalScheduleDay(params.Params, paramEndsAt); err != nil {
		return nil, err
	}
	if req.DocumentID, err = toolparams.OptionalID(params.Params, workforcekit.ParamDocument); err != nil {
		return nil, err
	}
	hours, err := toolparams.OptionalIntInRange(
		params.Params,
		paramEligibilityHours,
		0,
		maxEligibilityHours,
	)
	if err != nil {
		return nil, err
	}
	if hours != nil {
		value := int32(*hours) //nolint:gosec // bounded by the schema's maximum
		req.EligibilityHoursWorked = &value
	}
	return req, nil
}

func renderLeaveCaseChange(
	summary string,
	volatile []string,
	fields ...string,
) func(*workerleaveservice.CaseChange) (*agentpreview.ToolPreview, error) {
	options := append(workforcekit.Options(fields...), toolpreview.Volatile(volatile...))
	return func(change *workerleaveservice.CaseChange) (*agentpreview.ToolPreview, error) {
		recorded, err := toolpreview.Changed(leaveCaseRecord(change.Before), change.Before,
			change.After, options...)
		if err != nil {
			return nil, err
		}
		return toolpreview.Build(summary, recorded), nil
	}
}

func newUpdateLeaveCaseTool(cases LeaveKeeper, scope LeaveScope) agenttoolports.AgentTool {
	properties := leaveCaseProperties()
	properties[paramLeaveCaseID] = leaveCaseIDProperty()
	properties[paramStartsAt] = agenttoolschema.Date("The first day of leave.")
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_update_leave_case",
		"Correct or fill in a leave case as it goes: the kind, how it is taken, the dates, "+
			"hours worked for eligibility, a document, notes. Give only what changes. It never "+
			"decides the case.",
		"Corrects a leave case inside Trenova; nothing is decided or sent, and it is "+
			"corrected again the same way.",
		permission.ResourceWorkerLeave,
		permission.OpUpdate,
	), properties, paramLeaveCaseID), paramLeaveCaseID, permission.ResourceWorkerLeave)
	spec.SearchTerms = []string{"leave dates", "fix leave case"}
	render := renderLeaveCaseChange("Would correct the leave case.", nil, leaveCaseFields...)

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*workerleaveservice.UpdateCaseRequest, *workerleaveservice.CaseChange,
	]{
		Request: leaveCaseUpdateFrom,
		Settle: func(
			ctx context.Context,
			req *workerleaveservice.UpdateCaseRequest,
			params *agenttoolports.ToolExecuteParams,
		) error {
			return scope.leaveCase(ctx, params, permission.OpUpdate, req.CaseID)
		},
		Plan: func(
			ctx context.Context,
			req *workerleaveservice.UpdateCaseRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerleaveservice.CaseChange, error) {
			return cases.PlanUpdateCase(ctx, req)
		},
		Refused: func(*workerleaveservice.UpdateCaseRequest) string {
			return "Would correct a leave case."
		},
		Render: func(
			_ *workerleaveservice.UpdateCaseRequest,
			change *workerleaveservice.CaseChange,
		) (*agentpreview.ToolPreview, error) {
			return render(change)
		},
		Run: func(
			ctx context.Context,
			req *workerleaveservice.UpdateCaseRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			updated, err := cases.UpdateCase(ctx, req)
			if err != nil {
				return nil, err
			}
			return workforcekit.Result("updated", kindLeaveCase, paramLeaveCaseID, updated.ID,
				updated.WorkerID), nil
		},
	})
}

func newCloseLeaveCaseTool(cases LeaveKeeper, scope LeaveScope) agenttoolports.AgentTool {
	spec := workforcekit.PersonOnly(
		workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
			"worker_close_leave_case",
			"Close a decided leave case once the leave is over. The days recorded against it "+
				"stay counted. A case still waiting for a decision is refused. Only a person's "+
				"approval runs it.",
			"Closes a leave case under the leave approval grant, so it runs as the person who "+
				"approves it; the days it drew down stay drawn down.",
			permission.ResourceWorkerLeave,
			permission.OpApprove,
		), map[string]any{paramLeaveCaseID: leaveCaseIDProperty()}, paramLeaveCaseID),
			paramLeaveCaseID, permission.ResourceWorkerLeave),
	)
	spec.SearchTerms = []string{"returned from leave", "leave ended", "close fmla"}
	spec.Recipe = []string{"worker_list_leave_cases", "worker_close_leave_case"}
	render := renderLeaveCaseChange(
		"Would close the leave case.",
		[]string{
			workforcekit.FieldClosedAt,
		},
		changepreview.FieldStatus,
		workforcekit.FieldClosedAt,
		"endsAt",
	)

	return toolspec.NewReportingReceivableTool(
		spec,
		toolspec.ReceivablePlan[*workforcekit.RecordDelete, *workerleaveservice.CaseChange]{
			Request: workforcekit.RecordDeleteFrom(paramLeaveCaseID),
			Settle: func(
				ctx context.Context,
				req *workforcekit.RecordDelete,
				params *agenttoolports.ToolExecuteParams,
			) error {
				return scope.leaveCase(ctx, params, permission.OpApprove, req.ID)
			},
			Plan: func(
				ctx context.Context,
				req *workforcekit.RecordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*workerleaveservice.CaseChange, error) {
				return cases.PlanCloseCase(ctx, req.Tenant, req.ID)
			},
			Refused: func(*workforcekit.RecordDelete) string { return "Would close a leave case." },
			Render: func(_ *workforcekit.RecordDelete, change *workerleaveservice.CaseChange) (*agentpreview.ToolPreview, error) {
				return render(change)
			},
			Run: func(
				ctx context.Context,
				req *workforcekit.RecordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				closed, err := cases.CloseCase(ctx, req.Tenant, req.ID, req.UserID)
				if err != nil {
					return nil, err
				}
				return workforcekit.Result("closed", kindLeaveCase, paramLeaveCaseID, closed.ID,
					closed.WorkerID), nil
			},
		},
	)
}

func newRequestLeaveCertificationTool(
	cases LeaveKeeper,
	scope LeaveScope,
) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_request_leave_certification",
		"Record that medical certification was asked for on a leave case and start its "+
			"clock. It is due in the organization's certification window, fifteen days by "+
			"default, or on the day given. Asking the worker is done by the person; this keeps the date.",
		"Starts the certification clock on a leave case inside Trenova; nothing is sent to "+
			"the worker, and a later request restarts it.",
		permission.ResourceWorkerLeave,
		permission.OpUpdate,
	), map[string]any{
		paramLeaveCaseID:      leaveCaseIDProperty(),
		paramCertificationDue: agenttoolschema.Date("When it is due, when not the usual window."),
	}, paramLeaveCaseID), paramLeaveCaseID, permission.ResourceWorkerLeave)
	spec.SearchTerms = []string{"fmla certification", "certification clock"}
	render := renderLeaveCaseChange("Would record that certification was requested.",
		[]string{workforcekit.FieldCertRequestedAt}, "certificationStatus",
		workforcekit.FieldCertRequestedAt, "certificationDueAt")

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*workerleaveservice.RequestCertificationRequest, *workerleaveservice.CaseChange,
	]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*workerleaveservice.RequestCertificationRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramLeaveCaseID)
			if err != nil {
				return nil, err
			}
			due, err := toolparams.OptionalScheduleDay(params.Params, paramCertificationDue)
			if err != nil {
				return nil, err
			}
			return &workerleaveservice.RequestCertificationRequest{
				TenantInfo: toolparams.TenantFrom(*params),
				CaseID:     id,
				DueAt:      due,
				UserID:     params.Actor.UserID,
			}, nil
		},
		Settle: func(
			ctx context.Context,
			req *workerleaveservice.RequestCertificationRequest,
			params *agenttoolports.ToolExecuteParams,
		) error {
			return scope.leaveCase(ctx, params, permission.OpUpdate, req.CaseID)
		},
		Plan: func(
			ctx context.Context,
			req *workerleaveservice.RequestCertificationRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerleaveservice.CaseChange, error) {
			return cases.PlanRequestCertification(ctx, req)
		},
		Refused: func(*workerleaveservice.RequestCertificationRequest) string {
			return "Would record a certification request."
		},
		Render: func(
			_ *workerleaveservice.RequestCertificationRequest,
			change *workerleaveservice.CaseChange,
		) (*agentpreview.ToolPreview, error) {
			return render(change)
		},
		Run: func(
			ctx context.Context,
			req *workerleaveservice.RequestCertificationRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			updated, err := cases.RequestCertification(ctx, req)
			if err != nil {
				return nil, err
			}
			return workforcekit.Result(
				"requested certification on",
				kindLeaveCase,
				paramLeaveCaseID,
				updated.ID,
				updated.WorkerID,
			), nil
		},
	})
}

func requireHours(params map[string]any) (decimal.Decimal, error) {
	hours, present, err := toolparams.OptionalDecimal(params, paramLeaveHours)
	if err != nil {
		return decimal.Zero, err
	}
	if !present || !hours.IsPositive() {
		return decimal.Zero, fmt.Errorf("parameter %q must be a positive number of hours",
			paramLeaveHours)
	}
	return hours, nil
}

func leaveDayOptions() []toolpreview.Option {
	return append(
		workforcekit.Options(leaveDayFields...),
		toolpreview.WithRefs(map[string]permission.Resource{
			paramPTOLink:  permission.ResourceWorkerPTO,
			"leaveCaseId": permission.ResourceWorkerLeave,
		}),
	)
}

func recordLeaveDayRequest(
	params *agenttoolports.ToolExecuteParams,
) (*workerleaveservice.RecordDayRequest, error) {
	caseID, err := toolparams.RequirePulid(params.Params, paramLeaveCaseID)
	if err != nil {
		return nil, err
	}
	usedOn, err := toolparams.RequireScheduleDay(params.Params, paramUsedOn)
	if err != nil {
		return nil, err
	}
	hours, err := requireHours(params.Params)
	if err != nil {
		return nil, err
	}
	ptoID, err := toolparams.OptionalID(params.Params, paramPTOLink)
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
	return &workerleaveservice.RecordDayRequest{
		TenantInfo: toolparams.TenantFrom(*params),
		CaseID:     caseID,
		UsedOn:     usedOn,
		Hours:      hours,
		PTOID:      ptoID,
		Notes:      notes,
		UserID:     params.Actor.UserID,
	}, nil
}

func newRecordLeaveDayTool(cases LeaveKeeper, scope LeaveScope) agenttoolports.AgentTool {
	spec := workforcekit.WithSchema(workforcekit.Spec(
		"worker_record_leave_day",
		"Record a day of leave taken against an open or approved leave case, with the "+
			"hours. Whether it counts against the FMLA entitlement follows the case as it "+
			"stands today.",
		"Records a day taken against a leave case inside Trenova; nothing is sent, and "+
			"worker_update_leave_day or worker_delete_leave_day corrects it.",
		permission.ResourceWorkerLeave,
		permission.OpCreate,
	), map[string]any{
		paramLeaveCaseID: leaveCaseIDProperty(),
		paramUsedOn:      agenttoolschema.Date("The day the leave was taken."),
		paramLeaveHours:  toolparams.AmountProperty("Hours taken that day, such as 8 or 4.5."),
		paramPTOLink: agenttoolschema.RecordIDText(permission.ResourceWorkerPTO,
			"Paid time off booked for the same day, from worker_list_time_off, when the leave "+
				"runs concurrently."),
		toolparams.ParamNote: workforcekit.NoteProperty("Anything the day should say."),
	}, paramLeaveCaseID, paramUsedOn, paramLeaveHours)

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*workerleaveservice.RecordDayRequest, *workerleave.WorkerLeaveEntry,
	]{
		Request: recordLeaveDayRequest,
		Settle: func(
			ctx context.Context,
			req *workerleaveservice.RecordDayRequest,
			params *agenttoolports.ToolExecuteParams,
		) error {
			return scope.leaveCase(ctx, params, permission.OpCreate, req.CaseID)
		},
		Plan: func(
			ctx context.Context,
			req *workerleaveservice.RecordDayRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerleave.WorkerLeaveEntry, error) {
			return cases.PlanRecordDay(ctx, req)
		},
		Refused: func(*workerleaveservice.RecordDayRequest) string {
			return "Would record a day of leave."
		},
		Render: func(
			_ *workerleaveservice.RecordDayRequest,
			planned *workerleave.WorkerLeaveEntry,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Create(
				workforcekit.Record(permission.ResourceWorkerLeave, pulid.Nil,
					"Leave on "+changepreview.DayText(planned.UsedOn), 0),
				planned, leaveDayOptions()...)
			if err != nil {
				return nil, err
			}
			counts := "does not count"
			if planned.CountsAgainstEntitlement {
				counts = "counts"
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would record %s hours of leave on %s; it %s against the entitlement.",
				planned.Hours.String(), changepreview.DayText(planned.UsedOn), counts), change), nil
		},
		Run: func(
			ctx context.Context,
			req *workerleaveservice.RecordDayRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			created, err := cases.RecordDay(ctx, req)
			if err != nil {
				return nil, err
			}
			return workforcekit.Result("recorded", kindLeaveDay, paramLeaveEntryID, created.ID,
				created.WorkerID), nil
		},
	})
}

func newUpdateLeaveDayTool(cases LeaveKeeper, scope LeaveScope) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_update_leave_day",
		"Correct a day of leave already recorded: the hours, whether it counts against the "+
			"entitlement, the time off it runs with, or its note. Give only what changes.",
		"Corrects a recorded day of leave inside Trenova; nothing is sent, and it is corrected "+
			"again the same way.",
		permission.ResourceWorkerLeave,
		permission.OpUpdate,
	), map[string]any{
		paramLeaveEntryID: leaveDayIDProperty(),
		paramLeaveHours:   toolparams.AmountProperty("Hours taken that day, such as 8 or 4.5."),
		paramCounts: toolparams.BooleanProperty("Whether this day counts against the FMLA " +
			"entitlement."),
		paramPTOLink: agenttoolschema.RecordIDText(
			permission.ResourceWorkerPTO,
			"Paid time off booked for the same day, from worker_list_time_off.",
		),
		toolparams.ParamNote: workforcekit.NoteProperty("The day's note."),
	}, paramLeaveEntryID), paramLeaveEntryID, permission.ResourceWorkerLeave)

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*workerleaveservice.UpdateDayRequest, *workerleaveservice.EntryChange,
	]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*workerleaveservice.UpdateDayRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramLeaveEntryID)
			if err != nil {
				return nil, err
			}
			req := &workerleaveservice.UpdateDayRequest{
				TenantInfo: toolparams.TenantFrom(*params),
				EntryID:    id,
				UserID:     params.Actor.UserID,
			}
			if _, given := params.Params[paramLeaveHours]; given {
				hours, hoursErr := requireHours(params.Params)
				if hoursErr != nil {
					return nil, hoursErr
				}
				req.Hours = &hours
			}
			if req.Counts, err = toolparams.OptionalBoolPointer(
				params.Params,
				paramCounts,
			); err != nil {
				return nil, err
			}
			if req.PTOID, err = toolparams.OptionalID(params.Params, paramPTOLink); err != nil {
				return nil, err
			}
			if req.Notes, err = toolparams.OptionalBoundedText(params.Params, toolparams.ParamNote,
				workforcekit.NoteChars); err != nil {
				return nil, err
			}
			return req, nil
		},
		Settle: func(
			ctx context.Context,
			req *workerleaveservice.UpdateDayRequest,
			params *agenttoolports.ToolExecuteParams,
		) error {
			return scope.entry(ctx, params, permission.OpUpdate, req.EntryID)
		},
		Plan: func(
			ctx context.Context,
			req *workerleaveservice.UpdateDayRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerleaveservice.EntryChange, error) {
			return cases.PlanUpdateDay(ctx, req)
		},
		Refused: func(*workerleaveservice.UpdateDayRequest) string {
			return "Would correct a day of leave."
		},
		Render: func(
			_ *workerleaveservice.UpdateDayRequest,
			change *workerleaveservice.EntryChange,
		) (*agentpreview.ToolPreview, error) {
			recorded, err := toolpreview.Changed(leaveDayRecord(change.Before), change.Before,
				change.After, leaveDayOptions()...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build("Would correct the day of leave.", recorded), nil
		},
		Run: func(
			ctx context.Context,
			req *workerleaveservice.UpdateDayRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			updated, err := cases.UpdateDay(ctx, req)
			if err != nil {
				return nil, err
			}
			return workforcekit.Result("updated", kindLeaveDay, paramLeaveEntryID, updated.ID,
				updated.WorkerID), nil
		},
	})
}

func newDeleteLeaveDayTool(cases LeaveKeeper, scope LeaveScope) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_delete_leave_day",
		"Remove a day of leave recorded in error, which gives its hours back to the "+
			"worker's entitlement.",
		"Removes a recorded day of leave; the audit trail keeps what was removed, and "+
			"worker_record_leave_day records it again.",
		permission.ResourceWorkerLeave,
		permission.OpUpdate,
	), map[string]any{paramLeaveEntryID: leaveDayIDProperty()}, paramLeaveEntryID),
		paramLeaveEntryID, permission.ResourceWorkerLeave)
	spec.SearchTerms = []string{"remove leave day"}
	spec.MaxTier = agent.TierPropose
	spec.Reversible = false

	return toolspec.NewReceivableTool(
		spec,
		toolspec.ReceivablePlan[*workforcekit.RecordDelete, *workerleave.WorkerLeaveEntry]{
			Request: workforcekit.RecordDeleteFrom(paramLeaveEntryID),
			Settle: func(
				ctx context.Context,
				req *workforcekit.RecordDelete,
				params *agenttoolports.ToolExecuteParams,
			) error {
				return scope.entry(ctx, params, permission.OpUpdate, req.ID)
			},
			Plan: func(
				ctx context.Context,
				req *workforcekit.RecordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*workerleave.WorkerLeaveEntry, error) {
				return cases.PlanDeleteDay(ctx, req.Tenant, req.ID)
			},
			Refused: func(*workforcekit.RecordDelete) string { return "Would remove a day of leave." },
			Render: func(_ *workforcekit.RecordDelete, entry *workerleave.WorkerLeaveEntry) (*agentpreview.ToolPreview, error) {
				change, err := toolpreview.Delete(
					leaveDayRecord(entry),
					entry,
					leaveDayOptions()...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would remove %s hours of leave on %s and give them back to the entitlement.",
					entry.Hours.String(), changepreview.DayText(entry.UsedOn)), change), nil
			},
			Run: func(
				ctx context.Context,
				req *workforcekit.RecordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				return nil, cases.DeleteDay(ctx, req.Tenant, req.ID, req.UserID)
			},
		},
	)
}

func provideOpenLeaveCaseTool(
	s *workerleaveservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agenttoolports.AgentTool {
	return newOpenLeaveCaseTool(s, NewLeaveScope(permissions, teams, s))
}

func provideUpdateLeaveCaseTool(
	s *workerleaveservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agenttoolports.AgentTool {
	return newUpdateLeaveCaseTool(s, NewLeaveScope(permissions, teams, s))
}

func provideCloseLeaveCaseTool(
	s *workerleaveservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agenttoolports.AgentTool {
	return newCloseLeaveCaseTool(s, NewLeaveScope(permissions, teams, s))
}

func provideRequestLeaveCertificationTool(
	s *workerleaveservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agenttoolports.AgentTool {
	return newRequestLeaveCertificationTool(s, NewLeaveScope(permissions, teams, s))
}

func provideRecordLeaveDayTool(
	s *workerleaveservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agenttoolports.AgentTool {
	return newRecordLeaveDayTool(s, NewLeaveScope(permissions, teams, s))
}

func provideUpdateLeaveDayTool(
	s *workerleaveservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agenttoolports.AgentTool {
	return newUpdateLeaveDayTool(s, NewLeaveScope(permissions, teams, s))
}

func provideDeleteLeaveDayTool(
	s *workerleaveservice.Service,
	permissions permissionports.PermissionEngine,
	teams *orgstructureservice.Service,
) agenttoolports.AgentTool {
	return newDeleteLeaveDayTool(s, NewLeaveScope(permissions, teams, s))
}
