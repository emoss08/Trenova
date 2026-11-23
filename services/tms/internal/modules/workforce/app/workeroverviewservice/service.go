package workeroverviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paybenefit"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/portalinvitation"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/safetyscorecard"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercompliance"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/portalaccessports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/ptoledgerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/selfserviceports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerclearanceports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workeremploymentports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerleaveports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerreviewports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerscheduleports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type CredentialSummarizer interface {
	SummaryFor(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		wrk *worker.Worker,
		asOf int64,
	) (*workercredential.WorkerCredentialSummary, error)
}

type TrainingSummarizer interface {
	SummaryFor(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		wrk *worker.Worker,
		asOf int64,
	) (*workertraining.WorkerTrainingSummary, error)
}

type SafetyReader interface {
	Evidence(
		ctx context.Context,
		req *workersafetyservice.EvidenceRequest,
	) (*workersafetyservice.Evidence, error)
}

type ChecklistLister interface {
	ListForWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		includeClosed bool,
	) ([]*workerchecklist.WorkerChecklist, error)
	CountClosed(ctx context.Context, tenantInfo pagination.TenantInfo, workerID pulid.ID) (int, error)
}

type BalanceReader interface {
	GetBalances(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) ([]*ptoledgerservice.BalanceView, error)
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*ptoledgerports.PTOBalanceSummary, error)
}

type ReviewCounter interface {
	CountReviews(
		ctx context.Context,
		req *workerreviewports.CountPerformanceReviewsRequest,
	) (int, error)
}

type LeaveCounter interface {
	CountCases(
		ctx context.Context,
		req *workerleaveports.ListLeaveCasesRequest,
	) (int, error)
}

type LeaveEntitler interface {
	EntitlementFor(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		wrk *worker.Worker,
		asOf int64,
	) (workerleave.LeaveEntitlement, error)
}

type ReviewLister interface {
	ListReviews(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		statuses []workerreview.ReviewStatus,
	) ([]*workerreview.PerformanceReview, error)
}

type DrugAlcoholStander interface {
	Standing(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (workerdot.DrugAlcoholStanding, error)
}

type QualificationFiler interface {
	FileFor(
		ctx context.Context,
		req *workerdqfservice.FileForRequest,
	) (*workerdqfservice.FileParts, error)
}

type EmploymentSummarizer interface {
	Summarize(
		ctx context.Context,
		req *workeremploymentports.SummarizeWorkerEmploymentEventsRequest,
	) (*workeremploymentports.EmploymentEventSummary, error)
}

type ShiftLister interface {
	ListAssignments(
		ctx context.Context,
		req *workerscheduleports.ListShiftAssignmentsRequest,
	) ([]*workerschedule.WorkerShiftAssignment, error)
}

type HOSLinker interface {
	WorkerHOSLink(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (*telematics.HOSLinkState, error)
}

type TimeOffLister interface {
	List(
		ctx context.Context,
		req *workerptoports.ListPTORequest,
	) (*pagination.CursorListResult[*worker.WorkerPTO], error)
}

type TractorReader interface {
	GetByID(ctx context.Context, req tractorports.GetTractorByIDRequest) (*tractor.Tractor, error)
}

type PayAssignmentReader interface {
	GetEffectiveForWorker(
		ctx context.Context,
		req driverpayports.GetWorkerPayAssignmentRequest,
	) (*payassignment.WorkerPayAssignment, error)
}

type BenefitReader interface {
	ListPlans(
		ctx context.Context,
		req *driverpayports.ListBenefitPlansRequest,
	) ([]*paybenefit.BenefitPlan, error)
	ListEnrollments(
		ctx context.Context,
		req *driverpayports.ListBenefitEnrollmentsRequest,
	) ([]*paybenefit.WorkerBenefitEnrollment, error)
}

type PortalInvitationReader interface {
	GetPendingInvitationForWorker(
		ctx context.Context,
		req portalaccessports.GetPendingPortalInvitationRequest,
	) (*portalinvitation.PortalInvitation, error)
}

type ProfileChangeLister interface {
	ListChangeRequests(
		ctx context.Context,
		req *selfserviceports.ListProfileChangeRequestsRequest,
	) ([]*profilechange.WorkerProfileChangeRequest, error)
}

type MemoryLister interface {
	ListDesk(
		ctx context.Context,
		req *agentmemoryports.ListDeskMemoriesRequest,
	) (*agentmemoryports.DeskMemoryPage, error)
}

type Sections struct {
	Credentials    bool
	Qualification  bool
	DrugAlcohol    bool
	Training       bool
	Safety         bool
	Discipline     bool
	Employment     bool
	Checklists     bool
	PTO            bool
	Leave          bool
	Reviews        bool
	Schedule       bool
	HOS            bool
	Tractor        bool
	Pay            bool
	Benefits       bool
	Portal         bool
	ProfileChanges bool
	Documents      bool
	AgentMemory    bool
}

func All() Sections {
	return Sections{
		Credentials:    true,
		Qualification:  true,
		DrugAlcohol:    true,
		Training:       true,
		Safety:         true,
		Discipline:     true,
		Employment:     true,
		Checklists:     true,
		PTO:            true,
		Leave:          true,
		Reviews:        true,
		Schedule:       true,
		HOS:            true,
		Tractor:        true,
		Pay:            true,
		Benefits:       true,
		Portal:         true,
		ProfileChanges: true,
		Documents:      true,
		AgentMemory:    true,
	}
}

type Overview struct {
	Worker            *worker.Worker
	AsOf              int64
	Standing          workercompliance.Standing
	Concerns          []workercompliance.Concern
	ConcernCount      int
	Clearance         *workercompliance.Clearance
	Credentials       *workercredential.WorkerCredentialSummary
	QualificationFile *workercompliance.DQFFile
	DrugAlcohol       *DrugAlcoholSection
	Training          *workertraining.WorkerTrainingSummary
	Safety            *safetyscorecard.SafetyScorecard
	Employment        *EmploymentSection
	Checklist         *workerchecklist.WorkerChecklist
	PTO               []*ptoledgerservice.BalanceView
	Leave             *workerleave.LeaveEntitlement
	OpenReview        *workerreview.PerformanceReview
	LastReview        *workerreview.PerformanceReview
	NextReviewAt      *int64
	Schedule          *ScheduleSection
	HOS               *HOSSection
	Pay               *PaySection
	Benefits          *BenefitsSection
	Portal            *PortalSection
	NextTimeOff       *worker.WorkerPTO
	DocumentCount     *int
	AgentMemoryCount  *int
}

type GetRequest struct {
	TenantInfo  pagination.TenantInfo
	WorkerID    pulid.ID
	Sections    Sections
	MemoryActor *agentmemoryports.DeskMemoryActor
}

type Params struct {
	fx.In

	Logger         *zap.Logger
	Clock          timeutils.Clock
	Features       platformcatalogports.FeatureGate
	WorkerRepo     workerports.WorkerRepository
	ReviewCounts   ReviewCounter
	Credentials    CredentialSummarizer
	Training       TrainingSummarizer
	Safety         SafetyReader
	Checklists     ChecklistLister
	PTO            BalanceReader
	Reviews        ReviewLister
	Leave          LeaveCounter
	LeaveStanding  LeaveEntitler
	DrugAlcohol    DrugAlcoholStander
	Qualification  QualificationFiler
	Employment     EmploymentSummarizer
	Shifts         ShiftLister
	HOS            HOSLinker
	Tractors       TractorReader
	TimeOff        TimeOffLister
	Pay            PayAssignmentReader
	Benefits       BenefitReader
	Invitations    PortalInvitationReader
	ProfileChanges ProfileChangeLister
	Documents      documentports.ResourceDocumentCounter
	Memories       MemoryLister
	Dispatch       workerclearanceports.DispatchCheck
	Organizations  tenantports.OrganizationRepository
}

type Service struct {
	l              *zap.Logger
	now            timeutils.Clock
	features       platformcatalogports.FeatureGate
	workerRepo     workerports.WorkerRepository
	reviewCounts   ReviewCounter
	credentials    CredentialSummarizer
	training       TrainingSummarizer
	safety         SafetyReader
	checklists     ChecklistLister
	pto            BalanceReader
	reviews        ReviewLister
	leave          LeaveCounter
	leaveStanding  LeaveEntitler
	drugAlcohol    DrugAlcoholStander
	qualification  QualificationFiler
	employment     EmploymentSummarizer
	shifts         ShiftLister
	hos            HOSLinker
	tractors       TractorReader
	timeOff        TimeOffLister
	pay            PayAssignmentReader
	benefits       BenefitReader
	invitations    PortalInvitationReader
	profileChanges ProfileChangeLister
	documents      documentports.ResourceDocumentCounter
	memories       MemoryLister
	dispatch       workerclearanceports.DispatchCheck
	organizations  tenantports.OrganizationReader
}

func New(p Params) *Service {
	return &Service{
		l:              p.Logger.Named("service.worker-overview"),
		now:            p.Clock,
		features:       p.Features,
		workerRepo:     p.WorkerRepo,
		reviewCounts:   p.ReviewCounts,
		credentials:    p.Credentials,
		training:       p.Training,
		safety:         p.Safety,
		checklists:     p.Checklists,
		pto:            p.PTO,
		reviews:        p.Reviews,
		leave:          p.Leave,
		leaveStanding:  p.LeaveStanding,
		drugAlcohol:    p.DrugAlcohol,
		qualification:  p.Qualification,
		employment:     p.Employment,
		shifts:         p.Shifts,
		hos:            p.HOS,
		tractors:       p.Tractors,
		timeOff:        p.TimeOff,
		pay:            p.Pay,
		benefits:       p.Benefits,
		invitations:    p.Invitations,
		profileChanges: p.ProfileChanges,
		documents:      p.Documents,
		memories:       p.Memories,
		dispatch:       p.Dispatch,
		organizations:  p.Organizations,
	}
}

// attentionHorizonDays matches the roster's expiring-soon view, so the home
// tile and the list it links to never disagree about what "soon" means.
const attentionHorizonDays = 30

// RosterAttention is where HR needs to look today across the organisation. The
// worker counts come from the roll-up columns in one pass; reviews and the PTO
// liability come from their own areas, because neither is a property of a
// worker's profile.
type RosterAttention struct {
	ActiveWorkers int
	NonCompliant  int
	// TrainingOverdue is nil for a reader without worker_training read, and
	// when the plan leaves out the HR Suite talent feature.
	TrainingOverdue *int
	AtRisk          int
	ExpiringSoon    int
	// ReviewsAwaitingSignOff is nil for a reader who may not read reviews.
	ReviewsAwaitingSignOff *int
	// PTOLiabilityDays is nil for a reader without worker_pto read, and when
	// the plan leaves out the HR Suite time off feature. The PTO ledger is not
	// read for such a reader.
	PTOLiabilityDays *decimal.Decimal
	// LeaveCertificationsOutstanding is how many leave cases are waiting on a
	// medical certification the employee owes. Past the deadline the employer
	// may deny the leave (29 CFR 825.313), so it is a queue, not a statistic.
	// It is nil for a reader who may not read leave, and counts only their
	// team's cases when their leave grant reaches only their team.
	LeaveCertificationsOutstanding *int
}

// AttentionAccess is what the reader of the home tile may count: reviews and
// leave are granted on their own, so a worker reader is not told how many
// there are.
type AttentionAccess struct {
	Reviews         bool
	Leave           bool
	PTO             bool
	Training        bool
	LeaveManagerIDs []pulid.ID
}

// RosterAttention gathers the home screen's workforce tile. The three reads
// run concurrently because they touch unrelated tables and the tile shows them
// together.
func (s *Service) RosterAttention(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	access AttentionAccess,
) (*RosterAttention, error) {
	var (
		counts     *workerports.RosterAttention
		reviews    int
		ptoTotals  *ptoledgerports.PTOBalanceSummary
		leaveCerts int
	)
	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		var err error
		counts, err = s.workerRepo.CountRosterAttention(
			groupCtx,
			&workerports.CountRosterAttentionRequest{
				TenantInfo:    tenantInfo,
				ExpiryHorizon: s.now() + attentionHorizonDays*timeutils.SecondsPerDay,
			},
		)
		return err
	})
	group.Go(func() error {
		if !access.Reviews {
			return nil
		}
		var err error
		// Submitted is the state that waits on somebody else: the manager has
		// written it and the worker has not signed. A draft is still being
		// written and is nobody's queue but its author's.
		reviews, err = s.reviewCounts.CountReviews(
			groupCtx,
			&workerreviewports.CountPerformanceReviewsRequest{
				TenantInfo: tenantInfo,
				Statuses:   []workerreview.ReviewStatus{workerreview.ReviewStatusSubmitted},
			},
		)
		return err
	})
	group.Go(func() error {
		if !access.PTO {
			return nil
		}
		var err error
		ptoTotals, err = s.pto.Summary(groupCtx, tenantInfo)
		return err
	})
	group.Go(func() error {
		if !access.Leave {
			return nil
		}
		var err error
		leaveCerts, err = s.leave.CountCases(groupCtx, &workerleaveports.ListLeaveCasesRequest{
			TenantInfo:                   tenantInfo,
			CertificationOutstandingOnly: true,
			ManagerIDs:                   access.LeaveManagerIDs,
		})
		return err
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	attention := &RosterAttention{}
	if access.Reviews {
		attention.ReviewsAwaitingSignOff = &reviews
	}
	if access.Leave {
		attention.LeaveCertificationsOutstanding = &leaveCerts
	}
	if counts != nil {
		attention.ActiveWorkers = counts.ActiveWorkers
		attention.NonCompliant = counts.NonCompliant
		attention.AtRisk = counts.AtRisk
		attention.ExpiringSoon = counts.ExpiringSoon
	}
	if access.Training {
		overdue := 0
		if counts != nil {
			overdue = counts.TrainingOverdue
		}
		attention.TrainingOverdue = &overdue
	}
	if access.PTO {
		liability := decimal.Zero
		if ptoTotals != nil {
			liability = ptoTotals.TotalBalanceDays
		}
		attention.PTOLiabilityDays = &liability
	}
	return attention, nil
}

// newestOpenChecklist picks the one the office should be working on. A worker
// can have several open at once — onboarding plus an equipment handover, say —
// and the most recently started is the live one.
func newestOpenChecklist(
	checklists []*workerchecklist.WorkerChecklist,
) *workerchecklist.WorkerChecklist {
	var newest *workerchecklist.WorkerChecklist
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
	reviews []*workerreview.PerformanceReview,
) (open *workerreview.PerformanceReview, last *workerreview.PerformanceReview, nextAt *int64) {
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
