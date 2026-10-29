package qualificationtools

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
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramVerificationID = "verificationId"

const paramEmployerName = "employerName"

const paramEmployerDOT = "employerDotNumber"

const paramEmployerMC = "employerMcNumber"

const paramContactName = "contactName"

const paramContactPhone = "contactPhone"

const paramContactEmail = "contactEmail"

const paramEmployedFrom = "employedFrom"

const paramEmployedTo = "employedTo"

const paramWasDOTRegulated = "wasDotRegulated"

const paramVerifyMethod = "method"

const paramVerifyStatus = "status"

const paramResponseAt = "responseReceivedAt"

const paramDAResponseAt = "drugAlcoholResponseReceivedAt"

const paramHadAccidents = "hadAccidents"

const paramAccidentCount = "accidentCount"

const paramHadDAViolations = "hadDrugAlcoholViolations"

const paramFindings = "findings"

const paramRequestMove = "action"

const kindVerification = "previous employer verification"

const maxVerificationField = 150

const maxRegistrationNumber = 20

const maxAccidents = 100

const maxPhoneChars = 30

type verificationRequestMove string

const verificationRequested = verificationRequestMove("Requested")

const verificationFollowUp = verificationRequestMove("FollowUp")

var verificationStatuses = agenttoolschema.Source(
	"worker.employmentVerificationStatus",
	employment.EmploymentVerificationStatusValues(),
)

var verificationMethods = agenttoolschema.Source(
	"worker.employmentVerificationMethod",
	employment.EmploymentVerificationMethodValues(),
)

var verificationRequestMoves = agenttoolschema.Source(
	"employmentVerification.agentRequest",
	[]verificationRequestMove{verificationRequested, verificationFollowUp},
)

var verificationFields = []string{
	workforcekit.FieldWorkerID,
	paramEmployerName,
	paramEmployerDOT,
	paramEmployerMC,
	paramContactName,
	paramContactPhone,
	paramContactEmail,
	paramEmployedFrom,
	paramEmployedTo,
	paramWasDOTRegulated,
	changepreview.FieldStatus,
	paramVerifyMethod,
	workforcekit.FieldRequestedAt,
	"responseReceivedAt",
	"drugAlcoholResponseReceivedAt",
	"followUpCount",
	"lastFollowUpAt",
	paramHadAccidents,
	paramAccidentCount,
	paramHadDAViolations,
	paramFindings,
	workforcekit.FieldNotes,
	workforcekit.FieldDocument,
}

type VerificationKeeper interface {
	PlanRecordVerification(
		ctx context.Context,
		entity *employment.WorkerEmploymentVerification,
		userID pulid.ID,
	) (*employment.WorkerEmploymentVerification, error)
	RecordVerification(
		ctx context.Context,
		entity *employment.WorkerEmploymentVerification,
		userID pulid.ID,
	) (*employment.WorkerEmploymentVerification, error)
	PlanUpdateVerification(
		ctx context.Context,
		req *workerdqfservice.UpdateVerificationRequest,
	) (*workerdqfservice.VerificationChange, error)
	UpdateVerification(
		ctx context.Context,
		req *workerdqfservice.UpdateVerificationRequest,
	) (*employment.WorkerEmploymentVerification, error)
	PlanMarkRequested(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*workerdqfservice.VerificationChange, error)
	MarkRequested(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) (*employment.WorkerEmploymentVerification, error)
	PlanRecordFollowUp(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*workerdqfservice.VerificationChange, error)
	RecordFollowUp(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) (*employment.WorkerEmploymentVerification, error)
	PlanDeleteVerification(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*employment.WorkerEmploymentVerification, error)
	DeleteVerification(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) error
}

func DqfToolProviders() []any {
	return []any{
		provideRecordEmploymentVerificationTool,
		provideUpdateEmploymentVerificationTool,
		provideLogEmploymentVerificationRequestTool,
		provideDeleteEmploymentVerificationTool,
	}
}

func verificationIDProperty() map[string]any {
	return agenttoolschema.KindID(
		agenttoolschema.IDDescription("The previous employer's verification",
			"worker_list_employment_verifications"),
		recordkind.EmploymentVerification,
	)
}

func verificationRecord(entity *employment.WorkerEmploymentVerification) toolpreview.Record {
	return workforcekit.Record(permission.ResourceQualification, entity.ID, entity.EmployerName,
		entity.Version)
}

func verificationEmployerProperties() map[string]any {
	return map[string]any{
		paramEmployerName: toolparams.StringProperty("The previous employer's name.",
			maxVerificationField),
		paramEmployerDOT: toolparams.StringProperty("Their USDOT number.", maxRegistrationNumber),
		paramEmployerMC:  toolparams.StringProperty("Their MC number.", maxRegistrationNumber),
		paramContactName: toolparams.StringProperty("Who to ask there.", maxVerificationField),
		paramContactPhone: toolparams.StringProperty(
			"Their phone number, as the application gives "+
				"it.",
			maxPhoneChars,
		),
		paramContactEmail: toolparams.StringProperty(
			"Their email address, as the application gives "+
				"it.",
			maxVerificationField,
		),
		paramEmployedFrom: agenttoolschema.Date("When the worker started there."),
		paramEmployedTo:   agenttoolschema.Date("When the worker left."),
		paramWasDOTRegulated: toolparams.BooleanProperty(
			"Whether the job was DOT-regulated, which " +
				"brings the drug and alcohol questions in.",
		),
		paramVerifyMethod: agenttoolschema.Enum("How the request goes out.",
			verificationMethods),
		toolparams.ParamNote: workforcekit.NoteProperty("Anything the file should say."),
	}
}

func newRecordEmploymentVerificationTool(dqf VerificationKeeper) agenttoolports.AgentTool {
	properties := verificationEmployerProperties()
	properties[toolparams.ParamWorkerID] = toolparams.WorkerProperty()
	spec := workforcekit.WithSchema(workforcekit.Spec(
		"worker_record_employment_verification",
		"File a previous employer on a driver's qualification file so their safety "+
			"performance history can be asked for (49 CFR 391.23), with who to ask and the "+
			"dates of employment from the application. It starts Pending; "+
			"worker_record_employment_verification_request records the request going out.",
		"Adds a previous employer to the qualification file inside Trenova; nothing is sent "+
			"to the employer, and it is corrected or removed the same way.",
		permission.ResourceQualification,
		permission.OpCreate,
	), properties, toolparams.ParamWorkerID, paramEmployerName)
	spec.SearchTerms = []string{"previous employer", "safety performance history",
		"qualification file", "DQ file"}

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*employment.WorkerEmploymentVerification, *employment.WorkerEmploymentVerification,
	]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*employment.WorkerEmploymentVerification, error) {
			workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
			if err != nil {
				return nil, err
			}
			entity := &employment.WorkerEmploymentVerification{
				OrganizationID: params.OrganizationID,
				BusinessUnitID: params.BusinessUnitID,
				WorkerID:       workerID,
				Status:         employment.VerificationPending,
			}
			req := &workerdqfservice.UpdateVerificationRequest{}
			if err = applyVerificationEmployer(req, params.Params); err != nil {
				return nil, err
			}
			if req.EmployerName == nil || *req.EmployerName == "" {
				return nil, fmt.Errorf("missing required parameter %q", paramEmployerName)
			}
			copyVerificationEmployer(entity, req)
			return entity, nil
		},
		Plan: func(
			ctx context.Context,
			entity *employment.WorkerEmploymentVerification,
			params *agenttoolports.ToolExecuteParams,
		) (*employment.WorkerEmploymentVerification, error) {
			return dqf.PlanRecordVerification(ctx, entity, params.Actor.UserID)
		},
		Refused: func(*employment.WorkerEmploymentVerification) string {
			return "Would file a previous employer on the qualification file."
		},
		Render: func(
			_ *employment.WorkerEmploymentVerification,
			planned *employment.WorkerEmploymentVerification,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Create(
				workforcekit.Record(
					permission.ResourceQualification,
					pulid.Nil,
					planned.EmployerName,
					0,
				),
				planned,
				workforcekit.Options(verificationFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would file %s as a previous employer to verify.", planned.EmployerName),
				change), nil
		},
		Run: func(
			ctx context.Context,
			entity *employment.WorkerEmploymentVerification,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			created, err := dqf.RecordVerification(ctx, entity, params.Actor.UserID)
			if err != nil {
				return nil, err
			}
			return workforcekit.Result("filed", kindVerification, paramVerificationID, created.ID,
				created.WorkerID), nil
		},
	})
}

func applyVerificationEmployer(
	req *workerdqfservice.UpdateVerificationRequest,
	params map[string]any,
) error {
	for _, target := range []struct {
		key   string
		dest  **string
		limit int
	}{
		{paramEmployerName, &req.EmployerName, maxVerificationField},
		{paramEmployerDOT, &req.EmployerDOTNumber, maxRegistrationNumber},
		{paramEmployerMC, &req.EmployerMCNumber, maxRegistrationNumber},
		{paramContactName, &req.ContactName, maxVerificationField},
		{paramContactPhone, &req.ContactPhone, maxPhoneChars},
		{paramContactEmail, &req.ContactEmail, maxVerificationField},
		{toolparams.ParamNote, &req.Notes, workforcekit.NoteChars},
	} {
		text, err := toolparams.OptionalBoundedText(params, target.key, target.limit)
		if err != nil {
			return err
		}
		*target.dest = text
	}
	var err error
	if req.EmployedFrom, err = toolparams.OptionalScheduleDay(
		params,
		paramEmployedFrom,
	); err != nil {
		return err
	}
	if req.EmployedTo, err = toolparams.OptionalScheduleDay(params, paramEmployedTo); err != nil {
		return err
	}
	if req.WasDOTRegulated, err = toolparams.OptionalBoolPointer(
		params,
		paramWasDOTRegulated,
	); err != nil {
		return err
	}
	if method, given, methodErr := toolparams.OptionalEnum(params, paramVerifyMethod,
		verificationMethods.Values); methodErr != nil {
		return methodErr
	} else if given {
		req.Method = &method
	}
	return nil
}

func copyVerificationEmployer(
	entity *employment.WorkerEmploymentVerification,
	req *workerdqfservice.UpdateVerificationRequest,
) {
	for _, field := range []struct {
		dest  *string
		value *string
	}{
		{&entity.EmployerName, req.EmployerName},
		{&entity.EmployerDOTNumber, req.EmployerDOTNumber},
		{&entity.EmployerMCNumber, req.EmployerMCNumber},
		{&entity.ContactName, req.ContactName},
		{&entity.ContactPhone, req.ContactPhone},
		{&entity.ContactEmail, req.ContactEmail},
		{&entity.Notes, req.Notes},
	} {
		if field.value != nil {
			*field.dest = *field.value
		}
	}
	entity.EmployedFrom = req.EmployedFrom
	entity.EmployedTo = req.EmployedTo
	if req.WasDOTRegulated != nil {
		entity.WasDOTRegulated = *req.WasDOTRegulated
	}
	if req.Method != nil {
		entity.Method = *req.Method
	}
}

func applyVerificationResponse(
	req *workerdqfservice.UpdateVerificationRequest,
	params map[string]any,
) error {
	var err error
	if status, given, statusErr := toolparams.OptionalEnum(params, paramVerifyStatus,
		verificationStatuses.Values); statusErr != nil {
		return statusErr
	} else if given {
		req.Status = &status
	}
	if req.ResponseReceivedAt, err = toolparams.OptionalScheduleDay(
		params,
		paramResponseAt,
	); err != nil {
		return err
	}
	if req.DrugAlcoholResponseReceivedAt, err = toolparams.OptionalScheduleDay(params,
		paramDAResponseAt); err != nil {
		return err
	}
	if req.HadAccidents, err = toolparams.OptionalBoolPointer(
		params,
		paramHadAccidents,
	); err != nil {
		return err
	}
	if req.HadDrugAlcoholViolations, err = toolparams.OptionalBoolPointer(params,
		paramHadDAViolations); err != nil {
		return err
	}
	if req.Findings, err = toolparams.OptionalBoundedText(
		params,
		paramFindings,
		workforcekit.NoteChars,
	); err != nil {
		return err
	}
	if req.DocumentID, err = toolparams.OptionalID(params, workforcekit.ParamDocument); err != nil {
		return err
	}
	count, err := toolparams.OptionalIntInRange(params, paramAccidentCount, 0, maxAccidents)
	if err != nil {
		return err
	}
	if count != nil {
		value := int32(*count) //nolint:gosec // bounded by the schema's maximum
		req.AccidentCount = &value
	}
	return nil
}

func newUpdateEmploymentVerificationTool(dqf VerificationKeeper) agenttoolports.AgentTool {
	properties := verificationEmployerProperties()
	properties[paramVerificationID] = verificationIDProperty()
	properties[paramVerifyStatus] = agenttoolschema.Enum("Where the request stands.",
		verificationStatuses)
	properties[paramResponseAt] = agenttoolschema.Date("When the employer's answer came back.")
	properties[paramDAResponseAt] = agenttoolschema.Date(
		"When their drug and alcohol answer came " +
			"back.",
	)
	properties[paramHadAccidents] = toolparams.BooleanProperty("Whether they reported accidents.")
	properties[paramAccidentCount] = toolparams.IntegerProperty(
		"How many accidents they reported.",
		0,
		maxAccidents,
	)
	properties[paramHadDAViolations] = toolparams.BooleanProperty(
		"Whether they reported a drug or " +
			"alcohol violation.",
	)
	properties[paramFindings] = workforcekit.NoteProperty(
		"What the employer's answer says, from the " +
			"document they sent.",
	)
	properties[workforcekit.ParamDocument] = workforcekit.DocumentProperty()
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_update_employment_verification",
		"Correct a previous employer on the qualification file, or record their answer. "+
			"The answer is when it came back, the accidents and drug and alcohol violations "+
			"it reports, the findings and the document. Give only what changes, from the answer "+
			"itself.",
		"Updates the previous employer's record in the qualification file inside Trenova; "+
			"nothing is sent, and it is corrected again the same way.",
		permission.ResourceQualification,
		permission.OpUpdate,
	), properties, paramVerificationID), paramVerificationID, permission.ResourceQualification)
	spec.SearchTerms = []string{"previous employer answered", "employer response", "dq file"}

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*workerdqfservice.UpdateVerificationRequest, *workerdqfservice.VerificationChange,
	]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*workerdqfservice.UpdateVerificationRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramVerificationID)
			if err != nil {
				return nil, err
			}
			req := &workerdqfservice.UpdateVerificationRequest{
				TenantInfo:     toolparams.TenantFrom(params),
				VerificationID: id,
				UserID:         params.Actor.UserID,
			}
			if err = applyVerificationEmployer(req, params.Params); err != nil {
				return nil, err
			}
			return req, applyVerificationResponse(req, params.Params)
		},
		Plan: func(
			ctx context.Context,
			req *workerdqfservice.UpdateVerificationRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerdqfservice.VerificationChange, error) {
			return dqf.PlanUpdateVerification(ctx, req)
		},
		Refused: func(*workerdqfservice.UpdateVerificationRequest) string {
			return "Would update a previous employer's verification."
		},
		Render: func(
			_ *workerdqfservice.UpdateVerificationRequest,
			change *workerdqfservice.VerificationChange,
		) (*agentpreview.ToolPreview, error) {
			recorded, err := toolpreview.Changed(verificationRecord(change.Before),
				change.Before, change.After, workforcekit.Options(verificationFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would update the verification with %s.",
				change.Before.EmployerName), recorded), nil
		},
		Run: func(
			ctx context.Context,
			req *workerdqfservice.UpdateVerificationRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			updated, err := dqf.UpdateVerification(ctx, req)
			if err != nil {
				return nil, err
			}
			return workforcekit.Result("updated", kindVerification, paramVerificationID, updated.ID,
				updated.WorkerID), nil
		},
	})
}

type verificationRequestLog struct {
	move   verificationRequestMove
	id     pulid.ID
	tenant pagination.TenantInfo
	userID pulid.ID
}

func newLogEmploymentVerificationRequestTool(dqf VerificationKeeper) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_record_employment_verification_request",
		"Record that a previous employer was asked for the driver's safety performance "+
			"history today (Requested), or chased again when they have not answered "+
			"(FollowUp). Each follow-up is counted and dated as the good-faith effort 49 CFR "+
			"391.23 asks for. Sending the request is done by the person; this keeps the date.",
		"Stamps the request date or counts a follow-up inside Trenova; nothing is sent to "+
			"the employer.",
		permission.ResourceQualification,
		permission.OpUpdate,
	), map[string]any{
		paramVerificationID: verificationIDProperty(),
		paramRequestMove: agenttoolschema.Enum("Requested or FollowUp.",
			verificationRequestMoves),
	}, paramVerificationID, paramRequestMove), paramVerificationID,
		permission.ResourceQualification)
	spec.Reversible = false

	spec.SearchTerms = []string{"followed up", "follow up with employer", "chase previous employer"}

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*verificationRequestLog, *workerdqfservice.VerificationChange,
	]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*verificationRequestLog, error) {
			id, err := toolparams.RequirePulid(params.Params, paramVerificationID)
			if err != nil {
				return nil, err
			}
			move, err := toolparams.RequireEnum(params.Params, paramRequestMove,
				verificationRequestMoves.Values)
			if err != nil {
				return nil, err
			}
			return &verificationRequestLog{
				move:   move,
				id:     id,
				tenant: toolparams.TenantFrom(params),
				userID: params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			log *verificationRequestLog,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerdqfservice.VerificationChange, error) {
			if log.move == verificationRequested {
				return dqf.PlanMarkRequested(ctx, log.tenant, log.id)
			}
			return dqf.PlanRecordFollowUp(ctx, log.tenant, log.id)
		},
		Refused: func(*verificationRequestLog) string {
			return "Would log a verification request."
		},
		Render: func(
			log *verificationRequestLog,
			change *workerdqfservice.VerificationChange,
		) (*agentpreview.ToolPreview, error) {
			options := append(
				workforcekit.Options(
					changepreview.FieldStatus,
					workforcekit.FieldRequestedAt,
					"followUpCount",
					"lastFollowUpAt",
				),
				toolpreview.Volatile(workforcekit.FieldRequestedAt, "lastFollowUpAt"),
			)
			recorded, err := toolpreview.Changed(verificationRecord(change.Before),
				change.Before, change.After, options...)
			if err != nil {
				return nil, err
			}
			summary := fmt.Sprintf("Would record that %s was asked today.",
				change.Before.EmployerName)
			if log.move == verificationFollowUp {
				summary = fmt.Sprintf("Would count follow-up %d with %s.",
					change.After.FollowUpCount, change.Before.EmployerName)
			}
			return toolpreview.Build(summary, recorded), nil
		},
		Run: func(
			ctx context.Context,
			log *verificationRequestLog,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			var saved *employment.WorkerEmploymentVerification
			var err error
			action := "marked requested"
			if log.move == verificationRequested {
				saved, err = dqf.MarkRequested(ctx, log.tenant, log.id, log.userID)
			} else {
				action = "followed up"
				saved, err = dqf.RecordFollowUp(ctx, log.tenant, log.id, log.userID)
			}
			if err != nil {
				return nil, err
			}
			return workforcekit.Result(action, kindVerification, paramVerificationID, saved.ID,
				saved.WorkerID), nil
		},
	})
}

func newDeleteEmploymentVerificationTool(dqf VerificationKeeper) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_delete_employment_verification",
		"Remove a previous employer filed on the qualification file in error, such as a "+
			"duplicate. A settled investigation is corrected, not removed.",
		"Removes a previous employer from the qualification file; the audit trail keeps what "+
			"was removed.",
		permission.ResourceQualification,
		permission.OpDelete,
	), map[string]any{paramVerificationID: verificationIDProperty()}, paramVerificationID),
		paramVerificationID, permission.ResourceQualification)
	spec.SearchTerms = []string{"previous employer", "duplicate employer", "remove employer"}
	spec.MaxTier = agent.TierPropose
	spec.Reversible = false

	return toolspec.NewReceivableTool(
		spec,
		toolspec.ReceivablePlan[*workforcekit.RecordDelete, *employment.WorkerEmploymentVerification]{
			Request: workforcekit.RecordDeleteFrom(paramVerificationID),
			Plan: func(
				ctx context.Context,
				req *workforcekit.RecordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*employment.WorkerEmploymentVerification, error) {
				return dqf.PlanDeleteVerification(ctx, req.Tenant, req.ID)
			},
			Refused: func(*workforcekit.RecordDelete) string { return "Would remove a previous employer." },
			Render: func(
				_ *workforcekit.RecordDelete,
				entity *employment.WorkerEmploymentVerification,
			) (*agentpreview.ToolPreview, error) {
				change, err := toolpreview.Delete(verificationRecord(entity), entity,
					workforcekit.Options(verificationFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would remove %s from the qualification file.",
					entity.EmployerName,
				), change), nil
			},
			Run: func(
				ctx context.Context,
				req *workforcekit.RecordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				return nil, dqf.DeleteVerification(ctx, req.Tenant, req.ID, req.UserID)
			},
		},
	)
}

func provideRecordEmploymentVerificationTool(s *workerdqfservice.Service) agenttoolports.AgentTool {
	return newRecordEmploymentVerificationTool(s)
}

func provideUpdateEmploymentVerificationTool(s *workerdqfservice.Service) agenttoolports.AgentTool {
	return newUpdateEmploymentVerificationTool(s)
}

func provideLogEmploymentVerificationRequestTool(
	s *workerdqfservice.Service,
) agenttoolports.AgentTool {
	return newLogEmploymentVerificationRequestTool(s)
}

func provideDeleteEmploymentVerificationTool(s *workerdqfservice.Service) agenttoolports.AgentTool {
	return newDeleteEmploymentVerificationTool(s)
}
