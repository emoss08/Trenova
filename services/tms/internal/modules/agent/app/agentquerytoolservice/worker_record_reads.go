package agentquerytoolservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
)

const (
	toolListWorkerChecklists         = "worker_list_checklists"
	toolListWorkerTraining           = "worker_list_training"
	toolListWorkerPerformanceReviews = "worker_list_performance_reviews"

	opDeskChecklistTemplates = "DeskWorkerChecklistTemplates"
	opDeskReviewTemplates    = "DeskPerformanceReviewTemplates"

	inputStatus  = "status"
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
	noteOpenOnly = "Defaults to false, the open ones only."

	toolListWorkerInjuries = "worker_list_injuries"
	toolListWorkerDOTTests = "worker_list_dot_tests"

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
	toolListWorkerSafetyEvents = "worker_list_safety_events"
	toolWorkerListCredentials  = "worker_list_credentials"
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

	workerCredentialsSummary = "List worker credentials: one worker's licences, medical " +
		"cards and endorsements, or those falling due across the organization. With a worker " +
		"and no expiry window it returns every credential the worker holds, with its dates " +
		"and whether it is verified, and the credential types the organization tracks; a " +
		"required type with nothing on file was never recorded, which is worse than expiring " +
		"soon. Without a worker, or with expiringWithinDays or includeExpired, it lists " +
		"active credentials expiring within that many days from now, soonest first, a page " +
		"at a time: the tool for any " +
		"question about who is expiring, lapsed or out of compliance by a date. Narrow to one " +
		"kind with credentialTypeCode, such as MED_CARD for a medical card or CDL for a " +
		"licence. It yields the credentialId and credentialTypeId the credential tools take."
	noteFallingDue   = "Leave it out for the credentials falling due across the organization."
	noteEveryWorker  = "Leave it out for every worker's."
	fieldPrivacyCase = "privacyCase"
	fieldWorker      = "worker"
)

// workerReference is a worker id parameter: a worker, from the worker list.
var workerReference = idReference{
	resource: permission.ResourceWorker,
	what:     "The worker",
	source:   toolWorkerList,
}

// theWorker is the worker a per-worker list is keyed by.
var theWorker = map[string]idReference{paramWorkerID: workerReference}

// listDefaults are what a call to a list keyed by an optional worker reads
// with when it names no limit: a page, as the repository tools read.
var listDefaults = map[string]any{paramLimit: graphQLListLimit}

// workerRecordReadSpecs are the lists of one worker's own records, each read
// whole through the operation the worker's page sends, keyed by the worker.
// A list that yields the id of a template its next call starts from reads
// the active templates alongside, where the reader may read them, as the
// repository tool did.
func workerRecordReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		{
			name:         toolListWorkerChecklists,
			entityPlural: "checklists",
			summary:      workerChecklistsSummary,
			resource:     permission.ResourceWorkerChecklist,
			references:   theWorker,
			notes:        map[string]string{wfParamClosed: noteOpenOnly},
			defaults:     map[string]any{wfParamClosed: false},
			alongside: []alongsideSpec{{
				key:       "templatesStartedByHand",
				operation: opDeskChecklistTemplates,
				what:      "checklist templates",
				inputs: map[string]any{
					inputStatus:  activeStatus,
					inputTrigger: string(worker.ChecklistTriggerManual),
				},
			}},
		},
		{
			name:         toolListWorkerTraining,
			entityPlural: "training assignments",
			summary:      workerTrainingSummary,
			resource:     permission.ResourceWorkerTraining,
			references:   theWorker,
			notes:        map[string]string{wfParamClosed: noteOpenOnly},
			defaults:     map[string]any{wfParamClosed: false},
		},
		{
			name:         toolListWorkerPerformanceReviews,
			entityPlural: "performance reviews",
			summary:      workerReviewsSummary,
			resource:     permission.ResourcePerformanceReview,
			references:   theWorker,
			alongside: []alongsideSpec{{
				key:       "templates",
				operation: opDeskReviewTemplates,
				what:      "review templates",
				inputs:    map[string]any{inputStatus: activeStatus},
			}},
		},
		{
			name:         toolListWorkerInjuries,
			entityPlural: "injury cases",
			summary:      workerInjuriesSummary,
			resource:     permission.ResourceWorkerInjury,
			references:   theWorker,
			notes:        map[string]string{paramWorkerID: noteEveryWorker},
			defaults:     listDefaults,
			shape:        withholdPrivacyCaseNames,
		},
		{
			name:         toolListWorkerDOTTests,
			entityPlural: "drug and alcohol tests",
			summary:      workerDOTTestsSummary,
			resource:     permission.ResourceWorkerDOTTest,
			references:   theWorker,
			notes:        map[string]string{paramWorkerID: noteEveryWorker},
			defaults:     listDefaults,
		},
		{
			name:         toolListWorkerLeaveCases,
			entityPlural: "leave cases",
			summary:      workerLeaveCasesSummary,
			resource:     permission.ResourceWorkerLeave,
			references:   theWorker,
			notes:        map[string]string{paramWorkerID: noteEveryWorker},
			defaults:     listDefaults,
		},
		{
			name:         toolListWorkerSafetyEvents,
			entityPlural: "safety events",
			summary:      workerSafetyEventsSummary,
			resource:     permission.ResourceWorkerSafetyEvent,
			references:   theWorker,
			notes: map[string]string{
				paramWorkerID:   noteEveryWorker,
				wfParamOpenOnly: noteOpenAcrossTheFleet,
			},
			defaults: listDefaults,
			unscoped: map[string]any{wfParamOpenOnly: true},
			alongside: []alongsideSpec{{
				key:       "recognitions",
				operation: opDeskWorkerRecognitions,
				what:      "recognitions",
			}},
		},
		{
			name:         toolWorkerListCredentials,
			entityPlural: credentialEntity,
			summary:      workerCredentialsSummary,
			searchTerms: []string{
				"expiring licenses", "credentials due", "about to lapse", "lapsed",
			},
			resource:   permission.ResourceWorkerCredential,
			references: theWorker,
			notes:      map[string]string{paramWorkerID: noteFallingDue},
			defaults: map[string]any{
				paramLimit:           graphQLListLimit,
				paramIncludeArchived: false,
			},
			alongside: []alongsideSpec{{
				key:       "credentialTypes",
				operation: opDeskCredentialTypes,
				what:      "credential types",
				inputs:    map[string]any{inputStatus: activeStatus},
				scoped:    true,
			}},
		},
		{
			name:         toolListEmploymentVerifications,
			entityPlural: "previous employers",
			summary:      employmentVerificationsSummary,
			resource:     permission.ResourceQualification,
			references:   theWorker,
			notes:        map[string]string{paramWorkerID: noteEveryWorker},
			defaults:     listDefaults,
		},
	}
}

// withholdPrivacyCaseNames leaves out the worker of a privacy concern case
// (29 CFR 1904.29(b)(6)), whose name the log itself withholds, as the
// repository tool did; the case keeps its workerId for the tools that act on
// it.
func withholdPrivacyCaseNames(row map[string]any, _ map[string]struct{}, _ clock) {
	if private, _ := row[fieldPrivacyCase].(bool); private {
		delete(row, fieldWorker)
	}
}
