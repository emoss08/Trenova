package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const safetySeedDay = int64(86400)

type WorkerSafetySeed struct {
	seedhelpers.BaseSeed
}

// WorkerSafetySeed spreads a realistic safety record across the seeded
// drivers — clean inspections, a couple of preventable accidents, a citation,
// rungs on the discipline ladder, some kudos — plus a default review template
// and reviews at every stage, so the Safety and Reviews tabs and the Dash
// cards all have something to show.
//
// Depends on:
//   - Worker: the drivers whose records are seeded
func NewWorkerSafetySeed() *WorkerSafetySeed {
	seed := &WorkerSafetySeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"WorkerSafety",
		"1.0.0",
		"Seeds safety events, discipline, recognition, a review template and reviews per worker",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedWorker)
	return seed
}

type safetySeedRefs struct {
	orgID    pulid.ID
	buID     pulid.ID
	adminID  pulid.ID
	now      int64
	template *worker.PerformanceReviewTemplate
}

func (s *WorkerSafetySeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}
			admin, err := sc.GetUserByUsername(ctx, "admin")
			if err != nil {
				return fmt.Errorf("get admin user: %w", err)
			}
			refs := &safetySeedRefs{
				orgID:   org.ID,
				buID:    org.BusinessUnitID,
				adminID: admin.ID,
				now:     timeutils.NowUnix(),
			}
			if err = s.ensureTemplate(ctx, tx, refs); err != nil {
				return fmt.Errorf("ensure review template: %w", err)
			}

			workers, err := s.loadWorkers(ctx, tx, refs)
			if err != nil {
				return fmt.Errorf("load workers: %w", err)
			}
			for i, wrk := range workers {
				if err = s.seedWorker(ctx, tx, sc, refs, wrk, i); err != nil {
					return fmt.Errorf("seed safety for worker %s: %w", wrk.ID, err)
				}
			}
			return nil
		},
	)
}

func (s *WorkerSafetySeed) ensureTemplate(ctx context.Context, tx bun.Tx, refs *safetySeedRefs) error {
	cadence := int32(12)
	template := &worker.PerformanceReviewTemplate{
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		Code:           "DRIVER-ANNUAL",
		Name:           "Annual Driver Review",
		Description:    "Yearly review covering safety, service, compliance and teamwork.",
		Status:         domaintypes.StatusActive,
		IsDefault:      true,
		CadenceMonths:  &cadence,
		Items: []worker.ReviewItem{
			{Key: "safety", Label: "Safe driving", Description: "Inspections, incidents, defensive habits", Weight: 3},
			{Key: "service", Label: "Customer service", Description: "On-time, professional at the dock", Weight: 2},
			{Key: "compliance", Label: "Compliance & paperwork", Description: "HOS, logs, BOLs, expenses", Weight: 2},
			{Key: "teamwork", Label: "Teamwork", Description: "Communication with dispatch and peers", Weight: 1},
		},
	}
	if _, err := tx.NewInsert().Model(template).On("CONFLICT DO NOTHING").Exec(ctx); err != nil {
		return err
	}
	existing := new(worker.PerformanceReviewTemplate)
	cols := buncolgen.PerformanceReviewTemplateColumns
	if err := tx.NewSelect().
		Model(existing).
		Where(cols.OrganizationID.Eq(), refs.orgID).
		Where(cols.BusinessUnitID.Eq(), refs.buID).
		Where(cols.Code.Eq(), template.Code).
		Scan(ctx); err != nil {
		return err
	}
	refs.template = existing
	return nil
}

func (s *WorkerSafetySeed) loadWorkers(
	ctx context.Context,
	tx bun.Tx,
	refs *safetySeedRefs,
) ([]*worker.Worker, error) {
	workers := make([]*worker.Worker, 0, 32)
	cols := buncolgen.WorkerColumns
	err := tx.NewSelect().
		Model(&workers).
		Where(cols.OrganizationID.Eq(), refs.orgID).
		Where(cols.BusinessUnitID.Eq(), refs.buID).
		Where(cols.Status.Eq(), domaintypes.StatusActive).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	return workers, err
}

func (s *WorkerSafetySeed) seedWorker(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *safetySeedRefs,
	wrk *worker.Worker,
	index int,
) error {
	exists, err := tx.NewSelect().
		Model((*worker.WorkerSafetyEvent)(nil)).
		Where(buncolgen.WorkerSafetyEventColumns.WorkerID.Eq(), wrk.ID).
		Exists(ctx)
	if err != nil || exists {
		return err
	}

	events := s.eventsFor(refs, wrk, index)
	for _, event := range events {
		if _, err = tx.NewInsert().Model(event).Exec(ctx); err != nil {
			return err
		}
		if err = sc.TrackCreated(ctx, "worker_safety_events", event.ID, s.Name()); err != nil {
			return err
		}
	}
	for _, action := range s.actionsFor(refs, wrk, index, events) {
		if _, err = tx.NewInsert().Model(action).Exec(ctx); err != nil {
			return err
		}
		if err = sc.TrackCreated(ctx, "worker_disciplinary_actions", action.ID, s.Name()); err != nil {
			return err
		}
	}
	for _, recognition := range s.recognitionsFor(refs, wrk, index) {
		if _, err = tx.NewInsert().Model(recognition).Exec(ctx); err != nil {
			return err
		}
		if err = sc.TrackCreated(ctx, "worker_recognitions", recognition.ID, s.Name()); err != nil {
			return err
		}
	}
	if review := s.reviewFor(refs, wrk, index); review != nil {
		if _, err = tx.NewInsert().Model(review).Exec(ctx); err != nil {
			return err
		}
		if err = sc.TrackCreated(ctx, "performance_reviews", review.ID, s.Name()); err != nil {
			return err
		}
	}
	return nil
}

func (s *WorkerSafetySeed) event(
	refs *safetySeedRefs,
	wrk *worker.Worker,
	kind worker.SafetyEventKind,
	severity worker.SafetySeverity,
	daysAgo int64,
	description string,
	mutate func(*worker.WorkerSafetyEvent),
) *worker.WorkerSafetyEvent {
	e := &worker.WorkerSafetyEvent{
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		WorkerID:       wrk.ID,
		Kind:           kind,
		Severity:       severity,
		Status:         worker.SafetyEventStatusClosed,
		OccurredAt:     refs.now - daysAgo*safetySeedDay,
		Description:    description,
		RecordedByID:   refs.adminID,
		Resolution:     "Reviewed with the driver.",
	}
	if mutate != nil {
		mutate(e)
	}
	e.Points = worker.DefaultSafetyPoints(e.Kind, e.Severity, e.Preventable, e.InspectionResult)
	e.DefaultPointsExpiry()
	if e.Status == worker.SafetyEventStatusClosed {
		closed := e.OccurredAt + 5*safetySeedDay
		e.ClosedAt = &closed
		e.ClosedByID = refs.adminID
	} else {
		e.Resolution = ""
	}
	return e
}

func (s *WorkerSafetySeed) eventsFor(refs *safetySeedRefs, wrk *worker.Worker, index int) []*worker.WorkerSafetyEvent {
	level := int16(2)
	events := []*worker.WorkerSafetyEvent{
		s.event(refs, wrk, worker.SafetyEventInspection, worker.SafetySeverityMinor, 45+int64(index)*7,
			"Level 2 roadside inspection at the Ohio scale house.",
			func(e *worker.WorkerSafetyEvent) {
				e.InspectionResult = worker.InspectionResultPass
				e.InspectionLevel = &level
				e.Location = "I-70 EB, Cambridge OH"
			}),
	}
	switch index % 6 {
	case 1:
		events = append(events, s.event(refs, wrk, worker.SafetyEventAccident, worker.SafetySeverityModerate, 120,
			"Backed into a dock door at the consignee; trailer door damage.",
			func(e *worker.WorkerSafetyEvent) {
				e.Preventable = true
				e.Location = "Consignee yard, Indianapolis IN"
				e.CostAmount = decimal.NewNullDecimal(decimal.NewFromInt(1850))
			}))
	case 2:
		events = append(events, s.event(refs, wrk, worker.SafetyEventCitation, worker.SafetySeverityMinor, 30,
			"Speeding citation, 9 over in a 55 zone.",
			func(e *worker.WorkerSafetyEvent) {
				e.Status = worker.SafetyEventStatusUnderReview
				e.ReferenceNumber = fmt.Sprintf("CIT-%05d", 40000+index*13)
				e.FineAmount = decimal.NewNullDecimal(decimal.NewFromInt(175))
				e.Location = "US-30, Fort Wayne IN"
			}))
	case 3:
		events = append(events, s.event(refs, wrk, worker.SafetyEventInspection, worker.SafetySeverityMajor, 12,
			"Level 1 inspection; brake adjustment out of service.",
			func(e *worker.WorkerSafetyEvent) {
				one := int16(1)
				e.InspectionResult = worker.InspectionResultOutOfService
				e.InspectionLevel = &one
				e.Status = worker.SafetyEventStatusOpen
				e.Location = "I-80 WB, Joliet IL"
			}))
	case 4:
		events = append(events, s.event(refs, wrk, worker.SafetyEventNearMiss, worker.SafetySeverityMinor, 8,
			"Lane departure alert on I-65; driver reported drowsiness and pulled off.", nil))
	case 5:
		events = append(events, s.event(refs, wrk, worker.SafetyEventIncident, worker.SafetySeverityMinor, 200,
			"Cargo shift on a partial load; no damage, restrapped at the next stop.", nil))
	}
	return events
}

func (s *WorkerSafetySeed) actionsFor(
	refs *safetySeedRefs,
	wrk *worker.Worker,
	index int,
	events []*worker.WorkerSafetyEvent,
) []*worker.WorkerDisciplinaryAction {
	action := func(level worker.DisciplinaryLevel, daysAgo int64, reason string, mutate func(*worker.WorkerDisciplinaryAction)) *worker.WorkerDisciplinaryAction {
		a := &worker.WorkerDisciplinaryAction{
			OrganizationID: refs.orgID,
			BusinessUnitID: refs.buID,
			WorkerID:       wrk.ID,
			Level:          level,
			Status:         worker.DisciplinaryStatusActive,
			Reason:         reason,
			IssuedAt:       refs.now - daysAgo*safetySeedDay,
			IssuedByID:     refs.adminID,
		}
		if mutate != nil {
			mutate(a)
		}
		a.DefaultExpiry()
		if a.ExpiresAt != nil && *a.ExpiresAt < refs.now && a.Status == worker.DisciplinaryStatusActive {
			a.Status = worker.DisciplinaryStatusExpired
		}
		return a
	}
	switch index % 6 {
	case 1:
		acked := refs.now - 110*safetySeedDay
		return []*worker.WorkerDisciplinaryAction{
			action(worker.DisciplinaryLevelCoaching, 115, "Preventable dock strike; backing procedure reviewed.",
				func(a *worker.WorkerDisciplinaryAction) {
					a.SafetyEventID = events[len(events)-1].ID
					a.AcknowledgedAt = &acked
				}),
		}
	case 3:
		return []*worker.WorkerDisciplinaryAction{
			action(worker.DisciplinaryLevelVerbalWarning, 400, "Missed pre-trip inspection.", nil),
			action(worker.DisciplinaryLevelWrittenWarning, 10, "Out-of-service brake adjustment found at roadside.",
				func(a *worker.WorkerDisciplinaryAction) { a.SafetyEventID = events[len(events)-1].ID }),
		}
	case 5:
		rescinded := refs.now - 150*safetySeedDay
		return []*worker.WorkerDisciplinaryAction{
			action(worker.DisciplinaryLevelVerbalWarning, 160, "Late departure reported by dispatch.",
				func(a *worker.WorkerDisciplinaryAction) {
					a.Status = worker.DisciplinaryStatusRescinded
					a.RescindedAt = &rescinded
					a.RescindedByID = refs.adminID
					a.RescindReason = "Departure was delayed by the shipper; not the driver's fault"
				}),
		}
	}
	return nil
}

func (s *WorkerSafetySeed) recognitionsFor(refs *safetySeedRefs, wrk *worker.Worker, index int) []*worker.WorkerRecognition {
	give := func(kind worker.RecognitionKind, daysAgo int64, title, message string) *worker.WorkerRecognition {
		return &worker.WorkerRecognition{
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
		return []*worker.WorkerRecognition{
			give(worker.RecognitionKindSafetyMilestone, 20, "One year accident-free", "Twelve months without a preventable. Thank you for the care you take out there."),
		}
	case 2:
		return []*worker.WorkerRecognition{
			give(worker.RecognitionKindCustomerPraise, 6, "Praised by the receiver in Columbus", "The dock lead called to say the load was the smoothest delivery they had all week."),
			give(worker.RecognitionKindTeamPlayer, 90, "Covered a sick call on a Sunday", ""),
		}
	}
	return nil
}

func (s *WorkerSafetySeed) reviewFor(refs *safetySeedRefs, wrk *worker.Worker, index int) *worker.PerformanceReview {
	if refs.template == nil || index%3 == 2 {
		return nil
	}
	score := func(v int32) *int32 { return &v }
	review := &worker.PerformanceReview{
		OrganizationID: refs.orgID,
		BusinessUnitID: refs.buID,
		WorkerID:       wrk.ID,
		TemplateID:     refs.template.ID,
		ReviewerID:     refs.adminID,
		Title:          "Annual review — 2025",
		PeriodStart:    refs.now - 365*safetySeedDay,
		PeriodEnd:      refs.now - 30*safetySeedDay,
		Ratings:        worker.RatingsFromTemplate(refs.template),
		Goals:          []worker.ReviewGoal{},
	}
	scores := []int32{4, 4, 3, 5}
	if index%2 == 1 {
		scores = []int32{5, 3, 4, 4}
	}
	for i := range review.Ratings {
		review.Ratings[i].Score = score(scores[i%len(scores)])
	}
	review.OverallScore = worker.ComputeOverallScore(review.Ratings)
	review.Summary = "Dependable, safe, and easy to dispatch. Keep the paperwork tight and the year looks even better."
	review.Strengths = "Clean inspections; customers ask for this driver by name."
	review.Improvements = "Turn in fuel receipts the same day."
	review.Goals = []worker.ReviewGoal{
		{ID: pulid.MustNew("goal_").String(), Title: "Zero late expense submissions for a quarter", Status: worker.ReviewGoalStatusOpen},
	}
	submitted := refs.now - 20*safetySeedDay
	switch index % 3 {
	case 0:
		review.Status = worker.ReviewStatusSubmitted
		review.SubmittedAt = &submitted
	case 1:
		acked := refs.now - 15*safetySeedDay
		closed := refs.now - 10*safetySeedDay
		review.Status = worker.ReviewStatusClosed
		review.SubmittedAt = &submitted
		review.AcknowledgedAt = &acked
		review.WorkerComment = "Fair review — I will get the receipts in on time."
		review.ClosedAt = &closed
		review.ClosedByID = refs.adminID
		review.NextReviewAt = review.NextReviewDate(refs.template)
	}
	return review
}

func (s *WorkerSafetySeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *WorkerSafetySeed) CanRollback() bool {
	return true
}
