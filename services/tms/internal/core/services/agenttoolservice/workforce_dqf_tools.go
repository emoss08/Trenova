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
	"github.com/emoss08/trenova/internal/core/services/workerdqfservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramVerificationID   = "verificationId"
	paramEmployerName     = "employerName"
	paramEmployerDOT      = "employerDotNumber"
	paramEmployerMC       = "employerMcNumber"
	paramContactName      = "contactName"
	paramContactPhone     = "contactPhone"
	paramContactEmail     = "contactEmail"
	paramEmployedFrom     = "employedFrom"
	paramEmployedTo       = "employedTo"
	paramWasDOTRegulated  = "wasDotRegulated"
	paramVerifyMethod     = "method"
	paramVerifyStatus     = "status"
	paramResponseAt       = "responseReceivedAt"
	paramDAResponseAt     = "drugAlcoholResponseReceivedAt"
	paramHadAccidents     = "hadAccidents"
	paramAccidentCount    = "accidentCount"
	paramHadDAViolations  = "hadDrugAlcoholViolations"
	paramFindings         = "findings"
	paramRequestMove      = "action"
	kindVerification      = "previous employer verification"
	maxVerificationField  = 150
	maxRegistrationNumber = 20
	maxAccidents          = 100
	maxPhoneChars         = 30
)

type verificationRequestMove string

const (
	verificationRequested = verificationRequestMove("Requested")
	verificationFollowUp  = verificationRequestMove("FollowUp")
)

var (
	verificationStatuses = agenttoolschema.Source(
		"worker.employmentVerificationStatus",
		worker.EmploymentVerificationStatusValues(),
	)
	verificationMethods = agenttoolschema.Source(
		"worker.employmentVerificationMethod",
		worker.EmploymentVerificationMethodValues(),
	)
	verificationRequestMoves = agenttoolschema.Source(
		"employmentVerification.agentRequest",
		[]verificationRequestMove{verificationRequested, verificationFollowUp},
	)
	verificationFields = []string{
		wfFieldWorkerID, paramEmployerName, paramEmployerDOT, paramEmployerMC, paramContactName,
		paramContactPhone, paramContactEmail, paramEmployedFrom, paramEmployedTo,
		paramWasDOTRegulated, fieldStatus, paramVerifyMethod, wfFieldRequestedAt,
		"responseReceivedAt", "drugAlcoholResponseReceivedAt", "followUpCount",
		"lastFollowUpAt", paramHadAccidents, paramAccidentCount, paramHadDAViolations,
		paramFindings, wfFieldNotes, wfFieldDocument,
	}
)

type verificationKeeper interface {
	PlanRecordVerification(
		ctx context.Context,
		entity *worker.WorkerEmploymentVerification,
		userID pulid.ID,
	) (*worker.WorkerEmploymentVerification, error)
	RecordVerification(
		ctx context.Context,
		entity *worker.WorkerEmploymentVerification,
		userID pulid.ID,
	) (*worker.WorkerEmploymentVerification, error)
	PlanUpdateVerification(
		ctx context.Context,
		req *workerdqfservice.UpdateVerificationRequest,
	) (*workerdqfservice.VerificationChange, error)
	UpdateVerification(
		ctx context.Context,
		req *workerdqfservice.UpdateVerificationRequest,
	) (*worker.WorkerEmploymentVerification, error)
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
	) (*worker.WorkerEmploymentVerification, error)
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
	) (*worker.WorkerEmploymentVerification, error)
	PlanDeleteVerification(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.WorkerEmploymentVerification, error)
	DeleteVerification(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) error
}

var _ verificationKeeper = (*workerdqfservice.Service)(nil)

func dqfToolProviders() []any {
	return []any{
		provideRecordEmploymentVerificationTool,
		provideUpdateEmploymentVerificationTool,
		provideLogEmploymentVerificationRequestTool,
		provideDeleteEmploymentVerificationTool,
	}
}

func verificationIDProperty() map[string]any {
	return idProperty("The previous employer's verification, from " +
		"list_employment_verifications. Never guess one.")
}

func verificationRecord(entity *worker.WorkerEmploymentVerification) toolpreview.Record {
	return wfRecord(permission.ResourceQualification, entity.ID, entity.EmployerName,
		entity.Version)
}

func verificationEmployerProperties() map[string]any {
	return map[string]any{
		paramEmployerName: stringProperty("The previous employer's name.",
			maxVerificationField),
		paramEmployerDOT: stringProperty("Their USDOT number.", maxRegistrationNumber),
		paramEmployerMC:  stringProperty("Their MC number.", maxRegistrationNumber),
		paramContactName: stringProperty("Who to ask there.", maxVerificationField),
		paramContactPhone: stringProperty("Their phone number, as the application gives "+
			"it.", maxPhoneChars),
		paramContactEmail: stringProperty("Their email address, as the application gives "+
			"it.", maxVerificationField),
		paramEmployedFrom: dayProperty("When the worker started there."),
		paramEmployedTo:   dayProperty("When the worker left."),
		paramWasDOTRegulated: booleanProperty("Whether the job was DOT-regulated, which " +
			"brings the drug and alcohol questions in."),
		paramVerifyMethod: agenttoolschema.Enum("How the request goes out.",
			verificationMethods),
		fieldNotes: wfNoteProperty("Anything the file should say."),
	}
}

func newRecordEmploymentVerificationTool(dqf verificationKeeper) serviceports.AgentTool {
	properties := verificationEmployerProperties()
	properties[paramWorkerID] = workerProperty()
	spec := withSchema(wfSpec(
		"record_employment_verification",
		"File a previous employer on a driver's qualification file so their safety "+
			"performance history can be asked for (49 CFR 391.23), with who to ask and the "+
			"dates of employment from the application. It starts Pending; "+
			"log_employment_verification_request records the request going out.",
		"Adds a previous employer to the qualification file inside Trenova; nothing is sent "+
			"to the employer, and it is corrected or removed the same way.",
		permission.ResourceQualification,
		permission.OpCreate,
	), properties, paramWorkerID, paramEmployerName)
	spec.searchTerms = []string{"previous employer", "safety performance history",
		"qualification file", "DQ file"}

	return newReportingReceivableTool(spec, receivablePlan[
		*worker.WorkerEmploymentVerification, *worker.WorkerEmploymentVerification,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*worker.WorkerEmploymentVerification, error) {
			workerID, err := requirePulid(params.Params, paramWorkerID)
			if err != nil {
				return nil, err
			}
			entity := &worker.WorkerEmploymentVerification{
				OrganizationID: params.OrganizationID,
				BusinessUnitID: params.BusinessUnitID,
				WorkerID:       workerID,
				Status:         worker.VerificationPending,
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
		plan: func(
			ctx context.Context,
			entity *worker.WorkerEmploymentVerification,
			params *serviceports.ToolExecuteParams,
		) (*worker.WorkerEmploymentVerification, error) {
			return dqf.PlanRecordVerification(ctx, entity, params.Actor.UserID)
		},
		refused: func(*worker.WorkerEmploymentVerification) string {
			return "Would file a previous employer on the qualification file."
		},
		render: func(
			_ *worker.WorkerEmploymentVerification,
			planned *worker.WorkerEmploymentVerification,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(
				wfRecord(permission.ResourceQualification, pulid.Nil, planned.EmployerName, 0),
				planned, wfOptions(verificationFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would file %s as a previous employer to verify.", planned.EmployerName),
				change), nil
		},
		run: func(
			ctx context.Context,
			entity *worker.WorkerEmploymentVerification,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := dqf.RecordVerification(ctx, entity, params.Actor.UserID)
			if err != nil {
				return nil, err
			}
			return wfResult("filed", kindVerification, paramVerificationID, created.ID,
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
		{fieldNotes, &req.Notes, wfNoteChars},
	} {
		text, err := optionalBoundedText(params, target.key, target.limit)
		if err != nil {
			return err
		}
		*target.dest = text
	}
	var err error
	if req.EmployedFrom, err = optionalScheduleDay(params, paramEmployedFrom); err != nil {
		return err
	}
	if req.EmployedTo, err = optionalScheduleDay(params, paramEmployedTo); err != nil {
		return err
	}
	if req.WasDOTRegulated, err = optionalBoolPointer(params, paramWasDOTRegulated); err != nil {
		return err
	}
	if method, given, methodErr := optionalEnum(params, paramVerifyMethod,
		verificationMethods.Values); methodErr != nil {
		return methodErr
	} else if given {
		req.Method = &method
	}
	return nil
}

func copyVerificationEmployer(
	entity *worker.WorkerEmploymentVerification,
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
	if status, given, statusErr := optionalEnum(params, paramVerifyStatus,
		verificationStatuses.Values); statusErr != nil {
		return statusErr
	} else if given {
		req.Status = &status
	}
	if req.ResponseReceivedAt, err = optionalScheduleDay(params, paramResponseAt); err != nil {
		return err
	}
	if req.DrugAlcoholResponseReceivedAt, err = optionalScheduleDay(params,
		paramDAResponseAt); err != nil {
		return err
	}
	if req.HadAccidents, err = optionalBoolPointer(params, paramHadAccidents); err != nil {
		return err
	}
	if req.HadDrugAlcoholViolations, err = optionalBoolPointer(params,
		paramHadDAViolations); err != nil {
		return err
	}
	if req.Findings, err = optionalBoundedText(params, paramFindings, wfNoteChars); err != nil {
		return err
	}
	if req.DocumentID, err = optionalID(params, wfParamDocument); err != nil {
		return err
	}
	count, err := optionalIntInRange(params, paramAccidentCount, 0, maxAccidents)
	if err != nil {
		return err
	}
	if count != nil {
		value := int32(*count) //nolint:gosec // bounded by the schema's maximum
		req.AccidentCount = &value
	}
	return nil
}

func newUpdateEmploymentVerificationTool(dqf verificationKeeper) serviceports.AgentTool {
	properties := verificationEmployerProperties()
	properties[paramVerificationID] = verificationIDProperty()
	properties[paramVerifyStatus] = agenttoolschema.Enum("Where the request stands.",
		verificationStatuses)
	properties[paramResponseAt] = dayProperty("When the employer's answer came back.")
	properties[paramDAResponseAt] = dayProperty("When their drug and alcohol answer came " +
		"back.")
	properties[paramHadAccidents] = booleanProperty("Whether they reported accidents.")
	properties[paramAccidentCount] = integerProperty("How many accidents they reported.", 0,
		maxAccidents)
	properties[paramHadDAViolations] = booleanProperty("Whether they reported a drug or " +
		"alcohol violation.")
	properties[paramFindings] = wfNoteProperty("What the employer's answer says, from the " +
		"document they sent.")
	properties[wfParamDocument] = wfDocumentProperty()
	spec := targeting(withSchema(wfSpec(
		"update_employment_verification",
		"Correct a previous employer on the qualification file, or record their answer. "+
			"The answer is when it came back, the accidents and drug and alcohol violations "+
			"it reports, the findings and the document. Give only what changes, from the answer "+
			"itself.",
		"Updates the previous employer's record in the qualification file inside Trenova; "+
			"nothing is sent, and it is corrected again the same way.",
		permission.ResourceQualification,
		permission.OpUpdate,
	), properties, paramVerificationID), paramVerificationID, permission.ResourceQualification)

	return newReportingReceivableTool(spec, receivablePlan[
		*workerdqfservice.UpdateVerificationRequest, *workerdqfservice.VerificationChange,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*workerdqfservice.UpdateVerificationRequest, error) {
			id, err := requirePulid(params.Params, paramVerificationID)
			if err != nil {
				return nil, err
			}
			req := &workerdqfservice.UpdateVerificationRequest{
				TenantInfo:     tenantFrom(*params),
				VerificationID: id,
				UserID:         params.Actor.UserID,
			}
			if err = applyVerificationEmployer(req, params.Params); err != nil {
				return nil, err
			}
			return req, applyVerificationResponse(req, params.Params)
		},
		plan: func(
			ctx context.Context,
			req *workerdqfservice.UpdateVerificationRequest,
			_ *serviceports.ToolExecuteParams,
		) (*workerdqfservice.VerificationChange, error) {
			return dqf.PlanUpdateVerification(ctx, req)
		},
		refused: func(*workerdqfservice.UpdateVerificationRequest) string {
			return "Would update a previous employer's verification."
		},
		render: func(
			_ *workerdqfservice.UpdateVerificationRequest,
			change *workerdqfservice.VerificationChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(verificationRecord(change.Before),
				change.Before, change.After, wfOptions(verificationFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would update the verification with %s.",
				change.Before.EmployerName), recorded), nil
		},
		run: func(
			ctx context.Context,
			req *workerdqfservice.UpdateVerificationRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := dqf.UpdateVerification(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("updated", kindVerification, paramVerificationID, updated.ID,
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

func newLogEmploymentVerificationRequestTool(dqf verificationKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"log_employment_verification_request",
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
	spec.reversible = false

	return newReportingReceivableTool(spec, receivablePlan[
		*verificationRequestLog, *workerdqfservice.VerificationChange,
	]{
		request: func(params *serviceports.ToolExecuteParams) (*verificationRequestLog, error) {
			id, err := requirePulid(params.Params, paramVerificationID)
			if err != nil {
				return nil, err
			}
			move, err := requireEnum(params.Params, paramRequestMove,
				verificationRequestMoves.Values)
			if err != nil {
				return nil, err
			}
			return &verificationRequestLog{move: move, id: id, tenant: tenantFrom(*params),
				userID: params.Actor.UserID}, nil
		},
		plan: func(
			ctx context.Context,
			log *verificationRequestLog,
			_ *serviceports.ToolExecuteParams,
		) (*workerdqfservice.VerificationChange, error) {
			if log.move == verificationRequested {
				return dqf.PlanMarkRequested(ctx, log.tenant, log.id)
			}
			return dqf.PlanRecordFollowUp(ctx, log.tenant, log.id)
		},
		refused: func(*verificationRequestLog) string {
			return "Would log a verification request."
		},
		render: func(
			log *verificationRequestLog,
			change *workerdqfservice.VerificationChange,
		) (*agent.ToolPreview, error) {
			options := append(
				wfOptions(fieldStatus, wfFieldRequestedAt, "followUpCount", "lastFollowUpAt"),
				toolpreview.Volatile(wfFieldRequestedAt, "lastFollowUpAt"),
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
		run: func(
			ctx context.Context,
			log *verificationRequestLog,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			var saved *worker.WorkerEmploymentVerification
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
			return wfResult(action, kindVerification, paramVerificationID, saved.ID,
				saved.WorkerID), nil
		},
	})
}

func newDeleteEmploymentVerificationTool(dqf verificationKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"delete_employment_verification",
		"Remove a previous employer filed on the qualification file in error, such as a "+
			"duplicate. A settled investigation is corrected, not removed.",
		"Removes a previous employer from the qualification file; the audit trail keeps what "+
			"was removed.",
		permission.ResourceQualification,
		permission.OpDelete,
	), map[string]any{paramVerificationID: verificationIDProperty()}, paramVerificationID),
		paramVerificationID, permission.ResourceQualification)
	spec.maxTier = agent.TierPropose
	spec.reversible = false

	return newReceivableTool(
		spec,
		receivablePlan[*recordDelete, *worker.WorkerEmploymentVerification]{
			request: recordDeleteFrom(paramVerificationID),
			plan: func(
				ctx context.Context,
				req *recordDelete,
				_ *serviceports.ToolExecuteParams,
			) (*worker.WorkerEmploymentVerification, error) {
				return dqf.PlanDeleteVerification(ctx, req.tenant, req.id)
			},
			refused: func(*recordDelete) string { return "Would remove a previous employer." },
			render: func(
				_ *recordDelete,
				entity *worker.WorkerEmploymentVerification,
			) (*agent.ToolPreview, error) {
				change, err := toolpreview.Delete(verificationRecord(entity), entity,
					wfOptions(verificationFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would remove %s from the qualification file.",
					entity.EmployerName,
				), change), nil
			},
			run: func(
				ctx context.Context,
				req *recordDelete,
				_ *serviceports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				return nil, dqf.DeleteVerification(ctx, req.tenant, req.id, req.userID)
			},
		},
	)
}

func provideRecordEmploymentVerificationTool(s *workerdqfservice.Service) serviceports.AgentTool {
	return newRecordEmploymentVerificationTool(s)
}

func provideUpdateEmploymentVerificationTool(s *workerdqfservice.Service) serviceports.AgentTool {
	return newUpdateEmploymentVerificationTool(s)
}

func provideLogEmploymentVerificationRequestTool(
	s *workerdqfservice.Service,
) serviceports.AgentTool {
	return newLogEmploymentVerificationRequestTool(s)
}

func provideDeleteEmploymentVerificationTool(s *workerdqfservice.Service) serviceports.AgentTool {
	return newDeleteEmploymentVerificationTool(s)
}
