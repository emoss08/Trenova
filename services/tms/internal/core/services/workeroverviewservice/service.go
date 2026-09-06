// Package workeroverviewservice answers one question — is this worker in good
// standing right now — by composing the roll-ups each HR area already builds.
// It owns no data and writes nothing. Sections the signed-in user cannot read
// are left out rather than faked, and the standing is computed from whatever
// arrived, so a dispatcher and an HR manager see the same verdict drawn from
// different evidence.
package workeroverviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ptoledgerservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// CredentialSummarizer is the slice of the credential service this needs.
type CredentialSummarizer interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (*worker.WorkerCredentialSummary, error)
}

// TrainingSummarizer is the slice of the training service this needs.
type TrainingSummarizer interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (*worker.WorkerTrainingSummary, error)
}

// SafetyScorer is the slice of the safety service this needs.
type SafetyScorer interface {
	Scorecard(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (*worker.SafetyScorecard, error)
}

// ChecklistLister is the slice of the checklist service this needs.
type ChecklistLister interface {
	ListForWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		includeClosed bool,
	) ([]*worker.WorkerChecklist, error)
}

// BalanceReader is the slice of the PTO ledger this needs: one worker's
// balances for the overview, and the organisation's totals for the home tile.
type BalanceReader interface {
	GetBalances(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) ([]*ptoledgerservice.BalanceView, error)
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*repositories.PTOBalanceSummary, error)
}

// ReviewCounter is the slice of the review repository the home tile needs: a
// count across the organisation, which the per-worker list cannot answer.
type ReviewCounter interface {
	CountReviews(
		ctx context.Context,
		req *repositories.CountPerformanceReviewsRequest,
	) (int, error)
}

// LeaveCounter is the slice of the leave service the home tile needs: how many
// cases across the organisation are still waiting on paperwork the employee
// owes. Counted rather than listed — the tile shows only the number.
type LeaveCounter interface {
	CountCases(
		ctx context.Context,
		req *repositories.ListLeaveCasesRequest,
	) (int, error)
}

// ReviewLister is the slice of the performance review service this needs.
type ReviewLister interface {
	ListReviews(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		statuses []worker.ReviewStatus,
	) ([]*worker.PerformanceReview, error)
}

// Sections says which parts of the record the caller may read. The resolver
// fills it from the signed-in user's permissions, so the service never has to
// know about authorization itself.
type Sections struct {
	Credentials bool
	Training    bool
	Safety      bool
	Checklists  bool
	PTO         bool
	Reviews     bool
}

// All is every section, for callers that have already been authorized in full.
func All() Sections {
	return Sections{
		Credentials: true,
		Training:    true,
		Safety:      true,
		Checklists:  true,
		PTO:         true,
		Reviews:     true,
	}
}

// Overview is the composed answer. Every section pointer is nil when the
// caller could not read it, which is deliberately the same shape as "there is
// nothing there" — the standing treats both as contributing no concerns.
type Overview struct {
	Worker       *worker.Worker
	AsOf         int64
	Standing     worker.Standing
	Concerns     []worker.Concern
	Credentials  *worker.WorkerCredentialSummary
	Training     *worker.WorkerTrainingSummary
	Safety       *worker.SafetyScorecard
	Checklist    *worker.WorkerChecklist
	PTO          []*ptoledgerservice.BalanceView
	OpenReview   *worker.PerformanceReview
	LastReview   *worker.PerformanceReview
	NextReviewAt *int64
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	WorkerRepo   repositories.WorkerRepository
	ReviewCounts ReviewCounter
	Credentials  CredentialSummarizer
	Training     TrainingSummarizer
	Safety       SafetyScorer
	Checklists   ChecklistLister
	PTO          BalanceReader
	Reviews      ReviewLister
	Leave        LeaveCounter
}

type Service struct {
	l            *zap.Logger
	workerRepo   repositories.WorkerRepository
	reviewCounts ReviewCounter
	credentials  CredentialSummarizer
	training     TrainingSummarizer
	safety       SafetyScorer
	checklists   ChecklistLister
	pto          BalanceReader
	reviews      ReviewLister
	leave        LeaveCounter
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.worker-overview"),
		workerRepo:   p.WorkerRepo,
		reviewCounts: p.ReviewCounts,
		credentials:  p.Credentials,
		training:     p.Training,
		safety:       p.Safety,
		checklists:   p.Checklists,
		pto:          p.PTO,
		reviews:      p.Reviews,
		leave:        p.Leave,
	}
}

// Deps lets tests assemble the service from narrow fakes without Uber FX.
type Deps struct {
	Logger       *zap.Logger
	WorkerRepo   repositories.WorkerRepository
	ReviewCounts ReviewCounter
	Credentials  CredentialSummarizer
	Training     TrainingSummarizer
	Safety       SafetyScorer
	Checklists   ChecklistLister
	PTO          BalanceReader
	Reviews      ReviewLister
	Leave        LeaveCounter
}

func NewWithDeps(d Deps) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		l:            logger.Named("service.worker-overview"),
		workerRepo:   d.WorkerRepo,
		reviewCounts: d.ReviewCounts,
		credentials:  d.Credentials,
		training:     d.Training,
		safety:       d.Safety,
		checklists:   d.Checklists,
		pto:          d.PTO,
		reviews:      d.Reviews,
		leave:        d.Leave,
	}
}

// Get composes the overview. The sections are read concurrently because this
// backs a single panel view and running six roll-ups in series would make the
// panel feel slower than the tabs it replaces. Any section failing fails the
// whole call: a silently missing section would read as "nothing wrong here".
func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	sections Sections,
) (*Overview, error) {
	wrk, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:             workerID,
		TenantInfo:     tenantInfo,
		IncludeProfile: true,
		IncludeState:   true,
	})
	if err != nil {
		return nil, err
	}

	// Each goroutine writes its own local and nothing is read until Wait
	// returns, so there is no shared state to reason about.
	var (
		credentials  *worker.WorkerCredentialSummary
		training     *worker.WorkerTrainingSummary
		safety       *worker.SafetyScorecard
		checklist    *worker.WorkerChecklist
		balances     []*ptoledgerservice.BalanceView
		openReview   *worker.PerformanceReview
		lastReview   *worker.PerformanceReview
		nextReviewAt *int64
	)
	group, groupCtx := errgroup.WithContext(ctx)

	if sections.Credentials {
		group.Go(func() error {
			var sErr error
			credentials, sErr = s.credentials.Summary(groupCtx, tenantInfo, workerID)
			return sErr
		})
	}
	if sections.Training {
		group.Go(func() error {
			var sErr error
			training, sErr = s.training.Summary(groupCtx, tenantInfo, workerID)
			return sErr
		})
	}
	if sections.Safety {
		group.Go(func() error {
			var sErr error
			safety, sErr = s.safety.Scorecard(groupCtx, tenantInfo, workerID)
			return sErr
		})
	}
	if sections.Checklists {
		group.Go(func() error {
			checklists, sErr := s.checklists.ListForWorker(groupCtx, tenantInfo, workerID, false)
			if sErr != nil {
				return sErr
			}
			checklist = newestOpenChecklist(checklists)
			return nil
		})
	}
	if sections.PTO {
		group.Go(func() error {
			var sErr error
			balances, sErr = s.pto.GetBalances(groupCtx, tenantInfo, workerID)
			return sErr
		})
	}
	if sections.Reviews {
		group.Go(func() error {
			reviews, sErr := s.reviews.ListReviews(groupCtx, tenantInfo, workerID, nil)
			if sErr != nil {
				return sErr
			}
			openReview, lastReview, nextReviewAt = summariseReviews(reviews)
			return nil
		})
	}

	if err = group.Wait(); err != nil {
		return nil, err
	}

	overview := &Overview{
		Worker:       wrk,
		AsOf:         timeutils.NowUnix(),
		Credentials:  credentials,
		Training:     training,
		Safety:       safety,
		Checklist:    checklist,
		PTO:          balances,
		OpenReview:   openReview,
		LastReview:   lastReview,
		NextReviewAt: nextReviewAt,
	}
	overview.Standing, overview.Concerns = worker.BuildStanding(worker.StandingInput{
		Worker:      wrk,
		Credentials: credentials,
		Training:    training,
		Safety:      safety,
		Checklist:   checklist,
		ReviewDueAt: nextReviewAt,
		Now:         overview.AsOf,
	})
	return overview, nil
}

// attentionHorizonDays matches the roster's expiring-soon view, so the home
// tile and the list it links to never disagree about what "soon" means.
const attentionHorizonDays = 30

// RosterAttention is where HR needs to look today across the organisation. The
// worker counts come from the roll-up columns in one pass; reviews and the PTO
// liability come from their own areas, because neither is a property of a
// worker's profile.
type RosterAttention struct {
	ActiveWorkers          int
	NonCompliant           int
	TrainingOverdue        int
	AtRisk                 int
	ExpiringSoon           int
	ReviewsAwaitingSignOff int
	PTOLiabilityDays       decimal.Decimal
	// LeaveCertificationsOutstanding is how many leave cases are waiting on a
	// medical certification the employee owes. Past the deadline the employer
	// may deny the leave (29 CFR 825.313), so it is a queue, not a statistic.
	LeaveCertificationsOutstanding int
}

// RosterAttention gathers the home screen's workforce tile. The three reads
// run concurrently because they touch unrelated tables and the tile shows them
// together.
func (s *Service) RosterAttention(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*RosterAttention, error) {
	var (
		counts     *repositories.RosterAttention
		reviews    int
		ptoTotals  *repositories.PTOBalanceSummary
		leaveCerts int
	)
	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		var err error
		counts, err = s.workerRepo.CountRosterAttention(
			groupCtx,
			&repositories.CountRosterAttentionRequest{
				TenantInfo:    tenantInfo,
				ExpiryHorizon: timeutils.NowUnix() + attentionHorizonDays*24*60*60,
			},
		)
		return err
	})
	group.Go(func() error {
		var err error
		// Submitted is the state that waits on somebody else: the manager has
		// written it and the worker has not signed. A draft is still being
		// written and is nobody's queue but its author's.
		reviews, err = s.reviewCounts.CountReviews(
			groupCtx,
			&repositories.CountPerformanceReviewsRequest{
				TenantInfo: tenantInfo,
				Statuses:   []worker.ReviewStatus{worker.ReviewStatusSubmitted},
			},
		)
		return err
	})
	group.Go(func() error {
		var err error
		ptoTotals, err = s.pto.Summary(groupCtx, tenantInfo)
		return err
	})
	group.Go(func() error {
		if s.leave == nil {
			return nil
		}
		var err error
		leaveCerts, err = s.leave.CountCases(groupCtx, &repositories.ListLeaveCasesRequest{
			TenantInfo:                   tenantInfo,
			CertificationOutstandingOnly: true,
		})
		return err
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	attention := &RosterAttention{
		ReviewsAwaitingSignOff:         reviews,
		LeaveCertificationsOutstanding: leaveCerts,
	}
	if counts != nil {
		attention.ActiveWorkers = counts.ActiveWorkers
		attention.NonCompliant = counts.NonCompliant
		attention.TrainingOverdue = counts.TrainingOverdue
		attention.AtRisk = counts.AtRisk
		attention.ExpiringSoon = counts.ExpiringSoon
	}
	if ptoTotals != nil {
		attention.PTOLiabilityDays = ptoTotals.TotalBalanceDays
	}
	return attention, nil
}

// newestOpenChecklist picks the one the office should be working on. A worker
// can have several open at once — onboarding plus an equipment handover, say —
// and the most recently started is the live one.
func newestOpenChecklist(checklists []*worker.WorkerChecklist) *worker.WorkerChecklist {
	var newest *worker.WorkerChecklist
	for _, checklist := range checklists {
		if checklist == nil || !checklist.IsOpen() {
			continue
		}
		if newest == nil || checklist.StartedAt > newest.StartedAt {
			newest = checklist
		}
	}
	return newest
}

// summariseReviews finds the review in flight, the last one that closed, and
// when the next one falls due. The due date comes from the last closed review
// rather than the open one: a review already under way is not also overdue.
func summariseReviews(
	reviews []*worker.PerformanceReview,
) (open *worker.PerformanceReview, last *worker.PerformanceReview, nextAt *int64) {
	for _, review := range reviews {
		if review == nil {
			continue
		}
		if review.Status.IsOpen() {
			if open == nil || review.PeriodEnd > open.PeriodEnd {
				open = review
			}
			continue
		}
		if last == nil || review.PeriodEnd > last.PeriodEnd {
			last = review
		}
	}
	if open == nil && last != nil {
		nextAt = last.NextReviewAt
	}
	return open, last, nextAt
}
