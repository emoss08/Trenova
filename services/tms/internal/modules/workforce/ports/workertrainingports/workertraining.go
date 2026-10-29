package workertrainingports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListTrainingCoursesRequest struct {
	Filter   *pagination.QueryOptions `json:"filter"`
	Cursor   pagination.CursorInfo    `json:"cursor"`
	Status   string                   `json:"status"`
	Category string                   `json:"category"`
}

type GetTrainingCourseByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetTrainingCoursesByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CourseIDs  []pulid.ID            `json:"courseIds"`
}

type TrainingCourseSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
}

type TrainingCourseCodeExistsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Code       string                `json:"code"`
	ExcludeID  pulid.ID              `json:"excludeId"`
}

type CountTrainingRecordsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CourseID   pulid.ID              `json:"courseId"`
	OpenOnly   bool                  `json:"openOnly"`
}

type CountTrainingRecordsByCourseIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CourseIDs  []pulid.ID            `json:"courseIds"`
	OpenOnly   bool                  `json:"openOnly"`
}

type ListWorkerTrainingRequest struct {
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	WorkerID        pulid.ID              `json:"workerId"`
	IncludeClosed   bool                  `json:"includeClosed"`
	IncludeCourse   bool                  `json:"includeCourse"`
	IncludeDocument bool                  `json:"includeDocument"`
	IncludeActors   bool                  `json:"includeActors"`
}

type GetWorkerTrainingByIDRequest struct {
	ID              pulid.ID              `json:"id"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	IncludeCourse   bool                  `json:"includeCourse"`
	IncludeWorker   bool                  `json:"includeWorker"`
	IncludeDocument bool                  `json:"includeDocument"`
}

// ListTrainingRemindersRequest walks records the nightly sweep cares about:
// open assignments due inside [now - GraceDays, now + HorizonDays] and
// completions that lapse in the same window. A zero TenantInfo crosses every
// tenant; AfterID pages by record id.
type ListTrainingRemindersRequest struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	HorizonDays int                   `json:"horizonDays"`
	GraceDays   int                   `json:"graceDays"`
	AfterID     pulid.ID              `json:"afterId"`
	Limit       int                   `json:"limit"`
}

type WorkerTrainingRepository interface {
	ListCourses(
		ctx context.Context,
		req *ListTrainingCoursesRequest,
	) (*pagination.CursorListResult[*workertraining.TrainingCourse], error)
	ListActiveCourses(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*workertraining.TrainingCourse, error)
	CourseSelectOptions(
		ctx context.Context,
		req *TrainingCourseSelectOptionsRequest,
	) (*pagination.ListResult[*workertraining.TrainingCourse], error)
	GetCourseByID(
		ctx context.Context,
		req *GetTrainingCourseByIDRequest,
	) (*workertraining.TrainingCourse, error)
	GetCoursesByIDs(
		ctx context.Context,
		req *GetTrainingCoursesByIDsRequest,
	) ([]*workertraining.TrainingCourse, error)
	CourseCodeExists(ctx context.Context, req *TrainingCourseCodeExistsRequest) (bool, error)
	CreateCourse(
		ctx context.Context,
		entity *workertraining.TrainingCourse,
	) (*workertraining.TrainingCourse, error)
	UpdateCourse(
		ctx context.Context,
		entity *workertraining.TrainingCourse,
	) (*workertraining.TrainingCourse, error)
	CountRecordsByCourse(ctx context.Context, req *CountTrainingRecordsRequest) (int, error)
	CountRecordsByCourseIDs(
		ctx context.Context,
		req *CountTrainingRecordsByCourseIDsRequest,
	) (map[pulid.ID]int, error)

	ListForWorker(
		ctx context.Context,
		req *ListWorkerTrainingRequest,
	) ([]*workertraining.WorkerTrainingRecord, error)
	GetByID(
		ctx context.Context,
		req *GetWorkerTrainingByIDRequest,
	) (*workertraining.WorkerTrainingRecord, error)
	Create(
		ctx context.Context,
		entity *workertraining.WorkerTrainingRecord,
	) (*workertraining.WorkerTrainingRecord, error)
	Update(
		ctx context.Context,
		entity *workertraining.WorkerTrainingRecord,
	) (*workertraining.WorkerTrainingRecord, error)
	ListDue(
		ctx context.Context,
		req *ListTrainingRemindersRequest,
	) ([]*workertraining.WorkerTrainingRecord, error)
	ListExpiring(
		ctx context.Context,
		req *ListTrainingRemindersRequest,
	) ([]*workertraining.WorkerTrainingRecord, error)
}
