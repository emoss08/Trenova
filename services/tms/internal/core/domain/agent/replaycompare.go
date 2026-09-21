package agent

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// ReplayVerdict is what one original proposal, set against the replay and
// against how a person decided on it, says about the agent as it is now.
type ReplayVerdict string

const (
	// VerdictAgreed: a person approved it and the replay proposes it again.
	VerdictAgreed = ReplayVerdict("Agreed")
	// VerdictImproved: a person rejected it and the replay no longer proposes it.
	VerdictImproved = ReplayVerdict("Improved")
	// VerdictRegressed: a person approved it and the replay no longer proposes it.
	VerdictRegressed = ReplayVerdict("Regressed")
	// VerdictRepeated: a person rejected it and the replay proposes it again.
	VerdictRepeated = ReplayVerdict("Repeated")
	// VerdictChanged: the replay proposes the same tool with different parameters.
	VerdictChanged = ReplayVerdict("Changed")
	// VerdictUndecided: nobody decided on the original, so the replay's
	// keeping or dropping it says nothing yet.
	VerdictUndecided = ReplayVerdict("Undecided")
	// VerdictAdded: the replay proposes something the original did not.
	VerdictAdded = ReplayVerdict("Added")
)

// OriginalProposal is what the recorded run proposed and what became of it.
type OriginalProposal struct {
	ID       pulid.ID
	ToolName string
	Params   map[string]any
	Status   ProposalStatus
	Decision DecisionType
}

// ReplayMatch pairs an original proposal with the replay's answer to it, or
// a replay action with nothing in the original.
type ReplayMatch struct {
	ToolName           string         `json:"toolName"`
	OriginalProposalID pulid.ID       `json:"originalProposalId,omitempty"`
	OriginalParams     map[string]any `json:"originalParams,omitempty"`
	ReplayParams       map[string]any `json:"replayParams,omitempty"`
	OriginalOutcome    string         `json:"originalOutcome,omitempty"`
	Verdict            ReplayVerdict  `json:"verdict"`
	Changes            []FieldChange  `json:"changes,omitempty"`
}

// ReplayComparison is the whole replay set against the original, with the
// counts the list page reads and a score over the proposals people decided.
type ReplayComparison struct {
	Matches   []ReplayMatch `json:"matches"`
	Agreed    int           `json:"agreed"`
	Improved  int           `json:"improved"`
	Regressed int           `json:"regressed"`
	Repeated  int           `json:"repeated"`
	Changed   int           `json:"changed"`
	Added     int           `json:"added"`
	Undecided int           `json:"undecided"`
	Decided   int           `json:"decided"`
	// Score is the share of decided originals the replay got right: agreed
	// plus improved over those kept or dropped. A changed proposal is
	// decided but neither, so it is left out of both sides. Nil when
	// nothing counts.
	Score *float64 `json:"score"`
}

const (
	outcomeAccepted  = "accepted"
	outcomeRejected  = "rejected"
	outcomeUndecided = "undecided"
)

// outcomeOf reads a proposal's fate as a person's verdict on it. A write
// the agent made on its own, or one simulated, counts as accepted: the
// system stood behind it and nobody took it back.
func outcomeOf(original OriginalProposal) string {
	switch original.Decision {
	case DecisionAccepted, DecisionModified:
		return outcomeAccepted
	case DecisionRejected:
		return outcomeRejected
	}

	switch original.Status {
	case ProposalStatusAccepted,
		ProposalStatusModified,
		ProposalStatusExecuted,
		ProposalStatusSimulated:
		return outcomeAccepted
	case ProposalStatusRejected:
		return outcomeRejected
	default:
		return outcomeUndecided
	}
}

// CompareReplay matches each original proposal to the replay action that
// answers it and judges the pair by the original's outcome.
//
// Matching is by tool, and within a tool by identical parameters first, so a
// replay that proposes the same two assignments in a different order still
// agrees with both. What is left in the replay is added; what is left in the
// original is dropped, and dropped is an improvement or a regression by how
// a person decided on it.
func CompareReplay(originals []OriginalProposal, replay []ReplayAction) *ReplayComparison {
	out := &ReplayComparison{Matches: make([]ReplayMatch, 0, len(originals)+len(replay))}
	used := make([]bool, len(replay))

	pick := func(original OriginalProposal) (int, bool) {
		exact := -1
		sameTool := -1
		for i, action := range replay {
			if used[i] || action.ToolName != original.ToolName {
				continue
			}
			if reflect.DeepEqual(normalize(action.Arguments), normalize(original.Params)) {
				exact = i
				break
			}
			if sameTool < 0 {
				sameTool = i
			}
		}
		if exact >= 0 {
			return exact, true
		}
		if sameTool >= 0 {
			return sameTool, false
		}

		return -1, false
	}

	for _, original := range originals {
		match := ReplayMatch{
			ToolName:           original.ToolName,
			OriginalProposalID: original.ID,
			OriginalParams:     original.Params,
			OriginalOutcome:    outcomeOf(original),
		}
		index, exact := pick(original)
		switch {
		case index < 0:
			match.Verdict = droppedVerdict(match.OriginalOutcome)
		case exact:
			used[index] = true
			match.ReplayParams = replay[index].Arguments
			match.Verdict = keptVerdict(match.OriginalOutcome)
		default:
			used[index] = true
			match.ReplayParams = replay[index].Arguments
			match.Verdict = VerdictChanged
			match.Changes = diffParams(original.Params, replay[index].Arguments)
		}
		out.count(match)
		out.Matches = append(out.Matches, match)
	}

	for i, action := range replay {
		if used[i] {
			continue
		}
		match := ReplayMatch{
			ToolName:     action.ToolName,
			ReplayParams: action.Arguments,
			Verdict:      VerdictAdded,
		}
		out.count(match)
		out.Matches = append(out.Matches, match)
	}

	if out.Decided > 0 {
		score := float64(out.Agreed+out.Improved) / float64(out.Decided)
		out.Score = &score
	}

	return out
}

func keptVerdict(outcome string) ReplayVerdict {
	switch outcome {
	case outcomeAccepted:
		return VerdictAgreed
	case outcomeRejected:
		return VerdictRepeated
	default:
		return VerdictUndecided
	}
}

func droppedVerdict(outcome string) ReplayVerdict {
	switch outcome {
	case outcomeAccepted:
		return VerdictRegressed
	case outcomeRejected:
		return VerdictImproved
	default:
		return VerdictUndecided
	}
}

func (c *ReplayComparison) count(match ReplayMatch) {
	switch match.Verdict {
	case VerdictAgreed:
		c.Agreed++
	case VerdictImproved:
		c.Improved++
	case VerdictRegressed:
		c.Regressed++
	case VerdictRepeated:
		c.Repeated++
	case VerdictChanged:
		c.Changed++
	case VerdictAdded:
		c.Added++
	case VerdictUndecided:
		c.Undecided++
	}
	switch match.Verdict {
	case VerdictAgreed, VerdictImproved, VerdictRegressed, VerdictRepeated:
		c.Decided++
	}
}

// normalize makes two parameter maps comparable whatever JSON round trip
// they took: numbers come back as float64 from storage and as int from a
// model, and an absent key is the same as an empty one.
func normalize(params map[string]any) map[string]any {
	out := make(map[string]any, len(params))
	for key, value := range params {
		if value == nil {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			continue
		}
		out[key] = fmt.Sprint(normalizeValue(value))
	}

	return out
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	case string:
		return strings.TrimSpace(v)
	default:
		encoded, err := sonic.MarshalString(v)
		if err != nil {
			return fmt.Sprint(v)
		}

		return encoded
	}
}

func diffParams(original, replay map[string]any) []FieldChange {
	keys := make(map[string]struct{}, len(original)+len(replay))
	for key := range original {
		keys[key] = struct{}{}
	}
	for key := range replay {
		keys[key] = struct{}{}
	}
	sorted := make([]string, 0, len(keys))
	for key := range keys {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)

	before := normalize(original)
	after := normalize(replay)
	changes := make([]FieldChange, 0, len(sorted))
	for _, key := range sorted {
		if before[key] == after[key] {
			continue
		}
		changes = append(changes, FieldChange{
			Field: key,
			From:  displayValue(before[key]),
			To:    displayValue(after[key]),
		})
	}

	return changes
}

func displayValue(value any) string {
	if value == nil {
		return "nothing"
	}

	return stringutils.Ellipsize(fmt.Sprint(value), 80)
}
