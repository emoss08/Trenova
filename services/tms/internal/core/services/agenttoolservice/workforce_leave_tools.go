package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/internal/core/services/workerleaveservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	paramLeaveCaseID       = "leaveCaseId"
	paramLeaveEntryID      = "leaveDayId"
	paramLeaveType         = "leaveType"
	paramLeaveFrequency    = "frequency"
	paramMilitaryCaregiver = "militaryCaregiver"
	paramStartsAt          = "startsAt"
	paramEndsAt            = "endsAt"
	paramRequestedAt       = wfFieldRequestedAt
	paramEligibilityHours  = "eligibilityHoursWorked"
	paramUsedOn            = "usedOn"
	paramLeaveHours        = "hours"
	paramCounts            = "countsAgainstEntitlement"
	paramPTOLink           = "ptoId"
	paramCertificationDue  = "dueDate"
	kindLeaveCase          = "leave case"
	kindLeaveDay           = "day of leave"
	maxEligibilityHours    = 10000
)

var (
	leaveTypes       = agenttoolschema.Source("worker.leaveType", worker.LeaveTypeValues())
	leaveFrequencies = agenttoolschema.Source(
		"worker.leaveFrequency",
		worker.LeaveFrequencyValues(),
	)
	leaveCaseFields = []string{
		wfFieldWorkerID, paramLeaveType, fieldStatus, paramLeaveFrequency, fieldReason,
		paramMilitaryCaregiver, wfFieldRequestedAt, paramStartsAt, paramEndsAt, wfFieldClosedAt,
		"certificationStatus", wfFieldCertRequestedAt, "certificationDueAt",
		paramEligibilityHours, wfFieldDocument, wfFieldNotes,
	}
	leaveDayFields = []string{
		wfFieldWorkerID, paramLeaveCaseID, paramUsedOn, paramLeaveHours, paramCounts, paramPTOLink,
		wfFieldNotes,
	}
)

type leaveKeeper interface {
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

var _ leaveKeeper = (*workerleaveservice.Service)(nil)

func leaveToolProviders() []any {
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
	return idProperty("The leave case, from list_worker_leave_cases. Never guess one.")
}

func leaveDayIDProperty() map[string]any {
	return idProperty("The day of leave, from list_worker_leave_cases. Never guess one.")
}

func leaveCaseRecord(entity *worker.WorkerLeaveCase) toolpreview.Record {
	return wfRecord(permission.ResourceWorkerLeave, entity.ID,
		fmt.Sprintf("%s leave from %s", entity.LeaveType, dayText(entity.StartsAt)),
		entity.Version)
}

func leaveDayRecord(entry *worker.WorkerLeaveEntry) toolpreview.Record {
	return wfRecord(permission.ResourceWorkerLeave, entry.ID,
		"Leave on "+dayText(entry.UsedOn), entry.Version)
}

func leaveCaseProperties() map[string]any {
	return map[string]any{
		paramLeaveType:      agenttoolschema.Enum("The kind of leave.", leaveTypes),
		paramLeaveFrequency: agenttoolschema.Enum("How the leave is taken.", leaveFrequencies),
		fieldReason: stringProperty("Why, as the worker gave it. Keep medical detail out; "+
			"the certification carries that.", wfShortChars),
		paramMilitaryCaregiver: booleanProperty("Whether it is leave to care for a covered " +
			"servicemember."),
		paramEndsAt: dayProperty("The last day of leave, when it is known."),
		paramEligibilityHours: integerProperty("Hours worked in the twelve months before, "+
			"for eligibility.", 0, maxEligibilityHours),
		wfParamDocument: wfDocumentProperty(),
		fieldNotes:      wfNoteProperty("Anything the case should say."),
	}
}

func applyLeaveCaseOpen(entity *worker.WorkerLeaveCase, params map[string]any) error {
	var err error
	if kind, given, kindErr := optionalEnum(params, paramLeaveType,
		leaveTypes.Values); kindErr != nil {
		return kindErr
	} else if given {
		entity.LeaveType = kind
	}
	if frequency, given, freqErr := optionalEnum(params, paramLeaveFrequency,
		leaveFrequencies.Values); freqErr != nil {
		return freqErr
	} else if given {
		entity.Frequency = frequency
	}
	if entity.Reason, err = boundedText(params, fieldReason, wfShortChars); err != nil {
		return err
	}
	if entity.MilitaryCaregiver, err = optionalBoolParam(params, paramMilitaryCaregiver,
		false); err != nil {
		return err
	}
	if entity.EndsAt, err = optionalScheduleDay(params, paramEndsAt); err != nil {
		return err
	}
	if entity.DocumentID, err = optionalID(params, wfParamDocument); err != nil {
		return err
	}
	if entity.Notes, err = boundedText(params, fieldNotes, wfNoteChars); err != nil {
		return err
	}
	hours, err := optionalIntInRange(params, paramEligibilityHours, 0, maxEligibilityHours)
	if err != nil {
		return err
	}
	if hours != nil {
		value := int32(*hours) //nolint:gosec // bounded by the schema's maximum
		entity.EligibilityHoursWorked = &value
	}
	return nil
}

func newOpenLeaveCaseTool(cases leaveKeeper) serviceports.AgentTool {
	properties := leaveCaseProperties()
	properties[paramWorkerID] = workerProperty()
	properties[paramStartsAt] = dayProperty("The first day of leave.")
	properties[paramRequestedAt] = dayProperty("The day the worker asked. Defaults to today.")
	spec := withSchema(wfSpec(
		"open_leave_case",
		"Open a leave case for a worker who asked for FMLA, medical, military, parental or "+
			"personal leave, from the day it starts. The case waits for a decision a person "+
			"makes; the days taken are recorded against it with record_leave_day.",
		"Files a pending leave request inside Trenova; nothing is decided or sent, and the "+
			"case is corrected with update_leave_case.",
		permission.ResourceWorkerLeave,
		permission.OpCreate,
	), properties, paramWorkerID, paramStartsAt)
	spec.searchTerms = []string{"FMLA", "leave of absence", "medical leave"}

	return newReportingReceivableTool(spec,
		receivablePlan[*worker.WorkerLeaveCase, *worker.WorkerLeaveCase]{
			request: func(params *serviceports.ToolExecuteParams) (*worker.WorkerLeaveCase, error) {
				workerID, err := requirePulid(params.Params, paramWorkerID)
				if err != nil {
					return nil, err
				}
				starts, err := requireScheduleDay(params.Params, paramStartsAt)
				if err != nil {
					return nil, err
				}
				requested, err := optionalScheduleDay(params.Params, paramRequestedAt)
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
			plan: func(
				ctx context.Context,
				entity *worker.WorkerLeaveCase,
				params *serviceports.ToolExecuteParams,
			) (*worker.WorkerLeaveCase, error) {
				return cases.PlanOpenCase(ctx, entity, params.Actor.UserID)
			},
			refused: func(*worker.WorkerLeaveCase) string { return "Would open a leave case." },
			render: func(
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
					planned.LeaveType, dayText(planned.StartsAt)), change), nil
			},
			run: func(
				ctx context.Context,
				entity *worker.WorkerLeaveCase,
				params *serviceports.ToolExecuteParams,
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
	params *serviceports.ToolExecuteParams,
) (*workerleaveservice.UpdateCaseRequest, error) {
	id, err := requirePulid(params.Params, paramLeaveCaseID)
	if err != nil {
		return nil, err
	}
	req := &workerleaveservice.UpdateCaseRequest{
		TenantInfo: tenantFrom(*params),
		CaseID:     id,
		UserID:     params.Actor.UserID,
	}
	if kind, given, kindErr := optionalEnum(params.Params, paramLeaveType,
		leaveTypes.Values); kindErr != nil {
		return nil, kindErr
	} else if given {
		req.LeaveType = &kind
	}
	if frequency, given, freqErr := optionalEnum(params.Params, paramLeaveFrequency,
		leaveFrequencies.Values); freqErr != nil {
		return nil, freqErr
	} else if given {
		req.Frequency = &frequency
	}
	if req.Reason, err = optionalBoundedText(params.Params, fieldReason, wfShortChars); err != nil {
		return nil, err
	}
	if req.Notes, err = optionalBoundedText(params.Params, fieldNotes, wfNoteChars); err != nil {
		return nil, err
	}
	if req.MilitaryCaregiver, err = optionalBoolPointer(params.Params,
		paramMilitaryCaregiver); err != nil {
		return nil, err
	}
	if req.StartsAt, err = optionalScheduleDay(params.Params, paramStartsAt); err != nil {
		return nil, err
	}
	if req.EndsAt, err = optionalScheduleDay(params.Params, paramEndsAt); err != nil {
		return nil, err
	}
	if req.DocumentID, err = optionalID(params.Params, wfParamDocument); err != nil {
		return nil, err
	}
	hours, err := optionalIntInRange(params.Params, paramEligibilityHours, 0, maxEligibilityHours)
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

func newUpdateLeaveCaseTool(cases leaveKeeper) serviceports.AgentTool {
	properties := leaveCaseProperties()
	properties[paramLeaveCaseID] = leaveCaseIDProperty()
	properties[paramStartsAt] = dayProperty("The first day of leave.")
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
	render := renderLeaveCaseChange("Would correct the leave case.", nil, leaveCaseFields...)

	return newReportingReceivableTool(spec, receivablePlan[
		*workerleaveservice.UpdateCaseRequest, *workerleaveservice.CaseChange,
	]{
		request: leaveCaseUpdateFrom,
		plan: func(
			ctx context.Context,
			req *workerleaveservice.UpdateCaseRequest,
			_ *serviceports.ToolExecuteParams,
		) (*workerleaveservice.CaseChange, error) {
			return cases.PlanUpdateCase(ctx, req)
		},
		refused: func(*workerleaveservice.UpdateCaseRequest) string {
			return "Would correct a leave case."
		},
		render: func(
			_ *workerleaveservice.UpdateCaseRequest,
			change *workerleaveservice.CaseChange,
		) (*agent.ToolPreview, error) {
			return render(change)
		},
		run: func(
			ctx context.Context,
			req *workerleaveservice.UpdateCaseRequest,
			_ *serviceports.ToolExecuteParams,
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

func newCloseLeaveCaseTool(cases leaveKeeper) serviceports.AgentTool {
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
	render := renderLeaveCaseChange("Would close the leave case.",
		[]string{wfFieldClosedAt}, fieldStatus, wfFieldClosedAt, "endsAt")

	return newReportingReceivableTool(
		spec,
		receivablePlan[*recordDelete, *workerleaveservice.CaseChange]{
			request: recordDeleteFrom(paramLeaveCaseID),
			plan: func(
				ctx context.Context,
				req *recordDelete,
				_ *serviceports.ToolExecuteParams,
			) (*workerleaveservice.CaseChange, error) {
				return cases.PlanCloseCase(ctx, req.tenant, req.id)
			},
			refused: func(*recordDelete) string { return "Would close a leave case." },
			render: func(_ *recordDelete, change *workerleaveservice.CaseChange) (*agent.ToolPreview, error) {
				return render(change)
			},
			run: func(
				ctx context.Context,
				req *recordDelete,
				_ *serviceports.ToolExecuteParams,
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

func newRequestLeaveCertificationTool(cases leaveKeeper) serviceports.AgentTool {
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
		paramCertificationDue: dayProperty("When it is due, when not the usual window."),
	}, paramLeaveCaseID), paramLeaveCaseID, permission.ResourceWorkerLeave)
	render := renderLeaveCaseChange("Would record that certification was requested.",
		[]string{wfFieldCertRequestedAt}, "certificationStatus",
		wfFieldCertRequestedAt, "certificationDueAt")

	return newReportingReceivableTool(spec, receivablePlan[
		*workerleaveservice.RequestCertificationRequest, *workerleaveservice.CaseChange,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*workerleaveservice.RequestCertificationRequest, error) {
			id, err := requirePulid(params.Params, paramLeaveCaseID)
			if err != nil {
				return nil, err
			}
			due, err := optionalScheduleDay(params.Params, paramCertificationDue)
			if err != nil {
				return nil, err
			}
			return &workerleaveservice.RequestCertificationRequest{
				TenantInfo: tenantFrom(*params),
				CaseID:     id,
				DueAt:      due,
				UserID:     params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *workerleaveservice.RequestCertificationRequest,
			_ *serviceports.ToolExecuteParams,
		) (*workerleaveservice.CaseChange, error) {
			return cases.PlanRequestCertification(ctx, req)
		},
		refused: func(*workerleaveservice.RequestCertificationRequest) string {
			return "Would record a certification request."
		},
		render: func(
			_ *workerleaveservice.RequestCertificationRequest,
			change *workerleaveservice.CaseChange,
		) (*agent.ToolPreview, error) {
			return render(change)
		},
		run: func(
			ctx context.Context,
			req *workerleaveservice.RequestCertificationRequest,
			_ *serviceports.ToolExecuteParams,
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
	hours, present, err := optionalDecimal(params, paramLeaveHours)
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

func newRecordLeaveDayTool(cases leaveKeeper) serviceports.AgentTool {
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
		paramUsedOn:      dayProperty("The day the leave was taken."),
		paramLeaveHours:  amountProperty("Hours taken that day, such as 8 or 4.5."),
		paramPTOLink: idProperty("Paid time off booked for the same day, from " +
			"list_time_off, when the leave runs concurrently."),
		fieldNotes: wfNoteProperty("Anything the day should say."),
	}, paramLeaveCaseID, paramUsedOn, paramLeaveHours)

	return newReportingReceivableTool(spec, receivablePlan[
		*workerleaveservice.RecordDayRequest, *worker.WorkerLeaveEntry,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*workerleaveservice.RecordDayRequest, error) {
			caseID, err := requirePulid(params.Params, paramLeaveCaseID)
			if err != nil {
				return nil, err
			}
			usedOn, err := requireScheduleDay(params.Params, paramUsedOn)
			if err != nil {
				return nil, err
			}
			hours, err := requireHours(params.Params)
			if err != nil {
				return nil, err
			}
			ptoID, err := optionalID(params.Params, paramPTOLink)
			if err != nil {
				return nil, err
			}
			notes, err := boundedText(params.Params, fieldNotes, wfNoteChars)
			if err != nil {
				return nil, err
			}
			return &workerleaveservice.RecordDayRequest{
				TenantInfo: tenantFrom(*params),
				CaseID:     caseID,
				UsedOn:     usedOn,
				Hours:      hours,
				PTOID:      ptoID,
				Notes:      notes,
				UserID:     params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *workerleaveservice.RecordDayRequest,
			_ *serviceports.ToolExecuteParams,
		) (*worker.WorkerLeaveEntry, error) {
			return cases.PlanRecordDay(ctx, req)
		},
		refused: func(*workerleaveservice.RecordDayRequest) string {
			return "Would record a day of leave."
		},
		render: func(
			_ *workerleaveservice.RecordDayRequest,
			planned *worker.WorkerLeaveEntry,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(
				wfRecord(permission.ResourceWorkerLeave, pulid.Nil,
					"Leave on "+dayText(planned.UsedOn), 0),
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
				planned.Hours.String(), dayText(planned.UsedOn), counts), change), nil
		},
		run: func(
			ctx context.Context,
			req *workerleaveservice.RecordDayRequest,
			_ *serviceports.ToolExecuteParams,
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

func newUpdateLeaveDayTool(cases leaveKeeper) serviceports.AgentTool {
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
		paramLeaveHours:   amountProperty("Hours taken that day, such as 8 or 4.5."),
		paramCounts: booleanProperty("Whether this day counts against the FMLA " +
			"entitlement."),
		paramPTOLink: idProperty("Paid time off booked for the same day, from list_time_off."),
		fieldNotes:   wfNoteProperty("The day's note."),
	}, paramLeaveEntryID), paramLeaveEntryID, permission.ResourceWorkerLeave)

	return newReportingReceivableTool(spec, receivablePlan[
		*workerleaveservice.UpdateDayRequest, *workerleaveservice.EntryChange,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*workerleaveservice.UpdateDayRequest, error) {
			id, err := requirePulid(params.Params, paramLeaveEntryID)
			if err != nil {
				return nil, err
			}
			req := &workerleaveservice.UpdateDayRequest{
				TenantInfo: tenantFrom(*params),
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
			if req.Counts, err = optionalBoolPointer(params.Params, paramCounts); err != nil {
				return nil, err
			}
			if req.PTOID, err = optionalID(params.Params, paramPTOLink); err != nil {
				return nil, err
			}
			if req.Notes, err = optionalBoundedText(params.Params, fieldNotes,
				wfNoteChars); err != nil {
				return nil, err
			}
			return req, nil
		},
		plan: func(
			ctx context.Context,
			req *workerleaveservice.UpdateDayRequest,
			_ *serviceports.ToolExecuteParams,
		) (*workerleaveservice.EntryChange, error) {
			return cases.PlanUpdateDay(ctx, req)
		},
		refused: func(*workerleaveservice.UpdateDayRequest) string {
			return "Would correct a day of leave."
		},
		render: func(
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
		run: func(
			ctx context.Context,
			req *workerleaveservice.UpdateDayRequest,
			_ *serviceports.ToolExecuteParams,
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

func newDeleteLeaveDayTool(cases leaveKeeper) serviceports.AgentTool {
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
	spec.maxTier = agent.TierPropose
	spec.reversible = false

	return newReceivableTool(spec, receivablePlan[*recordDelete, *worker.WorkerLeaveEntry]{
		request: recordDeleteFrom(paramLeaveEntryID),
		plan: func(
			ctx context.Context,
			req *recordDelete,
			_ *serviceports.ToolExecuteParams,
		) (*worker.WorkerLeaveEntry, error) {
			return cases.PlanDeleteDay(ctx, req.tenant, req.id)
		},
		refused: func(*recordDelete) string { return "Would remove a day of leave." },
		render: func(_ *recordDelete, entry *worker.WorkerLeaveEntry) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(leaveDayRecord(entry), entry, leaveDayOptions()...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would remove %s hours of leave on %s and give them back to the entitlement.",
				entry.Hours.String(), dayText(entry.UsedOn)), change), nil
		},
		run: func(
			ctx context.Context,
			req *recordDelete,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return nil, cases.DeleteDay(ctx, req.tenant, req.id, req.userID)
		},
	})
}

func provideOpenLeaveCaseTool(s *workerleaveservice.Service) serviceports.AgentTool {
	return newOpenLeaveCaseTool(s)
}

func provideUpdateLeaveCaseTool(s *workerleaveservice.Service) serviceports.AgentTool {
	return newUpdateLeaveCaseTool(s)
}

func provideCloseLeaveCaseTool(s *workerleaveservice.Service) serviceports.AgentTool {
	return newCloseLeaveCaseTool(s)
}

func provideRequestLeaveCertificationTool(s *workerleaveservice.Service) serviceports.AgentTool {
	return newRequestLeaveCertificationTool(s)
}

func provideRecordLeaveDayTool(s *workerleaveservice.Service) serviceports.AgentTool {
	return newRecordLeaveDayTool(s)
}

func provideUpdateLeaveDayTool(s *workerleaveservice.Service) serviceports.AgentTool {
	return newUpdateLeaveDayTool(s)
}

func provideDeleteLeaveDayTool(s *workerleaveservice.Service) serviceports.AgentTool {
	return newDeleteLeaveDayTool(s)
}
