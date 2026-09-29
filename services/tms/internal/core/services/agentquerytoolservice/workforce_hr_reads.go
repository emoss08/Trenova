package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/performancereviewservice"
	"github.com/emoss08/trenova/internal/core/services/workerchecklistservice"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/internal/core/services/workerdqfservice"
	"github.com/emoss08/trenova/internal/core/services/workerinjuryservice"
	"github.com/emoss08/trenova/internal/core/services/workerleaveservice"
	"github.com/emoss08/trenova/internal/core/services/workertrainingservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxCatalogRows       = 100
	activeStatus         = "Active"
	minCaseYear          = 2000
	maxCaseYear          = 2100
	paramCaseYear        = "caseYear"
	paramRecordable      = "recordableOnly"
	paramOutstanding     = "outstandingOnly"
	paramCertOutstanding = "certificationOutstandingOnly"
	paramIncludeArchived = "includeArchived"
)

type wfLeaveReader interface {
	ListCases(
		ctx context.Context,
		req *repositories.ListLeaveCasesRequest,
	) ([]*worker.WorkerLeaveCase, error)
}

type wfTrainingReader interface {
	ListForWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		includeClosed bool,
	) ([]*worker.WorkerTrainingRecord, error)
	ListCourses(
		ctx context.Context,
		req *repositories.ListTrainingCoursesRequest,
	) (*pagination.CursorListResult[*worker.TrainingCourse], error)
}

type wfChecklistReader interface {
	ListForWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		includeClosed bool,
	) ([]*worker.WorkerChecklist, error)
	ListTemplates(
		ctx context.Context,
		req *repositories.ListChecklistTemplatesRequest,
	) (*pagination.CursorListResult[*worker.WorkerChecklistTemplate], error)
}

type wfReviewReader interface {
	ListReviews(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		statuses []worker.ReviewStatus,
	) ([]*worker.PerformanceReview, error)
	ListTemplates(
		ctx context.Context,
		req *repositories.ListReviewTemplatesRequest,
	) (*pagination.CursorListResult[*worker.PerformanceReviewTemplate], error)
}

type wfVerificationReader interface {
	ListVerifications(
		ctx context.Context,
		req *repositories.ListEmploymentVerificationsRequest,
	) ([]*worker.WorkerEmploymentVerification, error)
}

type wfCredentialReader interface {
	ListForWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		includeArchived bool,
	) ([]*worker.WorkerCredential, error)
	ListTypes(
		ctx context.Context,
		req *repositories.ListCredentialTypesRequest,
	) (*pagination.CursorListResult[*worker.WorkerCredentialType], error)
}

type wfInjuryReader interface {
	ListInjuries(
		ctx context.Context,
		req *repositories.ListWorkerInjuriesRequest,
	) ([]*worker.WorkerInjury, error)
}

var (
	_ wfLeaveReader        = (*workerleaveservice.Service)(nil)
	_ wfTrainingReader     = (*workertrainingservice.Service)(nil)
	_ wfChecklistReader    = (*workerchecklistservice.Service)(nil)
	_ wfReviewReader       = (*performancereviewservice.Service)(nil)
	_ wfVerificationReader = (*workerdqfservice.Service)(nil)
	_ wfCredentialReader   = (*workercredentialservice.Service)(nil)
	_ wfInjuryReader       = (*workerinjuryservice.Service)(nil)
)

func catalogFilter(params *serviceports.QueryToolParams) *pagination.QueryOptions {
	return &pagination.QueryOptions{
		TenantInfo: tenantOf(params),
		Pagination: pagination.Info{Limit: maxCatalogRows},
	}
}

type leaveDayRow struct {
	ID                       string       `json:"id"`
	UsedOn                   optionalDate `json:"usedOn"`
	Hours                    string       `json:"hours"`
	CountsAgainstEntitlement bool         `json:"countsAgainstEntitlement"`
	PTOID                    string       `json:"ptoId,omitempty"`
}

type leaveCaseRow struct {
	ID                  string        `json:"id"`
	WorkerID            string        `json:"workerId"`
	Worker              string        `json:"worker,omitempty"`
	LeaveType           string        `json:"leaveType"`
	Status              string        `json:"status"`
	Frequency           string        `json:"frequency,omitempty"`
	FMLADesignated      bool          `json:"fmlaDesignated"`
	Reason              string        `json:"reason,omitempty"`
	RequestedAt         optionalDate  `json:"requestedAt"`
	StartsAt            optionalDate  `json:"startsAt"`
	EndsAt              optionalDate  `json:"endsAt"`
	CertificationStatus string        `json:"certificationStatus,omitempty"`
	CertificationDueAt  optionalDate  `json:"certificationDueAt"`
	Days                []leaveDayRow `json:"days,omitempty"`
}

func newListWorkerLeaveCasesTool(
	cases wfLeaveReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &workforceRead{
		name: "list_worker_leave_cases",
		description: "List FMLA, medical, military, parental and personal leave cases. " +
			"Cases for one worker or the whole organization come with their status, " +
			"certification and each day taken against them. It yields the leaveCaseId and leaveDayId the " +
			"leave tools take.",
		resource: permission.ResourceWorkerLeave,
		properties: map[string]any{
			paramWorkerID:   wfWorkerProperty("Narrow to one worker"),
			wfParamOpenOnly: wfBoolProperty("Only cases still pending or approved."),
			paramCertOutstanding: wfBoolProperty("Only cases still waiting on the " +
				"certification the worker owes."),
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
			found, err := cases.ListCases(ctx, &repositories.ListLeaveCasesRequest{
				TenantInfo:                   tenantOf(params),
				WorkerID:                     workerID,
				OpenOnly:                     optionalBool(params.Params, wfParamOpenOnly),
				CertificationOutstandingOnly: optionalBool(params.Params, paramCertOutstanding),
				IncludeWorker:                true,
				IncludeEntries:               true,
				Limit:                        receivableLimit(params.Params),
			})
			if err != nil {
				return nil, err
			}
			rows := make([]leaveCaseRow, 0, len(found))
			for _, leave := range found {
				row := leaveCaseRow{
					ID:                  leave.ID.String(),
					WorkerID:            leave.WorkerID.String(),
					Worker:              workerName(leave.Worker),
					LeaveType:           string(leave.LeaveType),
					Status:              string(leave.Status),
					Frequency:           string(leave.Frequency),
					FMLADesignated:      leave.FMLADesignated,
					Reason:              gatedText(gate, wfFieldReason, leave.Reason),
					RequestedAt:         recordedDate(leave.RequestedAt),
					StartsAt:            recordedDate(leave.StartsAt),
					EndsAt:              expectedDate(derefInt64(leave.EndsAt), absentOpenEnded),
					CertificationStatus: string(leave.CertificationStatus),
					CertificationDueAt: expectedDate(
						derefInt64(leave.CertificationDueAt),
						absentNotDue,
					),
					Days: make([]leaveDayRow, 0, len(leave.Entries)),
				}
				for _, entry := range leave.Entries {
					row.Days = append(row.Days, leaveDayRow{
						ID:                       entry.ID.String(),
						UsedOn:                   recordedDate(entry.UsedOn),
						Hours:                    entry.Hours.String(),
						CountsAgainstEntitlement: entry.CountsAgainstEntitlement,
						PTOID:                    pulidString(entry.PTOID),
					})
				}
				rows = append(rows, row)
			}
			return newReceivableList(rows, gate), nil
		},
	}
}

type trainingRow struct {
	ID          string       `json:"id"`
	CourseID    string       `json:"courseId"`
	Course      string       `json:"course,omitempty"`
	Status      string       `json:"status"`
	AssignedAt  optionalDate `json:"assignedAt"`
	DueAt       optionalDate `json:"dueAt"`
	CompletedAt optionalDate `json:"completedAt"`
	ExpiresAt   optionalDate `json:"expiresAt"`
	Score       string       `json:"score,omitempty"`
	Passed      string       `json:"passed,omitempty"`
}

func newListWorkerTrainingTool(
	training wfTrainingReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &workforceRead{
		name: "list_worker_training",
		description: "List one worker's training assignments with each course, its " +
			"status, due date, completion, score and when it expires. It yields the " +
			"trainingRecordId the training tools take.",
		resource: permission.ResourceWorkerTraining,
		properties: map[string]any{
			paramWorkerID: wfWorkerProperty("The worker"),
			wfParamClosed: wfBoolProperty("Also return completed, waived, cancelled and " +
				"expired assignments. Defaults to open ones only."),
		},
		required: []string{paramWorkerID},
		access:   newFieldAccess(permissions),
		run: func(
			ctx context.Context,
			params *serviceports.QueryToolParams,
			gate *fieldGate,
		) (any, error) {
			workerID, err := requirePulid(params.Params, paramWorkerID)
			if err != nil {
				return nil, err
			}
			records, err := training.ListForWorker(ctx, tenantOf(params), workerID,
				optionalBool(params.Params, wfParamClosed))
			if err != nil {
				return nil, err
			}
			rows := make([]trainingRow, 0, len(records))
			for _, record := range records {
				row := trainingRow{
					ID:          record.ID.String(),
					CourseID:    record.CourseID.String(),
					Status:      string(record.Status),
					AssignedAt:  recordedDate(record.AssignedAt),
					DueAt:       expectedDate(derefInt64(record.DueAt), absentNotDue),
					CompletedAt: expectedDate(derefInt64(record.CompletedAt), absentNotCompleted),
					ExpiresAt:   expectedDate(derefInt64(record.ExpiresAt), absentNotExpiring),
					Score:       decimalText(record.Score),
					Passed:      boolRef(record.Passed),
				}
				if record.Course != nil {
					row.Course = record.Course.Name
				}
				rows = append(rows, row)
			}
			return newReceivableList(rows, gate), nil
		},
	}
}

type trainingCourseRow struct {
	ID             string `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Delivery       string `json:"delivery"`
	PassingScore   string `json:"passingScore,omitempty"`
	ValidityMonths *int32 `json:"validityMonths,omitempty"`
	IsRequired     bool   `json:"isRequired"`
	DueDays        int32  `json:"dueDaysAfterAssignment"`
}

func newListTrainingCoursesTool(
	training wfTrainingReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &workforceRead{
		name: "list_training_courses",
		description: "List the organization's active training courses with their category, " +
			"delivery, passing score, how long a pass lasts and whether they are required. " +
			"It yields the courseId the training tools take.",
		resource:   permission.ResourceTrainingCourse,
		properties: map[string]any{},
		access:     newFieldAccess(permissions),
		run: func(
			ctx context.Context,
			params *serviceports.QueryToolParams,
			gate *fieldGate,
		) (any, error) {
			courses, err := training.ListCourses(ctx, &repositories.ListTrainingCoursesRequest{
				Filter: catalogFilter(params),
				Cursor: pagination.CursorInfo{Limit: maxCatalogRows},
				Status: activeStatus,
			})
			if err != nil {
				return nil, err
			}
			rows := make([]trainingCourseRow, 0, len(courses.Items))
			for _, course := range courses.Items {
				rows = append(rows, trainingCourseRow{
					ID:             course.ID.String(),
					Code:           course.Code,
					Name:           course.Name,
					Category:       string(course.Category),
					Delivery:       string(course.Delivery),
					PassingScore:   decimalText(course.PassingScore),
					ValidityMonths: course.ValidityMonths,
					IsRequired:     course.IsRequired,
					DueDays:        course.DueDaysAfterAssignment,
				})
			}
			return newReceivableList(rows, gate), nil
		},
	}
}

type checklistItemRow struct {
	ID          string       `json:"id"`
	Label       string       `json:"label"`
	Kind        string       `json:"kind"`
	Required    bool         `json:"required"`
	Status      string       `json:"status"`
	DueAt       optionalDate `json:"dueAt"`
	CompletedAt optionalDate `json:"completedAt"`
	Note        string       `json:"note,omitempty"`
}

type checklistRow struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Kind      string             `json:"kind"`
	Status    string             `json:"status"`
	StartedAt optionalDate       `json:"startedAt"`
	DueAt     optionalDate       `json:"dueAt"`
	Items     []checklistItemRow `json:"items"`
}

type checklistTemplateRow struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Trigger string `json:"trigger"`
}

type workerChecklists struct {
	Checklists []checklistRow         `json:"checklists"`
	Templates  []checklistTemplateRow `json:"templatesStartedByHand,omitempty"`
	Withheld   []string               `json:"withheldByAccess,omitempty"`
}

func newListWorkerChecklistsTool(
	checklists wfChecklistReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	access := newFieldAccess(permissions)
	return &workforceRead{
		name: "list_worker_checklists",
		description: "List one worker's onboarding, offboarding and other checklists " +
			"with every item and where it stands, and the templates a person can start by " +
			"hand. It yields the checklistId, checklistItemId and templateId the checklist " +
			"tools take.",
		resource: permission.ResourceWorkerChecklist,
		properties: map[string]any{
			paramWorkerID: wfWorkerProperty("The worker"),
			wfParamClosed: wfBoolProperty("Also return completed and cancelled checklists."),
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
			found, err := checklists.ListForWorker(ctx, tenantOf(params), workerID,
				optionalBool(params.Params, wfParamClosed))
			if err != nil {
				return nil, err
			}
			result := &workerChecklists{Checklists: make([]checklistRow, 0, len(found))}
			for _, checklist := range found {
				row := checklistRow{
					ID:        checklist.ID.String(),
					Name:      checklist.Name,
					Kind:      string(checklist.Kind),
					Status:    string(checklist.Status),
					StartedAt: recordedDate(checklist.StartedAt),
					DueAt:     expectedDate(derefInt64(checklist.DueAt), absentNotDue),
					Items:     make([]checklistItemRow, 0, len(checklist.Items)),
				}
				for _, item := range checklist.Items {
					row.Items = append(row.Items, checklistItemRow{
						ID:          item.ID.String(),
						Label:       item.Label,
						Kind:        string(item.Kind),
						Required:    item.Required,
						Status:      string(item.Status),
						DueAt:       expectedDate(derefInt64(item.DueAt), absentNotDue),
						CompletedAt: expectedDate(derefInt64(item.CompletedAt), absentNotCompleted),
						Note:        gatedText(gate, "note", item.Note),
					})
				}
				result.Checklists = append(result.Checklists, row)
			}
			if access.mayRead(ctx, params, permission.ResourceWorkerChecklistTemplate) {
				templates, tmplErr := checklists.ListTemplates(ctx,
					&repositories.ListChecklistTemplatesRequest{
						Filter:  catalogFilter(params),
						Cursor:  pagination.CursorInfo{Limit: maxCatalogRows},
						Status:  activeStatus,
						Trigger: string(worker.ChecklistTriggerManual),
					})
				if tmplErr != nil {
					return nil, tmplErr
				}
				result.Templates = make([]checklistTemplateRow, 0, len(templates.Items))
				for _, template := range templates.Items {
					result.Templates = append(result.Templates, checklistTemplateRow{
						ID:      template.ID.String(),
						Code:    template.Code,
						Name:    template.Name,
						Kind:    string(template.Kind),
						Trigger: string(template.Trigger),
					})
				}
			}
			result.Withheld = gate.Withheld()
			return result, nil
		},
	}
}

type reviewRatingRow struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Weight  int32  `json:"weight"`
	Score   *int32 `json:"score"`
	Comment string `json:"comment,omitempty"`
}

type reviewRow struct {
	ID           string            `json:"id"`
	TemplateID   string            `json:"templateId"`
	Title        string            `json:"title"`
	Status       string            `json:"status"`
	PeriodStart  optionalDate      `json:"periodStart"`
	PeriodEnd    optionalDate      `json:"periodEnd"`
	OverallScore string            `json:"overallScore,omitempty"`
	Ratings      []reviewRatingRow `json:"ratings"`
	Summary      string            `json:"summary,omitempty"`
	Goals        []string          `json:"goals,omitempty"`
}

type reviewTemplateRow struct {
	ID            string `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	IsDefault     bool   `json:"isDefault"`
	CadenceMonths *int32 `json:"cadenceMonths,omitempty"`
}

type workerReviews struct {
	Reviews   []reviewRow         `json:"reviews"`
	Templates []reviewTemplateRow `json:"templates,omitempty"`
	Withheld  []string            `json:"withheldByAccess,omitempty"`
}

func newListPerformanceReviewsTool(
	reviews wfReviewReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	access := newFieldAccess(permissions)
	return &workforceRead{
		name: "list_performance_reviews",
		description: "List one worker's performance reviews with each item's key, score " +
			"and comment, the summary and goals, and the review templates in use. It " +
			"yields the reviewId and templateId the review tools take, and the rating " +
			"keys draft_performance_review scores.",
		resource: permission.ResourcePerformanceReview,
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
			found, err := reviews.ListReviews(ctx, tenantOf(params), workerID, nil)
			if err != nil {
				return nil, err
			}
			result := &workerReviews{Reviews: make([]reviewRow, 0, len(found))}
			for _, review := range found {
				row := reviewRow{
					ID:           review.ID.String(),
					TemplateID:   review.TemplateID.String(),
					Title:        review.Title,
					Status:       string(review.Status),
					PeriodStart:  recordedDate(review.PeriodStart),
					PeriodEnd:    recordedDate(review.PeriodEnd),
					OverallScore: decimalText(review.OverallScore),
					Ratings:      make([]reviewRatingRow, 0, len(review.Ratings)),
					Summary:      gatedText(gate, "summary", review.Summary),
				}
				for _, rating := range review.Ratings {
					row.Ratings = append(row.Ratings, reviewRatingRow{
						Key:     rating.Key,
						Label:   rating.Label,
						Weight:  rating.Weight,
						Score:   rating.Score,
						Comment: gatedText(gate, "ratings", rating.Comment),
					})
				}
				for _, goal := range review.Goals {
					row.Goals = append(row.Goals, goal.Title+" ("+string(goal.Status)+")")
				}
				result.Reviews = append(result.Reviews, row)
			}
			if access.mayRead(ctx, params, permission.ResourcePerformanceReviewTemplate) {
				templates, tmplErr := reviews.ListTemplates(ctx,
					&repositories.ListReviewTemplatesRequest{
						Filter: catalogFilter(params),
						Cursor: pagination.CursorInfo{Limit: maxCatalogRows},
						Status: activeStatus,
					})
				if tmplErr != nil {
					return nil, tmplErr
				}
				result.Templates = make([]reviewTemplateRow, 0, len(templates.Items))
				for _, template := range templates.Items {
					result.Templates = append(result.Templates, reviewTemplateRow{
						ID:            template.ID.String(),
						Code:          template.Code,
						Name:          template.Name,
						IsDefault:     template.IsDefault,
						CadenceMonths: template.CadenceMonths,
					})
				}
			}
			result.Withheld = gate.Withheld()
			return result, nil
		},
	}
}

type verificationRow struct {
	ID                       string       `json:"id"`
	WorkerID                 string       `json:"workerId"`
	Worker                   string       `json:"worker,omitempty"`
	EmployerName             string       `json:"employerName"`
	EmployerDOTNumber        string       `json:"employerDotNumber,omitempty"`
	EmployedFrom             optionalDate `json:"employedFrom"`
	EmployedTo               optionalDate `json:"employedTo"`
	WasDOTRegulated          bool         `json:"wasDotRegulated"`
	Status                   string       `json:"status"`
	Method                   string       `json:"method,omitempty"`
	RequestedAt              optionalDate `json:"requestedAt"`
	ResponseReceivedAt       optionalDate `json:"responseReceivedAt"`
	FollowUpCount            int32        `json:"followUpCount"`
	HadAccidents             bool         `json:"hadAccidents"`
	AccidentCount            int32        `json:"accidentCount"`
	HadDrugAlcoholViolations bool         `json:"hadDrugAlcoholViolations"`
	Findings                 string       `json:"findings,omitempty"`
}

func newListEmploymentVerificationsTool(
	verifications wfVerificationReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &workforceRead{
		name: "list_employment_verifications",
		description: "List the previous employers on drivers' qualification files. Each " +
			"shows where its safety performance history request stands: asked when, " +
			"followed up how often, and what came back. It yields the verificationId the " +
			"verification tools take.",
		resource: permission.ResourceQualification,
		properties: map[string]any{
			paramWorkerID:    wfWorkerProperty("Narrow to one driver"),
			paramOutstanding: wfBoolProperty("Only employers who have not answered yet."),
			paramLimit:       limitProperty(),
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
			found, err := verifications.ListVerifications(ctx,
				&repositories.ListEmploymentVerificationsRequest{
					TenantInfo:      tenantOf(params),
					WorkerID:        workerID,
					OutstandingOnly: optionalBool(params.Params, paramOutstanding),
					IncludeWorker:   true,
					Limit:           receivableLimit(params.Params),
				})
			if err != nil {
				return nil, err
			}
			rows := make([]verificationRow, 0, len(found))
			for _, entry := range found {
				rows = append(rows, verificationRow{
					ID:                entry.ID.String(),
					WorkerID:          entry.WorkerID.String(),
					Worker:            workerName(entry.Worker),
					EmployerName:      entry.EmployerName,
					EmployerDOTNumber: entry.EmployerDOTNumber,
					EmployedFrom:      pointerDate(entry.EmployedFrom),
					EmployedTo:        pointerDate(entry.EmployedTo),
					WasDOTRegulated:   entry.WasDOTRegulated,
					Status:            string(entry.Status),
					Method:            string(entry.Method),
					RequestedAt: expectedDate(
						derefInt64(entry.RequestedAt),
						absentNotRequested,
					),
					ResponseReceivedAt: expectedDate(
						derefInt64(entry.ResponseReceivedAt),
						absentNotReceived,
					),
					FollowUpCount:            entry.FollowUpCount,
					HadAccidents:             entry.HadAccidents,
					AccidentCount:            entry.AccidentCount,
					HadDrugAlcoholViolations: entry.HadDrugAlcoholViolations,
					Findings:                 gatedText(gate, "findings", entry.Findings),
				})
			}
			return newReceivableList(rows, gate), nil
		},
	}
}

type credentialRow struct {
	ID               string       `json:"id"`
	CredentialTypeID string       `json:"credentialTypeId"`
	CredentialType   string       `json:"credentialType,omitempty"`
	Status           string       `json:"status"`
	Number           string       `json:"number,omitempty"`
	IssuingAuthority string       `json:"issuingAuthority,omitempty"`
	IssuedAt         optionalDate `json:"issuedAt"`
	ExpiresAt        optionalDate `json:"expiresAt"`
	Verified         bool         `json:"verified"`
	HasDocument      bool         `json:"hasDocument"`
}

type credentialTypeRow struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	IsRequired bool   `json:"isRequired"`
}

type workerCredentials struct {
	Credentials []credentialRow     `json:"credentials"`
	Types       []credentialTypeRow `json:"credentialTypes,omitempty"`
	Withheld    []string            `json:"withheldByAccess,omitempty"`
}

func newListWorkerCredentialsTool(
	credentials wfCredentialReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	access := newFieldAccess(permissions)
	return &workforceRead{
		name: "list_worker_credentials",
		description: "List one worker's licences, medical cards, endorsements and other " +
			"credentials. Each has its dates and whether it is verified; the credential " +
			"types the organization tracks come with them. It yields the credentialId and " +
			"credentialTypeId the credential tools take.",
		resource: permission.ResourceWorkerCredential,
		properties: map[string]any{
			paramWorkerID:        wfWorkerProperty("The worker"),
			paramIncludeArchived: wfBoolProperty("Also return archived credentials."),
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
			found, err := credentials.ListForWorker(ctx, tenantOf(params), workerID,
				optionalBool(params.Params, paramIncludeArchived))
			if err != nil {
				return nil, err
			}
			result := &workerCredentials{Credentials: make([]credentialRow, 0, len(found))}
			for _, credential := range found {
				row := credentialRow{
					ID:               credential.ID.String(),
					CredentialTypeID: credential.CredentialTypeID.String(),
					Status:           string(credential.Status),
					Number:           gatedText(gate, "number", credential.Number),
					IssuingAuthority: credential.IssuingAuthority,
					IssuedAt:         pointerDate(credential.IssuedAt),
					ExpiresAt: expectedDate(
						derefInt64(credential.ExpiresAt),
						absentNotExpiring,
					),
					Verified:    credential.IsVerified(),
					HasDocument: credential.DocumentID.IsNotNil(),
				}
				if credential.CredentialType != nil {
					row.CredentialType = credential.CredentialType.Name
				}
				result.Credentials = append(result.Credentials, row)
			}
			if access.mayRead(ctx, params, permission.ResourceWorkerCredentialType) {
				types, typeErr := credentials.ListTypes(
					ctx,
					&repositories.ListCredentialTypesRequest{
						Filter: catalogFilter(params),
						Cursor: pagination.CursorInfo{Limit: maxCatalogRows},
						Status: activeStatus,
					},
				)
				if typeErr != nil {
					return nil, typeErr
				}
				result.Types = make([]credentialTypeRow, 0, len(types.Items))
				for _, kind := range types.Items {
					result.Types = append(result.Types, credentialTypeRow{
						ID:         kind.ID.String(),
						Code:       kind.Code,
						Name:       kind.Name,
						Category:   string(kind.Category),
						IsRequired: kind.IsRequired,
					})
				}
			}
			result.Withheld = gate.Withheld()
			return result, nil
		},
	}
}

type injuryRow struct {
	ID               string       `json:"id"`
	WorkerID         string       `json:"workerId"`
	Worker           string       `json:"worker,omitempty"`
	CaseNumber       int32        `json:"caseNumber"`
	CaseYear         int16        `json:"caseYear"`
	Classification   string       `json:"classification"`
	IllnessType      string       `json:"illnessType"`
	Treatment        string       `json:"treatment,omitempty"`
	Status           string       `json:"status"`
	OccurredAt       optionalDate `json:"occurredAt"`
	ReturnedToWorkAt optionalDate `json:"returnedToWorkAt"`
	Description      string       `json:"description,omitempty"`
	BodyPart         string       `json:"bodyPart,omitempty"`
	DaysAway         int32        `json:"daysAway"`
	DaysRestricted   int32        `json:"daysRestricted"`
	PrivacyCase      bool         `json:"privacyCase"`
	ClaimStatus      string       `json:"claimStatus,omitempty"`
}

func newListWorkerInjuriesTool(
	injuries wfInjuryReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &workforceRead{
		name: "list_worker_injuries",
		description: "List injury and illness cases on the OSHA log, for one worker or a " +
			"year, with the classification, days away or restricted and the workers' comp " +
			"claim. Medical detail follows the injury permission. It yields the injuryId " +
			"the injury tools take.",
		resource: permission.ResourceWorkerInjury,
		properties: map[string]any{
			paramWorkerID: wfWorkerProperty("Narrow to one worker"),
			paramCaseYear: map[string]any{
				toolschema.KeyType:        toolschema.TypeInteger,
				toolschema.KeyDescription: "Narrow to one calendar year's log, such as 2026.",
				toolschema.KeyMinimum:     minCaseYear,
				toolschema.KeyMaximum:     maxCaseYear,
			},
			paramRecordable: wfBoolProperty("Only the cases that belong on the 300 log."),
			wfParamOpenOnly: wfBoolProperty("Only cases still open."),
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
			year := optionalInt(params.Params, paramCaseYear, 0)
			if year != 0 && (year < minCaseYear || year > maxCaseYear) {
				return nil, fmt.Errorf("parameter %q must be a year from %d to %d",
					paramCaseYear, minCaseYear, maxCaseYear)
			}
			caseYear := int16(year) //nolint:gosec // checked against the year range above
			found, err := injuries.ListInjuries(ctx, &repositories.ListWorkerInjuriesRequest{
				TenantInfo:     tenantOf(params),
				WorkerID:       workerID,
				CaseYear:       caseYear,
				RecordableOnly: optionalBool(params.Params, paramRecordable),
				OpenOnly:       optionalBool(params.Params, wfParamOpenOnly),
				IncludeWorker:  true,
			})
			if err != nil {
				return nil, err
			}
			rows := make([]injuryRow, 0, len(found))
			for _, injury := range found {
				row := injuryRow{
					ID:             injury.ID.String(),
					WorkerID:       injury.WorkerID.String(),
					CaseNumber:     injury.CaseNumber,
					CaseYear:       injury.CaseYear,
					Classification: string(injury.Classification),
					IllnessType:    string(injury.IllnessType),
					Status:         string(injury.Status),
					OccurredAt:     recordedDate(injury.OccurredAt),
					ReturnedToWorkAt: expectedDate(
						derefInt64(injury.ReturnedToWorkAt),
						absentNotReturned,
					),
					DaysAway:       injury.DaysAway,
					DaysRestricted: injury.DaysRestricted,
					PrivacyCase:    injury.PrivacyCase,
					ClaimStatus:    string(injury.ClaimStatus),
					Treatment:      gatedText(gate, "treatment", string(injury.Treatment)),
					Description:    gatedText(gate, wfFieldDescription, injury.Description),
					BodyPart:       gatedText(gate, "bodyPart", injury.BodyPart),
				}
				if !injury.PrivacyCase {
					row.Worker = workerName(injury.Worker)
				}
				rows = append(rows, row)
			}
			return newReceivableList(rows, gate), nil
		},
	}
}

func provideListWorkerLeaveCasesTool(
	s *workerleaveservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListWorkerLeaveCasesTool(s, permissions)
}

func provideListWorkerTrainingTool(
	s *workertrainingservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListWorkerTrainingTool(s, permissions)
}

func provideListTrainingCoursesTool(
	s *workertrainingservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListTrainingCoursesTool(s, permissions)
}

func provideListWorkerChecklistsTool(
	s *workerchecklistservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListWorkerChecklistsTool(s, permissions)
}

func provideListPerformanceReviewsTool(
	s *performancereviewservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListPerformanceReviewsTool(s, permissions)
}

func provideListEmploymentVerificationsTool(
	s *workerdqfservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListEmploymentVerificationsTool(s, permissions)
}

func provideListWorkerCredentialsTool(
	s *workercredentialservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListWorkerCredentialsTool(s, permissions)
}

func provideListWorkerInjuriesTool(
	s *workerinjuryservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListWorkerInjuriesTool(s, permissions)
}
