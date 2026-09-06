package workersafetyservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) ListRecognitions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	visibleOnly bool,
) ([]*worker.WorkerRecognition, error) {
	return s.repo.ListRecognitions(ctx, &repositories.ListWorkerRecognitionsRequest{
		TenantInfo:    tenantInfo,
		WorkerID:      workerID,
		VisibleOnly:   visibleOnly,
		IncludeActors: true,
	})
}

// GiveRecognition records praise and, when it is visible, tells the driver.
func (s *Service) GiveRecognition(
	ctx context.Context,
	entity *worker.WorkerRecognition,
	userID pulid.ID,
) (*worker.WorkerRecognition, error) {
	log := s.l.With(zap.String("operation", "GiveRecognition"), zap.String("workerId", entity.WorkerID.String()))

	wrk, err := s.loadWorker(ctx, recognitionTenant(entity), entity.WorkerID)
	if err != nil {
		return nil, err
	}
	entity.Title = strings.TrimSpace(entity.Title)
	entity.Message = strings.TrimSpace(entity.Message)
	entity.AwardedByID = userID
	if entity.OccurredAt <= 0 {
		entity.OccurredAt = timeutils.NowUnix()
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.CreateRecognition(ctx, entity)
	if err != nil {
		log.Error("failed to record recognition", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource: permission.ResourceWorkerRecognition, resourceID: created.GetResourceID(),
		operation: permission.OpCreate, userID: userID, tenant: recognitionTenant(created),
		current: created, comment: "Recognition given: " + created.Title, log: log,
	})
	s.publish(ctx, recognitionTenant(created), realtimeRecognition, permission.OpCreate, created.ID, userID)
	s.refreshRollupQuietly(ctx, recognitionTenant(created), created.WorkerID)

	if created.VisibleToWorker && !wrk.UserID.IsNil() {
		s.notifyDriver(ctx, recognitionTenant(created), created.WorkerID, eventRecognition, notification.PriorityMedium,
			documenttemplate.DriverNotificationContext{
				RecognitionTitle:   created.Title,
				RecognitionMessage: created.Message,
			},
			fmt.Sprintf("recognition-%s", created.ID),
		)
	}
	return created, nil
}

func (s *Service) DeleteRecognition(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) error {
	log := s.l.With(zap.String("operation", "DeleteRecognition"), zap.String("id", id.String()))

	original, err := s.repo.GetRecognitionByID(ctx, &repositories.GetWorkerRecognitionByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if err = s.repo.DeleteRecognition(ctx, &repositories.GetWorkerRecognitionByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	}); err != nil {
		log.Error("failed to delete recognition", zap.Error(err))
		return err
	}

	s.audit(&auditParams{
		resource: permission.ResourceWorkerRecognition, resourceID: original.GetResourceID(),
		operation: permission.OpDelete, userID: userID, tenant: tenantInfo,
		current: original, comment: "Recognition removed", log: log,
	})
	s.publish(ctx, tenantInfo, realtimeRecognition, permission.OpDelete, original.ID, userID)
	s.refreshRollupQuietly(ctx, tenantInfo, original.WorkerID)
	return nil
}
