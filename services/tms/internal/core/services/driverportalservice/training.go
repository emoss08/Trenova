package driverportalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// PortalTraining is the driver's view of one course: what to take, by when,
// how to open it, and where a completed course stands.
type PortalTraining struct {
	ID                      pulid.ID                `json:"id"`
	CourseID                pulid.ID                `json:"courseId"`
	Name                    string                  `json:"name"`
	Description             string                  `json:"description"`
	Category                worker.TrainingCategory `json:"category"`
	Delivery                worker.TrainingDelivery `json:"delivery"`
	ContentURL              string                  `json:"contentUrl"`
	DurationMinutes         int32                   `json:"durationMinutes"`
	Status                  worker.TrainingStatus   `json:"status"`
	Health                  worker.TrainingHealth   `json:"health"`
	Required                bool                    `json:"required"`
	RequiresAcknowledgement bool                    `json:"requiresAcknowledgement"`
	// Scored courses need the office to enter a result after the driver
	// acknowledges them; unscored self-serve courses complete on acknowledgement.
	Scored          bool   `json:"scored"`
	DueAt           *int64 `json:"dueAt"`
	DaysUntilDue    *int64 `json:"daysUntilDue"`
	StartedAt       *int64 `json:"startedAt"`
	CompletedAt     *int64 `json:"completedAt"`
	ExpiresAt       *int64 `json:"expiresAt"`
	DaysUntilExpiry *int64 `json:"daysUntilExpiry"`
	AcknowledgedAt  *int64 `json:"acknowledgedAt"`
	Score           string `json:"score"`
}

func portalTrainingView(item *worker.TrainingSummaryItem) *PortalTraining {
	course := item.Course
	view := &PortalTraining{
		CourseID:                course.ID,
		Name:                    course.Name,
		Description:             course.Description,
		Category:                course.Category,
		Delivery:                course.Delivery,
		ContentURL:              course.ContentURL,
		DurationMinutes:         course.DurationMinutes,
		Health:                  item.Health,
		Required:                item.Required,
		RequiresAcknowledgement: course.RequiresAcknowledgement,
		Scored:                  course.PassingScore.Valid,
		DaysUntilDue:            item.DaysUntilDue,
		DaysUntilExpiry:         item.DaysUntilExpiry,
	}
	if record := item.Record; record != nil {
		view.ID = record.ID
		view.Status = record.Status
		view.DueAt = record.DueAt
		view.StartedAt = record.StartedAt
		view.CompletedAt = record.CompletedAt
		view.ExpiresAt = record.ExpiresAt
		view.AcknowledgedAt = record.AcknowledgedAt
		if record.Score.Valid {
			view.Score = record.Score.Decimal.StringFixed(2)
		}
	}
	return view
}

func portalTrainingFromRecord(record *worker.WorkerTrainingRecord, wrk *worker.Worker) *PortalTraining {
	now := timeutils.NowUnix()
	return portalTrainingView(&worker.TrainingSummaryItem{
		Course:          record.Course,
		Record:          record,
		Health:          record.Health(now),
		DaysUntilDue:    record.DaysUntilDue(now),
		DaysUntilExpiry: record.DaysUntilExpiry(now),
		Required:        record.Course.AppliesTo(wrk),
	})
}

// MyTraining lists the signed-in driver's courses: open assignments first,
// then current certifications, then anything required they have not been
// assigned yet.
func (s *Service) MyTraining(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*PortalTraining, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	summary, err := s.training.Summary(ctx, tenantInfo, wrk.ID)
	if err != nil {
		return nil, err
	}

	views := make([]*PortalTraining, 0, len(summary.Items))
	for _, item := range summary.Items {
		views = append(views, portalTrainingView(item))
	}
	return views, nil
}

// StartMyTraining marks the course in progress as the driver opens it and
// returns the link to follow.
func (s *Service) StartMyTraining(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	recordID pulid.ID,
) (*PortalTraining, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	record, err := s.training.Start(ctx, tenantInfo, recordID, wrk.ID)
	if err != nil {
		return nil, err
	}
	return portalTrainingFromRecord(record, wrk), nil
}

// AcknowledgeMyTraining records that the driver took the course. When the
// office still has to score it, dispatch is told so the result gets entered.
func (s *Service) AcknowledgeMyTraining(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	recordID pulid.ID,
) (*PortalTraining, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	record, err := s.training.Acknowledge(ctx, tenantInfo, recordID, wrk.ID)
	if err != nil {
		return nil, err
	}
	if record.IsOpen() {
		s.notifyDispatch(
			ctx,
			tenantInfo,
			"training_acknowledged",
			"Training awaiting a result",
			wrk.FirstName+" "+wrk.LastName+" finished "+record.Course.Name+
				". Enter the score to close the assignment.",
			"/hr/workers?tab=training",
			map[string]any{"workerId": wrk.ID.String(), "trainingRecordId": record.ID.String()},
		)
	}
	return portalTrainingFromRecord(record, wrk), nil
}
