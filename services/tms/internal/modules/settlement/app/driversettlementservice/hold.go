package driversettlementservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type HoldRequest struct {
	TenantInfo   pagination.TenantInfo
	SettlementID pulid.ID
	Reason       driversettlement.HoldReason
	Note         string
	Release      bool
}

func PlanHold(
	entity *driversettlement.Settlement,
	req *HoldRequest,
	userID pulid.ID,
	now int64,
) error {
	if req.Release {
		return entity.ReleaseHold(userID, now)
	}

	return entity.PlaceHold(&driversettlement.HoldInput{
		Reason: req.Reason,
		Note:   req.Note,
		UserID: userID,
		At:     now,
	})
}

func (s *Service) PlanHoldChange(ctx context.Context, req *HoldRequest) (*ActionPlan, error) {
	entity, err := s.GetForUpdate(ctx, req.TenantInfo, req.SettlementID)
	if err != nil {
		return nil, err
	}
	plan := &ActionPlan{Before: entity, After: CloneSettlement(entity)}
	plan.Refusal = PlanHold(plan.After, req, req.TenantInfo.UserID, timeutils.NowUnix())

	return plan, nil
}

func (s *Service) ChangeHold(
	ctx context.Context,
	req *HoldRequest,
	actor *permissionports.RequestActor,
) (*driversettlement.Settlement, error) {
	if err := settlementshared.RequireActor(actor, "Settlement hold"); err != nil {
		return nil, err
	}
	if !req.Release {
		if err := driversettlement.CheckHoldInput(req.Reason, req.Note); err != nil {
			return nil, err
		}
	}

	var updated *driversettlement.Settlement
	var previous driversettlement.Settlement
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		entity, txErr := s.GetForUpdate(txCtx, req.TenantInfo, req.SettlementID)
		if txErr != nil {
			return txErr
		}
		previous = *entity
		if txErr = PlanHold(entity, req, actor.UserID, timeutils.NowUnix()); txErr != nil {
			return txErr
		}
		if updated, txErr = s.settlementRepo.Update(txCtx, entity); txErr != nil {
			return txErr
		}

		return s.refreshBatch(txCtx, req.TenantInfo, updated.BatchID)
	})
	if err != nil {
		return nil, err
	}
	s.LogSettlementAudit(ctx, updated, &previous, actor.UserID, permission.OpHold,
		holdComment(req, &previous))

	return updated, nil
}

var holdReasonWords = map[driversettlement.HoldReason]string{
	driversettlement.HoldReasonRateDispute:      "rate dispute",
	driversettlement.HoldReasonMissingPaperwork: "missing paperwork",
	driversettlement.HoldReasonNegativeNet:      "negative net",
	driversettlement.HoldReasonDeductionReview:  "deduction under review",
	driversettlement.HoldReasonOther:            "other",
}

func holdComment(req *HoldRequest, previous *driversettlement.Settlement) string {
	if req.Release {
		return "Settlement hold released (" + holdReasonWords[previous.HoldReason] + ")"
	}
	comment := "Settlement held: " + holdReasonWords[req.Reason]
	if note := strings.TrimSpace(req.Note); note != "" {
		comment += ": " + note
	}

	return comment
}

func (s *Service) refreshBatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	batchID *pulid.ID,
) error {
	if batchID == nil || batchID.IsNil() {
		return nil
	}
	_, err := s.batchRepo.RecalculateAggregates(ctx, tenantInfo, *batchID)

	return err
}
