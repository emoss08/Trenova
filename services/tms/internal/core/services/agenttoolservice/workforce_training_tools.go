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
	"github.com/emoss08/trenova/internal/core/services/workertrainingservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	paramTrainingID      = "trainingRecordId"
	paramCourseID        = "courseId"
	paramCourseIDs       = "courseIds"
	paramTrainingWorkers = "workerIds"
	paramCompletedAt     = wfFieldCompletedAt
	paramScore           = "score"
	paramTrainingClose   = "action"
	kindTraining         = "training assignment"
	maxTrainingWorkers   = 50
	maxTrainingCourses   = 10
)

type trainingClose string

const (
	trainingWaive  = trainingClose("Waive")
	trainingCancel = trainingClose("Cancel")
)

var (
	trainingCloses = agenttoolschema.Source(
		"workerTraining.agentClose",
		[]trainingClose{trainingWaive, trainingCancel},
	)
	trainingFields = []string{
		wfFieldWorkerID, paramCourseID, fieldStatus, "assignedAt", "dueAt", wfFieldCompletedAt,
		wfFieldExpiresAt, paramScore, "passed", wfFieldDocument, wfFieldNotes, "waivedReason",
	}
)

type trainingKeeper interface {
	PlanAssign(
		ctx context.Context,
		req *workertrainingservice.AssignRequest,
	) (*worker.WorkerTrainingRecord, error)
	Assign(
		ctx context.Context,
		req *workertrainingservice.AssignRequest,
	) (*worker.WorkerTrainingRecord, error)
	PlanBulkAssign(
		ctx context.Context,
		req *workertrainingservice.BulkAssignRequest,
	) (*workertrainingservice.BulkAssignPlan, error)
	BulkAssign(
		ctx context.Context,
		req *workertrainingservice.BulkAssignRequest,
	) (*workertrainingservice.BulkAssignResult, error)
	PlanAssignRequired(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) ([]*worker.TrainingCourse, error)
	AssignRequired(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		userID pulid.ID,
	) ([]*worker.WorkerTrainingRecord, error)
	PlanComplete(
		ctx context.Context,
		req *workertrainingservice.CompleteRequest,
	) (*workertrainingservice.RecordChange, error)
	Complete(
		ctx context.Context,
		req *workertrainingservice.CompleteRequest,
	) (*worker.WorkerTrainingRecord, error)
	PlanWaive(
		ctx context.Context,
		req *workertrainingservice.StatusRequest,
	) (*workertrainingservice.RecordChange, error)
	Waive(
		ctx context.Context,
		req *workertrainingservice.StatusRequest,
	) (*worker.WorkerTrainingRecord, error)
	PlanCancel(
		ctx context.Context,
		req *workertrainingservice.StatusRequest,
	) (*workertrainingservice.RecordChange, error)
	Cancel(
		ctx context.Context,
		req *workertrainingservice.StatusRequest,
	) (*worker.WorkerTrainingRecord, error)
	PlanAttachDocument(
		ctx context.Context,
		req *workertrainingservice.AttachDocumentRequest,
	) (*workertrainingservice.RecordChange, error)
	AttachDocument(
		ctx context.Context,
		req *workertrainingservice.AttachDocumentRequest,
	) (*worker.WorkerTrainingRecord, error)
}

var _ trainingKeeper = (*workertrainingservice.Service)(nil)

func trainingToolProviders() []any {
	return []any{
		provideAssignWorkerTrainingTool,
		provideAssignRequiredWorkerTrainingTool,
		provideRecordTrainingCompletionTool,
		provideAttachWorkerTrainingDocumentTool,
		provideCloseWorkerTrainingTool,
	}
}

func trainingIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceWorkerTraining, "The training assignment",
		"list_worker_training")
}

func trainingRecord(record *worker.WorkerTrainingRecord) toolpreview.Record {
	label := "Training"
	if record.Course != nil {
		label = record.Course.Name
	}
	return wfRecord(permission.ResourceWorkerTraining, record.ID, label, record.Version)
}

func trainingOptions(fields ...string) []toolpreview.Option {
	return append(wfOptions(fields...), toolpreview.Volatile("assignedAt"))
}

type trainingAssignment struct {
	single *workertrainingservice.AssignRequest
	bulk   *workertrainingservice.BulkAssignRequest
}

func trainingAssignmentFrom(params *serviceports.ToolExecuteParams) (*trainingAssignment, error) {
	workers, err := requirePulidSlice(params.Params, paramTrainingWorkers, maxTrainingWorkers)
	if err != nil {
		return nil, err
	}
	courses, err := requirePulidSlice(params.Params, paramCourseIDs, maxTrainingCourses)
	if err != nil {
		return nil, err
	}
	due, err := optionalScheduleDay(params.Params, wfParamDueDate)
	if err != nil {
		return nil, err
	}
	notes, err := boundedText(params.Params, fieldNotes, wfNoteChars)
	if err != nil {
		return nil, err
	}
	if len(workers) == 1 && len(courses) == 1 {
		return &trainingAssignment{single: &workertrainingservice.AssignRequest{
			TenantInfo: tenantFrom(*params),
			WorkerID:   workers[0],
			CourseID:   courses[0],
			DueAt:      due,
			Notes:      notes,
			UserID:     params.Actor.UserID,
		}}, nil
	}
	return &trainingAssignment{bulk: &workertrainingservice.BulkAssignRequest{
		TenantInfo: tenantFrom(*params),
		WorkerIDs:  workers,
		CourseIDs:  courses,
		DueAt:      due,
		Notes:      notes,
		UserID:     params.Actor.UserID,
	}}, nil
}

func (a *trainingAssignment) plan(
	ctx context.Context,
	training trainingKeeper,
) (*workertrainingservice.BulkAssignPlan, error) {
	if a.single != nil {
		record, err := training.PlanAssign(ctx, a.single)
		if err != nil {
			return nil, err
		}
		return &workertrainingservice.BulkAssignPlan{
			Records: []*worker.WorkerTrainingRecord{record},
		}, nil
	}
	return training.PlanBulkAssign(ctx, a.bulk)
}

func renderTrainingAssignment(
	_ *trainingAssignment,
	plan *workertrainingservice.BulkAssignPlan,
) (*agent.ToolPreview, error) {
	changes := make([]*agent.RecordChange, 0, len(plan.Records))
	for _, record := range plan.Records {
		change, err := toolpreview.Create(
			wfRecord(permission.ResourceWorkerTraining, pulid.Nil, trainingRecord(record).Label, 0),
			record, trainingOptions(wfFieldWorkerID, paramCourseID, fieldStatus, "dueAt",
				wfFieldNotes)...)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return toolpreview.Build(fmt.Sprintf(
		"Would assign %d course(s); the driver sees each in Dash. %d already open, %d refused.",
		len(plan.Records), len(plan.Skipped), len(plan.Failed)), changes...), nil
}

func newAssignWorkerTrainingTool(training trainingKeeper) serviceports.AgentTool {
	spec := withSchema(wfSpec(
		"assign_worker_training",
		"Assign one or more training courses to one or more workers, due on the day given "+
			"or on the course's own schedule. A course a worker already has open is left as it "+
			"is. The driver sees each assignment in Dash.",
		"Assigns courses the driver sees in Dash; cancel_worker_training withdraws one.",
		permission.ResourceWorkerTraining,
		permission.OpAssign,
	), map[string]any{
		paramTrainingWorkers: agenttoolschema.RecordIDs(
			permission.ResourceWorker,
			"The workers, from search_worker or "+
				"list_workers.",
			maxTrainingWorkers,
		),
		paramCourseIDs: agenttoolschema.IDList("The courses, from list_training_courses.",
			maxTrainingCourses),
		wfParamDueDate: agenttoolschema.Date("When it is due, when not the course's own schedule."),
		fieldNotes:     wfNoteProperty("Anything the assignment should say."),
	}, paramTrainingWorkers, paramCourseIDs)
	spec.egress = agent.EgressDriverVisible
	spec.artifact = ""
	spec.searchTerms = []string{"assign course", "training", "orientation"}

	return newReceivableTool(spec, receivablePlan[
		*trainingAssignment, *workertrainingservice.BulkAssignPlan,
	]{
		request: trainingAssignmentFrom,
		plan: func(
			ctx context.Context,
			assignment *trainingAssignment,
			_ *serviceports.ToolExecuteParams,
		) (*workertrainingservice.BulkAssignPlan, error) {
			plan, err := assignment.plan(ctx, training)
			if err != nil {
				return nil, err
			}
			if len(plan.Records) == 0 && len(plan.Skipped) == 0 && len(plan.Failed) > 0 {
				return nil, errortypes.NewValidationError(paramCourseIDs, errortypes.ErrInvalid,
					"None of these courses could be assigned: {0}", plan.Failed[0].Error)
			}
			return plan, nil
		},
		refused: func(*trainingAssignment) string { return "Would assign training." },
		render:  renderTrainingAssignment,
		run: func(
			ctx context.Context,
			assignment *trainingAssignment,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if assignment.single != nil {
				_, err := training.Assign(ctx, assignment.single)
				return nil, err
			}
			_, err := training.BulkAssign(ctx, assignment.bulk)
			return nil, err
		},
	})
}

type requiredTraining struct {
	workerID pulid.ID
	tenant   pagination.TenantInfo
	userID   pulid.ID
}

func newAssignRequiredWorkerTrainingTool(training trainingKeeper) serviceports.AgentTool {
	spec := withSchema(wfSpec(
		"assign_required_worker_training",
		"Assign a worker every course the organization requires of them that they are "+
			"missing, have failed or have let lapse. Courses already open are left alone. The "+
			"driver sees each assignment in Dash.",
		"Assigns the courses the worker's driver type requires; the driver sees them in Dash, "+
			"and cancel_worker_training withdraws one.",
		permission.ResourceWorkerTraining,
		permission.OpAssign,
	), map[string]any{paramWorkerID: workerProperty()}, paramWorkerID)
	spec.egress = agent.EgressDriverVisible

	return newReportingReceivableTool(
		spec,
		receivablePlan[*requiredTraining, []*worker.TrainingCourse]{
			request: func(params *serviceports.ToolExecuteParams) (*requiredTraining, error) {
				workerID, err := requirePulid(params.Params, paramWorkerID)
				if err != nil {
					return nil, err
				}
				return &requiredTraining{workerID: workerID, tenant: tenantFrom(*params),
					userID: params.Actor.UserID}, nil
			},
			plan: func(
				ctx context.Context,
				req *requiredTraining,
				_ *serviceports.ToolExecuteParams,
			) ([]*worker.TrainingCourse, error) {
				return training.PlanAssignRequired(ctx, req.tenant, req.workerID)
			},
			refused: func(*requiredTraining) string {
				return "Would assign the worker their required training."
			},
			render: func(req *requiredTraining, courses []*worker.TrainingCourse) (*agent.ToolPreview, error) {
				changes := make([]*agent.RecordChange, 0, len(courses))
				for _, course := range courses {
					change, err := toolpreview.Create(
						wfRecord(permission.ResourceWorkerTraining, pulid.Nil, course.Name, 0),
						&worker.WorkerTrainingRecord{
							WorkerID: req.workerID,
							CourseID: course.ID,
							Status:   worker.TrainingStatusAssigned,
						}, wfOptions(wfFieldWorkerID, paramCourseID, fieldStatus)...)
					if err != nil {
						return nil, err
					}
					changes = append(changes, change)
				}
				if len(courses) == 0 {
					return toolpreview.Build("The worker is missing no required course; " +
						"nothing would be assigned."), nil
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would assign %d required course(s); the driver sees each in Dash.",
					len(courses)), changes...), nil
			},
			run: func(
				ctx context.Context,
				req *requiredTraining,
				_ *serviceports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				if _, err := training.AssignRequired(ctx, req.tenant, req.workerID,
					req.userID); err != nil {
					return nil, err
				}
				return &agent.ToolExecutionResult{
					Action: "assigned required training to",
					Kind:   "worker",
					IDs:    map[string]string{paramWorkerID: req.workerID.String()},
					Record: recordOf(workerRecordEntity, req.workerID),
				}, nil
			},
		},
	)
}

func completionFrom(
	params *serviceports.ToolExecuteParams,
) (*workertrainingservice.CompleteRequest, error) {
	req := &workertrainingservice.CompleteRequest{
		TenantInfo: tenantFrom(*params),
		UserID:     params.Actor.UserID,
	}
	var err error
	if req.ID, err = optionalID(params.Params, paramTrainingID); err != nil {
		return nil, err
	}
	if req.WorkerID, err = optionalID(params.Params, paramWorkerID); err != nil {
		return nil, err
	}
	if req.CourseID, err = optionalID(params.Params, paramCourseID); err != nil {
		return nil, err
	}
	if req.ID.IsNil() == (req.WorkerID.IsNil() || req.CourseID.IsNil()) {
		return nil, fmt.Errorf("name either %q, or %q with %q", paramTrainingID,
			paramWorkerID, paramCourseID)
	}
	completed, err := optionalScheduleDay(params.Params, paramCompletedAt)
	if err != nil {
		return nil, err
	}
	if completed != nil {
		req.CompletedAt = *completed
	}
	score, present, err := optionalDecimal(params.Params, paramScore)
	if err != nil {
		return nil, err
	}
	if present {
		req.Score = decimal.NewNullDecimal(score)
	}
	if req.DocumentID, err = optionalID(params.Params, wfParamDocument); err != nil {
		return nil, err
	}
	if req.Notes, err = boundedText(params.Params, fieldNotes, wfNoteChars); err != nil {
		return nil, err
	}
	return req, nil
}

func newRecordTrainingCompletionTool(training trainingKeeper) serviceports.AgentTool {
	spec := withSchema(wfSpec(
		"record_training_completion",
		"Record that a worker finished a training course. Give the open assignment's id, "+
			"or the worker and course directly for a classroom session recorded after the "+
			"fact. A scored course needs the score and fails below its passing mark; a pass sets "+
			"when a recurring course next expires.",
		"Records a completion on the worker's training record inside Trenova; nothing is "+
			"sent, and a completion recorded in error is corrected by a person.",
		permission.ResourceWorkerTraining,
		permission.OpUpdate,
	), map[string]any{
		paramTrainingID: trainingIDProperty(),
		paramWorkerID:   workerProperty(),
		paramCourseID:   agenttoolschema.IDText("The course, from list_training_courses."),
		paramCompletedAt: agenttoolschema.Date(
			"The day it was finished. Defaults to today; never a " +
				"day to come.",
		),
		paramScore:      amountProperty("The score, such as 92, for a scored course."),
		wfParamDocument: wfDocumentProperty(),
		fieldNotes:      wfNoteProperty("Anything the record should say."),
	})
	spec.target = func(params map[string]any) (serviceports.ToolTarget, bool) {
		return targetOf(params, paramTrainingID, permission.ResourceWorkerTraining)
	}
	spec.reversible = false

	spec.searchTerms = []string{
		"completed training", "course completed", "driver finished training",
	}

	return newReportingReceivableTool(spec, receivablePlan[
		*workertrainingservice.CompleteRequest, *workertrainingservice.RecordChange,
	]{
		request: completionFrom,
		plan: func(
			ctx context.Context,
			req *workertrainingservice.CompleteRequest,
			_ *serviceports.ToolExecuteParams,
		) (*workertrainingservice.RecordChange, error) {
			return training.PlanComplete(ctx, req)
		},
		refused: func(*workertrainingservice.CompleteRequest) string {
			return "Would record a training completion."
		},
		render: func(
			_ *workertrainingservice.CompleteRequest,
			change *workertrainingservice.RecordChange,
		) (*agent.ToolPreview, error) {
			var recorded *agent.RecordChange
			var err error
			if change.Before == nil {
				recorded, err = toolpreview.Create(
					wfRecord(permission.ResourceWorkerTraining, pulid.Nil,
						trainingRecord(change.After).Label, 0),
					change.After, trainingOptions(trainingFields...)...)
			} else {
				recorded, err = toolpreview.Changed(trainingRecord(change.Before), change.Before,
					change.After, trainingOptions(trainingFields...)...)
			}
			if err != nil {
				return nil, err
			}
			outcome := "a pass"
			if change.After.Status == worker.TrainingStatusFailed {
				outcome = "a fail below the passing mark"
			}
			return toolpreview.Build(fmt.Sprintf("Would record %s as %s.",
				trainingRecord(change.After).Label, outcome), recorded), nil
		},
		run: func(
			ctx context.Context,
			req *workertrainingservice.CompleteRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			saved, err := training.Complete(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("recorded", kindTraining, paramTrainingID, saved.ID,
				saved.WorkerID), nil
		},
	})
}

func newAttachWorkerTrainingDocumentTool(training trainingKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"attach_worker_training_document",
		"Attach a certificate or sign-in sheet already filed on the worker to one of their "+
			"training assignments.",
		"Links a filed document to a training record inside Trenova; attaching another "+
			"replaces it.",
		permission.ResourceWorkerTraining,
		permission.OpUpdate,
	), map[string]any{
		paramTrainingID: trainingIDProperty(),
		wfParamDocument: wfDocumentProperty(),
	}, paramTrainingID, wfParamDocument), paramTrainingID, permission.ResourceWorkerTraining)

	return newReportingReceivableTool(spec, receivablePlan[
		*workertrainingservice.AttachDocumentRequest, *workertrainingservice.RecordChange,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*workertrainingservice.AttachDocumentRequest, error) {
			id, err := requirePulid(params.Params, paramTrainingID)
			if err != nil {
				return nil, err
			}
			documentID, err := requirePulid(params.Params, wfParamDocument)
			if err != nil {
				return nil, err
			}
			return &workertrainingservice.AttachDocumentRequest{
				ID:         id,
				DocumentID: documentID,
				TenantInfo: tenantFrom(*params),
				UserID:     params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *workertrainingservice.AttachDocumentRequest,
			_ *serviceports.ToolExecuteParams,
		) (*workertrainingservice.RecordChange, error) {
			return training.PlanAttachDocument(ctx, req)
		},
		refused: func(*workertrainingservice.AttachDocumentRequest) string {
			return "Would attach a document to a training assignment."
		},
		render: func(
			_ *workertrainingservice.AttachDocumentRequest,
			change *workertrainingservice.RecordChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(trainingRecord(change.Before), change.Before,
				change.After, wfOptions(wfFieldDocument)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build("Would attach the document to "+
				trainingRecord(change.Before).Label+".", recorded), nil
		},
		run: func(
			ctx context.Context,
			req *workertrainingservice.AttachDocumentRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			saved, err := training.AttachDocument(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("attached a document to", kindTraining, paramTrainingID, saved.ID,
				saved.WorkerID), nil
		},
	})
}

type trainingClosing struct {
	close trainingClose
	req   *workertrainingservice.StatusRequest
}

func newCloseWorkerTrainingTool(training trainingKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"close_worker_training",
		"Close an open training assignment without a completion. Waive satisfies the "+
			"requirement for a stated reason, such as an equivalent certificate on file; "+
			"Cancel withdraws an assignment that should never have been made.",
		"Closes a training assignment inside Trenova; the reason stays on the record, and "+
			"the course is assigned again the same way.",
		permission.ResourceWorkerTraining,
		permission.OpCancel,
	), map[string]any{
		paramTrainingID:    trainingIDProperty(),
		paramTrainingClose: agenttoolschema.Enum("Waive or Cancel.", trainingCloses),
		fieldReason: stringProperty("Why. Required to waive; what the record will "+
			"say.", wfShortChars),
	}, paramTrainingID, paramTrainingClose), paramTrainingID, permission.ResourceWorkerTraining)
	spec.searchTerms = []string{"retired course", "close without completion"}
	spec.maxTier = agent.TierPropose

	return newReportingReceivableTool(spec, receivablePlan[
		*trainingClosing, *workertrainingservice.RecordChange,
	]{
		request: func(params *serviceports.ToolExecuteParams) (*trainingClosing, error) {
			id, err := requirePulid(params.Params, paramTrainingID)
			if err != nil {
				return nil, err
			}
			action, err := requireEnum(params.Params, paramTrainingClose, trainingCloses.Values)
			if err != nil {
				return nil, err
			}
			reason, err := boundedText(params.Params, fieldReason, wfShortChars)
			if err != nil {
				return nil, err
			}
			return &trainingClosing{close: action, req: &workertrainingservice.StatusRequest{
				ID:         id,
				TenantInfo: tenantFrom(*params),
				Reason:     reason,
				UserID:     params.Actor.UserID,
			}}, nil
		},
		plan: func(
			ctx context.Context,
			closing *trainingClosing,
			_ *serviceports.ToolExecuteParams,
		) (*workertrainingservice.RecordChange, error) {
			if closing.close == trainingWaive {
				return training.PlanWaive(ctx, closing.req)
			}
			return training.PlanCancel(ctx, closing.req)
		},
		refused: func(closing *trainingClosing) string {
			return fmt.Sprintf("Would %s a training assignment.",
				strings.ToLower(string(closing.close)))
		},
		render: func(
			closing *trainingClosing,
			change *workertrainingservice.RecordChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(trainingRecord(change.Before), change.Before,
				change.After, wfOptions(fieldStatus, "waivedReason", wfFieldNotes)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would %s %s.",
				strings.ToLower(string(closing.close)), trainingRecord(change.Before).Label),
				recorded), nil
		},
		run: func(
			ctx context.Context,
			closing *trainingClosing,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			var saved *worker.WorkerTrainingRecord
			var err error
			action := "waived"
			if closing.close == trainingWaive {
				saved, err = training.Waive(ctx, closing.req)
			} else {
				action = "cancelled"
				saved, err = training.Cancel(ctx, closing.req)
			}
			if err != nil {
				return nil, err
			}
			return wfResult(action, kindTraining, paramTrainingID, saved.ID, saved.WorkerID), nil
		},
	})
}

func provideAssignWorkerTrainingTool(s *workertrainingservice.Service) serviceports.AgentTool {
	return newAssignWorkerTrainingTool(s)
}

func provideAssignRequiredWorkerTrainingTool(
	s *workertrainingservice.Service,
) serviceports.AgentTool {
	return newAssignRequiredWorkerTrainingTool(s)
}

func provideRecordTrainingCompletionTool(
	s *workertrainingservice.Service,
) serviceports.AgentTool {
	return newRecordTrainingCompletionTool(s)
}

func provideAttachWorkerTrainingDocumentTool(
	s *workertrainingservice.Service,
) serviceports.AgentTool {
	return newAttachWorkerTrainingDocumentTool(s)
}

func provideCloseWorkerTrainingTool(s *workertrainingservice.Service) serviceports.AgentTool {
	return newCloseWorkerTrainingTool(s)
}
