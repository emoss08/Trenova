package workertrainingservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) ListCourses(
	ctx context.Context,
	req *repositories.ListTrainingCoursesRequest,
) (*pagination.CursorListResult[*worker.TrainingCourse], error) {
	return s.repo.ListCourses(ctx, req)
}

func (s *Service) GetCourse(
	ctx context.Context,
	req *repositories.GetTrainingCourseByIDRequest,
) (*worker.TrainingCourse, error) {
	return s.repo.GetCourseByID(ctx, req)
}

func (s *Service) ActiveCourses(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.TrainingCourse, error) {
	return s.repo.ListActiveCourses(ctx, tenantInfo)
}

func (s *Service) validateCourse(ctx context.Context, entity *worker.TrainingCourse) error {
	entity.NormalizeCode()
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if entity.Code != "" {
		exists, err := s.repo.CourseCodeExists(ctx, &repositories.TrainingCourseCodeExistsRequest{
			TenantInfo: courseTenant(entity),
			Code:       entity.Code,
			ExcludeID:  entity.ID,
		})
		if err != nil {
			return err
		}
		if exists {
			multiErr.Add("code", errortypes.ErrDuplicate, "A course with this code already exists")
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) CreateCourse(
	ctx context.Context,
	entity *worker.TrainingCourse,
	userID pulid.ID,
) (*worker.TrainingCourse, error) {
	log := s.l.With(zap.String("operation", "CreateCourse"))

	if err := s.validateCourse(ctx, entity); err != nil {
		return nil, err
	}

	created, err := s.repo.CreateCourse(ctx, entity)
	if err != nil {
		log.Error("failed to create training course", zap.Error(err))
		return nil, err
	}

	s.auditCourse(created, nil, permission.OpCreate, userID, "Training course created", log)
	s.publish(
		ctx,
		courseTenant(created),
		realtimeResourceCourse,
		permission.OpCreate,
		created.ID,
		userID,
	)

	return created, nil
}

func (s *Service) UpdateCourse(
	ctx context.Context,
	entity *worker.TrainingCourse,
	userID pulid.ID,
) (*worker.TrainingCourse, error) {
	log := s.l.With(zap.String("operation", "UpdateCourse"), zap.String("id", entity.ID.String()))

	original, err := s.repo.GetCourseByID(ctx, &repositories.GetTrainingCourseByIDRequest{
		ID:         entity.ID,
		TenantInfo: courseTenant(entity),
	})
	if err != nil {
		return nil, err
	}
	if err = s.validateCourse(ctx, entity); err != nil {
		return nil, err
	}
	if original.Status == domaintypes.StatusActive && entity.Status != domaintypes.StatusActive {
		if err = s.requireCourseRetirable(ctx, original); err != nil {
			return nil, err
		}
	}

	updated, err := s.repo.UpdateCourse(ctx, entity)
	if err != nil {
		log.Error("failed to update training course", zap.Error(err))
		return nil, err
	}

	s.auditCourse(updated, original, permission.OpUpdate, userID, "Training course updated", log)
	s.publish(
		ctx,
		courseTenant(updated),
		realtimeResourceCourse,
		permission.OpUpdate,
		updated.ID,
		userID,
	)

	return updated, nil
}

type CourseStatusChangeRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Version    int64
	UserID     pulid.ID
}

func (s *Service) ArchiveCourse(
	ctx context.Context,
	req *CourseStatusChangeRequest,
) (*worker.TrainingCourse, error) {
	return s.changeCourseStatus(ctx, req, domaintypes.StatusInactive, permission.OpArchive)
}

func (s *Service) RestoreCourse(
	ctx context.Context,
	req *CourseStatusChangeRequest,
) (*worker.TrainingCourse, error) {
	return s.changeCourseStatus(ctx, req, domaintypes.StatusActive, permission.OpRestore)
}

func (s *Service) changeCourseStatus(
	ctx context.Context,
	req *CourseStatusChangeRequest,
	target domaintypes.Status,
	operation permission.Operation,
) (*worker.TrainingCourse, error) {
	log := s.l.With(zap.String("operation", string(operation)), zap.String("id", req.ID.String()))

	original, err := s.repo.GetCourseByID(ctx, &repositories.GetTrainingCourseByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if original.Status == target {
		return original, nil
	}
	if req.Version > 0 && original.Version != req.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Course was changed by someone else. Reload and try again",
		)
	}
	if target == domaintypes.StatusInactive {
		if err = s.requireCourseRetirable(ctx, original); err != nil {
			return nil, err
		}
	}

	updated := *original
	updated.Status = target
	saved, err := s.repo.UpdateCourse(ctx, &updated)
	if err != nil {
		log.Error("failed to change training course status", zap.Error(err))
		return nil, err
	}

	s.auditCourse(
		saved,
		original,
		operation,
		req.UserID,
		fmt.Sprintf("Training course %s", target),
		log,
	)
	s.publish(ctx, courseTenant(saved), realtimeResourceCourse, operation, saved.ID, req.UserID)

	return saved, nil
}

// requireCourseRetirable keeps a course with open assignments from being
// switched off underneath the workers still taking it.
func (s *Service) requireCourseRetirable(ctx context.Context, entity *worker.TrainingCourse) error {
	count, err := s.repo.CountRecordsByCourse(ctx, &repositories.CountTrainingRecordsRequest{
		TenantInfo: courseTenant(entity),
		CourseID:   entity.ID,
		OpenOnly:   true,
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			fmt.Sprintf(
				"%d worker%s still %s this course assigned. Complete, waive or cancel those first",
				count,
				plural(count, "", "s"),
				plural(count, "has", "have"),
			),
		)
	}
	return nil
}

func (s *Service) CountOpenRecords(
	ctx context.Context,
	entity *worker.TrainingCourse,
) (int, error) {
	return s.repo.CountRecordsByCourse(ctx, &repositories.CountTrainingRecordsRequest{
		TenantInfo: courseTenant(entity),
		CourseID:   entity.ID,
		OpenOnly:   true,
	})
}

func (s *Service) CountOpenRecordsByCourses(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	courseIDs []pulid.ID,
) (map[pulid.ID]int, error) {
	return s.repo.CountRecordsByCourseIDs(ctx, &repositories.CountTrainingRecordsByCourseIDsRequest{
		TenantInfo: tenantInfo,
		CourseIDs:  courseIDs,
		OpenOnly:   true,
	})
}

func plural(count int, one, many string) string {
	if count == 1 {
		return one
	}
	return many
}
