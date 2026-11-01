package billingqueueassist

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueassist/reviewprompt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	generationPage = 100
	// generationRest is how long after a generation that left items it could
	// not review opening the queue leaves them to the schedule, so a queue
	// whose items keep failing is not sent round again by every reader.
	generationRest = int64(10 * 60)
)

// NovaGeneration is the queue's generation under way. Opening a queue with
// reviews or a brief missing starts one, unless one ended moments ago leaving
// items it could not review: those are reported as left, with when opening
// the queue tries them again.
func (s *Service) NovaGeneration(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*billingqueueports.NovaGeneration, error) {
	if s.generator == nil {
		return nil, nil //nolint:nilnil // nothing generates in bulk without the worker's generator
	}
	state, err := s.generator.Generation(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if state.Running != nil {
		return state.Running, nil
	}
	if state.LeftUnreviewed && timeutils.NowUnix()-state.EndedAt < generationRest {
		left, leftErr := s.HasUnreviewedItems(ctx, tenantInfo)
		if leftErr != nil {
			return nil, leftErr
		}
		if left {
			return &billingqueueports.NovaGeneration{
				Phase:   billingqueueports.NovaPhaseLeft,
				RetryAt: state.EndedAt + generationRest,
			}, nil
		}
	}

	needed, err := s.needsGeneration(ctx, tenantInfo)
	if err != nil || !needed {
		return nil, err
	}
	s.generator.RequestGeneration(ctx, tenantInfo, billingqueueports.NovaTriggerQueueOpened)

	return &billingqueueports.NovaGeneration{Phase: billingqueueports.NovaPhaseReviewing}, nil
}

func (s *Service) HasUnreviewedItems(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	available, err := s.Availability(ctx, tenantInfo)
	if err != nil || !available.Review {
		return false, err
	}

	return s.hasUnreviewed(ctx, tenantInfo)
}

func (s *Service) hasUnreviewed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	pending, err := s.repo.ListUnreviewedItems(ctx, &billingqueueports.ListUnreviewedItemsRequest{
		TenantInfo: tenantInfo,
		Limit:      1,
	})

	return len(pending) > 0, err
}

func (s *Service) needsGeneration(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	available, err := s.Availability(ctx, tenantInfo)
	if err != nil {
		return false, err
	}
	if available.Review {
		pending, listErr := s.hasUnreviewed(ctx, tenantInfo)
		if listErr != nil || pending {
			return pending, listErr
		}
	}
	if !available.Brief {
		return false, nil
	}
	if _, err = s.repo.GetBrief(ctx, tenantInfo); errortypes.IsNotFoundError(err) {
		return true, nil
	}

	return false, err
}

// SubmitReviews sends one page of the items Nova has not reviewed as a batch.
// A batch takes one answer, so each review is given what it would have read
// (readAhead) instead of reading it. A page
// someone is waiting on, or one a provider with no batch API cannot take, is
// left to be reviewed one item at a time, each gathering its own evidence.
func (s *Service) SubmitReviews(
	ctx context.Context,
	req *billingqueueports.SubmitReviewsRequest,
) (*billingqueueports.ReviewSubmission, error) {
	submission := &billingqueueports.ReviewSubmission{}
	available, err := s.Availability(ctx, req.TenantInfo)
	if err != nil || !available.Review {
		return submission, err
	}

	pending, err := s.repo.ListUnreviewedItems(ctx, &billingqueueports.ListUnreviewedItemsRequest{
		TenantInfo: req.TenantInfo,
		After:      req.After,
		Limit:      generationPage,
	})
	if err != nil {
		return nil, err
	}
	if len(pending) == generationPage {
		last := pending[len(pending)-1]
		submission.Next = &billingqueueports.ReviewCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	if req.Inline || s.batches == nil {
		submission.Inline = make([]pulid.ID, 0, len(pending))
		for _, unreviewed := range pending {
			submission.Inline = append(submission.Inline, unreviewed.ID)
		}

		return submission, nil
	}

	items := make([]llmports.NovaBatchItem, 0, len(pending))
	ahead := s.newReadAhead(req.TenantInfo)
	for _, unreviewed := range pending {
		evidence, gatherErr := s.gatherEvidence(ctx, req.TenantInfo, unreviewed.ID)
		if gatherErr != nil {
			s.l.Warn("an item's evidence could not be gathered for review",
				zap.String("billingQueueItemId", unreviewed.ID.String()), zap.Error(gatherErr))
			continue
		}
		if evidence == nil {
			continue
		}
		task := s.reviewTask(req.TenantInfo, evidence)
		task.Context.Sections = append(task.Context.Sections, ahead.sections(ctx, evidence)...)
		items = append(items, llmports.NovaBatchItem{CustomID: unreviewed.ID.String(), Task: task})
		submission.Tickets = append(submission.Tickets, billingqueueports.ReviewTicket{
			ItemID: unreviewed.ID,
			Hash:   reviewprompt.Hash(evidence),
		})
	}
	if len(items) == 0 {
		return submission, nil
	}

	handle, err := s.batches.SubmitTasks(ctx, &llmports.NovaBatch{
		TenantInfo: req.TenantInfo,
		Task:       aitask.TaskBillingDiagnosis,
		Items:      items,
	})
	if errors.Is(err, llmports.ErrBatchUnsupported) ||
		errors.Is(err, llmports.ErrNoProviderConfigured) {
		submission.Inline = ticketItems(submission.Tickets)
		submission.Tickets = nil

		return submission, nil
	}
	if err != nil {
		return nil, err
	}
	submission.Handle = &billingqueueports.ReviewBatch{
		ProviderID:  handle.ProviderID,
		BatchID:     handle.BatchID,
		SubmittedAt: handle.SubmittedAt,
	}

	return submission, nil
}

func (s *Service) ReviewBatchEnded(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	submission *billingqueueports.ReviewSubmission,
) (bool, error) {
	if submission.Handle == nil || s.batches == nil {
		return true, nil
	}
	status, err := s.batches.BatchStatus(ctx, &llmports.BatchStatusRequest{
		TenantInfo: tenantInfo,
		ProviderID: submission.Handle.ProviderID,
		BatchID:    submission.Handle.BatchID,
	})
	if err != nil {
		return false, err
	}

	return status.Ended, nil
}

// CancelReviews stops a batch's reviews that have not run, for someone now
// waiting on them; the batch ends with those that had, which ApplyReviews
// records, and the rest are reviewed one at a time.
func (s *Service) CancelReviews(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	submission *billingqueueports.ReviewSubmission,
) error {
	if submission.Handle == nil || s.batches == nil {
		return nil
	}

	return s.batches.CancelBatch(ctx, &llmports.BatchCancelRequest{
		TenantInfo: tenantInfo,
		ProviderID: submission.Handle.ProviderID,
		BatchID:    submission.Handle.BatchID,
	})
}

// ApplyReviews records an ended batch's reviews. An item reviewed since the
// batch was sent, or whose evidence has moved, keeps what it has; the answers
// are otherwise recorded exactly as a review run on its own would be, and an
// item Nova agrees with is offered to auto-approve.
func (s *Service) ApplyReviews(
	ctx context.Context,
	req *billingqueueports.ApplyReviewsRequest,
) (int, error) {
	submission := req.Submission
	if submission.Handle == nil || s.batches == nil {
		return 0, nil
	}

	attributions := make(map[string]aiusageports.AIUsageAttribution, len(submission.Tickets))
	for _, ticket := range submission.Tickets {
		attributions[ticket.ItemID.String()] = aiusageports.AIUsageAttribution{
			Feature: aiusage.FeatureBillingQueueReview,
		}
	}
	results, err := s.batches.BatchResults(ctx, &llmports.BatchResultsRequest{
		TenantInfo:   req.TenantInfo,
		Task:         aitask.TaskBillingDiagnosis,
		ProviderID:   submission.Handle.ProviderID,
		BatchID:      submission.Handle.BatchID,
		SubmittedAt:  submission.Handle.SubmittedAt,
		Attributions: attributions,
	})
	if err != nil {
		return 0, err
	}

	applied := 0
	ahead := s.newReadAhead(req.TenantInfo)
	for _, ticket := range submission.Tickets {
		result, ok := results[ticket.ItemID.String()]
		if ok && result.NotRun {
			continue
		}
		if !ok || !result.Succeeded() {
			s.l.Warn("a billing review in a batch failed",
				zap.String("billingQueueItemId", ticket.ItemID.String()),
				zap.String("code", result.FailureCode),
				zap.String("message", result.FailureMessage))
			continue
		}
		evidence, gatherErr := s.gatherEvidence(ctx, req.TenantInfo, ticket.ItemID)
		if gatherErr != nil {
			s.l.Warn("an item's evidence could not be gathered to record its review",
				zap.String("billingQueueItemId", ticket.ItemID.String()), zap.Error(gatherErr))
			continue
		}
		if evidence == nil || evidence.Item.AgentReviewHash != "" ||
			reviewprompt.Hash(evidence) != ticket.Hash {
			continue
		}

		answer := &reviewprompt.Answer{}
		decodeErr := llmports.DecodeAnswer(aitask.TaskBillingDiagnosis, result.Result.Text, answer)
		outcome, settleErr := s.settleReview(
			ctx,
			req.TenantInfo,
			evidence,
			ahead.taken(ctx, evidence),
			answer,
			decodeErr,
		)
		if settleErr != nil {
			return applied, settleErr
		}
		applied++
		if outcome.Verdict == billingqueue.AgentVerdictAgrees {
			s.offerAutoApprove(ctx, req.TenantInfo, ticket.ItemID)
		}
	}

	return applied, nil
}

func (s *Service) offerAutoApprove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	itemID pulid.ID,
) {
	if _, err := s.AutoApprove(ctx, tenantInfo, itemID); err != nil {
		s.l.Info("an item Nova agreed with was not auto-approved",
			zap.String("billingQueueItemId", itemID.String()), zap.Error(err))
	}
}

func ticketItems(tickets []billingqueueports.ReviewTicket) []pulid.ID {
	ids := make([]pulid.ID, 0, len(tickets))
	for _, ticket := range tickets {
		ids = append(ids, ticket.ItemID)
	}

	return ids
}
