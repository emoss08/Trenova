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
	"github.com/emoss08/trenova/internal/core/services/workerinjuryservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramInjuryID       = "injuryId"
	paramClassification = "classification"
	paramIllnessType    = "illnessType"
	paramTreatment      = "treatment"
	paramInjuryStatus   = "status"
	paramReportedAt     = "reportedAt"
	paramReturnedAt     = "returnedToWorkAt"
	paramBodyPart       = "bodyPart"
	paramHarmfulAgent   = "harmfulAgent"
	paramDaysAway       = "daysAway"
	paramDaysRestricted = "daysRestricted"
	paramPrivacyCase    = "privacyCase"
	paramClaimStatus    = "claimStatus"
	paramClaimNumber    = "claimNumber"
	paramClaimCarrier   = "claimCarrier"
	paramClaimFiledAt   = "claimFiledAt"
	paramClaimClosedAt  = "claimClosedAt"
	kindInjury          = "injury case"
	maxInjuryDays       = 180
	maxClaimField       = 150
)

var (
	oshaClassifications = agenttoolschema.Source(
		"worker.oshaCaseClassification",
		worker.OSHACaseClassificationValues(),
	)
	oshaIllnessTypes = agenttoolschema.Source(
		"worker.oshaIllnessType",
		worker.OSHAIllnessTypeValues(),
	)
	injuryTreatments = agenttoolschema.Source(
		"worker.injuryTreatment",
		worker.InjuryTreatmentValues(),
	)
	injuryStatuses = agenttoolschema.Source(
		"worker.injuryCaseStatus",
		worker.InjuryCaseStatusValues(),
	)
	claimStatuses = agenttoolschema.Source(
		"worker.workersCompClaimStatus",
		worker.WorkersCompClaimStatusValues(),
	)
	injuryFields = []string{
		wfFieldWorkerID, "caseNumber", "caseYear", paramClassification, paramIllnessType,
		paramTreatment, fieldStatus, wfParamOccurred, "reportedAt", "returnedToWorkAt",
		paramEventLocation, fieldDescription, paramBodyPart, paramHarmfulAgent, paramDaysAway,
		paramDaysRestricted, paramPrivacyCase, paramClaimStatus, paramClaimNumber,
		paramClaimCarrier, "claimFiledAt", "claimClosedAt", paramSafetyEventID,
		wfFieldDocument, wfFieldNotes,
	}
)

type injuryKeeper interface {
	PlanRecordInjury(
		ctx context.Context,
		entity *worker.WorkerInjury,
		userID pulid.ID,
	) (*worker.WorkerInjury, error)
	RecordInjury(
		ctx context.Context,
		entity *worker.WorkerInjury,
		userID pulid.ID,
	) (*worker.WorkerInjury, error)
	PlanUpdateInjury(
		ctx context.Context,
		req *workerinjuryservice.UpdateInjuryRequest,
	) (*workerinjuryservice.InjuryChange, error)
	UpdateInjury(
		ctx context.Context,
		req *workerinjuryservice.UpdateInjuryRequest,
	) (*worker.WorkerInjury, error)
	PlanDeleteInjury(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.WorkerInjury, error)
	DeleteInjury(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) error
}

var _ injuryKeeper = (*workerinjuryservice.Service)(nil)

func injuryToolProviders() []any {
	return []any{
		provideRecordWorkerInjuryTool,
		provideUpdateWorkerInjuryTool,
		provideDeleteWorkerInjuryTool,
	}
}

func injuryIDProperty() map[string]any {
	return idProperty("The injury case, from list_worker_injuries. Never guess one.")
}

func injuryRecord(injury *worker.WorkerInjury) toolpreview.Record {
	return wfRecord(permission.ResourceWorkerInjury, injury.ID,
		fmt.Sprintf("Case %d-%d", injury.CaseYear, injury.CaseNumber), injury.Version)
}

func injuryDetailProperties() map[string]any {
	return map[string]any{
		paramClassification: agenttoolschema.Enum("How the case is classified on the OSHA "+
			"300 log. Recordability is the employer's judgement: leave it out unless a person "+
			"gave it, and the case takes the classification the treatment suggests.",
			oshaClassifications),
		paramIllnessType: agenttoolschema.Enum(
			"Injury, or the kind of illness.",
			oshaIllnessTypes,
		),
		paramTreatment:     agenttoolschema.Enum("The most care it took.", injuryTreatments),
		paramReportedAt:    dayProperty("When the worker reported it."),
		paramReturnedAt:    dayProperty("When the worker came back to work."),
		paramEventLocation: stringProperty("Where it happened.", wfShortChars),
		paramBodyPart:      stringProperty("The part of the body affected.", maxCredentialField),
		paramHarmfulAgent: stringProperty("What harmed the worker, such as a load strap.",
			wfShortChars),
		paramDaysAway: integerProperty("Days away from work, capped at 180 as the log counts "+
			"them.", 0, maxInjuryDays),
		paramDaysRestricted: integerProperty("Days on restricted duty or job transfer.", 0,
			maxInjuryDays),
		paramPrivacyCase: booleanProperty("Whether the worker's name is withheld from the " +
			"log as a privacy case."),
		paramClaimStatus: agenttoolschema.Enum(
			"Where the workers' comp claim stands.",
			claimStatuses,
		),
		paramClaimNumber:  stringProperty("The claim number.", maxCredentialField),
		paramClaimCarrier: stringProperty("The workers' comp carrier.", maxClaimField),
		paramClaimFiledAt: dayProperty("When the claim was filed."),
		paramSafetyEventID: idProperty(
			"The accident it came from, from list_worker_safety_events.",
		),
		wfParamDocument: wfDocumentProperty(),
		fieldNotes:      wfNoteProperty("Anything the case should say."),
	}
}

func injuryUpdateFrom(
	params map[string]any,
	req *workerinjuryservice.UpdateInjuryRequest,
) error {
	if err := injuryEnumsFrom(params, req); err != nil {
		return err
	}
	if err := injuryTextFrom(params, req); err != nil {
		return err
	}
	if err := injuryDaysFrom(params, req); err != nil {
		return err
	}
	var err error
	if req.PrivacyCase, err = optionalBoolPointer(params, paramPrivacyCase); err != nil {
		return err
	}
	if req.SafetyEventID, err = optionalID(params, paramSafetyEventID); err != nil {
		return err
	}
	req.DocumentID, err = optionalID(params, wfParamDocument)
	return err
}

func injuryEnumsFrom(params map[string]any, req *workerinjuryservice.UpdateInjuryRequest) error {
	if value, given, err := optionalEnum(params, paramClassification,
		oshaClassifications.Values); err != nil {
		return err
	} else if given {
		req.Classification = &value
	}
	if value, given, err := optionalEnum(params, paramIllnessType,
		oshaIllnessTypes.Values); err != nil {
		return err
	} else if given {
		req.IllnessType = &value
	}
	if value, given, err := optionalEnum(params, paramTreatment,
		injuryTreatments.Values); err != nil {
		return err
	} else if given {
		req.Treatment = &value
	}
	if value, given, err := optionalEnum(params, paramInjuryStatus,
		injuryStatuses.Values); err != nil {
		return err
	} else if given {
		req.Status = &value
	}
	if value, given, err := optionalEnum(params, paramClaimStatus,
		claimStatuses.Values); err != nil {
		return err
	} else if given {
		req.ClaimStatus = &value
	}
	return nil
}

func injuryTextFrom(params map[string]any, req *workerinjuryservice.UpdateInjuryRequest) error {
	for _, field := range []struct {
		key   string
		dest  **string
		limit int
	}{
		{paramEventLocation, &req.Location, wfShortChars},
		{fieldDescription, &req.Description, wfNoteChars},
		{paramBodyPart, &req.BodyPart, maxCredentialField},
		{paramHarmfulAgent, &req.HarmfulAgent, wfShortChars},
		{paramClaimNumber, &req.ClaimNumber, maxCredentialField},
		{paramClaimCarrier, &req.ClaimCarrier, maxClaimField},
		{fieldNotes, &req.Notes, wfNoteChars},
	} {
		text, err := optionalBoundedText(params, field.key, field.limit)
		if err != nil {
			return err
		}
		*field.dest = text
	}
	return nil
}

func injuryDaysFrom(params map[string]any, req *workerinjuryservice.UpdateInjuryRequest) error {
	var err error
	for _, field := range []struct {
		key  string
		dest **int64
	}{
		{paramReportedAt, &req.ReportedAt},
		{paramReturnedAt, &req.ReturnedToWorkAt},
		{paramClaimFiledAt, &req.ClaimFiledAt},
		{paramClaimClosedAt, &req.ClaimClosedAt},
	} {
		if *field.dest, err = optionalScheduleDay(params, field.key); err != nil {
			return err
		}
	}
	for _, field := range []struct {
		key  string
		dest **int32
	}{
		{paramDaysAway, &req.DaysAway},
		{paramDaysRestricted, &req.DaysRestricted},
	} {
		days, dayErr := optionalIntInRange(params, field.key, 0, maxInjuryDays)
		if dayErr != nil {
			return dayErr
		}
		if days != nil {
			value := int32(*days) //nolint:gosec // bounded by the schema's maximum
			*field.dest = &value
		}
	}
	return nil
}

func newRecordWorkerInjuryTool(injuries injuryKeeper) serviceports.AgentTool {
	properties := injuryDetailProperties()
	properties[paramWorkerID] = workerProperty()
	properties[wfParamOccurred] = dateTimeProperty("When it happened.")
	properties[fieldDescription] = stringProperty("What happened and the injury, as "+
		"reported.", wfNoteChars)
	spec := withSchema(wfSpec(
		"record_worker_injury",
		"Enter a work-related injury or illness on the OSHA injury log as a new case. It "+
			"takes the next case number for its year and records when and how it happened, "+
			"the treatment, the days away or restricted, and the workers' comp claim. Medical detail stays with "+
			"people who hold the injury permission.",
		"Adds a case to the organization's OSHA log inside Trenova; nothing is sent, and it "+
			"is corrected with update_worker_injury.",
		permission.ResourceWorkerInjury,
		permission.OpCreate,
	), properties, paramWorkerID, wfParamOccurred, fieldDescription)
	spec.searchTerms = []string{"OSHA", "injury", "workers comp", "300 log"}

	return newReportingReceivableTool(
		spec,
		receivablePlan[*worker.WorkerInjury, *worker.WorkerInjury]{
			request: func(params *serviceports.ToolExecuteParams) (*worker.WorkerInjury, error) {
				workerID, err := requirePulid(params.Params, paramWorkerID)
				if err != nil {
					return nil, err
				}
				occurred, err := requireDateTime(params.Params, wfParamOccurred)
				if err != nil {
					return nil, err
				}
				req := &workerinjuryservice.UpdateInjuryRequest{}
				if err = injuryUpdateFrom(params.Params, req); err != nil {
					return nil, err
				}
				if req.Description == nil || *req.Description == "" {
					return nil, fmt.Errorf("missing required parameter %q", fieldDescription)
				}
				entity := &worker.WorkerInjury{
					OrganizationID: params.OrganizationID,
					BusinessUnitID: params.BusinessUnitID,
					WorkerID:       workerID,
					OccurredAt:     occurred,
				}
				workerinjuryservice.ApplyInjuryUpdate(entity, req)
				return entity, nil
			},
			plan: func(
				ctx context.Context,
				entity *worker.WorkerInjury,
				params *serviceports.ToolExecuteParams,
			) (*worker.WorkerInjury, error) {
				return injuries.PlanRecordInjury(ctx, entity, params.Actor.UserID)
			},
			refused: func(*worker.WorkerInjury) string {
				return "Would enter an injury case on the log."
			},
			render: func(_ *worker.WorkerInjury, planned *worker.WorkerInjury) (*agent.ToolPreview, error) {
				change, err := toolpreview.Create(
					wfRecord(permission.ResourceWorkerInjury, pulid.Nil,
						fmt.Sprintf("Case %d-%d", planned.CaseYear, planned.CaseNumber), 0),
					planned, wfOptions(injuryFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would enter case %d-%d on the injury log, classified %s.", planned.CaseYear,
					planned.CaseNumber, planned.Classification), change), nil
			},
			run: func(
				ctx context.Context,
				entity *worker.WorkerInjury,
				params *serviceports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				created, err := injuries.RecordInjury(ctx, entity, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return wfResult("recorded", kindInjury, paramInjuryID, created.ID,
					created.WorkerID), nil
			},
		},
	)
}

func newUpdateWorkerInjuryTool(injuries injuryKeeper) serviceports.AgentTool {
	properties := injuryDetailProperties()
	properties[paramInjuryID] = injuryIDProperty()
	properties[paramInjuryStatus] = agenttoolschema.Enum("Open or Closed.", injuryStatuses)
	properties[paramClaimClosedAt] = dayProperty("When the claim closed.")
	properties[wfParamOccurred] = dateTimeProperty("When it happened.")
	properties[fieldDescription] = stringProperty("What happened and the injury.", wfNoteChars)
	spec := targeting(withSchema(wfSpec(
		"update_worker_injury",
		"Correct or bring up to date an injury case: its classification, treatment, days "+
			"away or restricted, return to work, the claim, or close it. Give only what "+
			"changes. The log must stay accurate for five years after its year, so a closed "+
			"case is still corrected.",
		"Corrects a case on the OSHA log inside Trenova; nothing is sent, and it is "+
			"corrected again the same way.",
		permission.ResourceWorkerInjury,
		permission.OpUpdate,
	), properties, paramInjuryID), paramInjuryID, permission.ResourceWorkerInjury)

	return newReportingReceivableTool(spec, receivablePlan[
		*workerinjuryservice.UpdateInjuryRequest, *workerinjuryservice.InjuryChange,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*workerinjuryservice.UpdateInjuryRequest, error) {
			id, err := requirePulid(params.Params, paramInjuryID)
			if err != nil {
				return nil, err
			}
			req := &workerinjuryservice.UpdateInjuryRequest{
				TenantInfo: tenantFrom(*params),
				InjuryID:   id,
				UserID:     params.Actor.UserID,
			}
			if req.OccurredAt, err = optionalDateTime(params.Params, wfParamOccurred); err != nil {
				return nil, err
			}
			return req, injuryUpdateFrom(params.Params, req)
		},
		plan: func(
			ctx context.Context,
			req *workerinjuryservice.UpdateInjuryRequest,
			_ *serviceports.ToolExecuteParams,
		) (*workerinjuryservice.InjuryChange, error) {
			return injuries.PlanUpdateInjury(ctx, req)
		},
		refused: func(*workerinjuryservice.UpdateInjuryRequest) string {
			return "Would correct an injury case."
		},
		render: func(
			_ *workerinjuryservice.UpdateInjuryRequest,
			change *workerinjuryservice.InjuryChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(injuryRecord(change.Before), change.Before,
				change.After, wfOptions(injuryFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would correct case %d-%d.",
				change.Before.CaseYear, change.Before.CaseNumber), recorded), nil
		},
		run: func(
			ctx context.Context,
			req *workerinjuryservice.UpdateInjuryRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := injuries.UpdateInjury(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("updated", kindInjury, paramInjuryID, updated.ID,
				updated.WorkerID), nil
		},
	})
}

func newDeleteWorkerInjuryTool(injuries injuryKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"delete_worker_injury",
		"Delete an injury case entered on the log in error, such as a duplicate. Its case "+
			"number is not reused. A real case is corrected, never deleted.",
		"Removes a case from the OSHA log; the audit trail keeps what was removed, and the "+
			"number is not given out again.",
		permission.ResourceWorkerInjury,
		permission.OpDelete,
	), map[string]any{paramInjuryID: injuryIDProperty()}, paramInjuryID), paramInjuryID,
		permission.ResourceWorkerInjury)
	spec.searchTerms = []string{"duplicate injury", "remove injury case"}
	spec.maxTier = agent.TierPropose
	spec.reversible = false

	return newReceivableTool(spec, receivablePlan[*recordDelete, *worker.WorkerInjury]{
		request: recordDeleteFrom(paramInjuryID),
		plan: func(
			ctx context.Context,
			req *recordDelete,
			_ *serviceports.ToolExecuteParams,
		) (*worker.WorkerInjury, error) {
			return injuries.PlanDeleteInjury(ctx, req.tenant, req.id)
		},
		refused: func(*recordDelete) string { return "Would delete an injury case." },
		render: func(_ *recordDelete, injury *worker.WorkerInjury) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(injuryRecord(injury), injury,
				wfOptions(injuryFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would delete case %d-%d from the injury log.", injury.CaseYear,
				injury.CaseNumber), change), nil
		},
		run: func(
			ctx context.Context,
			req *recordDelete,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return nil, injuries.DeleteInjury(ctx, req.tenant, req.id, req.userID)
		},
	})
}

func provideRecordWorkerInjuryTool(s *workerinjuryservice.Service) serviceports.AgentTool {
	return newRecordWorkerInjuryTool(s)
}

func provideUpdateWorkerInjuryTool(s *workerinjuryservice.Service) serviceports.AgentTool {
	return newUpdateWorkerInjuryTool(s)
}

func provideDeleteWorkerInjuryTool(s *workerinjuryservice.Service) serviceports.AgentTool {
	return newDeleteWorkerInjuryTool(s)
}
