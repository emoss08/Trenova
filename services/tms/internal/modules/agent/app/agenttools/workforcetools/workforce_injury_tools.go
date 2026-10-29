package workforcetools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerinjuryservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerinjury"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramInjuryID = "injuryId"

const paramClassification = "classification"

const paramIllnessType = "illnessType"

const paramTreatment = "treatment"

const paramInjuryStatus = "status"

const paramReportedAt = "reportedAt"

const paramReturnedAt = "returnedToWorkAt"

const paramBodyPart = "bodyPart"

const paramHarmfulAgent = "harmfulAgent"

const paramDaysAway = "daysAway"

const paramDaysRestricted = "daysRestricted"

const paramPrivacyCase = "privacyCase"

const paramClaimStatus = "claimStatus"

const paramClaimNumber = "claimNumber"

const paramClaimCarrier = "claimCarrier"

const paramClaimFiledAt = "claimFiledAt"

const paramClaimClosedAt = "claimClosedAt"

const kindInjury = "injury case"

const maxInjuryDays = 180

const maxClaimField = 150

var oshaClassifications = agenttoolschema.Source(
	"worker.oshaCaseClassification",
	workerinjury.OSHACaseClassificationValues(),
)

var oshaIllnessTypes = agenttoolschema.Source(
	"worker.oshaIllnessType",
	workerinjury.OSHAIllnessTypeValues(),
)

var injuryTreatments = agenttoolschema.Source(
	"worker.injuryTreatment",
	workerinjury.InjuryTreatmentValues(),
)

var injuryStatuses = agenttoolschema.Source(
	"worker.injuryCaseStatus",
	workerinjury.InjuryCaseStatusValues(),
)

var claimStatuses = agenttoolschema.Source(
	"worker.workersCompClaimStatus",
	workerinjury.WorkersCompClaimStatusValues(),
)

var injuryFields = []string{
	wfFieldWorkerID, "caseNumber", "caseYear", paramClassification, paramIllnessType,
	paramTreatment, changepreview.FieldStatus, wfParamOccurred, "reportedAt", "returnedToWorkAt",
	paramEventLocation, changepreview.FieldDescription, paramBodyPart, paramHarmfulAgent, paramDaysAway,
	paramDaysRestricted, paramPrivacyCase, paramClaimStatus, paramClaimNumber,
	paramClaimCarrier, "claimFiledAt", "claimClosedAt", paramSafetyEventID,
	wfFieldDocument, wfFieldNotes,
}

type InjuryKeeper interface {
	PlanRecordInjury(
		ctx context.Context,
		entity *workerinjury.WorkerInjury,
		userID pulid.ID,
	) (*workerinjury.WorkerInjury, error)
	RecordInjury(
		ctx context.Context,
		entity *workerinjury.WorkerInjury,
		userID pulid.ID,
	) (*workerinjury.WorkerInjury, error)
	PlanUpdateInjury(
		ctx context.Context,
		req *workerinjuryservice.UpdateInjuryRequest,
	) (*workerinjuryservice.InjuryChange, error)
	UpdateInjury(
		ctx context.Context,
		req *workerinjuryservice.UpdateInjuryRequest,
	) (*workerinjury.WorkerInjury, error)
	PlanDeleteInjury(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*workerinjury.WorkerInjury, error)
	DeleteInjury(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) error
}

func InjuryToolProviders() []any {
	return []any{
		provideRecordWorkerInjuryTool,
		provideUpdateWorkerInjuryTool,
		provideDeleteWorkerInjuryTool,
	}
}

func injuryIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceWorkerInjury, "The injury case",
		"worker_list_injuries")
}

func injuryRecord(injury *workerinjury.WorkerInjury) toolpreview.Record {
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
		paramReportedAt:    agenttoolschema.Date("When the worker reported it."),
		paramReturnedAt:    agenttoolschema.Date("When the worker came back to work."),
		paramEventLocation: toolparams.StringProperty("Where it happened.", wfShortChars),
		paramBodyPart: toolparams.StringProperty(
			"The part of the body affected.",
			maxCredentialField,
		),
		paramHarmfulAgent: toolparams.StringProperty(
			"What harmed the worker, such as a load strap.",
			wfShortChars,
		),
		paramDaysAway: toolparams.IntegerProperty(
			"Days away from work, capped at 180 as the log counts "+
				"them.",
			0,
			maxInjuryDays,
		),
		paramDaysRestricted: toolparams.IntegerProperty(
			"Days on restricted duty or job transfer.",
			0,
			maxInjuryDays,
		),
		paramPrivacyCase: toolparams.BooleanProperty(
			"Whether the worker's name is withheld from the " +
				"log as a privacy case.",
		),
		paramClaimStatus: agenttoolschema.Enum(
			"Where the workers' comp claim stands.",
			claimStatuses,
		),
		paramClaimNumber:  toolparams.StringProperty("The claim number.", maxCredentialField),
		paramClaimCarrier: toolparams.StringProperty("The workers' comp carrier.", maxClaimField),
		paramClaimFiledAt: agenttoolschema.Date("When the claim was filed."),
		paramSafetyEventID: agenttoolschema.RecordIDText(permission.ResourceWorkerSafetyEvent,
			"The accident it came from, from worker_list_safety_events.",
		),
		wfParamDocument:      wfDocumentProperty(),
		toolparams.ParamNote: wfNoteProperty("Anything the case should say."),
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
	if req.PrivacyCase, err = toolparams.OptionalBoolPointer(params, paramPrivacyCase); err != nil {
		return err
	}
	if req.SafetyEventID, err = toolparams.OptionalID(params, paramSafetyEventID); err != nil {
		return err
	}
	req.DocumentID, err = toolparams.OptionalID(params, wfParamDocument)
	return err
}

func injuryEnumsFrom(params map[string]any, req *workerinjuryservice.UpdateInjuryRequest) error {
	if value, given, err := toolparams.OptionalEnum(params, paramClassification,
		oshaClassifications.Values); err != nil {
		return err
	} else if given {
		req.Classification = &value
	}
	if value, given, err := toolparams.OptionalEnum(params, paramIllnessType,
		oshaIllnessTypes.Values); err != nil {
		return err
	} else if given {
		req.IllnessType = &value
	}
	if value, given, err := toolparams.OptionalEnum(params, paramTreatment,
		injuryTreatments.Values); err != nil {
		return err
	} else if given {
		req.Treatment = &value
	}
	if value, given, err := toolparams.OptionalEnum(params, paramInjuryStatus,
		injuryStatuses.Values); err != nil {
		return err
	} else if given {
		req.Status = &value
	}
	if value, given, err := toolparams.OptionalEnum(params, paramClaimStatus,
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
		{changepreview.FieldDescription, &req.Description, wfNoteChars},
		{paramBodyPart, &req.BodyPart, maxCredentialField},
		{paramHarmfulAgent, &req.HarmfulAgent, wfShortChars},
		{paramClaimNumber, &req.ClaimNumber, maxCredentialField},
		{paramClaimCarrier, &req.ClaimCarrier, maxClaimField},
		{toolparams.ParamNote, &req.Notes, wfNoteChars},
	} {
		text, err := toolparams.OptionalBoundedText(params, field.key, field.limit)
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
		if *field.dest, err = toolparams.OptionalScheduleDay(params, field.key); err != nil {
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
		days, dayErr := toolparams.OptionalIntInRange(params, field.key, 0, maxInjuryDays)
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

func newRecordWorkerInjuryTool(injuries InjuryKeeper) agenttoolports.AgentTool {
	properties := injuryDetailProperties()
	properties[toolparams.ParamWorkerID] = toolparams.WorkerProperty()
	properties[wfParamOccurred] = agenttoolschema.DateTime("When it happened.")
	properties[changepreview.FieldDescription] = toolparams.StringProperty(
		"What happened and the injury, as "+
			"reported.",
		wfNoteChars,
	)
	spec := withSchema(wfSpec(
		"worker_record_injury",
		"Enter a work-related injury or illness on the OSHA injury log as a new case. It "+
			"takes the next case number for its year and records when and how it happened, "+
			"the treatment, the days away or restricted, and the workers' comp claim. Medical detail stays with "+
			"people who hold the injury permission.",
		"Adds a case to the organization's OSHA log inside Trenova; nothing is sent, and it "+
			"is corrected with worker_update_injury.",
		permission.ResourceWorkerInjury,
		permission.OpCreate,
	), properties, toolparams.ParamWorkerID, wfParamOccurred, changepreview.FieldDescription)
	spec.SearchTerms = []string{"OSHA", "injury", "workers comp", "300 log"}

	return toolspec.NewReportingReceivableTool(
		spec,
		toolspec.ReceivablePlan[*workerinjury.WorkerInjury, *workerinjury.WorkerInjury]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*workerinjury.WorkerInjury, error) {
				workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
				if err != nil {
					return nil, err
				}
				occurred, err := toolparams.RequireDateTime(params.Params, wfParamOccurred)
				if err != nil {
					return nil, err
				}
				req := &workerinjuryservice.UpdateInjuryRequest{}
				if err = injuryUpdateFrom(params.Params, req); err != nil {
					return nil, err
				}
				if req.Description == nil || *req.Description == "" {
					return nil, fmt.Errorf(
						"missing required parameter %q",
						changepreview.FieldDescription,
					)
				}
				entity := &workerinjury.WorkerInjury{
					OrganizationID: params.OrganizationID,
					BusinessUnitID: params.BusinessUnitID,
					WorkerID:       workerID,
					OccurredAt:     occurred,
				}
				workerinjuryservice.ApplyInjuryUpdate(entity, req)
				return entity, nil
			},
			Plan: func(
				ctx context.Context,
				entity *workerinjury.WorkerInjury,
				params *agenttoolports.ToolExecuteParams,
			) (*workerinjury.WorkerInjury, error) {
				return injuries.PlanRecordInjury(ctx, entity, params.Actor.UserID)
			},
			Refused: func(*workerinjury.WorkerInjury) string {
				return "Would enter an injury case on the log."
			},
			Render: func(_ *workerinjury.WorkerInjury, planned *workerinjury.WorkerInjury) (*agentpreview.ToolPreview, error) {
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
			Run: func(
				ctx context.Context,
				entity *workerinjury.WorkerInjury,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
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

func newUpdateWorkerInjuryTool(injuries InjuryKeeper) agenttoolports.AgentTool {
	properties := injuryDetailProperties()
	properties[paramInjuryID] = injuryIDProperty()
	properties[paramInjuryStatus] = agenttoolschema.Enum("Open or Closed.", injuryStatuses)
	properties[paramClaimClosedAt] = agenttoolschema.Date("When the claim closed.")
	properties[wfParamOccurred] = agenttoolschema.DateTime("When it happened.")
	properties[changepreview.FieldDescription] = toolparams.StringProperty(
		"What happened and the injury.",
		wfNoteChars,
	)
	spec := targeting(withSchema(wfSpec(
		"worker_update_injury",
		"Correct or bring up to date an injury case: its classification, treatment, days "+
			"away or restricted, return to work, the claim, or close it. Give only what "+
			"changes. The log must stay accurate for five years after its year, so a closed "+
			"case is still corrected.",
		"Corrects a case on the OSHA log inside Trenova; nothing is sent, and it is "+
			"corrected again the same way.",
		permission.ResourceWorkerInjury,
		permission.OpUpdate,
	), properties, paramInjuryID), paramInjuryID, permission.ResourceWorkerInjury)

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*workerinjuryservice.UpdateInjuryRequest, *workerinjuryservice.InjuryChange,
	]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*workerinjuryservice.UpdateInjuryRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramInjuryID)
			if err != nil {
				return nil, err
			}
			req := &workerinjuryservice.UpdateInjuryRequest{
				TenantInfo: toolparams.TenantFrom(*params),
				InjuryID:   id,
				UserID:     params.Actor.UserID,
			}
			if req.OccurredAt, err = toolparams.OptionalDateTime(
				params.Params,
				wfParamOccurred,
			); err != nil {
				return nil, err
			}
			return req, injuryUpdateFrom(params.Params, req)
		},
		Plan: func(
			ctx context.Context,
			req *workerinjuryservice.UpdateInjuryRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerinjuryservice.InjuryChange, error) {
			return injuries.PlanUpdateInjury(ctx, req)
		},
		Refused: func(*workerinjuryservice.UpdateInjuryRequest) string {
			return "Would correct an injury case."
		},
		Render: func(
			_ *workerinjuryservice.UpdateInjuryRequest,
			change *workerinjuryservice.InjuryChange,
		) (*agentpreview.ToolPreview, error) {
			recorded, err := toolpreview.Changed(injuryRecord(change.Before), change.Before,
				change.After, wfOptions(injuryFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would correct case %d-%d.",
				change.Before.CaseYear, change.Before.CaseNumber), recorded), nil
		},
		Run: func(
			ctx context.Context,
			req *workerinjuryservice.UpdateInjuryRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			updated, err := injuries.UpdateInjury(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("updated", kindInjury, paramInjuryID, updated.ID,
				updated.WorkerID), nil
		},
	})
}

func newDeleteWorkerInjuryTool(injuries InjuryKeeper) agenttoolports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"worker_delete_injury",
		"Delete an injury case entered on the log in error, such as a duplicate. Its case "+
			"number is not reused. A real case is corrected, never deleted.",
		"Removes a case from the OSHA log; the audit trail keeps what was removed, and the "+
			"number is not given out again.",
		permission.ResourceWorkerInjury,
		permission.OpDelete,
	), map[string]any{paramInjuryID: injuryIDProperty()}, paramInjuryID), paramInjuryID,
		permission.ResourceWorkerInjury)
	spec.SearchTerms = []string{"duplicate injury", "remove injury case"}
	spec.MaxTier = agent.TierPropose
	spec.Reversible = false

	return toolspec.NewReceivableTool(
		spec,
		toolspec.ReceivablePlan[*recordDelete, *workerinjury.WorkerInjury]{
			Request: recordDeleteFrom(paramInjuryID),
			Plan: func(
				ctx context.Context,
				req *recordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*workerinjury.WorkerInjury, error) {
				return injuries.PlanDeleteInjury(ctx, req.tenant, req.id)
			},
			Refused: func(*recordDelete) string { return "Would delete an injury case." },
			Render: func(_ *recordDelete, injury *workerinjury.WorkerInjury) (*agentpreview.ToolPreview, error) {
				change, err := toolpreview.Delete(injuryRecord(injury), injury,
					wfOptions(injuryFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would delete case %d-%d from the injury log.", injury.CaseYear,
					injury.CaseNumber), change), nil
			},
			Run: func(
				ctx context.Context,
				req *recordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				return nil, injuries.DeleteInjury(ctx, req.tenant, req.id, req.userID)
			},
		},
	)
}

func provideRecordWorkerInjuryTool(s *workerinjuryservice.Service) agenttoolports.AgentTool {
	return newRecordWorkerInjuryTool(s)
}

func provideUpdateWorkerInjuryTool(s *workerinjuryservice.Service) agenttoolports.AgentTool {
	return newUpdateWorkerInjuryTool(s)
}

func provideDeleteWorkerInjuryTool(s *workerinjuryservice.Service) agenttoolports.AgentTool {
	return newDeleteWorkerInjuryTool(s)
}
