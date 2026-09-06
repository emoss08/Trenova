package worker

import (
	"fmt"
	"sort"

	"github.com/emoss08/trenova/pkg/domaintypes"
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

// Concern is one thing wrong with a worker's record, named in the words the
// office would use and pointing at the tab that fixes it. Code is the stable
// key the client draws an icon from; Headline and Detail are prose.
type Concern struct {
	Severity ConcernSeverity
	Code     string
	Headline string
	Detail   string
	Tab      string
}

// Concern codes are part of the API contract: the client maps them to icons
// and the tests name them, so they must not be reworded casually.
const (
	ConcernNotEmployed        = "not_employed"
	ConcernNotAssignable      = "not_assignable"
	ConcernCredentialsExpired = "credentials_expired"
	ConcernCredentialsMissing = "credentials_missing"
	ConcernCredentialsSoon    = "credentials_expiring"
	ConcernTrainingOverdue    = "training_overdue"
	ConcernTrainingMissing    = "training_missing"
	ConcernTrainingExpired    = "training_expired"
	ConcernTrainingDue        = "training_due"
	ConcernSafetyRating       = "safety_rating"
	ConcernOutOfService       = "out_of_service"
	ConcernDiscipline         = "active_discipline"
	ConcernOpenSafetyEvents   = "open_safety_events"
	ConcernChecklistOverdue   = "checklist_overdue"
	ConcernChecklistOpen      = "checklist_open"
	ConcernReviewOverdue      = "review_overdue"
)

// StandingInput carries whichever roll-ups the caller was allowed to read. A
// nil section is one the signed-in user cannot see, and it contributes no
// concerns rather than being treated as clean.
type StandingInput struct {
	Worker      *Worker
	Credentials *WorkerCredentialSummary
	Training    *WorkerTrainingSummary
	Safety      *SafetyScorecard
	Checklist   *WorkerChecklist
	ReviewDueAt *int64
	Now         int64
}

// BuildStanding reduces the roll-ups to a verdict and the ranked list of
// reasons behind it. A worker who is not currently employed short-circuits: an
// expired licence on somebody who left last year is not a finding.
func BuildStanding(in StandingInput) (Standing, []Concern) {
	if in.Worker == nil {
		return StandingGood, []Concern{}
	}
	if in.Worker.Status != domaintypes.StatusActive {
		return StandingBlocked, []Concern{{
			Severity: ConcernInfo,
			Code:     ConcernNotEmployed,
			Headline: "Not currently employed",
			Detail:   "The record is kept for retention. Nothing else on it needs action.",
			Tab:      "timeline",
		}}
	}

	concerns := make([]Concern, 0, 8)
	blocked := !in.Worker.CanBeAssigned
	if blocked {
		concerns = append(concerns, Concern{
			Severity: ConcernCritical,
			Code:     ConcernNotAssignable,
			Headline: "Cannot be assigned",
			Detail:   "The worker is off the dispatch board until this is lifted.",
			Tab:      "timeline",
		})
	}

	concerns = appendCredentialConcerns(concerns, in.Credentials)
	if in.Credentials != nil && in.Credentials.ComplianceStatus == ComplianceStatusNonCompliant {
		blocked = true
	}
	concerns = appendTrainingConcerns(concerns, in.Training)
	concerns = appendSafetyConcerns(concerns, in.Safety)
	concerns = appendChecklistConcerns(concerns, in.Checklist, in.Now)
	concerns = appendReviewConcern(concerns, in.ReviewDueAt, in.Now)

	sort.SliceStable(concerns, func(i, j int) bool {
		return concernRank(concerns[i].Severity) < concernRank(concerns[j].Severity)
	})
	return standingFor(blocked, concerns), concerns
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

func appendCredentialConcerns(concerns []Concern, summary *WorkerCredentialSummary) []Concern {
	if summary == nil {
		return concerns
	}
	if summary.ExpiredCount > 0 {
		concerns = append(concerns, Concern{
			Severity: ConcernCritical,
			Code:     ConcernCredentialsExpired,
			Headline: plural(
				summary.ExpiredCount,
				"credential has expired",
				"credentials have expired",
			),
			Detail: "The worker is not qualified to drive until these are renewed.",
			Tab:    "credentials",
		})
	}
	if summary.MissingCount > 0 {
		concerns = append(concerns, Concern{
			Severity: ConcernCritical,
			Code:     ConcernCredentialsMissing,
			Headline: plural(
				summary.MissingCount,
				"required credential is missing",
				"required credentials are missing",
			),
			Detail: "Nothing has been recorded for these at all.",
			Tab:    "credentials",
		})
	}
	if summary.ExpiringCount > 0 {
		concerns = append(concerns, Concern{
			Severity: ConcernWarning,
			Code:     ConcernCredentialsSoon,
			Headline: plural(
				summary.ExpiringCount,
				"credential expires soon",
				"credentials expire soon",
			),
			Detail: "Start the renewal before it lapses.",
			Tab:    "credentials",
		})
	}
	return concerns
}

func appendTrainingConcerns(concerns []Concern, summary *WorkerTrainingSummary) []Concern {
	if summary == nil {
		return concerns
	}
	if summary.ExpiredCount > 0 {
		concerns = append(concerns, Concern{
			Severity: ConcernCritical,
			Code:     ConcernTrainingExpired,
			Headline: plural(
				summary.ExpiredCount,
				"certification has expired",
				"certifications have expired",
			),
			Detail: "Assign the renewal to bring the worker back into date.",
			Tab:    "training",
		})
	}
	if summary.MissingCount > 0 {
		concerns = append(concerns, Concern{
			Severity: ConcernCritical,
			Code:     ConcernTrainingMissing,
			Headline: plural(
				summary.MissingCount,
				"required course was never assigned",
				"required courses were never assigned",
			),
			Detail: "These are required for the worker's role and nothing is open.",
			Tab:    "training",
		})
	}
	if summary.OverdueCount > 0 {
		concerns = append(concerns, Concern{
			Severity: ConcernCritical,
			Code:     ConcernTrainingOverdue,
			Headline: plural(
				summary.OverdueCount,
				"course is past its due date",
				"courses are past their due date",
			),
			Detail: "Assigned but not finished by the date it was due.",
			Tab:    "training",
		})
	}
	if soon := summary.DueCount + summary.ExpiringCount; soon > 0 {
		concerns = append(concerns, Concern{
			Severity: ConcernWarning,
			Code:     ConcernTrainingDue,
			Headline: plural(
				soon,
				"course needs attention soon",
				"courses need attention soon",
			),
			Detail: "Due or expiring inside the reminder window.",
			Tab:    "training",
		})
	}
	return concerns
}

func appendSafetyConcerns(concerns []Concern, card *SafetyScorecard) []Concern {
	if card == nil {
		return concerns
	}
	switch card.Rating {
	case SafetyRatingAtRisk:
		concerns = append(concerns, Concern{
			Severity: ConcernCritical,
			Code:     ConcernSafetyRating,
			Headline: fmt.Sprintf("Safety score is %d, at risk", card.Score),
			Detail: fmt.Sprintf(
				"%d active points against a threshold of %d.",
				card.ActivePoints,
				card.PointsAtRiskThreshold,
			),
			Tab: "safety",
		})
	case SafetyRatingWatch:
		concerns = append(concerns, Concern{
			Severity: ConcernWarning,
			Code:     ConcernSafetyRating,
			Headline: fmt.Sprintf("Safety score is %d, worth watching", card.Score),
			Detail: fmt.Sprintf(
				"%d active points against a threshold of %d.",
				card.ActivePoints,
				card.PointsWatchThreshold,
			),
			Tab: "safety",
		})
	default:
	}
	if card.OutOfServiceOrders > 0 {
		concerns = append(concerns, Concern{
			Severity: ConcernCritical,
			Code:     ConcernOutOfService,
			Headline: plural(
				int(card.OutOfServiceOrders),
				"out-of-service order in the last year",
				"out-of-service orders in the last year",
			),
			Detail: "Roadside put the worker or the vehicle out of service.",
			Tab:    "safety",
		})
	}
	if card.ActiveDiscipline > 0 {
		concerns = append(concerns, Concern{
			Severity: ConcernWarning,
			Code:     ConcernDiscipline,
			Headline: plural(
				int(card.ActiveDiscipline),
				"disciplinary action is active",
				"disciplinary actions are active",
			),
			Detail: fmt.Sprintf(
				"The highest still standing is a %s.",
				disciplineWords(card.HighestDiscipline),
			),
			Tab: "safety",
		})
	}
	if card.OpenEvents > 0 {
		concerns = append(concerns, Concern{
			Severity: ConcernInfo,
			Code:     ConcernOpenSafetyEvents,
			Headline: plural(
				int(card.OpenEvents),
				"safety event is still open",
				"safety events are still open",
			),
			Detail: "They stay open until somebody records the outcome.",
			Tab:    "safety",
		})
	}
	return concerns
}

func appendChecklistConcerns(concerns []Concern, checklist *WorkerChecklist, now int64) []Concern {
	if checklist == nil || checklist.Status != ChecklistStatusOpen {
		return concerns
	}
	progress := checklist.Progress(now)
	if progress.Overdue > 0 {
		return append(concerns, Concern{
			Severity: ConcernWarning,
			Code:     ConcernChecklistOverdue,
			Headline: plural(
				progress.Overdue,
				"checklist item is overdue",
				"checklist items are overdue",
			),
			Detail: fmt.Sprintf("%s is %d%% done.", checklist.Name, progress.Percent),
			Tab:    "checklist",
		})
	}
	return append(concerns, Concern{
		Severity: ConcernInfo,
		Code:     ConcernChecklistOpen,
		Headline: fmt.Sprintf("%s is %d%% done", checklist.Name, progress.Percent),
		Detail: fmt.Sprintf(
			"%d of %d required items are settled.",
			progress.RequiredDone,
			progress.RequiredTotal,
		),
		Tab: "checklist",
	})
}

func appendReviewConcern(concerns []Concern, dueAt *int64, now int64) []Concern {
	if dueAt == nil || *dueAt <= 0 || *dueAt > now {
		return concerns
	}
	return append(concerns, Concern{
		Severity: ConcernWarning,
		Code:     ConcernReviewOverdue,
		Headline: "Performance review is overdue",
		Detail:   "The next review was due and has not been started.",
		Tab:      "reviews",
	})
}

func plural(count int, one string, many string) string {
	if count == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", count, many)
}

func disciplineWords(level DisciplinaryLevel) string {
	if level == "" {
		return "disciplinary action"
	}
	return level.Label()
}
