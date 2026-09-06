package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
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
	) (*pagination.CursorListResult[*worker.TrainingCourse], error)
	ListActiveCourses(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*worker.TrainingCourse, error)
	GetCourseByID(
		ctx context.Context,
		req *GetTrainingCourseByIDRequest,
	) (*worker.TrainingCourse, error)
	CourseCodeExists(ctx context.Context, req *TrainingCourseCodeExistsRequest) (bool, error)
	CreateCourse(ctx context.Context, entity *worker.TrainingCourse) (*worker.TrainingCourse, error)
	UpdateCourse(ctx context.Context, entity *worker.TrainingCourse) (*worker.TrainingCourse, error)
	CountRecordsByCourse(ctx context.Context, req *CountTrainingRecordsRequest) (int, error)
	CountRecordsByCourseIDs(
		ctx context.Context,
		req *CountTrainingRecordsByCourseIDsRequest,
	) (map[pulid.ID]int, error)

	ListForWorker(
		ctx context.Context,
		req *ListWorkerTrainingRequest,
	) ([]*worker.WorkerTrainingRecord, error)
	GetByID(
		ctx context.Context,
		req *GetWorkerTrainingByIDRequest,
	) (*worker.WorkerTrainingRecord, error)
	Create(
		ctx context.Context,
		entity *worker.WorkerTrainingRecord,
	) (*worker.WorkerTrainingRecord, error)
	Update(
		ctx context.Context,
		entity *worker.WorkerTrainingRecord,
	) (*worker.WorkerTrainingRecord, error)
	ListDue(
		ctx context.Context,
		req *ListTrainingRemindersRequest,
	) ([]*worker.WorkerTrainingRecord, error)
	ListExpiring(
		ctx context.Context,
		req *ListTrainingRemindersRequest,
	) ([]*worker.WorkerTrainingRecord, error)
}
