package worker

// DrugAlcoholStatus is the worker's testing record in one word, denormalised
// onto the profile so the roster and the dispatch check can read it without
// replaying every test. Unknown means nothing is on file, which is not the same
// as clear: a driver nobody has ever tested has not passed anything.
type DrugAlcoholStatus string

const (
	DrugAlcoholUnknown    = DrugAlcoholStatus("Unknown")
	DrugAlcoholClear      = DrugAlcoholStatus("Clear")
	DrugAlcoholPending    = DrugAlcoholStatus("Pending")
	DrugAlcoholProhibited = DrugAlcoholStatus("Prohibited")
)

func (s DrugAlcoholStatus) String() string { return string(s) }

func (s DrugAlcoholStatus) IsValid() bool {
	switch s {
	case DrugAlcoholUnknown, DrugAlcoholClear, DrugAlcoholPending, DrugAlcoholProhibited:
		return true
	default:
		return false
	}
}

// Normalized treats an unset value as Unknown, so a profile assembled in memory
// reads the same as one loaded from the database, where the column defaults to
// Unknown and can never be blank.
func (s DrugAlcoholStatus) Normalized() DrugAlcoholStatus {
	if s == "" {
		return DrugAlcoholUnknown
	}
	return s
}

// Blocks reports whether the status bars a driver from safety-sensitive duty.
// Only a standing prohibition does. Pending and Unknown are warnings: the
// office has work to do, but nothing has been found against the driver, and
// grounding a fleet over missing paperwork is not what the rule asks for.
func (s DrugAlcoholStatus) Blocks() bool { return s == DrugAlcoholProhibited }

func (s DrugAlcoholStatus) Label() string {
	switch s {
	case DrugAlcoholUnknown:
		return "Not on file"
	case DrugAlcoholClear:
		return "Clear"
	case DrugAlcoholPending:
		return "Awaiting result"
	case DrugAlcoholProhibited:
		return "Prohibited"
	default:
		return string(s)
	}
}

// DrugAlcoholStanding is everything the profile keeps about the testing record.
type DrugAlcoholStanding struct {
	Status                    DrugAlcoholStatus
	ReturnToDuty              ReturnToDutyStatus
	LastClearinghouseQueryAt  *int64
	NextClearinghouseQueryDue *int64
	// HasPreEmploymentTest and HasPreEmploymentQuery say whether the two gates
	// a driver must pass before their first dispatch have been cleared. They
	// are reported rather than folded into Status because a missing gate is a
	// hiring failure, not a prohibition, and the two want different words.
	HasPreEmploymentTest  bool
	HasPreEmploymentQuery bool
	OpenTestCount         int
	// PassedTestCount is how many collections came back negative. It is what
	// separates a driver with a clean record from one whose only test was
	// cancelled — a voided collection proves nothing either way.
	PassedTestCount int
}

// DrugAlcoholInput is the evidence the standing is derived from. Every slice
// may be empty; nothing here is required for the derivation to be meaningful,
// because "nothing on file" is itself an answer.
type DrugAlcoholInput struct {
	Tests         []*WorkerDOTTest
	OpenViolation *WorkerDOTViolation
	Queries       []*WorkerClearinghouseQuery
}

// EvaluateDrugAlcoholStanding reduces a worker's testing file to the roll-up
// the profile carries.
//
// The order matters: a standing prohibition outranks everything, because a
// prohibited driver stays prohibited however many negative tests sit behind the
// violation. Only once nothing is prohibiting does an outstanding collection
// make the record pending.
func EvaluateDrugAlcoholStanding(in DrugAlcoholInput) DrugAlcoholStanding {
	standing := DrugAlcoholStanding{
		Status:       DrugAlcoholUnknown,
		ReturnToDuty: ReturnToDutyNotRequired,
	}

	if in.OpenViolation != nil {
		standing.ReturnToDuty = ReturnToDutyStatusFor(in.OpenViolation.Status)
	}

	standing.applyTests(in)
	standing.applyQueries(in.Queries)

	switch {
	case in.OpenViolation != nil && in.OpenViolation.Prohibits():
		standing.Status = DrugAlcoholProhibited
	case standing.clearinghouseProhibits(in.Queries):
		standing.Status = DrugAlcoholProhibited
	case standing.OpenTestCount > 0:
		standing.Status = DrugAlcoholPending
	case standing.PassedTestCount > 0:
		standing.Status = DrugAlcoholClear
	default:
		standing.Status = DrugAlcoholUnknown
	}

	return standing
}

func (s *DrugAlcoholStanding) applyTests(in DrugAlcoholInput) {
	for _, test := range in.Tests {
		if test == nil {
			continue
		}
		if test.IsOpen() {
			s.OpenTestCount++
			continue
		}
		if !test.IsPassed() {
			continue
		}
		s.PassedTestCount++
		if test.TestType == DOTTestPreEmployment && test.Substance == DOTSubstanceDrug {
			s.HasPreEmploymentTest = true
		}
	}
}

// applyQueries tracks the twelve-month clock. The clock runs from the most
// recent answered query of any kind: a full query tells the employer everything
// a limited one would, so running one resets the year.
func (s *DrugAlcoholStanding) applyQueries(queries []*WorkerClearinghouseQuery) {
	for _, query := range queries {
		if query == nil || !query.Result.IsResolved() || query.CompletedAt == nil {
			continue
		}
		if query.QueryType == ClearinghouseQueryPreEmploymentFull &&
			query.Result == ClearinghouseResultNoViolations {
			s.HasPreEmploymentQuery = true
		}
		if s.LastClearinghouseQueryAt == nil || *query.CompletedAt > *s.LastClearinghouseQueryAt {
			completed := *query.CompletedAt
			s.LastClearinghouseQueryAt = &completed
			s.NextClearinghouseQueryDue = query.NextDueAt()
		}
	}
}

// clearinghouseProhibits reads only the latest answered query. An old
// ViolationsFound that a later query says is clear has been worked through, and
// holding it against the driver forever would make the return-to-duty process
// pointless.
func (s *DrugAlcoholStanding) clearinghouseProhibits(
	queries []*WorkerClearinghouseQuery,
) bool {
	var latest *WorkerClearinghouseQuery
	for _, query := range queries {
		if query == nil || !query.Result.IsResolved() || query.CompletedAt == nil {
			continue
		}
		if latest == nil || *query.CompletedAt > *latest.CompletedAt {
			latest = query
		}
	}
	return latest != nil && latest.Result.Prohibits()
}

// Equal reports whether two standings say the same thing, so a refresh that
// changes nothing can skip the write.
func (s DrugAlcoholStanding) Equal(other DrugAlcoholStanding) bool {
	return s.Status == other.Status &&
		s.ReturnToDuty == other.ReturnToDuty &&
		equalOptionalUnix(s.LastClearinghouseQueryAt, other.LastClearinghouseQueryAt) &&
		equalOptionalUnix(s.NextClearinghouseQueryDue, other.NextClearinghouseQueryDue)
}
