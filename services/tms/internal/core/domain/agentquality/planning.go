package agentquality

import (
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const secondsPerDay = int64(24 * 60 * 60)

type SamplingCase struct {
	ID           pulid.ID          `json:"id"           bun:"id"`
	Source       CaseSource        `json:"source"       bun:"source"`
	Version      int64             `json:"version"      bun:"version"`
	Trigger      agent.RunTrigger  `json:"trigger"      bun:"trigger"`
	SubjectType  agent.SubjectType `json:"subjectType"  bun:"subject_type"`
	SubjectID    pulid.ID          `json:"subjectId"    bun:"subject_id"`
	LastFailedAt int64             `json:"lastFailedAt" bun:"last_failed_at"`
}

func (c SamplingCase) Failing() bool { return c.LastFailedAt > 0 }

func SuiteRevision(cases []SamplingCase) string {
	lines := make([]string, 0, len(cases))
	for _, evalCase := range cases {
		lines = append(lines, evalCase.ID.String()+":"+strconv.FormatInt(evalCase.Version, 10))
	}
	slices.Sort(lines)

	return hashutils.SHA256Hex("suite/v1\n" + strings.Join(lines, "\n"))
}

func SeedFromID(id pulid.ID) int64 {
	sum := sha256.Sum256([]byte(id.String()))

	return int64(binary.BigEndian.Uint64(sum[:8]) & math.MaxInt64)
}

func newRand(seed int64) *rand.Rand {
	unsigned := uint64(seed)

	return rand.New(rand.NewPCG(unsigned, unsigned^0x9e3779b97f4a7c15))
}

type LastRun struct {
	FingerprintHash string
	SuiteRevision   string
	FinishedAt      int64
}

type SkipInput struct {
	FingerprintHash string
	SuiteRevision   string
	Last            *LastRun
	Now             int64
	ForceRerunDays  int
	Forced          bool
}

type SkipDecision struct {
	Skip   bool
	Reason string
}

func DecideSkip(in SkipInput) SkipDecision {
	if in.Forced || in.Last == nil {
		return SkipDecision{}
	}
	if in.Last.FingerprintHash != in.FingerprintHash {
		return SkipDecision{}
	}
	if in.Last.SuiteRevision != in.SuiteRevision {
		return SkipDecision{}
	}

	days := in.ForceRerunDays
	if days <= 0 {
		days = DefaultForceRerunDays
	}
	if in.Now-in.Last.FinishedAt >= int64(days)*secondsPerDay {
		return SkipDecision{}
	}

	return SkipDecision{
		Skip: true,
		Reason: "Nothing about the agent or its cases changed since its last run, " +
			"which is less than " + strconv.Itoa(days) + " days old.",
	}
}

func SampleCases(cases []SamplingCase, seed int64, limit int) []SamplingCase {
	if limit <= 0 || len(cases) == 0 {
		return []SamplingCase{}
	}

	ordered := slices.Clone(cases)
	slices.SortFunc(ordered, func(a, b SamplingCase) int {
		return cmp.Compare(a.ID.String(), b.ID.String())
	})

	failing := make([]SamplingCase, 0, len(ordered))
	rest := make([]SamplingCase, 0, len(ordered))
	for _, evalCase := range ordered {
		if evalCase.Failing() {
			failing = append(failing, evalCase)
		} else {
			rest = append(rest, evalCase)
		}
	}
	slices.SortStableFunc(failing, func(a, b SamplingCase) int {
		return cmp.Compare(b.LastFailedAt, a.LastFailedAt)
	})
	if len(failing) >= limit {
		return failing[:limit]
	}

	chosen := make([]SamplingCase, 0, min(limit, len(ordered)))
	chosen = append(chosen, failing...)

	return append(chosen, stratify(rest, newRand(seed), limit-len(failing))...)
}

func stratify(cases []SamplingCase, rng *rand.Rand, room int) []SamplingCase {
	if room <= 0 || len(cases) == 0 {
		return nil
	}

	strata := make([][]SamplingCase, 0, len(AllCaseSources()))
	for _, source := range AllCaseSources() {
		stratum := make([]SamplingCase, 0, len(cases))
		for _, evalCase := range cases {
			if evalCase.Source == source {
				stratum = append(stratum, evalCase)
			}
		}
		rng.Shuffle(len(stratum), func(i, j int) {
			stratum[i], stratum[j] = stratum[j], stratum[i]
		})
		strata = append(strata, stratum)
	}

	shares := allocate(strata, len(cases), min(room, len(cases)))
	picked := make([]SamplingCase, 0, min(room, len(cases)))
	for round := 0; ; round++ {
		added := false
		for idx, stratum := range strata {
			if round < shares[idx] {
				picked = append(picked, stratum[round])
				added = true
			}
		}
		if !added {
			return picked
		}
	}
}

func allocate(strata [][]SamplingCase, total, room int) []int {
	shares := make([]int, len(strata))
	remainders := make([]float64, len(strata))
	assigned := 0
	for idx, stratum := range strata {
		exact := float64(room) * float64(len(stratum)) / float64(total)
		shares[idx] = int(math.Floor(exact))
		remainders[idx] = exact - float64(shares[idx])
		assigned += shares[idx]
	}

	for assigned < room {
		best := -1
		for idx, stratum := range strata {
			if shares[idx] >= len(stratum) {
				continue
			}
			if best < 0 || remainders[idx] > remainders[best] {
				best = idx
			}
		}
		if best < 0 {
			break
		}
		shares[best]++
		remainders[best] = -1
		assigned++
	}

	return shares
}

func JudgeSample(ids []pulid.ID, seed int64, rate float64) []pulid.ID {
	if rate <= 0 || len(ids) == 0 {
		return []pulid.ID{}
	}

	count := min(len(ids), max(1, int(math.Ceil(rate*float64(len(ids))))))
	ordered := slices.Clone(ids)
	slices.SortFunc(ordered, func(a, b pulid.ID) int { return cmp.Compare(a.String(), b.String()) })
	rng := newRand(seed ^ 0x5bd1e995)
	rng.Shuffle(len(ordered), func(i, j int) { ordered[i], ordered[j] = ordered[j], ordered[i] })

	return ordered[:count]
}

type BudgetInput struct {
	NightlySpent decimal.Decimal
	MonthlySpent decimal.Decimal
	NightlyCap   decimal.Decimal
	MonthlyCap   decimal.Decimal
}

type BudgetDecision struct {
	Stop   bool
	Reason string
}

func CheckBudget(in BudgetInput) BudgetDecision {
	if !in.NightlySpent.LessThan(in.NightlyCap) {
		return BudgetDecision{
			Stop: true,
			Reason: "Tonight's evaluation budget of $" + in.NightlyCap.StringFixed(2) +
				" is spent ($" + in.NightlySpent.StringFixed(2) + ").",
		}
	}
	if !in.MonthlySpent.LessThan(in.MonthlyCap) {
		return BudgetDecision{
			Stop: true,
			Reason: "This month's evaluation budget of $" + in.MonthlyCap.StringFixed(2) +
				" is spent ($" + in.MonthlySpent.StringFixed(2) + ").",
		}
	}

	return BudgetDecision{}
}

func DescribeChanges(changes []agent.FingerprintChange) string {
	if len(changes) == 0 {
		return "Nothing about the agent changed; its cases or the records they read did."
	}

	parts := make([]string, 0, len(changes))
	for _, change := range changes {
		switch change.Field {
		case agent.FingerprintFieldDefinition:
			parts = append(parts, "the agent was saved (version "+change.From+" to "+change.To+")")
		case agent.FingerprintFieldPrompt:
			parts = append(parts, "its instructions or prompt changed")
		case agent.FingerprintFieldTools:
			parts = append(parts, "its tools or their rules changed")
		case agent.FingerprintFieldModel:
			parts = append(parts, "its model changed from "+orNone(change.From)+" to "+
				orNone(change.To))
		case agent.FingerprintFieldProvider:
			parts = append(parts, "it is served by a different provider")
		}
	}

	sentence := strings.Join(parts, "; ")

	return strings.ToUpper(sentence[:1]) + sentence[1:] + "."
}

func orNone(value string) string {
	if value == "" {
		return "none"
	}

	return value
}
