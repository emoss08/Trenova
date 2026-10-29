package workforcequerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

const (
	toolListTrainingCourses = "worker_list_training_courses"
	trainingCoursesSummary  = "List the organization's training courses with their category, " +
		"delivery, passing score, how long a pass lasts and whether they are required. " +
		"It yields the courseId the training tools take."
	noteActiveCourses = "Defaults to Active, the courses in use; name Inactive too for " +
		"retired ones."
)

// TrainingReadSpecs are the training catalog, read through the operation
// the training page sends. The courses a call reads by default are the
// active ones, as the repository tool read.
func TrainingReadSpecs() []readspec.List {
	return []readspec.List{{
		Name:         toolListTrainingCourses,
		EntityPlural: "training courses",
		Summary:      trainingCoursesSummary,
		Resource:     permission.ResourceTrainingCourse,
		Notes:        map[string]string{querytool.ParamStatus: noteActiveCourses},
		Defaults:     map[string]any{querytool.ParamStatus: []any{string(querytool.ActiveStatus)}},
	}}
}
