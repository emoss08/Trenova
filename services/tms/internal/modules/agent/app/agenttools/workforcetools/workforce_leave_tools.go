package workforcetools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerleaveservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
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

const paramRequestedAt = wfFieldRequestedAt

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
	worker.LeaveFrequencyValues(),
)

var leaveCaseFields = []string{
	wfFieldWorkerID, paramLeaveType, common.FieldStatus, paramLeaveFrequency, common.FieldReason,
	paramMilitaryCaregiver, wfFieldRequestedAt, paramStartsAt, paramEndsAt, wfFieldClosedAt,
	"certificationStatus", wfFieldCertRequestedAt, "certificationDueAt",
	paramEligibilityHours, wfFieldDocument, wfFieldNotes,
}

var leaveDayFields = []string{
	wfFieldWorkerID, paramLeaveCaseID, paramUsedOn, paramLeaveHours, paramCounts, paramPTOLink,
	wfFieldNotes,
}

type LeaveKeeper interface {
	PlanOpenCase(
		ctx context.Context,
		entity *worker.WorkerLeaveCase,
		userID pulid.ID,
	) (*worker.WorkerLeaveCase, error)
	OpenCase(
		ctx context.Context,
		entity *worker.WorkerLeaveCase,
		userID pulid.ID,
	) (*worker.WorkerLeaveCase, error)
	PlanUpdateCase(
		ctx context.Context,
		req *workerleaveservice.UpdateCaseRequest,
	) (*workerleaveservice.CaseChange, error)
	UpdateCase(
		ctx context.Context,
		req *workerleaveservice.UpdateCaseRequest,
	) (*worker.WorkerLeaveCase, error)
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
	) (*worker.WorkerLeaveCase, error)
	PlanRequestCertification(
		ctx context.Context,
		req *workerleaveservice.RequestCertificationRequest,
	) (*workerleaveservice.CaseChange, error)
	RequestCertification(
		ctx context.Context,
		req *workerleaveservice.RequestCertificationRequest,
	) (*worker.WorkerLeaveCase, error)
	PlanRecordDay(
		ctx context.Context,
		req *workerleaveservice.RecordDayRequest,
	) (*worker.WorkerLeaveEntry, error)
	RecordDay(
		ctx context.Context,
		req *workerleaveservice.RecordDayRequest,
	) (*worker.WorkerLeaveEntry, error)
	PlanUpdateDay(
		ctx context.Context,
		req *workerleaveservice.UpdateDayRequest,
	) (*workerleaveservice.EntryChange, error)
	UpdateDay(
		ctx context.Context,
		req *workerleaveservice.UpdateDayRequest,
	) (*worker.WorkerLeaveEntry, error)
	PlanDeleteDay(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.WorkerLeaveEntry, error)
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
		"list_worker_leave_cases")
}

func leaveDayIDProperty() map[string]any {
	return agenttoolschema.KindID(
		agenttoolschema.IDDescription("The day of leave", "list_worker_leave_cases"),
		permission.KindLeaveDay,
	)
}

func leaveCaseRecord(entity *worker.WorkerLeaveCase) toolpreview.Record {
	return wfRecord(permission.ResourceWorkerLeave, entity.ID,
		fmt.Sprintf("%s leave from %s", entity.LeaveType, common.DayText(entity.StartsAt)),
		entity.Version)
}

func leaveDayRecord(entry *worker.WorkerLeaveEntry) toolpreview.Record {
	return wfRecord(permission.ResourceWorkerLeave, entry.ID,
		"Leave on "+common.DayText(entry.UsedOn), entry.Version)
}

func leaveCaseProperties() map[string]any {
	return map[string]any{
		paramLeaveType:      agenttoolschema.Enum("The kind of leave.", leaveTypes),
		paramLeaveFrequency: agenttoolschema.Enum("How the leave is taken.", leaveFrequencies),
		common.FieldReason: common.StringProperty(
			"Why, as the worker gave it. Keep medical detail out; "+
				"the certification carries that.",
			wfShortChars,
		),
		paramMilitaryCaregiver: common.BooleanProperty(
			"Whether it is leave to care for a covered " +
				"servicemember.",
		),
		paramEndsAt: agenttoolschema.Date("The last day of leave, when it is known."),
		paramEligibilityHours: common.IntegerProperty("Hours worked in the twelve months before, "+
			"for eligibility.", 0, maxEligibilityHours),
		wfParamDocument:   wfDocumentProperty(),
		common.FieldNotes: wfNoteProperty("Anything the case should say."),
	}
}

func applyLeaveCaseOpen(entity *worker.WorkerLeaveCase, params map[string]any) error {
	var err error
	if kind, given, kindErr := common.OptionalEnum(params, paramLeaveType,
		leaveTypes.Values); kindErr != nil {
		return kindErr
	} else if given {
		entity.LeaveType = kind
	}
	if frequency, given, freqErr := common.OptionalEnum(params, paramLeaveFrequency,
		leaveFrequencies.Values); freqErr != nil {
		return freqErr
	} else if given {
		entity.Frequency = frequency
	}
	if entity.Reason, err = common.BoundedText(
		params,
		common.FieldReason,
		wfShortChars,
	); err != nil {
		return err
	}
	if entity.MilitaryCaregiver, err = optionalBoolParam(params, paramMilitaryCaregiver,
		false); err != nil {
		return err
	}
	if entity.EndsAt, err = common.OptionalScheduleDay(params, paramEndsAt); err != nil {
		return err
	}
	if entity.DocumentID, err = common.OptionalID(params, wfParamDocument); err != nil {
		return err
	}
	if entity.Notes, err = common.BoundedText(params, common.FieldNotes, wfNoteChars); err != nil {
		return err
	}
	hours, err := common.OptionalIntInRange(params, paramEligibilityHours, 0, maxEligibilityHours)
	if err != nil {
		return err
	}
	if hours != nil {
		value := int32(*hours) //nolint:gosec // bounded by the schema's maximum
		entity.EligibilityHoursWorked = &value
	}
	return nil
}

func newOpenLeaveCaseTool(cases LeaveKeeper) agentports.AgentTool {
	properties := leaveCaseProperties()
	properties[common.ParamWorkerID] = common.WorkerProperty()
	properties[paramStartsAt] = agenttoolschema.Date("The first day of leave.")
	properties[paramRequestedAt] = agenttoolschema.Date(
		"The day the worker asked. Defaults to today.",
	)
	spec := withSchema(wfSpec(
		"open_leave_case",
		"Open a leave case for a worker who asked for FMLA, medical, military, parental or "+
			"personal leave, from the day it starts. The case waits for a decision a person "+
			"makes; the days taken are recorded against it with record_leave_day.",
		"Files a pending leave request inside Trenova; nothing is decided or sent, and the "+
			"case is corrected with update_leave_case.",
		permission.ResourceWorkerLeave,
		permission.OpCreate,
	), properties, common.ParamWorkerID, paramStartsAt)
	spec.SearchTerms = []string{"FMLA", "leave of absence", "medical leave"}

	return common.NewReportingReceivableTool(spec,
		common.ReceivablePlan[*worker.WorkerLeaveCase, *worker.WorkerLeaveCase]{
			Request: func(params *agentports.ToolExecuteParams) (*worker.WorkerLeaveCase, error) {
				workerID, err := common.RequirePulid(params.Params, common.ParamWorkerID)
				if err != nil {
					return nil, err
				}
				starts, err := common.RequireScheduleDay(params.Params, paramStartsAt)
				if err != nil {
					return nil, err
				}
				requested, err := common.OptionalScheduleDay(params.Params, paramRequestedAt)
				if err != nil {
					return nil, err
				}
				entity := &worker.WorkerLeaveCase{
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
			Plan: func(
				ctx context.Context,
				entity *worker.WorkerLeaveCase,
				params *agentports.ToolExecuteParams,
			) (*worker.WorkerLeaveCase, error) {
				return cases.PlanOpenCase(ctx, entity, params.Actor.UserID)
			},
			Refused: func(*worker.WorkerLeaveCase) string { return "Would open a leave case." },
			Render: func(
				_ *worker.WorkerLeaveCase,
				planned *worker.WorkerLeaveCase,
			) (*agent.ToolPreview, error) {
				change, err := toolpreview.Create(
					wfRecord(permission.ResourceWorkerLeave, pulid.Nil, "New leave case", 0),
					planned, append(wfOptions(leaveCaseFields...),
						toolpreview.Volatile(wfFieldRequestedAt))...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would open a %s leave case from %s, waiting for a decision.",
					planned.LeaveType, common.DayText(planned.StartsAt)), change), nil
			},
			Run: func(
				ctx context.Context,
				entity *worker.WorkerLeaveCase,
				params *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				created, err := cases.OpenCase(ctx, entity, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return wfResult("opened", kindLeaveCase, paramLeaveCaseID, created.ID,
					created.WorkerID), nil
			},
		})
}

func leaveCaseUpdateFrom(
	params *agentports.ToolExecuteParams,
) (*workerleaveservice.UpdateCaseRequest, error) {
	id, err := common.RequirePulid(params.Params, paramLeaveCaseID)
	if err != nil {
		return nil, err
	}
	req := &workerleaveservice.UpdateCaseRequest{
		TenantInfo: common.TenantFrom(*params),
		CaseID:     id,
		UserID:     params.Actor.UserID,
	}
	if kind, given, kindErr := common.OptionalEnum(params.Params, paramLeaveType,
		leaveTypes.Values); kindErr != nil {
		return nil, kindErr
	} else if given {
		req.LeaveType = &kind
	}
	if frequency, given, freqErr := common.OptionalEnum(params.Params, paramLeaveFrequency,
		leaveFrequencies.Values); freqErr != nil {
		return nil, freqErr
	} else if given {
		req.Frequency = &frequency
	}
	if req.Reason, err = common.OptionalBoundedText(
		params.Params,
		common.FieldReason,
		wfShortChars,
	); err != nil {
		return nil, err
	}
	if req.Notes, err = common.OptionalBoundedText(
		params.Params,
		common.FieldNotes,
		wfNoteChars,
	); err != nil {
		return nil, err
	}
	if req.MilitaryCaregiver, err = common.OptionalBoolPointer(params.Params,
		paramMilitaryCaregiver); err != nil {
		return nil, err
	}
	if req.StartsAt, err = common.OptionalScheduleDay(params.Params, paramStartsAt); err != nil {
		return nil, err
	}
	if req.EndsAt, err = common.OptionalScheduleDay(params.Params, paramEndsAt); err != nil {
		return nil, err
	}
	if req.DocumentID, err = common.OptionalID(params.Params, wfParamDocument); err != nil {
		return nil, err
	}
	hours, err := common.OptionalIntInRange(
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
) func(*workerleaveservice.CaseChange) (*agent.ToolPreview, error) {
	options := append(wfOptions(fields...), toolpreview.Volatile(volatile...))
	return func(change *workerleaveservice.CaseChange) (*agent.ToolPreview, error) {
		recorded, err := toolpreview.Changed(leaveCaseRecord(change.Before), change.Before,
			change.After, options...)
		if err != nil {
			return nil, err
		}
		return toolpreview.Build(summary, recorded), nil
	}
}

func newUpdateLeaveCaseTool(cases LeaveKeeper) agentports.AgentTool {
	properties := leaveCaseProperties()
	properties[paramLeaveCaseID] = leaveCaseIDProperty()
	properties[paramStartsAt] = agenttoolschema.Date("The first day of leave.")
	spec := targeting(withSchema(wfSpec(
		"update_leave_case",
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

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
		*workerleaveservice.UpdateCaseRequest, *workerleaveservice.CaseChange,
	]{
		Request: leaveCaseUpdateFrom,
		Plan: func(
			ctx context.Context,
			req *workerleaveservice.UpdateCaseRequest,
			_ *agentports.ToolExecuteParams,
		) (*workerleaveservice.CaseChange, error) {
			return cases.PlanUpdateCase(ctx, req)
		},
		Refused: func(*workerleaveservice.UpdateCaseRequest) string {
			return "Would correct a leave case."
		},
		Render: func(
			_ *workerleaveservice.UpdateCaseRequest,
			change *workerleaveservice.CaseChange,
		) (*agent.ToolPreview, error) {
			return render(change)
		},
		Run: func(
			ctx context.Context,
			req *workerleaveservice.UpdateCaseRequest,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := cases.UpdateCase(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("updated", kindLeaveCase, paramLeaveCaseID, updated.ID,
				updated.WorkerID), nil
		},
	})
}

func newCloseLeaveCaseTool(cases LeaveKeeper) agentports.AgentTool {
	spec := personOnly(targeting(withSchema(wfSpec(
		"close_leave_case",
		"Close a decided leave case once the leave is over. The days recorded against it "+
			"stay counted. A case still waiting for a decision is refused. Only a person's "+
			"approval runs it.",
		"Closes a leave case under the leave approval grant, so it runs as the person who "+
			"approves it; the days it drew down stay drawn down.",
		permission.ResourceWorkerLeave,
		permission.OpApprove,
	), map[string]any{paramLeaveCaseID: leaveCaseIDProperty()}, paramLeaveCaseID),
		paramLeaveCaseID, permission.ResourceWorkerLeave))
	spec.SearchTerms = []string{"returned from leave", "leave ended", "close fmla"}
	spec.Recipe = []string{"list_worker_leave_cases", "close_leave_case"}
	render := renderLeaveCaseChange("Would close the leave case.",
		[]string{wfFieldClosedAt}, common.FieldStatus, wfFieldClosedAt, "endsAt")

	return common.NewReportingReceivableTool(
		spec,
		common.ReceivablePlan[*recordDelete, *workerleaveservice.CaseChange]{
			Request: recordDeleteFrom(paramLeaveCaseID),
			Plan: func(
				ctx context.Context,
				req *recordDelete,
				_ *agentports.ToolExecuteParams,
			) (*workerleaveservice.CaseChange, error) {
				return cases.PlanCloseCase(ctx, req.tenant, req.id)
			},
			Refused: func(*recordDelete) string { return "Would close a leave case." },
			Render: func(_ *recordDelete, change *workerleaveservice.CaseChange) (*agent.ToolPreview, error) {
				return render(change)
			},
			Run: func(
				ctx context.Context,
				req *recordDelete,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				closed, err := cases.CloseCase(ctx, req.tenant, req.id, req.userID)
				if err != nil {
					return nil, err
				}
				return wfResult("closed", kindLeaveCase, paramLeaveCaseID, closed.ID,
					closed.WorkerID), nil
			},
		},
	)
}

func newRequestLeaveCertificationTool(cases LeaveKeeper) agentports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"request_leave_certification",
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
		[]string{wfFieldCertRequestedAt}, "certificationStatus",
		wfFieldCertRequestedAt, "certificationDueAt")

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
		*workerleaveservice.RequestCertificationRequest, *workerleaveservice.CaseChange,
	]{
		Request: func(
			params *agentports.ToolExecuteParams,
		) (*workerleaveservice.RequestCertificationRequest, error) {
			id, err := common.RequirePulid(params.Params, paramLeaveCaseID)
			if err != nil {
				return nil, err
			}
			due, err := common.OptionalScheduleDay(params.Params, paramCertificationDue)
			if err != nil {
				return nil, err
			}
			return &workerleaveservice.RequestCertificationRequest{
				TenantInfo: common.TenantFrom(*params),
				CaseID:     id,
				DueAt:      due,
				UserID:     params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *workerleaveservice.RequestCertificationRequest,
			_ *agentports.ToolExecuteParams,
		) (*workerleaveservice.CaseChange, error) {
			return cases.PlanRequestCertification(ctx, req)
		},
		Refused: func(*workerleaveservice.RequestCertificationRequest) string {
			return "Would record a certification request."
		},
		Render: func(
			_ *workerleaveservice.RequestCertificationRequest,
			change *workerleaveservice.CaseChange,
		) (*agent.ToolPreview, error) {
			return render(change)
		},
		Run: func(
			ctx context.Context,
			req *workerleaveservice.RequestCertificationRequest,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := cases.RequestCertification(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("requested certification on", kindLeaveCase, paramLeaveCaseID,
				updated.ID, updated.WorkerID), nil
		},
	})
}

func requireHours(params map[string]any) (decimal.Decimal, error) {
	hours, present, err := common.OptionalDecimal(params, paramLeaveHours)
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
	return append(wfOptions(leaveDayFields...), toolpreview.WithRefs(map[string]permission.Resource{
		paramPTOLink:  permission.ResourceWorkerPTO,
		"leaveCaseId": permission.ResourceWorkerLeave,
	}))
}

func newRecordLeaveDayTool(cases LeaveKeeper) agentports.AgentTool {
	spec := withSchema(wfSpec(
		"record_leave_day",
		"Record a day of leave taken against an open or approved leave case, with the "+
			"hours. Whether it counts against the FMLA entitlement follows the case as it "+
			"stands today.",
		"Records a day taken against a leave case inside Trenova; nothing is sent, and "+
			"update_leave_day or delete_leave_day corrects it.",
		permission.ResourceWorkerLeave,
		permission.OpCreate,
	), map[string]any{
		paramLeaveCaseID: leaveCaseIDProperty(),
		paramUsedOn:      agenttoolschema.Date("The day the leave was taken."),
		paramLeaveHours:  common.AmountProperty("Hours taken that day, such as 8 or 4.5."),
		paramPTOLink: agenttoolschema.RecordIDText(permission.ResourceWorkerPTO,
			"Paid time off booked for the same day, from list_time_off, when the leave "+
				"runs concurrently."),
		common.FieldNotes: wfNoteProperty("Anything the day should say."),
	}, paramLeaveCaseID, paramUsedOn, paramLeaveHours)

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
		*workerleaveservice.RecordDayRequest, *worker.WorkerLeaveEntry,
	]{
		Request: func(
			params *agentports.ToolExecuteParams,
		) (*workerleaveservice.RecordDayRequest, error) {
			caseID, err := common.RequirePulid(params.Params, paramLeaveCaseID)
			if err != nil {
				return nil, err
			}
			usedOn, err := common.RequireScheduleDay(params.Params, paramUsedOn)
			if err != nil {
				return nil, err
			}
			hours, err := requireHours(params.Params)
			if err != nil {
				return nil, err
			}
			ptoID, err := common.OptionalID(params.Params, paramPTOLink)
			if err != nil {
				return nil, err
			}
			notes, err := common.BoundedText(params.Params, common.FieldNotes, wfNoteChars)
			if err != nil {
				return nil, err
			}
			return &workerleaveservice.RecordDayRequest{
				TenantInfo: common.TenantFrom(*params),
				CaseID:     caseID,
				UsedOn:     usedOn,
				Hours:      hours,
				PTOID:      ptoID,
				Notes:      notes,
				UserID:     params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *workerleaveservice.RecordDayRequest,
			_ *agentports.ToolExecuteParams,
		) (*worker.WorkerLeaveEntry, error) {
			return cases.PlanRecordDay(ctx, req)
		},
		Refused: func(*workerleaveservice.RecordDayRequest) string {
			return "Would record a day of leave."
		},
		Render: func(
			_ *workerleaveservice.RecordDayRequest,
			planned *worker.WorkerLeaveEntry,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(
				wfRecord(permission.ResourceWorkerLeave, pulid.Nil,
					"Leave on "+common.DayText(planned.UsedOn), 0),
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
				planned.Hours.String(), common.DayText(planned.UsedOn), counts), change), nil
		},
		Run: func(
			ctx context.Context,
			req *workerleaveservice.RecordDayRequest,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := cases.RecordDay(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("recorded", kindLeaveDay, paramLeaveEntryID, created.ID,
				created.WorkerID), nil
		},
	})
}

func newUpdateLeaveDayTool(cases LeaveKeeper) agentports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"update_leave_day",
		"Correct a day of leave already recorded: the hours, whether it counts against the "+
			"entitlement, the time off it runs with, or its note. Give only what changes.",
		"Corrects a recorded day of leave inside Trenova; nothing is sent, and it is corrected "+
			"again the same way.",
		permission.ResourceWorkerLeave,
		permission.OpUpdate,
	), map[string]any{
		paramLeaveEntryID: leaveDayIDProperty(),
		paramLeaveHours:   common.AmountProperty("Hours taken that day, such as 8 or 4.5."),
		paramCounts: common.BooleanProperty("Whether this day counts against the FMLA " +
			"entitlement."),
		paramPTOLink: agenttoolschema.RecordIDText(
			permission.ResourceWorkerPTO,
			"Paid time off booked for the same day, from list_time_off.",
		),
		common.FieldNotes: wfNoteProperty("The day's note."),
	}, paramLeaveEntryID), paramLeaveEntryID, permission.ResourceWorkerLeave)

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
		*workerleaveservice.UpdateDayRequest, *workerleaveservice.EntryChange,
	]{
		Request: func(
			params *agentports.ToolExecuteParams,
		) (*workerleaveservice.UpdateDayRequest, error) {
			id, err := common.RequirePulid(params.Params, paramLeaveEntryID)
			if err != nil {
				return nil, err
			}
			req := &workerleaveservice.UpdateDayRequest{
				TenantInfo: common.TenantFrom(*params),
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
			if req.Counts, err = common.OptionalBoolPointer(
				params.Params,
				paramCounts,
			); err != nil {
				return nil, err
			}
			if req.PTOID, err = common.OptionalID(params.Params, paramPTOLink); err != nil {
				return nil, err
			}
			if req.Notes, err = common.OptionalBoundedText(params.Params, common.FieldNotes,
				wfNoteChars); err != nil {
				return nil, err
			}
			return req, nil
		},
		Plan: func(
			ctx context.Context,
			req *workerleaveservice.UpdateDayRequest,
			_ *agentports.ToolExecuteParams,
		) (*workerleaveservice.EntryChange, error) {
			return cases.PlanUpdateDay(ctx, req)
		},
		Refused: func(*workerleaveservice.UpdateDayRequest) string {
			return "Would correct a day of leave."
		},
		Render: func(
			_ *workerleaveservice.UpdateDayRequest,
			change *workerleaveservice.EntryChange,
		) (*agent.ToolPreview, error) {
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
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := cases.UpdateDay(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("updated", kindLeaveDay, paramLeaveEntryID, updated.ID,
				updated.WorkerID), nil
		},
	})
}

func newDeleteLeaveDayTool(cases LeaveKeeper) agentports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"delete_leave_day",
		"Remove a day of leave recorded in error, which gives its hours back to the "+
			"worker's entitlement.",
		"Removes a recorded day of leave; the audit trail keeps what was removed, and "+
			"record_leave_day records it again.",
		permission.ResourceWorkerLeave,
		permission.OpUpdate,
	), map[string]any{paramLeaveEntryID: leaveDayIDProperty()}, paramLeaveEntryID),
		paramLeaveEntryID, permission.ResourceWorkerLeave)
	spec.SearchTerms = []string{"remove leave day"}
	spec.MaxTier = agent.TierPropose
	spec.Reversible = false

	return common.NewReceivableTool(
		spec,
		common.ReceivablePlan[*recordDelete, *worker.WorkerLeaveEntry]{
			Request: recordDeleteFrom(paramLeaveEntryID),
			Plan: func(
				ctx context.Context,
				req *recordDelete,
				_ *agentports.ToolExecuteParams,
			) (*worker.WorkerLeaveEntry, error) {
				return cases.PlanDeleteDay(ctx, req.tenant, req.id)
			},
			Refused: func(*recordDelete) string { return "Would remove a day of leave." },
			Render: func(_ *recordDelete, entry *worker.WorkerLeaveEntry) (*agent.ToolPreview, error) {
				change, err := toolpreview.Delete(
					leaveDayRecord(entry),
					entry,
					leaveDayOptions()...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would remove %s hours of leave on %s and give them back to the entitlement.",
					entry.Hours.String(), common.DayText(entry.UsedOn)), change), nil
			},
			Run: func(
				ctx context.Context,
				req *recordDelete,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				return nil, cases.DeleteDay(ctx, req.tenant, req.id, req.userID)
			},
		},
	)
}

func provideOpenLeaveCaseTool(s *workerleaveservice.Service) agentports.AgentTool {
	return newOpenLeaveCaseTool(s)
}

func provideUpdateLeaveCaseTool(s *workerleaveservice.Service) agentports.AgentTool {
	return newUpdateLeaveCaseTool(s)
}

func provideCloseLeaveCaseTool(s *workerleaveservice.Service) agentports.AgentTool {
	return newCloseLeaveCaseTool(s)
}

func provideRequestLeaveCertificationTool(s *workerleaveservice.Service) agentports.AgentTool {
	return newRequestLeaveCertificationTool(s)
}

func provideRecordLeaveDayTool(s *workerleaveservice.Service) agentports.AgentTool {
	return newRecordLeaveDayTool(s)
}

func provideUpdateLeaveDayTool(s *workerleaveservice.Service) agentports.AgentTool {
	return newUpdateLeaveDayTool(s)
}

func provideDeleteLeaveDayTool(s *workerleaveservice.Service) agentports.AgentTool {
	return newDeleteLeaveDayTool(s)
}
