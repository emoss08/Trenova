package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdiscipline"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdiscipline/workerdisciplinedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview/workerreviewdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety/workersafetydb"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const safetySeedDay = int64(86400)

type WorkerSafetySeed struct {
}

// WorkerSafetySeed spreads a realistic safety record across the seeded
// drivers — clean inspections, a couple of preventable accidents, a citation,
// rungs on the discipline ladder, some kudos — plus a default review template
// and reviews at every stage, so the Safety and Reviews tabs and the Dash
// cards all have something to show.
//
// Depends on:
//   - Worker: the drivers whose records are seeded
var WorkerSafety = Seeds.Add(seed.Seed{
	Name:        "worker_safety",
	Kind:        seed.Fixture,
	Description: "Seeds safety events, discipline, recognition, a review template and reviews per worker",
	After:       []*seed.Seed{Worker},
	Run:         (&WorkerSafetySeed{}).Run,
})

type safetySeedRefs struct {
	orgID    pulid.ID
	buID     pulid.ID
	adminID  pulid.ID
	now      int64
	template *workerreview.PerformanceReviewTemplate
}

func (s *WorkerSafetySeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	admin, err := userByUsername(ctx, db, "admin")
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}
	refs := &safetySeedRefs{
		orgID:   org.ID,
		buID:    org.BusinessUnitID,
		adminID: admin.ID,
		now:     timeutils.NowUnix(),
	}
	if err = s.ensureTemplate(ctx, db, refs); err != nil {
		return fmt.Errorf("ensure review template: %w", err)
	}

	workers, err := s.loadWorkers(ctx, db, refs)
	if err != nil {
		return fmt.Errorf("load workers: %w", err)
	}
	for i, wrk := range workers {
		if err = s.seedWorker(ctx, db, sr, refs, wrk, i); err != nil {
			return fmt.Errorf("seed safety for worker %s: %w", wrk.ID, err)
		}
	}
	return nil
}

func (s *WorkerSafetySeed) ensureTemplate(
	ctx context.Context,
	db *keel.DB,
	refs *safetySeedRefs,
) error {
	cadence := int32(12)
	template := &workerreview.PerformanceReviewTemplate{
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		Code:           "DRIVER-ANNUAL",
		Name:           "Annual Driver Review",
		Description:    "Yearly review covering safety, service, compliance and teamwork.",
		Status:         domaintypes.StatusActive,
		IsDefault:      true,
		CadenceMonths:  &cadence,
		Items: []workerreview.ReviewItem{
			{
				Key:         "safety",
				Label:       "Safe driving",
				Description: "Inspections, incidents, defensive habits",
				Weight:      3,
			},
			{
				Key:         "service",
				Label:       "Customer service",
				Description: "On-time, professional at the dock",
				Weight:      2,
			},
			{
				Key:         "compliance",
				Label:       "Compliance & paperwork",
				Description: "HOS, logs, BOLs, expenses",
				Weight:      2,
			},
			{
				Key:         "teamwork",
				Label:       "Teamwork",
				Description: "Communication with dispatch and peers",
				Weight:      1,
			},
		},
	}
	if _, err := workerreviewdb.PerformanceReviewTemplates.CreateOrSkip(
		ctx,
		db,
		[]*workerreview.PerformanceReviewTemplate{template},
	); err != nil {
		return err
	}
	existing, err := workerreviewdb.PerformanceReviewTemplates.AllTenants().Where(
		workerreviewdb.PerformanceReviewTemplates.OrganizationID.Eq(refs.orgID),
		workerreviewdb.PerformanceReviewTemplates.BusinessUnitID.Eq(refs.buID),
		workerreviewdb.PerformanceReviewTemplates.Code.Eq(template.Code),
	).First(ctx, db)
	if err != nil {
		return err
	}
	refs.template = existing
	return nil
}

func (s *WorkerSafetySeed) loadWorkers(
	ctx context.Context,
	db *keel.DB,
	refs *safetySeedRefs,
) ([]*worker.Worker, error) {
	workers, err := workerdb.Workers.AllTenants().Where(
		workerdb.Workers.OrganizationID.Eq(refs.orgID),
		workerdb.Workers.BusinessUnitID.Eq(refs.buID),
		workerdb.Workers.Status.Eq(domaintypes.StatusActive),
	).Order(workerdb.Workers.CreatedAt.Asc()).All(ctx, db)
	return workers, err
}

func (s *WorkerSafetySeed) seedWorker(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *safetySeedRefs,
	wrk *worker.Worker,
	index int,
) error {
	events := s.eventsFor(refs, wrk, index)
	if err := workersafetydb.WorkerSafetyEvents.CreateMany(ctx, db, events); err != nil {
		return err
	}
	actions := s.actionsFor(refs, wrk, index, events)
	if err := workerdisciplinedb.WorkerDisciplinaryActions.CreateMany(ctx, db, actions); err != nil {
		return err
	}
	if err := workerdisciplinedb.WorkerRecognitions.CreateMany(ctx, db, s.recognitionsFor(refs, wrk, index)); err != nil {
		return err
	}
	if review := s.reviewFor(refs, wrk, index); review != nil {
		if err := workerreviewdb.PerformanceReviews.Create(ctx, db, review); err != nil {
			return err
		}
	}
	sr.Count(int64(len(events) + len(actions)))
	return nil
}

func (s *WorkerSafetySeed) event(
	refs *safetySeedRefs,
	wrk *worker.Worker,
	kind workersafety.SafetyEventKind,
	severity workersafety.SafetySeverity,
	daysAgo int64,
	description string,
	mutate func(*workersafety.WorkerSafetyEvent),
) *workersafety.WorkerSafetyEvent {
	e := &workersafety.WorkerSafetyEvent{
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		WorkerID:       wrk.ID,
		Kind:           kind,
		Severity:       severity,
		Status:         workersafety.SafetyEventStatusClosed,
		OccurredAt:     refs.now - daysAgo*safetySeedDay,
		Description:    description,
		RecordedByID:   refs.adminID,
		Resolution:     "Reviewed with the driver.",
	}
	if mutate != nil {
		mutate(e)
	}
	e.Points = workersafety.DefaultSafetyPoints(
		e.Kind,
		e.Severity,
		e.Preventable,
		e.InspectionResult,
	)
	e.DefaultPointsExpiry()
	if e.Status == workersafety.SafetyEventStatusClosed {
		closed := e.OccurredAt + 5*safetySeedDay
		e.ClosedAt = &closed
		e.ClosedByID = refs.adminID
	} else {
		e.Resolution = ""
	}
	return e
}

func (s *WorkerSafetySeed) eventsFor(
	refs *safetySeedRefs,
	wrk *worker.Worker,
	index int,
) []*workersafety.WorkerSafetyEvent {
	level := int16(2)
	events := []*workersafety.WorkerSafetyEvent{
		s.event(
			refs,
			wrk,
			workersafety.SafetyEventInspection,
			workersafety.SafetySeverityMinor,
			45+int64(index)*7,
			"Level 2 roadside inspection at the Ohio scale house.",
			func(e *workersafety.WorkerSafetyEvent) {
				e.InspectionResult = workersafety.InspectionResultPass
				e.InspectionLevel = &level
				e.Location = "I-70 EB, Cambridge OH"
			},
		),
	}
	switch index % 6 {
	case 1:
		events = append(
			events,
			s.event(
				refs,
				wrk,
				workersafety.SafetyEventAccident,
				workersafety.SafetySeverityModerate,
				120,
				"Backed into a dock door at the consignee; trailer door damage.",
				func(e *workersafety.WorkerSafetyEvent) {
					e.Preventable = true
					e.Location = "Consignee yard, Indianapolis IN"
					e.CostAmount = decimal.NewNullDecimal(decimal.NewFromInt(1850))
				},
			),
		)
	case 2:
		events = append(
			events,
			s.event(
				refs,
				wrk,
				workersafety.SafetyEventCitation,
				workersafety.SafetySeverityMinor,
				30,
				"Speeding citation, 9 over in a 55 zone.",
				func(e *workersafety.WorkerSafetyEvent) {
					e.Status = workersafety.SafetyEventStatusUnderReview
					e.ReferenceNumber = fmt.Sprintf("CIT-%05d", 40000+index*13)
					e.FineAmount = decimal.NewNullDecimal(decimal.NewFromInt(175))
					e.Location = "US-30, Fort Wayne IN"
				},
			),
		)
	case 3:
		events = append(
			events,
			s.event(
				refs,
				wrk,
				workersafety.SafetyEventInspection,
				workersafety.SafetySeverityMajor,
				12,
				"Level 1 inspection; brake adjustment out of service.",
				func(e *workersafety.WorkerSafetyEvent) {
					one := int16(1)
					e.InspectionResult = workersafety.InspectionResultOutOfService
					e.InspectionLevel = &one
					e.Status = workersafety.SafetyEventStatusOpen
					e.Location = "I-80 WB, Joliet IL"
				},
			),
		)
	case 4:
		events = append(
			events,
			s.event(
				refs,
				wrk,
				workersafety.SafetyEventNearMiss,
				workersafety.SafetySeverityMinor,
				8,
				"Lane departure alert on I-65; driver reported drowsiness and pulled off.",
				nil,
			),
		)
	case 5:
		events = append(
			events,
			s.event(
				refs,
				wrk,
				workersafety.SafetyEventIncident,
				workersafety.SafetySeverityMinor,
				200,
				"Cargo shift on a partial load; no damage, restrapped at the next stop.",
				nil,
			),
		)
	}
	return events
}

func (s *WorkerSafetySeed) actionsFor(
	refs *safetySeedRefs,
	wrk *worker.Worker,
	index int,
	events []*workersafety.WorkerSafetyEvent,
) []*workerdiscipline.WorkerDisciplinaryAction {
	action := func(level workerdiscipline.DisciplinaryLevel, daysAgo int64, reason string, mutate func(*workerdiscipline.WorkerDisciplinaryAction)) *workerdiscipline.WorkerDisciplinaryAction {
		a := &workerdiscipline.WorkerDisciplinaryAction{
			OrganizationID: refs.orgID,
			BusinessUnitID: refs.buID,
			WorkerID:       wrk.ID,
			Level:          level,
			Status:         workerdiscipline.DisciplinaryStatusActive,
			Reason:         reason,
			IssuedAt:       refs.now - daysAgo*safetySeedDay,
			IssuedByID:     refs.adminID,
		}
		if mutate != nil {
			mutate(a)
		}
		a.DefaultExpiry()
		if a.ExpiresAt != nil && *a.ExpiresAt < refs.now &&
			a.Status == workerdiscipline.DisciplinaryStatusActive {
			a.Status = workerdiscipline.DisciplinaryStatusExpired
		}
		return a
	}
	switch index % 6 {
	case 1:
		acked := refs.now - 110*safetySeedDay
		return []*workerdiscipline.WorkerDisciplinaryAction{
			action(
				workerdiscipline.DisciplinaryLevelCoaching,
				115,
				"Preventable dock strike; backing procedure reviewed.",
				func(a *workerdiscipline.WorkerDisciplinaryAction) {
					a.SafetyEventID = events[len(events)-1].ID
					a.AcknowledgedAt = &acked
				},
			),
		}
	case 3:
		return []*workerdiscipline.WorkerDisciplinaryAction{
			action(
				workerdiscipline.DisciplinaryLevelVerbalWarning,
				400,
				"Missed pre-trip inspection.",
				nil,
			),
			action(
				workerdiscipline.DisciplinaryLevelWrittenWarning,
				10,
				"Out-of-service brake adjustment found at roadside.",
				func(a *workerdiscipline.WorkerDisciplinaryAction) { a.SafetyEventID = events[len(events)-1].ID },
			),
		}
	case 5:
		rescinded := refs.now - 150*safetySeedDay
		return []*workerdiscipline.WorkerDisciplinaryAction{
			action(
				workerdiscipline.DisciplinaryLevelVerbalWarning,
				160,
				"Late departure reported by dispatch.",
				func(a *workerdiscipline.WorkerDisciplinaryAction) {
					a.Status = workerdiscipline.DisciplinaryStatusRescinded
					a.RescindedAt = &rescinded
					a.RescindedByID = refs.adminID
					a.RescindReason = "Departure was delayed by the shipper; not the driver's fault"
				},
			),
		}
	}
	return nil
}

func (s *WorkerSafetySeed) recognitionsFor(
	refs *safetySeedRefs,
	wrk *worker.Worker,
	index int,
) []*workerdiscipline.WorkerRecognition {
	give := func(kind workerdiscipline.RecognitionKind, daysAgo int64, title, message string) *workerdiscipline.WorkerRecognition {
		return &workerdiscipline.WorkerRecognition{
			OrganizationID:  refs.orgID,
			BusinessUnitID:  refs.buID,
			WorkerID:        wrk.ID,
			Kind:            kind,
			Title:           title,
			Message:         message,
			OccurredAt:      refs.now - daysAgo*safetySeedDay,
			AwardedByID:     refs.adminID,
			VisibleToWorker: true,
		}
	}
	switch index % 4 {
	case 0:
		return []*workerdiscipline.WorkerRecognition{
			give(
				workerdiscipline.RecognitionKindSafetyMilestone,
				20,
				"One year accident-free",
				"Twelve months without a preventable. Thank you for the care you take out there.",
			),
		}
	case 2:
		return []*workerdiscipline.WorkerRecognition{
			give(
				workerdiscipline.RecognitionKindCustomerPraise,
				6,
				"Praised by the receiver in Columbus",
				"The dock lead called to say the load was the smoothest delivery they had all week.",
			),
			give(
				workerdiscipline.RecognitionKindTeamPlayer,
				90,
				"Covered a sick call on a Sunday",
				"",
			),
		}
	}
	return nil
}

func (s *WorkerSafetySeed) reviewFor(
	refs *safetySeedRefs,
	wrk *worker.Worker,
	index int,
) *workerreview.PerformanceReview {
	if refs.template == nil || index%3 == 2 {
		return nil
	}
	score := func(v int32) *int32 { return &v }
	review := &workerreview.PerformanceReview{
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		WorkerID:       wrk.ID,
		TemplateID:     refs.template.ID,
		ReviewerID:     refs.adminID,
		Title:          "Annual review — 2025",
		PeriodStart:    refs.now - 365*safetySeedDay,
		PeriodEnd:      refs.now - 30*safetySeedDay,
		Ratings:        workerreview.RatingsFromTemplate(refs.template),
		Goals:          []workerreview.ReviewGoal{},
		Status:         workerreview.ReviewStatusDraft,
	}
	scores := []int32{4, 4, 3, 5}
	if index%2 == 1 {
		scores = []int32{5, 3, 4, 4}
	}
	for i := range review.Ratings {
		review.Ratings[i].Score = score(scores[i%len(scores)])
	}
	review.OverallScore = workerreview.ComputeOverallScore(review.Ratings)
	review.Summary = "Dependable, safe, and easy to dispatch. Keep the paperwork tight and the year looks even better."
	review.Strengths = "Clean inspections; customers ask for this driver by name."
	review.Improvements = "Turn in fuel receipts the same day."
	review.Goals = []workerreview.ReviewGoal{
		{
			ID:     pulid.MustNew("goal_").String(),
			Title:  "Zero late expense submissions for a quarter",
			Status: workerreview.ReviewGoalStatusOpen,
		},
	}
	submitted := refs.now - 20*safetySeedDay
	switch index % 3 {
	case 0:
		review.Status = workerreview.ReviewStatusSubmitted
		review.SubmittedAt = &submitted
	case 1:
		acked := refs.now - 15*safetySeedDay
		closed := refs.now - 10*safetySeedDay
		review.Status = workerreview.ReviewStatusClosed
		review.SubmittedAt = &submitted
		review.AcknowledgedAt = &acked
		review.WorkerComment = "Fair review — I will get the receipts in on time."
		review.ClosedAt = &closed
		review.ClosedByID = refs.adminID
		review.NextReviewAt = review.NextReviewDate(refs.template)
	}
	return review
}
