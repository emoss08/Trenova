package carriersettlementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type ActionPlan = settlementshared.ActionPlan[*carriersettlement.CarrierSettlement]

func CloneSettlement(
	entity *carriersettlement.CarrierSettlement,
) *carriersettlement.CarrierSettlement {
	if entity == nil {
		return nil
	}
	clone := *entity
	clone.Lines = make([]*carriersettlement.CarrierSettlementLine, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		copied := *line
		clone.Lines = append(clone.Lines, &copied)
	}
	return &clone
}

func PlanSubmit(entity *carriersettlement.CarrierSettlement, userID pulid.ID, now int64) error {
	if !carriersettlement.IsAllowedTransition(
		entity.Status,
		carriersettlement.StatusPendingApproval,
	) || entity.Status == carriersettlement.StatusPendingApproval {
		return transitionError(entity.Status, carriersettlement.StatusPendingApproval)
	}
	entity.Status = carriersettlement.StatusPendingApproval
	entity.SubmittedByID = userID
	entity.SubmittedAt = &now
	return nil
}

func PlanApprove(entity *carriersettlement.CarrierSettlement, userID pulid.ID, now int64) error {
	if entity.Status != carriersettlement.StatusPendingApproval {
		return transitionError(entity.Status, carriersettlement.StatusApproved)
	}
	entity.Status = carriersettlement.StatusApproved
	entity.ApprovedByID = userID
	entity.ApprovedAt = &now
	return nil
}

func PlanReject(entity *carriersettlement.CarrierSettlement, reason string) error {
	if reason == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A rejection reason is required",
		)
	}
	if entity.Status != carriersettlement.StatusPendingApproval {
		return transitionError(entity.Status, carriersettlement.StatusDraft)
	}
	entity.Status = carriersettlement.StatusDraft
	entity.SubmittedByID = pulid.Nil
	entity.SubmittedAt = nil
	if entity.Notes != "" {
		entity.Notes += "\n"
	}
	entity.Notes += "Rejected: " + reason
	return nil
}

func PlanPost(entity *carriersettlement.CarrierSettlement, userID pulid.ID, now int64) error {
	if entity.Status != carriersettlement.StatusApproved {
		return transitionError(entity.Status, carriersettlement.StatusPosted)
	}
	entity.Status = carriersettlement.StatusPosted
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

func CheckMarkPaid(entity *carriersettlement.CarrierSettlement, paymentMethod string) error {
	if paymentMethod == "" {
		return errortypes.NewValidationError(
			"paymentMethod",
			errortypes.ErrRequired,
			"Payment method is required",
		)
	}
	if entity.Status != carriersettlement.StatusPosted {
		return transitionError(entity.Status, carriersettlement.StatusPaid)
	}
	return nil
}

func PlanMarkPaid(entity *carriersettlement.CarrierSettlement, input *MarkPaidInput) error {
	if err := CheckMarkPaid(entity, input.PaymentMethod); err != nil {
		return err
	}
	paidAt := input.PaidAt
	entity.Status = carriersettlement.StatusPaid
	entity.PaidAt = &paidAt
	entity.PaidByID = input.UserID
	entity.PaymentMethod = input.PaymentMethod
	entity.PaymentReference = input.PaymentReference
	return nil
}

func CheckVoid(entity *carriersettlement.CarrierSettlement, reason string) error {
	if reason == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"A void reason is required",
		)
	}
	if !carriersettlement.IsAllowedTransition(
		entity.Status,
		carriersettlement.StatusVoided,
	) || entity.Status == carriersettlement.StatusVoided {
		return transitionError(entity.Status, carriersettlement.StatusVoided)
	}
	return nil
}

func PlanVoid(
	entity *carriersettlement.CarrierSettlement,
	reason string,
	userID pulid.ID,
	now int64,
) error {
	if err := CheckVoid(entity, reason); err != nil {
		return err
	}
	entity.Status = carriersettlement.StatusVoided
	entity.VoidedByID = userID
	entity.VoidedAt = &now
	entity.VoidReason = reason
	return nil
}

func PlanRecalculate(entity *carriersettlement.CarrierSettlement) error {
	if entity.Status != carriersettlement.StatusDraft {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only draft carrier settlements can be recalculated",
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

func checkEditable(entity *carriersettlement.CarrierSettlement) error {
	if !entity.IsEditable() {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only draft or pending carrier settlements can be adjusted",
		)
	}
	return nil
}

func PlanAddAdjustment(
	entity *carriersettlement.CarrierSettlement,
	input *AdjustmentLineInput,
) error {
	if err := checkAdjustmentInput(input); err != nil {
		return err
	}
	if err := checkEditable(entity); err != nil {
		return err
	}
	var glAccountID *pulid.ID
	if input.GLAccountID != nil && !input.GLAccountID.IsNil() {
		glAccountID = input.GLAccountID
	}
	entity.Lines = append(entity.Lines, &carriersettlement.CarrierSettlementLine{
		EventType:   carriersettlement.CostEventTypeAdjustment,
		Description: input.Description,
		AmountMinor: input.AmountMinor,
		GLAccountID: glAccountID,
	})
	entity.SyncTotals()
	return nil
}

func PlanRemoveAdjustment(entity *carriersettlement.CarrierSettlement, lineID pulid.ID) error {
	if err := checkEditable(entity); err != nil {
		return err
	}
	found := false
	remaining := make([]*carriersettlement.CarrierSettlementLine, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		if line.ID == lineID {
			if !line.IsManualAdjustment() {
				return errortypes.NewValidationError(
					"lineId",
					errortypes.ErrInvalidOperation,
					"Only manual adjustment lines can be removed",
				)
			}
			found = true
			continue
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
	entity.SyncTotals()
	return nil
}

func MergeRebuilt(entity, rebuilt *carriersettlement.CarrierSettlement) {
	manualLines := make([]*carriersettlement.CarrierSettlementLine, 0)
	for _, line := range entity.Lines {
		if line != nil && line.IsManualAdjustment() {
			line.ID = pulid.Nil
			manualLines = append(manualLines, line)
		}
	}

	mergedLines := make(
		[]*carriersettlement.CarrierSettlementLine,
		0,
		len(rebuilt.Lines)+len(manualLines),
	)
	mergedLines = append(mergedLines, rebuilt.Lines...)
	mergedLines = append(mergedLines, manualLines...)
	entity.Lines = mergedLines
	entity.CurrencyCode = rebuilt.CurrencyCode
	entity.ShipmentCount = rebuilt.ShipmentCount
	entity.SyncTotals()
}

func adjustmentLineInput(input *settlementshared.AdjustmentInput) *AdjustmentLineInput {
	if input == nil {
		return &AdjustmentLineInput{}
	}
	return &AdjustmentLineInput{
		Description: input.Description,
		AmountMinor: input.AmountMinor,
		GLAccountID: input.GLAccountID,
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
		err = s.planMarkPaid(ctx, req, plan, now)
	case settlementshared.ActionVoid:
		err = s.planVoid(ctx, req, plan, now)
	case settlementshared.ActionRecalculate:
		err = s.planRecalculate(ctx, req, plan)
	case settlementshared.ActionAddAdjustment:
		plan.Refusal = PlanAddAdjustment(plan.After, adjustmentLineInput(req.Adjustment))
	case settlementshared.ActionRemoveAdjustment:
		plan.Refusal = PlanRemoveAdjustment(plan.After, req.LineID)
	default:
		plan.Refusal = errortypes.NewValidationError(
			"action",
			errortypes.ErrInvalid,
			"Carrier settlement action is invalid",
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

func (s *Service) keepJournal(plan *ActionPlan, draft *journalDraft, err error) error {
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
	return s.keepJournal(plan, draft, err)
}

func (s *Service) planMarkPaid(
	ctx context.Context,
	req *settlementshared.ActionRequest,
	plan *ActionPlan,
	now int64,
) error {
	if plan.Refusal = PlanMarkPaid(plan.After, &MarkPaidInput{
		PaymentMethod:    req.PaymentMethod,
		PaymentReference: req.PaymentReference,
		PaidAt:           now,
		UserID:           req.TenantInfo.UserID,
	}); plan.Refused() {
		return nil
	}
	draft, err := s.planPaymentJournal(ctx, plan.Before, req.TenantInfo.UserID, now)
	return s.keepJournal(plan, draft, err)
}

func (s *Service) planVoid(
	ctx context.Context,
	req *settlementshared.ActionRequest,
	plan *ActionPlan,
	now int64,
) error {
	wasPosted := plan.Before.Status == carriersettlement.StatusPosted
	if plan.Refusal = PlanVoid(plan.After, req.Reason, req.TenantInfo.UserID, now); plan.Refused() {
		return nil
	}
	if !wasPosted {
		return nil
	}
	draft, err := s.planSettlementJournal(ctx, plan.Before, req.TenantInfo.UserID, true)
	return s.keepJournal(plan, draft, err)
}

func (s *Service) planRecalculate(
	ctx context.Context,
	req *settlementshared.ActionRequest,
	plan *ActionPlan,
) error {
	if plan.Refusal = PlanRecalculate(plan.After); plan.Refused() {
		return nil
	}
	rebuilt, _, err := s.buildSettlement(ctx, &GenerateForCarrierRequest{
		TenantInfo:   req.TenantInfo,
		CarrierID:    plan.After.CarrierID,
		PeriodStart:  plan.After.PeriodStart,
		PeriodEnd:    plan.After.PeriodEnd,
		PayDate:      plan.After.PayDate,
		BatchID:      plan.After.BatchID,
		ReleasedFrom: plan.After.ID,
	})
	if err != nil {
		if settlementshared.IsRefusal(err) {
			plan.Refusal = err
			return nil
		}
		return err
	}
	if rebuilt == nil {
		plan.Refusal = errNoPendingCostEventsRemain()
		return nil
	}
	MergeRebuilt(plan.After, rebuilt)
	return nil
}

func (s *Service) Perform(
	ctx context.Context,
	req *settlementshared.ActionRequest,
	actor *serviceports.RequestActor,
) (*carriersettlement.CarrierSettlement, error) {
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
			"Carrier settlement action is invalid",
		)
	}
}

func errNoPendingCostEventsRemain() error {
	return errortypes.NewValidationError(
		"settlementId",
		errortypes.ErrInvalid,
		"No pending cost events remain for this settlement's period",
	)
}
