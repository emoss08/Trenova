package captureservice

import (
	"context"
	"fmt"
	"net/url"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	// staleProcessingSeconds is how long a batch may sit sealed or processing
	// before the sweep decides its run was lost and starts another. Reading
	// even a thousand-page batch finishes well inside it.
	staleProcessingSeconds = 30 * 60
	// staleReceivingSeconds is how long a batch may wait for its last page.
	// A device that went offline mid-upload keeps its pages locally and
	// resumes, so this is generous: past it, the upload is not coming back.
	staleReceivingSeconds = 7 * 24 * 60 * 60
	maintenanceBatchSize  = 200
)

// MaintenanceResult is what one sweep did.
type MaintenanceResult struct {
	PairingsExpired  int `json:"pairingsExpired"`
	RequestsExpired  int `json:"requestsExpired"`
	BatchesRestarted int `json:"batchesRestarted"`
	BatchesAbandoned int `json:"batchesAbandoned"`
	BatchesExpired   int `json:"batchesExpired"`
	BatchesPurged    int `json:"batchesPurged"`
	OwnersReminded   int `json:"ownersReminded"`
}

// Maintain runs every housekeeping pass. Each pass is independent, so one
// failing does not stop the others; the first error is returned after all ran.
func (s *Service) Maintain(ctx context.Context) (*MaintenanceResult, error) {
	result := new(MaintenanceResult)
	var firstErr error
	record := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	var err error
	result.PairingsExpired, err = s.ExpirePairings(ctx)
	record(err)
	result.RequestsExpired, err = s.ExpireRequests(ctx)
	record(err)
	result.BatchesRestarted, err = s.RestartStaleProcessing(ctx)
	record(err)
	result.BatchesAbandoned, err = s.AbandonStaleUploads(ctx)
	record(err)
	result.OwnersReminded, err = s.RemindRetention(ctx)
	record(err)
	result.BatchesExpired, result.BatchesPurged, err = s.EnforceRetention(ctx)
	record(err)

	return result, firstErr
}

// RestartStaleProcessing starts processing again for batches whose run was
// lost: sealed with no worker at the time, or a worker that died mid-read.
func (s *Service) RestartStaleProcessing(ctx context.Context) (int, error) {
	if s.workflows == nil || !s.workflows.Enabled() {
		return 0, nil
	}

	stale, err := s.batches.ListStale(ctx, repositories.ListStaleCaptureBatchesRequest{
		Statuses:      []capture.BatchStatus{capture.BatchSealed, capture.BatchProcessing},
		UpdatedBefore: timeutils.NowUnix() - staleProcessingSeconds,
		Limit:         maintenanceBatchSize,
	})
	if err != nil {
		return 0, err
	}

	for _, batch := range stale {
		s.startProcessing(ctx, batch)
	}

	return len(stale), nil
}

// AbandonStaleUploads fails batches whose device never finished sending them.
// The pages that did arrive are kept, and the batch can still be split and
// filed by hand.
func (s *Service) AbandonStaleUploads(ctx context.Context) (int, error) {
	stale, err := s.batches.ListStale(ctx, repositories.ListStaleCaptureBatchesRequest{
		Statuses:      []capture.BatchStatus{capture.BatchReceiving},
		UpdatedBefore: timeutils.NowUnix() - staleReceivingSeconds,
		Limit:         maintenanceBatchSize,
	})
	if err != nil {
		return 0, err
	}

	count := 0
	for _, batch := range stale {
		batch.Status = capture.BatchFailed
		batch.FailureMessage = "The device stopped sending pages before the batch was finished."
		if batch.ReceivedPageCount > 0 {
			batch.Status = capture.BatchSealed
			batch.FailureMessage = ""
		}

		updated, updateErr := s.batches.Update(ctx, batch)
		if updateErr != nil {
			s.l.Warn("could not close an abandoned upload",
				zap.String("batchId", batch.ID.String()), zap.Error(updateErr))

			continue
		}
		if updated.Status == capture.BatchSealed {
			s.startProcessing(ctx, updated)
		}
		s.publishBatch(ctx, updated, batchActionUpdated)
		count++
	}

	return count, nil
}

// EnforceRetention expires batches nobody finished filing in time, then
// removes the stored pages of every batch past its retention. Filed documents
// are untouched: they live on as documents, with their own copies.
func (s *Service) EnforceRetention(ctx context.Context) (expired, purged int, err error) {
	due, err := s.batches.ListRetentionDue(ctx, repositories.ListRetentionDueCaptureBatchesRequest{
		Now:   timeutils.NowUnix(),
		Limit: maintenanceBatchSize,
	})
	if err != nil {
		return 0, 0, err
	}

	for _, batch := range due {
		if !batch.Status.Terminal() {
			if _, discardErr := s.discardBatch(
				ctx,
				batch,
				capture.BatchExpired,
			); discardErr != nil {
				s.l.Warn("could not expire a capture batch",
					zap.String("batchId", batch.ID.String()), zap.Error(discardErr))

				continue
			}
			expired++
		}

		if purgeErr := s.purgeBatchPages(ctx, batch); purgeErr != nil {
			s.l.Warn("could not remove an expired batch's pages",
				zap.String("batchId", batch.ID.String()), zap.Error(purgeErr))

			continue
		}
		purged++
	}

	return expired, purged, nil
}

// purgeBatchPages deletes every stored object for a batch and then the batch
// row itself, which takes its pages and items with it.
func (s *Service) purgeBatchPages(ctx context.Context, batch *capture.CaptureBatch) error {
	tenantInfo := pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID}
	pages, err := s.pages.ListByBatch(ctx, repositories.ListCapturePagesRequest{
		BatchID:    batch.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	for _, page := range pages {
		if err = s.deleteObject(ctx, page.ThumbnailPath); err != nil {
			return err
		}
		if err = s.deleteObject(ctx, page.StoragePath); err != nil {
			return err
		}
	}

	return s.batches.Delete(
		ctx,
		repositories.DeleteCaptureBatchRequest{ID: batch.ID, TenantInfo: tenantInfo},
	)
}

const (
	// retentionReminderSeconds is how far ahead of deletion an owner is told:
	// a week, so someone back from a few days off still has time to file.
	retentionReminderSeconds = 7 * secondsPerDay
	retentionReminderEvent   = "capture.retention_reminder"
	retentionReminderSource  = "captureservice"
)

// RemindRetention tells the owner of each stack still waiting to be filed
// that its unfiled pages are deleted within the week. Each stack is claimed
// before its owner is told, so two sweeps running at once tell them once.
func (s *Service) RemindRetention(ctx context.Context) (int, error) {
	if s.notifications == nil {
		return 0, nil
	}

	now := timeutils.NowUnix()
	due, err := s.batches.ListRetentionReminders(
		ctx,
		repositories.ListRetentionReminderCaptureBatchesRequest{
			From:  now,
			Until: now + retentionReminderSeconds,
			Limit: maintenanceBatchSize,
		},
	)
	if err != nil {
		return 0, err
	}

	reminded := 0
	for _, batch := range due {
		tenantInfo := pagination.TenantInfo{OrgID: batch.OrganizationID, BuID: batch.BusinessUnitID}
		claimed, claimErr := s.batches.ClaimRetentionReminder(
			ctx,
			repositories.ClaimRetentionReminderRequest{
				ID:         batch.ID,
				TenantInfo: tenantInfo,
				At:         now,
			},
		)
		if claimErr != nil {
			s.l.Warn("could not claim a capture retention reminder",
				zap.String("batchId", batch.ID.String()), zap.Error(claimErr))

			continue
		}
		if !claimed {
			continue
		}

		if notifyErr := s.remindOwner(ctx, batch, now); notifyErr != nil {
			s.l.Warn("could not remind a capture batch owner",
				zap.String("batchId", batch.ID.String()), zap.Error(notifyErr))

			continue
		}
		reminded++
	}

	return reminded, nil
}

func (s *Service) remindOwner(ctx context.Context, batch *capture.CaptureBatch, now int64) error {
	daysLeft := max(1, (batch.RetainUntil-now+secondsPerDay-1)/secondsPerDay)
	waiting := max(batch.ItemCount-batch.FiledItemCount, 1)
	owner := batch.UserID
	buID := batch.BusinessUnitID
	correlation := batch.ID.String()
	expires := batch.RetainUntil

	_, err := s.notifications.Create(ctx, &notification.Notification{
		OrganizationID: batch.OrganizationID,
		BusinessUnitID: &buID,
		TargetUserID:   &owner,
		Channel:        notification.ChannelUser,
		EventType:      retentionReminderEvent,
		Priority:       notification.PriorityMedium,
		Title:          "Scanned pages will be deleted soon",
		Message: fmt.Sprintf(
			"%s has %d document(s) not filed yet. Unfiled pages are deleted in %d day(s).",
			batchName(batch), waiting, daysLeft,
		),
		Data: map[string]any{
			"link":        "/intake?view=all&batch=" + url.QueryEscape(batch.ID.String()),
			"batchId":     batch.ID.String(),
			"retainUntil": batch.RetainUntil,
		},
		RelatedEntities: map[string]any{"captureBatchId": batch.ID.String()},
		ExpiresAt:       &expires,
		Source:          retentionReminderSource,
		CorrelationID:   &correlation,
	})

	return err
}

// batchName is what the reminder calls a stack: the print job, else the
// scanner it came from, else what it is.
func batchName(batch *capture.CaptureBatch) string {
	switch {
	case batch.JobName != "":
		return batch.JobName
	case batch.SourceName != "":
		return "A scan from " + batch.SourceName
	case batch.Source == capture.SourcePrint:
		return "A printed document"
	default:
		return "A scanned stack"
	}
}
