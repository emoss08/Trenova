package invoiceadjustmentjobs

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Service servicesports.InvoiceAdjustmentService
	Repo    repositories.InvoiceAdjustmentRepository
	Logger  *zap.Logger
}

type Activities struct {
	service servicesports.InvoiceAdjustmentService
	repo    repositories.InvoiceAdjustmentRepository
	logger  *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		service: p.Service,
		repo:    p.Repo,
		logger:  p.Logger.Named("invoice-adjustment-batch-activities"),
	}
}

func (a *Activities) ProcessBatchItemActivity(
	ctx context.Context,
	payload *ProcessBatchItemPayload,
) (*ProcessBatchItemResult, error) {
	tenantInfo := pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}
	batch, err := a.repo.GetBatchByID(ctx, repositories.GetBatchRequest{
		ID:         payload.BatchID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	item := findBatchItem(batch.Items, payload.ItemID)
	if item == nil {
		return nil, errBatchItemNotFound(payload.ItemID)
	}
	if isTerminalBatchItemStatus(item.Status) {
		return &ProcessBatchItemResult{
			ItemID:       item.ID,
			AdjustmentID: item.AdjustmentID,
			FinalStatus:  string(item.Status),
			ErrorMessage: item.ErrorMessage,
			ProcessedAt:  timeutils.NowUnix(),
		}, nil
	}

	item.Status = invoiceadjustment.BatchItemStatusExecuting
	item.ErrorMessage = ""
	if _, err = a.repo.UpdateBatchItem(ctx, item); err != nil {
		return nil, err
	}

	req := new(servicesports.InvoiceAdjustmentRequest)
	if err = decodeRequestPayload(item.RequestPayload, req); err != nil {
		return nil, err
	}
	req.TenantInfo = tenantInfo

	actor := &servicesports.RequestActor{
		PrincipalType:  payload.PrincipalType,
		PrincipalID:    payload.PrincipalID,
		UserID:         payload.UserID,
		APIKeyID:       payload.APIKeyID,
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
	}

	adjustment, submitErr := a.service.Submit(ctx, req, actor)
	if submitErr != nil {
		item.Status = invoiceadjustment.BatchItemStatusFailed
		item.ErrorMessage = submitErr.Error()
		item.ResultPayload = map[string]any{}
	} else {
		item.AdjustmentID = adjustment.ID
		if adjustment.Status == invoiceadjustment.StatusPendingApproval {
			item.Status = invoiceadjustment.BatchItemStatusPendingApproval
		} else {
			item.Status = invoiceadjustment.BatchItemStatusExecuted
		}
		item.ResultPayload = map[string]any{
			"adjustmentId": adjustment.ID,
			"status":       adjustment.Status,
		}
	}
	if _, err = a.repo.UpdateBatchItem(ctx, item); err != nil {
		return nil, err
	}

	if _, err = a.refreshBatchProgress(ctx, payload.BatchID, tenantInfo); err != nil {
		return nil, err
	}

	return &ProcessBatchItemResult{
		ItemID:       item.ID,
		AdjustmentID: item.AdjustmentID,
		FinalStatus:  string(item.Status),
		ErrorMessage: item.ErrorMessage,
		ProcessedAt:  timeutils.NowUnix(),
	}, nil
}

func (a *Activities) refreshBatchProgress(
	ctx context.Context,
	batchID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*invoiceadjustment.Batch, error) {
	batch, err := a.repo.GetBatchByID(ctx, repositories.GetBatchRequest{
		ID:         batchID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	processed := 0
	succeeded := 0
	failed := 0
	for _, item := range batch.Items {
		if item == nil {
			continue
		}
		switch item.Status {
		case invoiceadjustment.BatchItemStatusExecuted, invoiceadjustment.BatchItemStatusPendingApproval, invoiceadjustment.BatchItemStatusRejected:
			processed++
			succeeded++
		case invoiceadjustment.BatchItemStatusFailed:
			processed++
			failed++
		}
	}

	batch.ProcessedCount = processed
	batch.SucceededCount = succeeded
	batch.FailedCount = failed
	switch {
	case processed == 0:
		batch.Status = invoiceadjustment.BatchStatusRunning
	case failed == 0 && processed == batch.TotalCount:
		batch.Status = invoiceadjustment.BatchStatusCompleted
	case succeeded == 0 && processed == batch.TotalCount:
		batch.Status = invoiceadjustment.BatchStatusFailed
	case processed == batch.TotalCount:
		batch.Status = invoiceadjustment.BatchStatusPartial
	default:
		batch.Status = invoiceadjustment.BatchStatusRunning
	}

	return a.repo.UpdateBatch(ctx, batch)
}

func decodeRequestPayload(payload map[string]any, req *servicesports.InvoiceAdjustmentRequest) error {
	raw, err := sonic.Marshal(payload)
	if err != nil {
		return err
	}
	return sonic.Unmarshal(raw, req)
}

func findBatchItem(items []*invoiceadjustment.BatchItem, itemID pulid.ID) *invoiceadjustment.BatchItem {
	for _, item := range items {
		if item != nil && item.ID == itemID {
			return item
		}
	}
	return nil
}

func isTerminalBatchItemStatus(status invoiceadjustment.BatchItemStatus) bool {
	switch status {
	case invoiceadjustment.BatchItemStatusExecuted, invoiceadjustment.BatchItemStatusPendingApproval, invoiceadjustment.BatchItemStatusRejected, invoiceadjustment.BatchItemStatusFailed:
		return true
	default:
		return false
	}
}

func errBatchItemNotFound(itemID pulid.ID) error {
	return fmt.Errorf("invoice adjustment batch item %s not found", itemID.String())
}
