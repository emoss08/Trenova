package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerinjuryservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerleaveservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
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
		req *workerports.ListLeaveCasesRequest,
	) ([]*worker.WorkerLeaveCase, error)
}

type wfVerificationReader interface {
	ListVerifications(
		ctx context.Context,
		req *workerports.ListEmploymentVerificationsRequest,
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
		req *workerports.ListCredentialTypesRequest,
	) (*pagination.CursorListResult[*worker.WorkerCredentialType], error)
}

type wfInjuryReader interface {
	ListInjuries(
		ctx context.Context,
		req *workerports.ListWorkerInjuriesRequest,
	) ([]*worker.WorkerInjury, error)
}

var (
	_ wfLeaveReader        = (*workerleaveservice.Service)(nil)
	_ wfVerificationReader = (*workerdqfservice.Service)(nil)
	_ wfCredentialReader   = (*workercredentialservice.Service)(nil)
	_ wfInjuryReader       = (*workerinjuryservice.Service)(nil)
)

func catalogFilter(params *agentports.QueryToolParams) *pagination.QueryOptions {
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
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &workforceRead{
		name: "worker_list_leave_cases",
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
			params *agentports.QueryToolParams,
			gate *fieldGate,
		) (any, error) {
			workerID, err := optionalWorker(params.Params)
			if err != nil {
				return nil, err
			}
			found, err := cases.ListCases(ctx, &workerports.ListLeaveCasesRequest{
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
					ID:             leave.ID.String(),
					WorkerID:       leave.WorkerID.String(),
					Worker:         workerName(leave.Worker),
					LeaveType:      string(leave.LeaveType),
					Status:         string(leave.Status),
					Frequency:      string(leave.Frequency),
					FMLADesignated: leave.FMLADesignated,
					Reason:         gatedText(gate, wfFieldReason, leave.Reason),
					RequestedAt:    recordedDate(leave.RequestedAt),
					StartsAt:       recordedDate(leave.StartsAt),
					EndsAt: expectedDate(
						typeutils.ValueOrZero(leave.EndsAt),
						absentOpenEnded,
					),
					CertificationStatus: string(leave.CertificationStatus),
					CertificationDueAt: expectedDate(
						typeutils.ValueOrZero(leave.CertificationDueAt),
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
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &workforceRead{
		name: "worker_list_employment_verifications",
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
			params *agentports.QueryToolParams,
			gate *fieldGate,
		) (any, error) {
			workerID, err := optionalWorker(params.Params)
			if err != nil {
				return nil, err
			}
			found, err := verifications.ListVerifications(ctx,
				&workerports.ListEmploymentVerificationsRequest{
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
						typeutils.ValueOrZero(entry.RequestedAt),
						absentNotRequested,
					),
					ResponseReceivedAt: expectedDate(
						typeutils.ValueOrZero(entry.ResponseReceivedAt),
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
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &workforceRead{
		name: "worker_list_injuries",
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
			params *agentports.QueryToolParams,
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
			found, err := injuries.ListInjuries(ctx, &workerports.ListWorkerInjuriesRequest{
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
						typeutils.ValueOrZero(injury.ReturnedToWorkAt),
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
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListWorkerLeaveCasesTool(s, permissions)
}

func provideListEmploymentVerificationsTool(
	s *workerdqfservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListEmploymentVerificationsTool(s, permissions)
}

func provideListWorkerInjuriesTool(
	s *workerinjuryservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListWorkerInjuriesTool(s, permissions)
}
