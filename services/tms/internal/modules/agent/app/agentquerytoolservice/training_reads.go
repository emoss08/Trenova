package agentquerytoolservice

import "github.com/emoss08/trenova/internal/modules/identity/domain/permission"

const (
	toolListTrainingCourses = "worker_list_training_courses"
	trainingCoursesSummary  = "List the organization's training courses with their category, " +
		"delivery, passing score, how long a pass lasts and whether they are required. " +
		"It yields the courseId the training tools take."
	noteActiveCourses = "Defaults to Active, the courses in use; name Inactive too for " +
		"retired ones."
)

// trainingReadSpecs are the training catalog, read through the operation
// the training page sends. The courses a call reads by default are the
// active ones, as the repository tool read.
func trainingReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{{
		name:         toolListTrainingCourses,
		entityPlural: "training courses",
		summary:      trainingCoursesSummary,
		resource:     permission.ResourceTrainingCourse,
		notes:        map[string]string{paramStatus: noteActiveCourses},
		defaults:     map[string]any{paramStatus: []any{string(activeStatus)}},
	}}
}
