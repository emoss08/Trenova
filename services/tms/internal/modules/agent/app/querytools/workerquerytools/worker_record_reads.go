package workerquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

const (
	ToolListWorkerChecklists         = "worker_list_checklists"
	ToolListWorkerTraining           = "worker_list_training"
	toolListWorkerPerformanceReviews = "worker_list_performance_reviews"

	opDeskChecklistTemplates = "DeskWorkerChecklistTemplates"
	OpDeskReviewTemplates    = "DeskPerformanceReviewTemplates"

	inputTrigger = "trigger"

	workerChecklistsSummary = "List one worker's onboarding, offboarding and other checklists " +
		"with every item and where it stands, and the templates a person can start by " +
		"hand. It yields the checklistId, checklistItemId and templateId the checklist " +
		"tools take."
	workerTrainingSummary = "List one worker's training assignments with each course, its " +
		"status, due date, completion, score and when it expires. It yields the " +
		"trainingRecordId the training tools take."
	workerReviewsSummary = "List one worker's performance reviews with each item's key, score " +
		"and comment, the summary and goals, and the review templates in use. It " +
		"yields the reviewId and templateId the review tools take, and the rating " +
		"keys worker_update_performance_review scores."
	NoteOpenOnly = "Defaults to false, the open ones only."

	ToolListWorkerInjuries = "worker_list_injuries"
	ToolListWorkerDOTTests = "worker_list_dot_tests"

	workerInjuriesSummary = "List injury and illness cases on the OSHA log, for one worker or a " +
		"year, with the classification, days away or restricted and the workers' comp " +
		"claim. Medical detail follows the injury permission. It yields the injuryId " +
		"the injury tools take."
	workerDOTTestsSummary = "List drug and alcohol tests, newest first: one worker's, or every " +
		"collection still open across the organization. Each row has the test type, why it " +
		"was ordered, its status and dates; the substance and result are shown only where " +
		"access reaches Restricted. It yields the testId worker_cancel_dot_test takes."
	toolListWorkerLeaveCases        = "worker_list_leave_cases"
	toolListEmploymentVerifications = "worker_list_employment_verifications"

	workerLeaveCasesSummary = "List FMLA, medical, military, parental and personal leave cases. " +
		"Cases for one worker or the whole organization come with their status, " +
		"certification and each day taken against them. It yields the leaveCaseId and " +
		"leaveDayId the leave tools take."
	employmentVerificationsSummary = "List the previous employers on drivers' qualification " +
		"files. Each shows where its safety performance history request stands: asked when, " +
		"followed up how often, and what came back. It yields the verificationId the " +
		"verification tools take."
	ToolListWorkerSafetyEvents = "worker_list_safety_events"
	ToolWorkerListCredentials  = "worker_list_credentials"
	credentialEntity           = "credentials"

	opDeskWorkerRecognitions = "DeskWorkerRecognitions"

	workerSafetyEventsSummary = "List safety events newest first: one worker's, or, leaving " +
		"out the worker, those still open across the fleet. One worker's come with their " +
		"roadside violations and recognitions; the fleet's each name the driver. Events are " +
		"accidents, inspections, citations, complaints and near misses with their points and " +
		"status. It yields the safetyEventId, violationId and recognitionId the safety tools take."
	noteOpenAcrossTheFleet = "Defaults to true when no worker is named, and to false for one " +
		"worker's; set it to false for the fleet's closed events too."
	opDeskCredentialTypes = "DeskWorkerCredentialTypes"

	workerCredentialsSummary = "List worker credentials: the licences, medical cards and " +
		"endorsements workers hold, those falling due, or who holds a type. With workerIds " +
		"and no expiry window: every credential those workers hold, in one call, with dates, " +
		"verification and the organization's types; a required type with nothing on file " +
		"was never recorded, worse than expiring soon. With expiringWithinDays or " +
		"includeExpired, or no worker and no type: active credentials expiring within that " +
		"many days, soonest first, paged: who is expiring, lapsed or out of compliance by a " +
		"date. credentialTypeCode alone: who holds that type. Who lacks one: worker_list " +
		"withoutCredential, never a call per worker. Each row's worker says if " +
		"dispatchable; canBeAssigned false is already off dispatch: no hold. It yields the " +
		"credentialId and credentialTypeId the credential tools take."
	noteFallingDue = "Leave it out for the credentials across the organization."
	paramWorkerIDs = "workerIds"

	paramCredentialTypeCode = "credentialTypeCode"
	noteEveryWorker         = "Leave it out for every worker's."
	fieldPrivacyCase        = "privacyCase"
	fieldWorker             = "worker"
)

// RecordReadSpecs are the lists of one worker's own records, each read
// whole through the operation the worker's page sends, keyed by the worker.
// A list that yields the id of a template its next call starts from reads
// the active templates alongside, where the reader may read them, as the
// repository tool did.
func RecordReadSpecs() []readspec.List {
	return []readspec.List{
		{
			Name:         ToolListWorkerChecklists,
			EntityPlural: "checklists",
			Summary:      workerChecklistsSummary,
			Resource:     permission.ResourceWorkerChecklist,
			References:   readspec.TheWorker,
			Notes:        map[string]string{querytool.WfParamClosed: NoteOpenOnly},
			Defaults:     map[string]any{querytool.WfParamClosed: false},
			Alongside: []readspec.Alongside{{
				Key:       "templatesStartedByHand",
				Operation: opDeskChecklistTemplates,
				What:      "checklist templates",
				Inputs: map[string]any{
					querytool.InputStatus: querytool.ActiveStatus,
					inputTrigger:          string(workerchecklist.ChecklistTriggerManual),
				},
			}},
		},
		{
			Name:         ToolListWorkerTraining,
			EntityPlural: "training assignments",
			Summary:      workerTrainingSummary,
			Resource:     permission.ResourceWorkerTraining,
			References:   readspec.TheWorker,
			Notes:        map[string]string{querytool.WfParamClosed: NoteOpenOnly},
			Defaults:     map[string]any{querytool.WfParamClosed: false},
		},
		{
			Name:         toolListWorkerPerformanceReviews,
			EntityPlural: "performance reviews",
			Summary:      workerReviewsSummary,
			Resource:     permission.ResourcePerformanceReview,
			References:   readspec.TheWorker,
			Alongside: []readspec.Alongside{{
				Key:       "templates",
				Operation: OpDeskReviewTemplates,
				What:      "review templates",
				Inputs:    map[string]any{querytool.InputStatus: querytool.ActiveStatus},
			}},
		},
		{
			Name:         ToolListWorkerInjuries,
			EntityPlural: "injury cases",
			Summary:      workerInjuriesSummary,
			Resource:     permission.ResourceWorkerInjury,
			References:   readspec.TheWorker,
			Notes:        map[string]string{querytool.ParamWorkerID: noteEveryWorker},
			Defaults:     querytool.ListDefaults,
			Shape:        withholdPrivacyCaseNames,
		},
		{
			Name:         ToolListWorkerDOTTests,
			EntityPlural: "drug and alcohol tests",
			Summary:      workerDOTTestsSummary,
			Resource:     permission.ResourceWorkerDOTTest,
			References:   readspec.TheWorker,
			Notes:        map[string]string{querytool.ParamWorkerID: noteEveryWorker},
			Defaults:     querytool.ListDefaults,
		},
		{
			Name:         toolListWorkerLeaveCases,
			EntityPlural: "leave cases",
			Summary:      workerLeaveCasesSummary,
			Resource:     permission.ResourceWorkerLeave,
			References:   readspec.TheWorker,
			Notes:        map[string]string{querytool.ParamWorkerID: noteEveryWorker},
			Defaults:     querytool.ListDefaults,
		},
		{
			Name:         ToolListWorkerSafetyEvents,
			EntityPlural: "safety events",
			Summary:      workerSafetyEventsSummary,
			Resource:     permission.ResourceWorkerSafetyEvent,
			References:   readspec.TheWorker,
			Notes: map[string]string{
				querytool.ParamWorkerID:   noteEveryWorker,
				querytool.WfParamOpenOnly: noteOpenAcrossTheFleet,
			},
			Defaults: querytool.ListDefaults,
			Unscoped: map[string]any{querytool.WfParamOpenOnly: true},
			Alongside: []readspec.Alongside{{
				Key:       "recognitions",
				Operation: opDeskWorkerRecognitions,
				What:      "recognitions",
			}},
		},
		{
			Name:         ToolWorkerListCredentials,
			EntityPlural: credentialEntity,
			Summary:      workerCredentialsSummary,
			SearchTerms: []string{
				"expiring licenses", "credentials due", "about to lapse", "lapsed",
			},
			Resource:   permission.ResourceWorkerCredential,
			References: map[string]readspec.Reference{paramWorkerIDs: readspec.WorkerReference},
			Notes:      map[string]string{paramWorkerIDs: noteFallingDue},
			Defaults: map[string]any{
				querytool.ParamLimit: querytool.GraphQLListLimit,
				paramIncludeArchived: false,
			},
			Shape: shapeCredentialWorker,
			Alongside: []readspec.Alongside{{
				Key:       "credentialTypes",
				Operation: opDeskCredentialTypes,
				What:      "credential types",
				Inputs:    map[string]any{querytool.InputStatus: querytool.ActiveStatus},
				Scoped:    true,
				Unless:    []string{paramCredentialTypeCode},
			}},
		},
		{
			Name:         toolListEmploymentVerifications,
			EntityPlural: "previous employers",
			Summary:      employmentVerificationsSummary,
			Resource:     permission.ResourceQualification,
			References:   readspec.TheWorker,
			Notes:        map[string]string{querytool.ParamWorkerID: noteEveryWorker},
			Defaults:     querytool.ListDefaults,
		},
	}
}

// withholdPrivacyCaseNames leaves out the worker of a privacy concern case
// (29 CFR 1904.29(b)(6)), whose name the log itself withholds, as the
// repository tool did; the case keeps its workerId for the tools that act on
// it.
func withholdPrivacyCaseNames(row map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
	if private, _ := row[fieldPrivacyCase].(bool); private {
		delete(row, fieldWorker)
	}
}

func shapeCredentialWorker(row map[string]any, _ map[string]struct{}, _ filtercatalog.Clock) {
	if held, ok := row[fieldWorker].(map[string]any); ok {
		dispatchStanding(held)
	}
}
