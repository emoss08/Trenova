package factoringservice

import (
	"context"
	"errors"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringremittanceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

type applyStep struct {
	kind         factoringremittance.RemittanceLineKind
	submissionID pulid.ID
	lines        []*factoringremittance.RemittanceLine
	items        map[pulid.ID]*factoringsubmission.SubmissionItem
}

type applyRun struct {
	rem      *factoringremittance.Remittance
	actor    *permissionports.RequestActor
	steps    []*applyStep
	versions map[pulid.ID]int64
}

func (s *RemittanceService) PlanApply(
	ctx context.Context,
	req *factoringremittanceports.RemittanceVersionRequest,
	actor *permissionports.RequestActor,
) (*factoringremittanceports.RemittancePlan, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireExecutor(actor, "Applying a remittance"); err != nil {
		return nil, err
	}
	rem, err := s.remittances.Get(ctx, factoringremittanceports.GetRemittanceRequest{
		TenantInfo:   req.TenantInfo,
		ID:           req.ID,
		IncludeLines: true,
	})
	if err != nil {
		return nil, err
	}
	if err = requireVersion(rem, req.Version); err != nil {
		return nil, err
	}
	run, err := s.prepareApply(ctx, rem, actor)
	if err != nil {
		return nil, err
	}
	plan := &factoringremittanceports.RemittancePlan{
		Remittance: withoutLines(rem),
		Lines:      make([]*factoringremittanceports.PlannedRemittanceLine, 0, rem.MatchedCount),
	}
	for _, step := range run.steps {
		if err = s.planStep(ctx, run, step, plan); err != nil {
			return nil, err
		}
	}
	for _, line := range rem.Lines {
		if line != nil && line.MatchStatus.IsException() {
			plan.LeftOpenCount++
		}
	}
	return plan, nil
}

func (s *RemittanceService) Apply(
	ctx context.Context,
	req *factoringremittanceports.RemittanceVersionRequest,
	actor *permissionports.RequestActor,
) (*factoringremittanceports.ApplyRemittanceResult, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if err := requireExecutor(actor, "Applying a remittance"); err != nil {
		return nil, err
	}
	result := &factoringremittanceports.ApplyRemittanceResult{}
	var before *factoringremittance.Remittance
	var booked int
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		rem, err := s.lock(txCtx, req.TenantInfo, req.ID, req.Version)
		if err != nil {
			return err
		}
		before = cloneRemittance(rem)
		run, err := s.prepareApply(txCtx, rem, actor)
		if err != nil {
			return err
		}
		applied := make([]*factoringremittance.RemittanceLine, 0, rem.MatchedCount)
		for _, step := range run.steps {
			entries, stepErr := s.runStep(txCtx, run, step)
			if stepErr != nil {
				return stepErr
			}
			result.Entries = append(result.Entries, entries...)
			applied = append(applied, step.lines...)
		}
		booked = len(applied)
		rem.MarkApplied(s.now(), actor.ExecutorUserID())
		if err = s.remittances.UpdateLines(txCtx, applied); err != nil {
			return err
		}
		result.Remittance, err = s.save(txCtx, rem)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.record(ctx, actor, before, result.Remittance, permission.OpImport,
		"Remittance "+remittanceName(result.Remittance)+" applied: "+
			strconv.Itoa(booked)+" lines booked, "+
			strconv.Itoa(result.Remittance.ExceptionCount)+" left to resolve")
	return result, nil
}

func (s *RemittanceService) prepareApply(
	ctx context.Context,
	rem *factoringremittance.Remittance,
	actor *permissionports.RequestActor,
) (*applyRun, error) {
	if err := rem.RequireApplicable(); err != nil {
		return nil, err
	}
	matched := make([]*factoringremittance.RemittanceLine, 0, rem.MatchedCount)
	itemIDs := make([]pulid.ID, 0, rem.MatchedCount)
	for _, line := range rem.Lines {
		if line != nil && line.MatchStatus == factoringremittance.RemittanceLineMatched {
			matched = append(matched, line)
			itemIDs = append(itemIDs, line.SubmissionItemID)
		}
	}
	candidates, err := s.remittances.ListCandidates(
		ctx,
		&factoringremittanceports.ListRemittanceCandidatesRequest{
			TenantInfo:         remittanceTenant(rem),
			FactoringCompanyID: rem.FactoringCompanyID,
			ItemIDs:            itemIDs,
			Limit:              len(itemIDs),
		},
	)
	if err != nil {
		return nil, err
	}
	byItem := make(map[pulid.ID]*factoringremittance.RemittanceCandidate, len(candidates))
	for _, candidate := range candidates {
		byItem[candidate.Item.ID] = candidate
	}
	steps, err := groupSteps(matched, byItem)
	if err != nil {
		return nil, err
	}
	versions, err := s.submissionVersions(ctx, rem, steps)
	if err != nil {
		return nil, err
	}
	return &applyRun{rem: rem, actor: actor, steps: steps, versions: versions}, nil
}

func groupSteps(
	lines []*factoringremittance.RemittanceLine,
	byItem map[pulid.ID]*factoringremittance.RemittanceCandidate,
) ([]*applyStep, error) {
	steps := make([]*applyStep, 0, len(lines))
	funding := make(map[pulid.ID]*applyStep, len(lines))
	for _, line := range lines {
		candidate := byItem[line.SubmissionItemID]
		if candidate == nil || !line.Kind.Accepts(candidate) {
			return nil, errortypes.NewBusinessError(
				"Row {0} matched an invoice that can no longer take {1}; match the remittance again",
				line.RowNumber,
				kindPhrase(line.Kind),
			)
		}
		item := candidate.Item
		switch line.Kind {
		case factoringremittance.RemittanceLineFunding, factoringremittance.RemittanceLineRejection:
			step, ok := funding[item.SubmissionID]
			if !ok {
				step = &applyStep{
					kind:         factoringremittance.RemittanceLineFunding,
					submissionID: item.SubmissionID,
					items:        make(map[pulid.ID]*factoringsubmission.SubmissionItem, 4),
				}
				funding[item.SubmissionID] = step
				steps = append(steps, step)
			}
			step.lines = append(step.lines, line)
			step.items[line.ID] = item
		case factoringremittance.RemittanceLineReserveRelease,
			factoringremittance.RemittanceLineChargeback:
			steps = append(steps, &applyStep{
				kind:         line.Kind,
				submissionID: item.SubmissionID,
				lines:        []*factoringremittance.RemittanceLine{line},
				items:        map[pulid.ID]*factoringsubmission.SubmissionItem{line.ID: item},
			})
		case factoringremittance.RemittanceLineOther:
		}
	}
	return steps, nil
}

func (s *RemittanceService) submissionVersions(
	ctx context.Context,
	rem *factoringremittance.Remittance,
	steps []*applyStep,
) (map[pulid.ID]int64, error) {
	ids := make([]pulid.ID, 0, len(steps))
	seen := make(map[pulid.ID]struct{}, len(steps))
	for _, step := range steps {
		if _, dup := seen[step.submissionID]; !dup {
			seen[step.submissionID] = struct{}{}
			ids = append(ids, step.submissionID)
		}
	}
	subs, err := s.submissions.GetByIDs(ctx, factoringsubmissionports.GetSubmissionsByIDsRequest{
		TenantInfo: remittanceTenant(rem),
		IDs:        ids,
	})
	if err != nil {
		return nil, err
	}
	versions := make(map[pulid.ID]int64, len(subs))
	for _, sub := range subs {
		versions[sub.ID] = sub.Version
	}
	return versions, nil
}

func (s *RemittanceService) planStep(
	ctx context.Context,
	run *applyRun,
	step *applyStep,
	plan *factoringremittanceports.RemittancePlan,
) error {
	switch step.kind {
	case factoringremittance.RemittanceLineFunding:
		funding, err := s.ledger.PlanFunding(ctx, s.fundingRequest(run, step), run.actor)
		if err != nil {
			return stepError(err)
		}
		plan.FundedCount += funding.FundedCount
		plan.RejectedCount += funding.RejectCount
		plan.AdvanceMinor += funding.AdvanceMinor
		plan.FeeMinor += funding.FeeMinor
		for _, line := range step.lines {
			planned := plannedLine(line, step)
			for _, movement := range funding.Movements {
				if movement.ItemID == planned.ItemID {
					planned.Movement = movement
				}
			}
			plan.Lines = append(plan.Lines, planned)
		}
		return nil
	case factoringremittance.RemittanceLineReserveRelease,
		factoringremittance.RemittanceLineChargeback:
		var movement *factoringledgerports.ReserveMovementPlan
		var err error
		req := s.movementRequest(run, step)
		if step.kind == factoringremittance.RemittanceLineReserveRelease {
			movement, err = s.ledger.PlanRelease(ctx, req, run.actor)
		} else {
			movement, err = s.ledger.PlanChargeback(ctx, req, run.actor)
		}
		if err != nil {
			return stepError(err)
		}
		planned := plannedLine(step.lines[0], step)
		planned.Movement = movement.Movement
		if movement.Movement != nil {
			planned.CashMinor = movementCash(movement.Movement)
		}
		if step.kind == factoringremittance.RemittanceLineReserveRelease {
			plan.ReleasedCount++
			plan.ReleasedMinor += planned.CashMinor
		} else {
			plan.ChargebackCount++
			plan.ChargebackMinor += planned.CashMinor
		}
		plan.Lines = append(plan.Lines, planned)
		return nil
	case factoringremittance.RemittanceLineRejection, factoringremittance.RemittanceLineOther:
	}
	return nil
}

func (s *RemittanceService) runStep(
	ctx context.Context,
	run *applyRun,
	step *applyStep,
) ([]*factoringledger.ReserveEntry, error) {
	switch step.kind {
	case factoringremittance.RemittanceLineFunding:
		funded, err := s.ledger.RecordFunding(ctx, s.fundingRequest(run, step), run.actor)
		if err != nil {
			return nil, stepError(err)
		}
		run.versions[step.submissionID] = funded.Submission.Version
		byItem := make(map[pulid.ID]pulid.ID, len(funded.Entries))
		for _, entry := range funded.Entries {
			byItem[entry.SubmissionItemID] = entry.ID
		}
		for _, line := range step.lines {
			line.MarkApplied(byItem[step.items[line.ID].ID])
		}
		return funded.Entries, nil
	case factoringremittance.RemittanceLineReserveRelease,
		factoringremittance.RemittanceLineChargeback:
		var moved *factoringledgerports.ReserveMovementResult
		var err error
		req := s.movementRequest(run, step)
		if step.kind == factoringremittance.RemittanceLineReserveRelease {
			moved, err = s.ledger.ReleaseReserve(ctx, req, run.actor)
		} else {
			moved, err = s.ledger.RecordChargeback(ctx, req, run.actor)
		}
		if err != nil {
			return nil, stepError(err)
		}
		run.versions[step.submissionID] = moved.Submission.Version
		entryID := pulid.Nil
		entries := make([]*factoringledger.ReserveEntry, 0, 1)
		if moved.Entry != nil {
			entryID = moved.Entry.ID
			entries = append(entries, moved.Entry)
		}
		step.lines[0].MarkApplied(entryID)
		return entries, nil
	case factoringremittance.RemittanceLineRejection, factoringremittance.RemittanceLineOther:
	}
	return nil, nil
}

func (s *RemittanceService) fundingRequest(
	run *applyRun,
	step *applyStep,
) *factoringledgerports.RecordFundingRequest {
	items := make([]factoringledgerports.FundingItemInput, 0, len(step.lines))
	fundedOn := int64(0)
	for _, line := range step.lines {
		fundedOn = max(fundedOn, s.lineDate(run.rem, line))
		input := factoringledgerports.FundingItemInput{
			ItemID:           step.items[line.ID].ID,
			Outcome:          factoringledger.FundingOutcomeFunded,
			RemittanceLineID: line.ID,
		}
		if line.Kind == factoringremittance.RemittanceLineRejection {
			input.Outcome = factoringledger.FundingOutcomeRejected
			input.Reason = "Declined on the factor's remittance " + remittanceName(run.rem) +
				", row " + strconv.Itoa(line.RowNumber)
		} else {
			input.AdvanceMinor = fundedAdvance(line)
			input.FeeMinor = line.FeeMinor
		}
		items = append(items, input)
	}
	return &factoringledgerports.RecordFundingRequest{
		TenantInfo:   remittanceTenant(run.rem),
		SubmissionID: step.submissionID,
		Version:      run.versions[step.submissionID],
		FundedOn:     fundedOn,
		Items:        items,
		Memo:         "Factor remittance " + remittanceName(run.rem),
	}
}

func (s *RemittanceService) movementRequest(
	run *applyRun,
	step *applyStep,
) *factoringledgerports.ReserveMovementRequest {
	line := step.lines[0]
	req := &factoringledgerports.ReserveMovementRequest{
		TenantInfo:       remittanceTenant(run.rem),
		SubmissionID:     step.submissionID,
		ItemID:           step.items[line.ID].ID,
		Version:          run.versions[step.submissionID],
		OccurredOn:       s.lineDate(run.rem, line),
		RemittanceLineID: line.ID,
	}
	source := line.CashMinor
	if source == nil && line.Kind == factoringremittance.RemittanceLineReserveRelease {
		source = line.ReserveMinor
	}
	if source != nil {
		req.CashMinor = new(intutils.Abs(*source))
	}
	if line.Kind == factoringremittance.RemittanceLineChargeback {
		req.Reason = "Charged back on the factor's remittance " + remittanceName(run.rem) +
			", row " + strconv.Itoa(line.RowNumber)
	} else {
		req.Reason = "Released on the factor's remittance " + remittanceName(run.rem) +
			", row " + strconv.Itoa(line.RowNumber)
	}
	return req
}

func fundedAdvance(line *factoringremittance.RemittanceLine) *int64 {
	switch {
	case line.AdvanceMinor != nil:
		return line.AdvanceMinor
	case line.CashMinor != nil:
		return new(intutils.Abs(*line.CashMinor))
	default:
		return nil
	}
}

func (s *RemittanceService) lineDate(
	rem *factoringremittance.Remittance,
	line *factoringremittance.RemittanceLine,
) int64 {
	now := s.now()
	for _, candidate := range []*int64{line.OccurredOn, rem.RemittedOn, rem.StagedAt} {
		if candidate != nil && *candidate > 0 && *candidate <= now+maxFutureMovementSeconds {
			return *candidate
		}
	}
	return now
}

func plannedLine(
	line *factoringremittance.RemittanceLine,
	step *applyStep,
) *factoringremittanceports.PlannedRemittanceLine {
	item := step.items[line.ID]
	return &factoringremittanceports.PlannedRemittanceLine{
		LineID:        line.ID,
		RowNumber:     line.RowNumber,
		Kind:          line.Kind,
		InvoiceNumber: item.InvoiceNumber,
		SubmissionID:  item.SubmissionID,
		ItemID:        item.ID,
		CashMinor:     line.SignedCash(),
	}
}

func movementCash(movement *factoringledgerports.PlannedMovement) int64 {
	var cash int64
	for _, line := range movement.Lines {
		if line.Role == factoring.AccountRoleCash {
			cash += line.Debit - line.Credit
		}
	}
	return intutils.Abs(cash)
}

func stepError(err error) error {
	var multiErr *errortypes.MultiError
	if errors.As(err, &multiErr) && len(multiErr.Errors) > 0 {
		first := multiErr.Errors[0]
		return errortypes.NewBusinessError(first.Message, first.Args...)
	}
	return err
}
