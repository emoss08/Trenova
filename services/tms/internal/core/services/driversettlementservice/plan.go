package driversettlementservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type ActionPlan = settlementshared.ActionPlan[*driversettlement.Settlement]

const manualAdjustmentMessage = "Settlement contains manual adjustment lines"

func CloneSettlement(entity *driversettlement.Settlement) *driversettlement.Settlement {
	if entity == nil {
		return nil
	}
	clone := *entity
	clone.Exceptions = slices.Clone(entity.Exceptions)
	clone.Lines = make([]*driversettlement.SettlementLine, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		copied := *line
		clone.Lines = append(clone.Lines, &copied)
	}
	return &clone
}

func PlanSubmit(entity *driversettlement.Settlement, userID pulid.ID, now int64) error {
	if !driversettlement.IsAllowedTransition(
		entity.Status,
		driversettlement.StatusPendingApproval,
	) || entity.Status == driversettlement.StatusPendingApproval {
		return transitionError(entity.Status, driversettlement.StatusPendingApproval)
	}
	entity.Status = driversettlement.StatusPendingApproval
	entity.SubmittedByID = userID
	entity.SubmittedAt = &now
	return nil
}

func PlanApprove(entity *driversettlement.Settlement, userID pulid.ID, now int64) error {
	if entity.Status != driversettlement.StatusPendingApproval {
		return transitionError(entity.Status, driversettlement.StatusApproved)
	}
	entity.Status = driversettlement.StatusApproved
	entity.ApprovedByID = userID
	entity.ApprovedAt = &now
	return nil
}

func PlanReject(entity *driversettlement.Settlement, reason string) error {
	if reason == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A rejection reason is required",
		)
	}
	if entity.Status != driversettlement.StatusPendingApproval {
		return transitionError(entity.Status, driversettlement.StatusDraft)
	}
	entity.Status = driversettlement.StatusDraft
	entity.SubmittedByID = pulid.Nil
	entity.SubmittedAt = nil
	if entity.Notes != "" {
		entity.Notes += "\n"
	}
	entity.Notes += "Rejected: " + reason
	return nil
}

func PlanPost(entity *driversettlement.Settlement, userID pulid.ID, now int64) error {
	if entity.Status != driversettlement.StatusApproved {
		return transitionError(entity.Status, driversettlement.StatusPosted)
	}
	entity.Status = driversettlement.StatusPosted
	entity.PostedByID = userID
	entity.PostedAt = &now
	return nil
}

type MarkPaidInput struct {
	PaymentMethod    string
	PaymentReference string
	PaidAt           int64
	UserID           pulid.ID
}

func PlanMarkPaid(entity *driversettlement.Settlement, input *MarkPaidInput) error {
	if input.PaymentMethod == "" {
		return errortypes.NewValidationError(
			"paymentMethod",
			errortypes.ErrRequired,
			"Payment method is required",
		)
	}
	if entity.Status != driversettlement.StatusPosted {
		return transitionError(entity.Status, driversettlement.StatusPaid)
	}
	paidAt := input.PaidAt
	entity.Status = driversettlement.StatusPaid
	entity.PaidAt = &paidAt
	entity.PaidByID = input.UserID
	entity.PaymentMethod = input.PaymentMethod
	entity.PaymentReference = input.PaymentReference
	return nil
}

func CheckVoid(entity *driversettlement.Settlement, reason string) error {
	if reason == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A void reason is required",
		)
	}
	if !driversettlement.IsAllowedTransition(
		entity.Status,
		driversettlement.StatusVoided,
	) || entity.Status == driversettlement.StatusVoided {
		return transitionError(entity.Status, driversettlement.StatusVoided)
	}
	return nil
}

func PlanVoid(
	entity *driversettlement.Settlement,
	reason string,
	userID pulid.ID,
	now int64,
) error {
	if err := CheckVoid(entity, reason); err != nil {
		return err
	}
	entity.Status = driversettlement.StatusVoided
	entity.VoidedByID = userID
	entity.VoidedAt = &now
	entity.VoidReason = reason
	return nil
}

func PlanRecalculate(entity *driversettlement.Settlement) error {
	if entity.Status != driversettlement.StatusDraft {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only draft settlements can be recalculated",
		)
	}
	return nil
}

func checkAdjustmentInput(input *AdjustmentLineInput) error {
	if input.Description == "" {
		return errortypes.NewValidationError(
			"description",
			errortypes.ErrRequired,
			"Adjustment description is required",
		)
	}
	if input.AmountMinor == 0 {
		return errortypes.NewValidationError(
			"amountMinor",
			errortypes.ErrInvalid,
			"Adjustment amount cannot be zero",
		)
	}
	return nil
}

func checkEditable(entity *driversettlement.Settlement) error {
	if !entity.IsEditable() {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only draft or pending settlements can be adjusted",
		)
	}
	return nil
}

func PlanAddAdjustment(entity *driversettlement.Settlement, input *AdjustmentLineInput) error {
	if err := checkAdjustmentInput(input); err != nil {
		return err
	}
	if err := checkEditable(entity); err != nil {
		return err
	}
	var payCodeID *pulid.ID
	if input.PayCodeID != nil && !input.PayCodeID.IsNil() {
		payCodeID = input.PayCodeID
	}
	entity.Lines = append(entity.Lines, &driversettlement.SettlementLine{
		Category:    driversettlement.LineCategoryAdjustment,
		Description: input.Description,
		AmountMinor: input.AmountMinor,
		Quantity:    input.Quantity,
		Rate:        input.Rate,
		PayCodeID:   payCodeID,
	})
	entity.AddException(
		driversettlement.ExceptionCodeManualAdjustment,
		driversettlement.ExceptionSeverityWarning,
		manualAdjustmentMessage,
	)
	entity.SyncTotals()
	return nil
}

func PlanRemoveAdjustment(entity *driversettlement.Settlement, lineID pulid.ID) error {
	if err := checkEditable(entity); err != nil {
		return err
	}

	found := false
	remaining := make([]*driversettlement.SettlementLine, 0, len(entity.Lines))
	hasOtherAdjustments := false
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		if line.ID == lineID {
			if line.Category != driversettlement.LineCategoryAdjustment {
				return errortypes.NewValidationError(
					"lineId",
					errortypes.ErrInvalidOperation,
					"Only manual adjustment lines can be removed",
				)
			}
			found = true
			continue
		}
		if line.Category == driversettlement.LineCategoryAdjustment {
			hasOtherAdjustments = true
		}
		remaining = append(remaining, line)
	}
	if !found {
		return errortypes.NewValidationError(
			"lineId",
			errortypes.ErrInvalid,
			"Adjustment line not found on this settlement",
		)
	}

	entity.Lines = remaining
	if !hasOtherAdjustments {
		filtered := make([]driversettlement.Exception, 0, len(entity.Exceptions))
		for _, exception := range entity.Exceptions {
			if exception.Code != driversettlement.ExceptionCodeManualAdjustment {
				filtered = append(filtered, exception)
			}
		}
		entity.Exceptions = filtered
		entity.HasExceptions = len(filtered) > 0
	}
	entity.SyncTotals()
	return nil
}

func MergeRebuilt(entity, rebuilt *driversettlement.Settlement) {
	manualLines := make([]*driversettlement.SettlementLine, 0)
	for _, line := range entity.Lines {
		if line != nil && line.Category == driversettlement.LineCategoryAdjustment {
			line.ID = pulid.Nil
			manualLines = append(manualLines, line)
		}
	}

	mergedLines := make(
		[]*driversettlement.SettlementLine,
		0,
		len(rebuilt.Lines)+len(manualLines),
	)
	mergedLines = append(mergedLines, rebuilt.Lines...)
	mergedLines = append(mergedLines, manualLines...)
	entity.Lines = mergedLines
	entity.ClearExceptions()
	entity.Exceptions = rebuilt.Exceptions
	entity.HasExceptions = rebuilt.HasExceptions
	entity.PayProfileID = rebuilt.PayProfileID
	entity.PayProfileName = rebuilt.PayProfileName
	entity.Classification = rebuilt.Classification
	entity.CurrencyCode = rebuilt.CurrencyCode
	entity.TotalMiles = rebuilt.TotalMiles
	entity.ShipmentCount = rebuilt.ShipmentCount
	if len(manualLines) > 0 {
		entity.AddException(
			driversettlement.ExceptionCodeManualAdjustment,
			driversettlement.ExceptionSeverityWarning,
			manualAdjustmentMessage,
		)
	}
	entity.SyncTotals()
}

func adjustmentLineInput(input *settlementshared.AdjustmentInput) *AdjustmentLineInput {
	if input == nil {
		return &AdjustmentLineInput{}
	}
	return &AdjustmentLineInput{
		Description: input.Description,
		AmountMinor: input.AmountMinor,
		Quantity:    input.Quantity,
		Rate:        input.Rate,
		PayCodeID:   input.PayCodeID,
	}
}

func (s *Service) PlanAction(
	ctx context.Context,
	req *settlementshared.ActionRequest,
) (*ActionPlan, error) {
	entity, err := s.getForUpdate(ctx, req.TenantInfo, req.SettlementID)
	if err != nil {
		return nil, err
	}

	plan := &ActionPlan{Before: entity, After: CloneSettlement(entity)}
	userID := req.TenantInfo.UserID
	now := timeutils.NowUnix()

	switch req.Action {
	case settlementshared.ActionSubmit:
		plan.Refusal = PlanSubmit(plan.After, userID, now)
	case settlementshared.ActionApprove:
		err = s.planApprove(ctx, req, plan, now)
	case settlementshared.ActionReject:
		plan.Refusal = PlanReject(plan.After, req.Reason)
	case settlementshared.ActionPost:
		err = s.planPost(ctx, plan, userID, now)
	case settlementshared.ActionMarkPaid:
		plan.Refusal = PlanMarkPaid(plan.After, &MarkPaidInput{
			PaymentMethod:    req.PaymentMethod,
			PaymentReference: req.PaymentReference,
			PaidAt:           now,
			UserID:           userID,
		})
	case settlementshared.ActionVoid:
		err = s.planVoid(ctx, req, plan, now)
	case settlementshared.ActionRecalculate:
		err = s.planRecalculate(ctx, req, plan)
	case settlementshared.ActionAddAdjustment:
		err = s.planAddAdjustment(ctx, req, plan)
	case settlementshared.ActionRemoveAdjustment:
		plan.Refusal = PlanRemoveAdjustment(plan.After, req.LineID)
	default:
		plan.Refusal = errortypes.NewValidationError(
			"action",
			errortypes.ErrInvalid,
			"Settlement action is invalid",
		)
	}
	if err != nil {
		return nil, err
	}

	return plan, nil
}

func (s *Service) planApprove(
	ctx context.Context,
	req *settlementshared.ActionRequest,
	plan *ActionPlan,
	now int64,
) error {
	if plan.Refusal = PlanApprove(plan.After, req.TenantInfo.UserID, now); plan.Refused() {
		return nil
	}
	control, err := s.settlementControl.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return err
	}
	plan.AutoPost = control.AutoPostOnApprove
	return nil
}

func (s *Service) planPost(
	ctx context.Context,
	plan *ActionPlan,
	userID pulid.ID,
	now int64,
) error {
	if plan.Refusal = PlanPost(plan.After, userID, now); plan.Refused() {
		return nil
	}
	draft, err := s.planSettlementJournal(ctx, plan.After, userID, false)
	if err != nil {
		if settlementshared.IsRefusal(err) {
			plan.Refusal = err
			return nil
		}
		return err
	}
	if draft != nil {
		plan.Journal = draft.plan
	}
	return nil
}

func (s *Service) planVoid(
	ctx context.Context,
	req *settlementshared.ActionRequest,
	plan *ActionPlan,
	now int64,
) error {
	wasPosted := plan.Before.Status == driversettlement.StatusPosted
	if plan.Refusal = PlanVoid(plan.After, req.Reason, req.TenantInfo.UserID, now); plan.Refused() {
		return nil
	}
	if !wasPosted {
		return nil
	}
	draft, err := s.planSettlementJournal(ctx, plan.Before, req.TenantInfo.UserID, true)
	if err != nil {
		if settlementshared.IsRefusal(err) {
			plan.Refusal = err
			return nil
		}
		return err
	}
	if draft != nil {
		plan.Journal = draft.plan
	}
	return nil
}

func (s *Service) planRecalculate(
	ctx context.Context,
	req *settlementshared.ActionRequest,
	plan *ActionPlan,
) error {
	if plan.Refusal = PlanRecalculate(plan.After); plan.Refused() {
		return nil
	}
	control, err := s.settlementControl.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return err
	}
	rebuilt, _, err := s.buildSettlement(ctx, &GenerateForWorkerRequest{
		TenantInfo:   req.TenantInfo,
		WorkerID:     plan.After.WorkerID,
		PeriodStart:  plan.After.PeriodStart,
		PeriodEnd:    plan.After.PeriodEnd,
		PayDate:      plan.After.PayDate,
		BatchID:      plan.After.BatchID,
		ReleasedFrom: plan.After.ID,
	}, control)
	if err != nil {
		if settlementshared.IsRefusal(err) {
			plan.Refusal = err
			return nil
		}
		return err
	}
	if rebuilt == nil {
		plan.Refusal = errNoAccruedEventsRemain()
		return nil
	}
	MergeRebuilt(plan.After, rebuilt)
	return nil
}

func (s *Service) planAddAdjustment(
	ctx context.Context,
	req *settlementshared.ActionRequest,
	plan *ActionPlan,
) error {
	input := adjustmentLineInput(req.Adjustment)
	if plan.Refusal = PlanAddAdjustment(plan.After, input); plan.Refused() {
		return nil
	}
	if input.PayCodeID == nil || input.PayCodeID.IsNil() {
		return nil
	}
	if _, err := s.payCodeRepo.GetByID(ctx, repositories.GetPayCodeByIDRequest{
		ID:         *input.PayCodeID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		if settlementshared.IsRefusal(err) {
			plan.Refusal = err
			return nil
		}
		return err
	}
	return nil
}

func (s *Service) Perform(
	ctx context.Context,
	req *settlementshared.ActionRequest,
	actor *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	switch req.Action {
	case settlementshared.ActionSubmit:
		return s.SubmitForApproval(ctx, req.TenantInfo, req.SettlementID, actor)
	case settlementshared.ActionApprove:
		return s.Approve(ctx, req.TenantInfo, req.SettlementID, actor)
	case settlementshared.ActionReject:
		return s.Reject(ctx, req.TenantInfo, req.SettlementID, req.Reason, actor)
	case settlementshared.ActionPost:
		return s.Post(ctx, req.TenantInfo, req.SettlementID, actor)
	case settlementshared.ActionMarkPaid:
		return s.MarkPaid(ctx, &serviceports.MarkSettlementPaidRequest{
			TenantInfo:       req.TenantInfo,
			SettlementID:     req.SettlementID,
			PaymentMethod:    req.PaymentMethod,
			PaymentReference: req.PaymentReference,
		}, actor)
	case settlementshared.ActionVoid:
		return s.Void(ctx, req.TenantInfo, req.SettlementID, req.Reason, actor)
	case settlementshared.ActionRecalculate:
		return s.Recalculate(ctx, req.TenantInfo, req.SettlementID, actor)
	case settlementshared.ActionAddAdjustment:
		return s.AddAdjustmentLine(
			ctx,
			req.TenantInfo,
			req.SettlementID,
			adjustmentLineInput(req.Adjustment),
			actor,
		)
	case settlementshared.ActionRemoveAdjustment:
		return s.RemoveAdjustmentLine(ctx, req.TenantInfo, req.SettlementID, req.LineID, actor)
	default:
		return nil, errortypes.NewValidationError(
			"action",
			errortypes.ErrInvalid,
			"Settlement action is invalid",
		)
	}
}

func errNoAccruedEventsRemain() error {
	return errortypes.NewValidationError(
		"settlementId",
		errortypes.ErrInvalid,
		"No accrued pay events remain for this settlement's period",
	)
}
