package captureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
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
