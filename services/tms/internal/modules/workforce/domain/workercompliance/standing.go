package workercompliance

import (
	"sort"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/safetyscorecard"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpage"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

// Standing is the one-word answer to "is this worker in good standing right
// now". It is derived, never stored: every input is itself a roll-up that is
// rebuilt on read, so persisting the verdict would only create a second thing
// to keep in sync.
type Standing string

const (
	StandingGood    = Standing("Good")
	StandingWatch   = Standing("Watch")
	StandingAtRisk  = Standing("AtRisk")
	StandingBlocked = Standing("Blocked")
)

func (s Standing) String() string { return string(s) }

// ConcernSeverity ranks a concern so the office reads the worst thing first.
type ConcernSeverity string

const (
	ConcernCritical = ConcernSeverity("Critical")
	ConcernWarning  = ConcernSeverity("Warning")
	ConcernInfo     = ConcernSeverity("Info")
)

func (c ConcernSeverity) String() string { return string(c) }

func concernRank(severity ConcernSeverity) int {
	switch severity {
	case ConcernCritical:
		return 0
	case ConcernWarning:
		return 1
	case ConcernInfo:
		return 2
	default:
		return 3
	}
}

type ConcernCode string

const (
	ConcernNotEmployed                       = ConcernCode("NotEmployed")
	ConcernNotAssignable                     = ConcernCode("NotAssignable")
	ConcernCredentialExpired                 = ConcernCode("CredentialExpired")
	ConcernCredentialMissing                 = ConcernCode("CredentialMissing")
	ConcernCredentialExpiring                = ConcernCode("CredentialExpiring")
	ConcernTrainingExpired                   = ConcernCode("TrainingExpired")
	ConcernTrainingMissing                   = ConcernCode("TrainingMissing")
	ConcernTrainingOverdue                   = ConcernCode("TrainingOverdue")
	ConcernTrainingDue                       = ConcernCode("TrainingDue")
	ConcernSafetyRating                      = ConcernCode("SafetyRating")
	ConcernOutOfService                      = ConcernCode("OutOfService")
	ConcernActiveDiscipline                  = ConcernCode("ActiveDiscipline")
	ConcernOpenSafetyEvent                   = ConcernCode("OpenSafetyEvent")
	ConcernChecklistOverdue                  = ConcernCode("ChecklistOverdue")
	ConcernChecklistOpen                     = ConcernCode("ChecklistOpen")
	ConcernReviewOverdue                     = ConcernCode("ReviewOverdue")
	ConcernReviewDue                         = ConcernCode("ReviewDue")
	ConcernClearinghouseQueryOverdue         = ConcernCode("ClearinghouseQueryOverdue")
	ConcernClearinghouseQueryDue             = ConcernCode("ClearinghouseQueryDue")
	ConcernDrugAlcoholProhibited             = ConcernCode("DrugAlcoholProhibited")
	ConcernEmploymentVerificationOutstanding = ConcernCode("EmploymentVerificationOutstanding")
)

func (c ConcernCode) String() string { return string(c) }

type Concern struct {
	Severity        ConcernSeverity
	Code            ConcernCode
	Subject         *string
	Count           int
	OccurredAt      *int64
	DueAt           *int64
	DaysUntil       *int64
	StopsAssignment bool
	Destination     workerpage.Destination
}

const (
	ClearinghouseDueSoonDays = 30
	reviewDueSoonDays        = 30
)

type StandingInput struct {
	Worker             *worker.Worker
	Credentials        *workercredential.WorkerCredentialSummary
	Training           *workertraining.WorkerTrainingSummary
	Safety             *safetyscorecard.SafetyScorecard
	SafetyEvents       []*workersafety.WorkerSafetyEvent
	Checklist          *workerchecklist.WorkerChecklist
	ReviewDueAt        *int64
	DrugAlcohol        *workerdot.DrugAlcoholStanding
	Verifications      []*employment.WorkerEmploymentVerification
	SafetyHistoryDueAt int64
	Dispatch           *DispatchVerdicts
	Now                int64
}

func BuildStanding(in StandingInput) (Standing, []Concern) {
	if in.Worker == nil {
		return StandingGood, []Concern{}
	}
	if in.Worker.Status != domaintypes.StatusActive {
		return StandingBlocked, []Concern{{
			Severity:        ConcernInfo,
			Code:            ConcernNotEmployed,
			Count:           1,
			StopsAssignment: true,
			Destination:     workerpage.AtView(workerpage.TabEmployment, workerpage.ViewHistory),
		}}
	}

	concerns := make([]Concern, 0, 16)
	blocked := !in.Worker.CanBeAssigned
	if blocked {
		concerns = append(concerns, Concern{
			Severity:        ConcernCritical,
			Code:            ConcernNotAssignable,
			Count:           1,
			StopsAssignment: true,
			Destination:     workerpage.At(workerpage.TabProfile),
		})
	}

	concerns = appendCredentialConcerns(concerns, in)
	if in.Credentials != nil &&
		in.Credentials.ComplianceStatus == worker.ComplianceStatusNonCompliant {
		blocked = true
	}
	concerns = appendTrainingConcerns(concerns, in)
	concerns = appendSafetyConcerns(concerns, in)
	concerns = appendChecklistConcerns(concerns, in.Checklist, in.Now)
	concerns = appendReviewConcern(concerns, in.ReviewDueAt, in.Now)
	concerns = appendDrugAlcoholConcerns(concerns, in)
	concerns = appendVerificationConcerns(concerns, in)

	rankConcerns(concerns)
	return standingFor(blocked, concerns), concerns
}

func rankConcerns(concerns []Concern) {
	sort.SliceStable(concerns, func(i, j int) bool {
		a, b := concerns[i], concerns[j]
		if rankA, rankB := concernRank(a.Severity), concernRank(b.Severity); rankA != rankB {
			return rankA < rankB
		}
		if a.StopsAssignment != b.StopsAssignment {
			return a.StopsAssignment
		}
		switch {
		case a.DueAt != nil && b.DueAt != nil:
			return *a.DueAt < *b.DueAt
		case a.DueAt != nil:
			return true
		default:
			return false
		}
	})
}

func standingFor(blocked bool, concerns []Concern) Standing {
	if blocked {
		return StandingBlocked
	}
	for _, concern := range concerns {
		if concern.Severity == ConcernCritical {
			return StandingAtRisk
		}
	}
	for _, concern := range concerns {
		if concern.Severity == ConcernWarning {
			return StandingWatch
		}
	}
	return StandingGood
}

func datedConcern(dueAt *int64, now int64, concern Concern) Concern {
	if dueAt == nil || *dueAt <= 0 {
		return concern
	}
	due := *dueAt
	days := workercredential.DaysUntil(due, now)
	concern.DueAt = &due
	concern.DaysUntil = &days
	return concern
}

func subjectOf(name string) *string {
	if name == "" {
		return nil
	}
	return &name
}

func appendCredentialConcerns(concerns []Concern, in StandingInput) []Concern {
	if in.Credentials == nil {
		return concerns
	}
	for _, item := range in.Credentials.Items {
		if item == nil || item.CredentialType == nil {
			continue
		}
		concern := Concern{
			Subject:     subjectOf(item.CredentialType.Name),
			Count:       1,
			Destination: credentialDestination(item),
		}
		switch item.Health {
		case workercredential.CredentialHealthExpired:
			concern.Severity = ConcernCritical
			concern.Code = ConcernCredentialExpired
		case workercredential.CredentialHealthMissing:
			concern.Severity = ConcernCritical
			concern.Code = ConcernCredentialMissing
		case workercredential.CredentialHealthExpiringSoon:
			concern.Severity = ConcernWarning
			concern.Code = ConcernCredentialExpiring
		case workercredential.CredentialHealthValid:
			continue
		default:
			continue
		}
		concern.StopsAssignment = credentialRuling(item, in.Dispatch).stopsAll()
		if item.Credential != nil {
			concern.OccurredAt = item.Credential.IssuedAt
			concern = datedConcern(item.Credential.ExpiresAt, in.Now, concern)
		}
		concerns = append(concerns, concern)
	}
	return concerns
}

func appendTrainingConcerns(concerns []Concern, in StandingInput) []Concern {
	if in.Training == nil {
		return concerns
	}
	for _, item := range in.Training.Items {
		if item == nil || item.Course == nil {
			continue
		}
		concern := Concern{
			Subject:     subjectOf(item.Course.Name),
			Count:       1,
			Destination: trainingDestination(item),
		}
		switch item.Health {
		case worker.TrainingHealthExpired:
			concern.Severity = ConcernCritical
			concern.Code = ConcernTrainingExpired
		case worker.TrainingHealthMissing, worker.TrainingHealthFailed:
			concern.Severity = ConcernCritical
			concern.Code = ConcernTrainingMissing
		case worker.TrainingHealthOverdue:
			concern.Severity = ConcernCritical
			concern.Code = ConcernTrainingOverdue
		case worker.TrainingHealthScheduled,
			worker.TrainingHealthDueSoon,
			worker.TrainingHealthExpiringSoon:
			concern.Severity = ConcernWarning
			concern.Code = ConcernTrainingDue
		case worker.TrainingHealthCurrent:
			continue
		default:
			continue
		}
		if item.Record != nil {
			assignedAt := item.Record.AssignedAt
			if assignedAt > 0 {
				concern.OccurredAt = &assignedAt
			}
			concern = datedConcern(trainingDueAt(item), in.Now, concern)
		}
		concerns = append(concerns, concern)
	}
	return concerns
}

func appendSafetyConcerns(concerns []Concern, in StandingInput) []Concern {
	card := in.Safety
	if card == nil {
		return concerns
	}
	safety := workerpage.At(workerpage.TabSafety)
	switch card.Rating {
	case worker.SafetyRatingAtRisk:
		concerns = append(concerns, Concern{
			Severity:    ConcernCritical,
			Code:        ConcernSafetyRating,
			Count:       int(card.ActivePoints),
			Destination: safety,
		})
	case worker.SafetyRatingWatch:
		concerns = append(concerns, Concern{
			Severity:    ConcernWarning,
			Code:        ConcernSafetyRating,
			Count:       int(card.ActivePoints),
			Destination: safety,
		})
	default:
	}
	windowStart := safetyscorecard.WindowStart(in.Now)
	for _, event := range in.SafetyEvents {
		if event == nil {
			continue
		}
		occurredAt := event.OccurredAt
		dest := safety.WithRecord(workerpage.RecordEvent, event.ID)
		if event.OccurredAt >= windowStart && safetyscorecard.IsOutOfServiceOrder(event) {
			concerns = append(concerns, Concern{
				Severity:    ConcernCritical,
				Code:        ConcernOutOfService,
				Count:       1,
				OccurredAt:  &occurredAt,
				Destination: dest,
			})
		}
		if !event.IsClosed() {
			concerns = append(concerns, Concern{
				Severity:    ConcernInfo,
				Code:        ConcernOpenSafetyEvent,
				Count:       1,
				OccurredAt:  &occurredAt,
				Destination: dest,
			})
		}
	}
	if card.ActiveDiscipline > 0 {
		concerns = append(concerns, Concern{
			Severity:    ConcernWarning,
			Code:        ConcernActiveDiscipline,
			Count:       int(card.ActiveDiscipline),
			Destination: safety,
		})
	}
	return concerns
}

func appendChecklistConcerns(
	concerns []Concern,
	checklist *workerchecklist.WorkerChecklist,
	now int64,
) []Concern {
	if checklist == nil || !checklist.IsOpen() {
		return concerns
	}
	startedAt := checklist.StartedAt
	concern := Concern{
		Severity:   ConcernInfo,
		Code:       ConcernChecklistOpen,
		Subject:    subjectOf(checklist.Name),
		Count:      1,
		OccurredAt: &startedAt,
		Destination: workerpage.AtView(workerpage.TabEmployment, workerpage.ViewChecklists).
			WithRecord(workerpage.RecordChecklist, checklist.ID),
	}
	concern = datedConcern(checklist.DueAt, now, concern)
	if progress := checklist.Progress(now); progress.Overdue > 0 {
		concern.Severity = ConcernWarning
		concern.Code = ConcernChecklistOverdue
		concern.Count = progress.Overdue
	}
	return append(concerns, concern)
}

func appendReviewConcern(concerns []Concern, dueAt *int64, now int64) []Concern {
	if dueAt == nil || *dueAt <= 0 {
		return concerns
	}
	dest := workerpage.AtView(workerpage.TabDevelopment, workerpage.ViewReviews)
	switch {
	case *dueAt < now:
		return append(concerns, datedConcern(dueAt, now, Concern{
			Severity:    ConcernWarning,
			Code:        ConcernReviewOverdue,
			Count:       1,
			Destination: dest,
		}))
	case *dueAt <= now+reviewDueSoonDays*timeutils.SecondsPerDay:
		return append(concerns, datedConcern(dueAt, now, Concern{
			Severity:    ConcernInfo,
			Code:        ConcernReviewDue,
			Count:       1,
			Destination: dest,
		}))
	default:
		return concerns
	}
}

func appendDrugAlcoholConcerns(concerns []Concern, in StandingInput) []Concern {
	standing := in.DrugAlcohol
	if standing == nil {
		return concerns
	}
	dest := workerpage.AtView(workerpage.TabQualification, workerpage.ViewDrugAndAlcohol)
	if standing.Status.Normalized() == worker.DrugAlcoholProhibited {
		concerns = append(concerns, Concern{
			Severity:        ConcernCritical,
			Code:            ConcernDrugAlcoholProhibited,
			Count:           1,
			StopsAssignment: in.Dispatch.RefusesTodayOn(DispatchRequirementDrugAlcohol),
			Destination:     dest,
		})
	}
	due := clearinghouseDue(standing)
	if due == nil {
		return concerns
	}
	concern := Concern{
		Count:           1,
		OccurredAt:      standing.LastClearinghouseQueryAt,
		StopsAssignment: clearinghouseRuling(*due, in.Dispatch).stopsAll(),
		Destination:     dest,
	}
	switch {
	case *due < in.Now:
		concern.Code = ConcernClearinghouseQueryOverdue
		concern.Severity = ConcernWarning
		if concern.StopsAssignment {
			concern.Severity = ConcernCritical
		}
	case *due <= in.Now+ClearinghouseDueSoonDays*timeutils.SecondsPerDay:
		concern.Code = ConcernClearinghouseQueryDue
		concern.Severity = ConcernInfo
	default:
		return concerns
	}
	return append(concerns, datedConcern(due, in.Now, concern))
}

func appendVerificationConcerns(concerns []Concern, in StandingInput) []Concern {
	for _, verification := range in.Verifications {
		if verification == nil {
			continue
		}
		awaitingDrugAlcohol := verification.Status == employment.VerificationReceived &&
			verification.NeedsDrugAlcoholAnswer()
		if !verification.IsOutstanding() && !awaitingDrugAlcohol {
			continue
		}
		concern := Concern{
			Severity:   ConcernInfo,
			Code:       ConcernEmploymentVerificationOutstanding,
			Subject:    subjectOf(verification.EmployerName),
			Count:      1,
			OccurredAt: verification.RequestedAt,
			Destination: workerpage.AtView(workerpage.TabQualification, workerpage.ViewDqFile).
				WithRecord(workerpage.RecordEmployer, verification.ID),
		}
		if in.SafetyHistoryDueAt > 0 {
			due := in.SafetyHistoryDueAt
			concern = datedConcern(&due, in.Now, concern)
			if due < in.Now {
				concern.Severity = ConcernWarning
			}
		}
		concerns = append(concerns, concern)
	}
	return concerns
}
