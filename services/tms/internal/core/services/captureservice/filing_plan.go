package captureservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type FilePlan struct {
	Item      *capture.CaptureItem
	Batch     *capture.CaptureBatch
	Target    capture.Target
	Unchanged bool
}

type DiscardItemPlan struct {
	Item  *capture.CaptureItem
	Batch *capture.CaptureBatch
}

type DiscardBatchPlan struct {
	Batch     *capture.CaptureBatch
	Open      []*capture.CaptureItem
	Unchanged bool
}

func (s *Service) PlanFileItem(ctx context.Context, in *FileItemInput) (*FilePlan, error) {
	item, err := s.items.GetByID(
		ctx,
		repositories.GetCaptureItemByIDRequest{ID: in.ItemID, TenantInfo: in.TenantInfo},
	)
	if err != nil {
		return nil, err
	}

	batch, err := s.visibleBatch(
		ctx,
		in.TenantInfo,
		permission.OpUpdate,
		&repositories.GetCaptureBatchByIDRequest{
			ID:           item.BatchID,
			IncludePages: true,
		},
	)
	if err != nil {
		return nil, err
	}
	if item.Status == capture.ItemFiled || item.Status == capture.ItemFiling {
		return &FilePlan{Item: item, Batch: batch, Unchanged: true}, nil
	}
	if !item.Status.Open() || !batch.Status.Editable() {
		return nil, errortypes.NewConflictError("This document can no longer be filed")
	}
	if !in.Automatic && item.Version != in.Version {
		return nil, errortypes.NewConflictError(
			"Somebody else changed this document; reload it and try again")
	}

	target := capture.Target{
		ResourceType:   in.TargetType,
		ResourceID:     &in.TargetID,
		DocumentTypeID: in.DocumentTypeID,
	}
	if err = s.checkTarget(ctx, in.TenantInfo, target, "targetType", "targetId"); err != nil {
		return nil, err
	}

	return &FilePlan{Item: item, Batch: batch, Target: target}, nil
}

func (s *Service) PlanDiscardItem(
	ctx context.Context,
	in *DiscardItemInput,
) (*DiscardItemPlan, error) {
	item, err := s.items.GetByID(
		ctx,
		repositories.GetCaptureItemByIDRequest{ID: in.ItemID, TenantInfo: in.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	batch, err := s.visibleBatch(
		ctx,
		in.TenantInfo,
		permission.OpDelete,
		&repositories.GetCaptureBatchByIDRequest{
			ID: item.BatchID,
		},
	)
	if err != nil {
		return nil, err
	}
	if !item.Status.Open() {
		return nil, errortypes.NewConflictError(
			"Only a document waiting to be filed can be discarded",
		)
	}
	if item.Version != in.Version {
		return nil, errortypes.NewConflictError(
			"Somebody else changed this document; reload it and try again")
	}

	return &DiscardItemPlan{Item: item, Batch: batch}, nil
}

func (s *Service) PlanDiscardBatch(
	ctx context.Context,
	in *DiscardBatchInput,
) (*DiscardBatchPlan, error) {
	batch, err := s.visibleBatch(
		ctx,
		in.TenantInfo,
		permission.OpDelete,
		&repositories.GetCaptureBatchByIDRequest{
			ID:           in.BatchID,
			IncludeItems: true,
		},
	)
	if err != nil {
		return nil, err
	}
	if batch.Status.Terminal() {
		return &DiscardBatchPlan{Batch: batch, Unchanged: true}, nil
	}
	if batch.Version != in.Version {
		return nil, errortypes.NewConflictError(
			"Somebody else changed this batch; reload it and try again")
	}
	if slices.ContainsFunc(batch.Items, func(item *capture.CaptureItem) bool {
		return item.Status == capture.ItemFiling
	}) {
		return nil, errortypes.NewConflictError(
			"A document in this batch is being filed; wait for it to finish",
		)
	}

	open := make([]*capture.CaptureItem, 0, len(batch.Items))
	for _, item := range batch.Items {
		if item.Status.Open() {
			open = append(open, item)
		}
	}

	return &DiscardBatchPlan{Batch: batch, Open: open}, nil
}

func (s *Service) GetItem(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	itemID pulid.ID,
) (*capture.CaptureItem, error) {
	item, err := s.items.GetByID(
		ctx,
		repositories.GetCaptureItemByIDRequest{ID: itemID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	if _, err = s.visibleBatch(
		ctx,
		tenantInfo,
		permission.OpRead,
		&repositories.GetCaptureBatchByIDRequest{ID: item.BatchID},
	); err != nil {
		return nil, err
	}

	return item, nil
}
