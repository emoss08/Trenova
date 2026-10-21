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
)

// workerReference is a worker id parameter: a worker, from the worker list.
var workerReference = idReference{
	resource: permission.ResourceWorker,
	what:     "The worker",
	source:   toolWorkerList,
}

// theWorker is the worker a per-worker list is keyed by.
var theWorker = map[string]idReference{paramWorkerID: workerReference}

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
	}
}
