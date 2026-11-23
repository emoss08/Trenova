package workeroverviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paybenefit"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercompliance"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerleave"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/portalaccessports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/selfserviceports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerclearanceports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workeremploymentports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerscheduleports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"golang.org/x/sync/errgroup"
)

const (
	overviewConcurrency = 6
	benefitPlanCap      = 200
	benefitEnrolmentCap = 200
	profileChangeCap    = 100
)

type gathered struct {
	credentials   *workercredential.WorkerCredentialSummary
	training      *workertraining.WorkerTrainingSummary
	safety        *workersafetyservice.Evidence
	checklists    []*workerchecklist.WorkerChecklist
	closedLists   int
	balances      []*ptoledgerservice.BalanceView
	reviews       []*workerreview.PerformanceReview
	drugAlcohol   *workerdot.DrugAlcoholStanding
	employment    *workeremploymentports.EmploymentEventSummary
	leave         *workerleave.LeaveEntitlement
	schedule      *ScheduleSection
	hos           *HOSSection
	pay           *PaySection
	benefits      *BenefitsSection
	portal        *PortalSection
	nextTimeOff   *worker.WorkerPTO
	documentCount *int
	memoryCount   *int
	qualification *workerdqfservice.FileParts
	dispatch      *workercompliance.DispatchVerdicts
}

type gatherInput struct {
	req   *GetRequest
	wrk   *worker.Worker
	now   int64
	today int64
}

func (s *Service) Get(ctx context.Context, req *GetRequest) (*Overview, error) {
	now := s.now()
	wrk, err := s.workerRepo.GetByID(ctx, workerports.GetWorkerByIDRequest{
		ID:                req.WorkerID,
		TenantInfo:        req.TenantInfo,
		IncludeProfile:    true,
		IncludeState:      true,
		IncludePortalUser: req.Sections.Portal,
	})
	if err != nil {
		return nil, err
	}

	loc, err := tenantports.OrganizationLocation(ctx, s.organizations, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	in := &gatherInput{
		req:   req,
		wrk:   wrk,
		now:   now,
		today: timeutils.LocalDateAsUTC(now, loc),
	}
	g := &gathered{}
	if err = s.gather(ctx, in, g); err != nil {
		return nil, err
	}
	if err = s.gatherDependent(ctx, in, g); err != nil {
		return nil, err
	}
	return assemble(in, g), nil
}

func (s *Service) gather(ctx context.Context, in *gatherInput, g *gathered) error {
	sections := in.req.Sections
	tenantInfo := in.req.TenantInfo
	workerID := in.wrk.ID
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(overviewConcurrency)

	if sections.Credentials || sections.Qualification {
		group.Go(func() error {
			var err error
			g.credentials, err = s.credentials.SummaryFor(ctx, tenantInfo, in.wrk, in.today)
			return err
		})
	}
	if sections.DrugAlcohol || sections.Qualification {
		group.Go(func() error {
			standing, err := s.drugAlcohol.Standing(ctx, tenantInfo, workerID)
			if err != nil {
				return err
			}
			g.drugAlcohol = &standing
			return nil
		})
	}
	if sections.Training {
		group.Go(func() error {
			var err error
			g.training, err = s.training.SummaryFor(ctx, tenantInfo, in.wrk, in.today)
			return err
		})
	}
	if sections.Safety {
		group.Go(func() error {
			var err error
			g.safety, err = s.safety.Evidence(ctx, &workersafetyservice.EvidenceRequest{
				TenantInfo: tenantInfo,
				WorkerID:   workerID,
				Discipline: sections.Discipline,
				Now:        in.now,
			})
			return err
		})
	}
	if sections.Checklists {
		group.Go(func() error {
			var err error
			g.checklists, err = s.checklists.ListForWorker(ctx, tenantInfo, workerID, false)
			return err
		})
		if sections.Employment {
			group.Go(func() error {
				var err error
				g.closedLists, err = s.checklists.CountClosed(ctx, tenantInfo, workerID)
				return err
			})
		}
	}
	if sections.PTO {
		group.Go(func() error {
			var err error
			g.balances, err = s.pto.GetBalances(ctx, tenantInfo, workerID)
			return err
		})
		group.Go(func() error {
			var err error
			g.nextTimeOff, err = s.nextTimeOff(ctx, in)
			return err
		})
	}
	if sections.Reviews {
		group.Go(func() error {
			var err error
			g.reviews, err = s.reviews.ListReviews(ctx, tenantInfo, workerID, nil)
			return err
		})
	}
	if sections.Employment {
		group.Go(func() error {
			var err error
			g.employment, err = s.employment.Summarize(
				ctx,
				&workeremploymentports.SummarizeWorkerEmploymentEventsRequest{
					TenantInfo: tenantInfo,
					WorkerID:   workerID,
				},
			)
			return err
		})
	}
	if sections.Leave {
		group.Go(func() error {
			entitlement, err := s.leaveStanding.EntitlementFor(ctx, tenantInfo, in.wrk, in.now)
			if err != nil {
				return err
			}
			g.leave = &entitlement
			return nil
		})
	}
	if sections.Schedule {
		group.Go(func() error {
			var err error
			g.schedule, err = s.scheduleSection(ctx, in)
			return err
		})
	}
	if sections.HOS {
		group.Go(func() error {
			var err error
			g.hos, err = s.hosSection(ctx, in)
			return err
		})
	}
	if sections.Pay {
		group.Go(func() error {
			var err error
			g.pay, err = s.paySection(ctx, in)
			return err
		})
	}
	if sections.Benefits {
		group.Go(func() error {
			var err error
			g.benefits, err = s.benefitsSection(ctx, in)
			return err
		})
	}
	if sections.Portal {
		group.Go(func() error {
			var err error
			g.portal, err = s.portalSection(ctx, in)
			return err
		})
	}
	if sections.Documents {
		group.Go(func() error {
			count, err := s.documents.CountByResource(
				ctx,
				&documentports.CountDocumentsByResourceRequest{
					TenantInfo:   tenantInfo,
					ResourceType: permission.ResourceWorker.String(),
					ResourceID:   workerID.String(),
				},
			)
			if err != nil {
				return err
			}
			g.documentCount = &count
			return nil
		})
	}
	if sections.AgentMemory && in.req.MemoryActor != nil {
		group.Go(func() error {
			var err error
			g.memoryCount, err = s.memoryCount(ctx, in)
			return err
		})
	}

	return group.Wait()
}

func (s *Service) gatherDependent(ctx context.Context, in *gatherInput, g *gathered) error {
	sections := in.req.Sections
	group, ctx := errgroup.WithContext(ctx)

	if sections.Qualification {
		group.Go(func() error {
			var err error
			g.qualification, err = s.qualification.FileFor(ctx, &workerdqfservice.FileForRequest{
				TenantInfo:  in.req.TenantInfo,
				Worker:      in.wrk,
				Credentials: g.credentials,
				DrugAlcohol: g.drugAlcohol,
				Now:         in.today,
			})
			return err
		})
	}
	if sections.Credentials || sections.DrugAlcohol {
		group.Go(func() error {
			var err error
			g.dispatch, err = s.dispatch.Verdicts(ctx, &workerclearanceports.DispatchRequest{
				TenantInfo: in.req.TenantInfo,
				Worker:     in.wrk,
				Now:        in.today,
				Probes: workercompliance.DispatchProbes(workercompliance.ClearanceInput{
					Worker:      in.wrk,
					Credentials: g.credentials,
					DrugAlcohol: visibleDrugAlcohol(sections, g.drugAlcohol),
					Now:         in.today,
				}),
			})
			return err
		})
	}

	return group.Wait()
}

func visibleDrugAlcohol(
	sections Sections,
	standing *workerdot.DrugAlcoholStanding,
) *workerdot.DrugAlcoholStanding {
	if !sections.DrugAlcohol {
		return nil
	}
	return standing
}

func visibleDispatch(
	sections Sections,
	verdicts *workercompliance.DispatchVerdicts,
) *workercompliance.DispatchVerdicts {
	if sections.DrugAlcohol {
		return verdicts
	}
	return verdicts.Without(
		workercompliance.DispatchRequirementDrugAlcohol,
		workercompliance.DispatchRequirementClearinghouseQuery,
	)
}

func assemble(in *gatherInput, g *gathered) *Overview {
	sections := in.req.Sections
	overview := &Overview{
		Worker:        in.wrk,
		AsOf:          in.now,
		Training:      g.training,
		Checklist:     newestOpenChecklist(g.checklists),
		PTO:           g.balances,
		Leave:         g.leave,
		Schedule:      g.schedule,
		HOS:           g.hos,
		Pay:           g.pay,
		Benefits:      g.benefits,
		Portal:        g.portal,
		NextTimeOff:   g.nextTimeOff,
		DocumentCount: g.documentCount,
	}
	if sections.AgentMemory {
		overview.AgentMemoryCount = g.memoryCount
	}
	if sections.Credentials {
		overview.Credentials = g.credentials
	}
	if sections.Safety && g.safety != nil {
		overview.Safety = g.safety.Scorecard
	}
	if sections.Reviews {
		overview.OpenReview, overview.LastReview, overview.NextReviewAt = summariseReviews(
			g.reviews,
		)
	}
	drugAlcohol := visibleDrugAlcohol(sections, g.drugAlcohol)
	dispatch := visibleDispatch(sections, g.dispatch)
	if drugAlcohol != nil {
		overview.DrugAlcohol = drugAlcoholSection(drugAlcohol, in.today)
	}
	if sections.Employment {
		overview.Employment = employmentSection(g.employment, g.checklists, sections.Checklists)
		if sections.Checklists {
			overview.Employment.ClosedChecklistCount = new(g.closedLists)
		}
	}

	standingIn := workercompliance.StandingInput{
		Worker:      in.wrk,
		Credentials: overview.Credentials,
		Training:    overview.Training,
		Safety:      overview.Safety,
		Checklist:   overview.Checklist,
		ReviewDueAt: overview.NextReviewAt,
		DrugAlcohol: drugAlcohol,
		Dispatch:    dispatch,
		Now:         in.today,
	}
	if sections.Safety && g.safety != nil {
		standingIn.SafetyEvents = g.safety.Events
	}
	if g.qualification != nil {
		overview.QualificationFile = g.qualification.File
		standingIn.Verifications = g.qualification.Verifications
		standingIn.SafetyHistoryDueAt = g.qualification.File.SafetyHistoryDueAt
	}
	overview.Standing, overview.Concerns = workercompliance.BuildStanding(standingIn)
	overview.ConcernCount = len(overview.Concerns)
	if len(overview.Concerns) > ConcernCap {
		overview.Concerns = overview.Concerns[:ConcernCap]
	}

	if overview.Credentials != nil {
		overview.Clearance = workercompliance.BuildClearance(workercompliance.ClearanceInput{
			Worker:      in.wrk,
			Credentials: overview.Credentials,
			Training:    overview.Training,
			DrugAlcohol: drugAlcohol,
			ReviewDueAt: overview.NextReviewAt,
			Dispatch:    dispatch,
			Now:         in.today,
		})
	}
	return overview
}

func drugAlcoholSection(standing *workerdot.DrugAlcoholStanding, now int64) *DrugAlcoholSection {
	section := &DrugAlcoholSection{Standing: standing}
	if due := standing.NextClearinghouseQueryDue; due != nil && *due > 0 {
		days := workercredential.DaysUntil(*due, now)
		section.NextQueryDaysUntil = &days
	}
	return section
}

func employmentSection(
	summary *workeremploymentports.EmploymentEventSummary,
	checklists []*workerchecklist.WorkerChecklist,
	checklistsVisible bool,
) *EmploymentSection {
	section := &EmploymentSection{}
	if summary != nil {
		section.EventCount = summary.EventCount
		section.HiredAt = summary.LastHiredAt
	}
	if checklistsVisible {
		open := 0
		for _, checklist := range checklists {
			if checklist != nil && checklist.IsOpen() {
				open++
			}
		}
		section.OpenChecklistCount = &open
	}
	return section
}

func (s *Service) scheduleSection(ctx context.Context, in *gatherInput) (*ScheduleSection, error) {
	assignments, err := s.shifts.ListAssignments(
		ctx,
		&workerscheduleports.ListShiftAssignmentsRequest{
			TenantInfo:      in.req.TenantInfo,
			WorkerID:        in.wrk.ID,
			ActiveAt:        in.now,
			IncludeTemplate: true,
			Limit:           1,
		},
	)
	if err != nil {
		return nil, err
	}
	section := &ScheduleSection{}
	for _, assignment := range assignments {
		if assignment != nil && assignment.ShiftTemplate != nil {
			name := assignment.ShiftTemplate.Name
			section.CurrentShiftName = &name
			break
		}
	}
	return section, nil
}

func (s *Service) hosSection(ctx context.Context, in *gatherInput) (*HOSSection, error) {
	onPlan, err := s.features.Enabled(ctx, platformcatalog.FeatureSamsaraIntegration)
	if err != nil {
		return nil, err
	}
	if !onPlan {
		return nil, nil //nolint:nilnil // a section the plan leaves out is null
	}
	state, err := s.hos.WorkerHOSLink(ctx, in.req.TenantInfo, in.wrk.ID)
	if err != nil {
		return nil, err
	}
	section := &HOSSection{
		Link:             state.Link,
		DutyStatus:       state.DutyStatus,
		DutyStatusSince:  state.DutyStatusSince,
		DriveRemainingMs: state.DriveRemainingMs,
		DriveLimitMs:     state.DriveLimitMs,
		ShiftRemainingMs: state.ShiftRemainingMs,
		ShiftLimitMs:     state.ShiftLimitMs,
		CycleRemainingMs: state.CycleRemainingMs,
		CycleLimitMs:     state.CycleLimitMs,
		BreakRemainingMs: state.BreakRemainingMs,
		BreakLimitMs:     state.BreakLimitMs,
		RecordedAt:       state.RecordedAt,
	}
	if !in.req.Sections.Tractor || state.CurrentTractorID.IsNil() {
		return section, nil
	}
	unit, err := s.tractors.GetByID(ctx, tractorports.GetTractorByIDRequest{
		ID:         state.CurrentTractorID,
		TenantInfo: in.req.TenantInfo,
	})
	switch {
	case errortypes.IsNotFoundError(err):
	case err != nil:
		return nil, err
	default:
		code := unit.Code
		section.TractorCode = &code
	}
	return section, nil
}

func (s *Service) nextTimeOff(ctx context.Context, in *gatherInput) (*worker.WorkerPTO, error) {
	result, err := s.timeOff.List(ctx, &workerptoports.ListPTORequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: in.req.TenantInfo,
			Sort: []domaintypes.SortField{
				{Field: "startDate", Direction: dbtype.SortDirectionAsc},
			},
		},
		Cursor:        pagination.CursorInfo{Limit: 1},
		Statuses:      []worker.PTOStatus{worker.PTOStatusRequested, worker.PTOStatusApproved},
		StartDateFrom: in.now,
		Overlapping:   true,
		WorkerID:      in.wrk.ID,
	})
	if err != nil {
		return nil, err
	}
	if len(result.Items) == 0 {
		return nil, nil //nolint:nilnil // no time off ahead is a null field
	}
	return result.Items[0], nil
}

func (s *Service) paySection(ctx context.Context, in *gatherInput) (*PaySection, error) {
	assignment, err := s.pay.GetEffectiveForWorker(
		ctx,
		driverpayports.GetWorkerPayAssignmentRequest{
			TenantInfo: in.req.TenantInfo,
			WorkerID:   in.wrk.ID,
			AsOf:       in.now,
		},
	)
	if errortypes.IsNotFoundError(err) {
		return nil, nil //nolint:nilnil // no assignment in force is a null section
	}
	if err != nil {
		return nil, err
	}
	if assignment.PayProfile == nil {
		return nil, nil //nolint:nilnil // an assignment without its profile has nothing to quote
	}
	return paySectionOf(assignment), nil
}

func (s *Service) benefitsSection(
	ctx context.Context,
	in *gatherInput,
) (*BenefitsSection, error) {
	onPlan, err := s.features.Enabled(ctx, platformcatalog.FeatureWorkforceBenefits)
	if err != nil {
		return nil, err
	}
	if !onPlan {
		return nil, nil //nolint:nilnil // a section the plan leaves out is null
	}
	plans, err := s.benefits.ListPlans(ctx, &driverpayports.ListBenefitPlansRequest{
		TenantInfo: in.req.TenantInfo,
		ActiveOnly: true,
		PlanYear:   planYearOf(in.today),
		Limit:      benefitPlanCap,
	})
	if err != nil {
		return nil, err
	}
	enrollments, err := s.benefits.ListEnrollments(
		ctx,
		&driverpayports.ListBenefitEnrollmentsRequest{
			TenantInfo: in.req.TenantInfo,
			WorkerID:   in.wrk.ID,
			Limit:      benefitEnrolmentCap,
		},
	)
	if err != nil {
		return nil, err
	}
	return benefitsSectionOf(plans, enrollments), nil
}

func benefitsSectionOf(
	plans []*paybenefit.BenefitPlan,
	enrollments []*paybenefit.WorkerBenefitEnrollment,
) *BenefitsSection {
	section := &BenefitsSection{}
	started := make(map[string]struct{}, len(enrollments))
	for _, enrollment := range enrollments {
		if enrollment == nil {
			continue
		}
		if enrollment.Status != paybenefit.EnrollmentEnded {
			started[enrollment.BenefitPlanID.String()] = struct{}{}
		}
		if enrollment.Status.Deducts() {
			section.CoveredCount++
			section.EmployeeCostPerPeriodMinor += enrollment.EmployeeCostMinor
		}
	}
	for _, plan := range plans {
		if plan == nil {
			continue
		}
		if _, ok := started[plan.ID.String()]; !ok {
			section.NotStartedCount++
		}
	}
	return section
}

func (s *Service) portalSection(ctx context.Context, in *gatherInput) (*PortalSection, error) {
	section := &PortalSection{Access: PortalAccessNotSetUp}
	if in.wrk.UserID.IsNotNil() {
		section.Access = PortalAccessLinked
		if in.wrk.PortalUser != nil {
			section.LastSignInAt = in.wrk.PortalUser.LastLoginAt
		}
	} else {
		invitation, err := s.invitations.GetPendingInvitationForWorker(
			ctx,
			portalaccessports.GetPendingPortalInvitationRequest{
				TenantInfo: in.req.TenantInfo,
				WorkerID:   in.wrk.ID,
			},
		)
		switch {
		case errortypes.IsNotFoundError(err):
		case err != nil:
			return nil, err
		case invitation != nil && invitation.IsAcceptable(in.now):
			section.Access = PortalAccessInvited
		}
	}
	if !in.req.Sections.ProfileChanges {
		return section, nil
	}
	pending, err := s.profileChanges.ListChangeRequests(
		ctx,
		&selfserviceports.ListProfileChangeRequestsRequest{
			TenantInfo: in.req.TenantInfo,
			WorkerID:   in.wrk.ID,
			Statuses:   []profilechange.ProfileChangeStatus{profilechange.ProfileChangePending},
			Limit:      profileChangeCap,
		},
	)
	if err != nil {
		return nil, err
	}
	count := len(pending)
	section.PendingProfileChangeCount = &count
	return section, nil
}

func (s *Service) memoryCount(ctx context.Context, in *gatherInput) (*int, error) {
	page, err := s.memories.ListDesk(ctx, &agentmemoryports.ListDeskMemoriesRequest{
		Actor:       in.req.MemoryActor,
		Limit:       1,
		SubjectType: agentmemory.MemorySubjectWorker,
		SubjectID:   in.wrk.ID,
	})
	if err != nil {
		return nil, err
	}
	count := page.All
	return &count, nil
}
