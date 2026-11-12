package recordcaseservice

import (
	"cmp"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/shared/recordids"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/textcase"
)

func exhibitsOf(built *composed) []recordcase.Exhibit {
	out := make(
		[]recordcase.Exhibit,
		0,
		len(built.subject.exhibits)+maxMessageExhibits+maxConditions,
	)
	timelines := make([]recordcase.Exhibit, 0, 1)
	for i := range built.subject.exhibits {
		if built.subject.exhibits[i].Kind == recordcase.ExhibitTimeline {
			timelines = append(timelines, built.subject.exhibits[i])

			continue
		}
		out = append(out, built.subject.exhibits[i])
	}
	out = append(out, messageExhibits(built)...)
	out = append(out, conditionExhibits(built)...)
	out = append(out, timelines...)
	for i := range out {
		out[i].Number = i + 1
		out[i].Claims = nonEmpty(out[i].Claims...)
	}

	return out
}

func messageExhibits(built *composed) []recordcase.Exhibit {
	out := make([]recordcase.Exhibit, 0, maxMessageExhibits)
	for _, item := range built.items {
		if item.SourceKind != watchtower.SourceInboundMessage || len(out) == maxMessageExhibits {
			continue
		}
		from := factValue(item, "From")
		title := "Message"
		if built.subject.customer != "" {
			title = built.subject.customer + "'s message"
		}
		exhibit := recordcase.Exhibit{
			Key:    "message:" + item.ID.String(),
			Kind:   recordcase.ExhibitMessage,
			Title:  title,
			Source: joined("Triage", lowerWords(factValue(item, "Kind"))),
			At:     item.OccurredAt,
			Path:   item.Path,
			Message: &recordcase.Message{
				Quote: stringutils.FirstNonEmptyTrimmed(item.Summary, item.Title),
				From:  from,
			},
			Claims: []string{stringutils.FirstNonEmptyTrimmed(item.Summary, item.Title)},
		}
		if item.HasAgent() {
			exhibit.Message.ReplyAgent = item.AgentDefinitionID
			exhibit.Message.Reply = "is working on this message."
			if built.decision != nil && built.decision.Phase == recordcase.PhaseAwaiting {
				exhibit.Message.Reply = "has this message. Its reply waits on this decision."
			}
		}
		out = append(out, exhibit)
	}

	return out
}

func conditions(built *composed) []*watchtower.Item {
	seen := map[string]struct{}{}
	out := make([]*watchtower.Item, 0, len(built.items))
	for _, item := range built.items {
		if item.SourceKind.DeskLens() != watchtower.DeskLensWatchtower {
			continue
		}
		key := string(item.SourceKind) + "\x1f" + item.Title
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	slices.SortStableFunc(out, func(a, b *watchtower.Item) int {
		if c := cmp.Compare(b.Severity.Rank(), a.Severity.Rank()); c != 0 {
			return c
		}

		return cmp.Compare(a.OccurredAt, b.OccurredAt)
	})

	return out
}

func conditionExhibits(built *composed) []recordcase.Exhibit {
	out := make([]recordcase.Exhibit, 0, maxConditions)
	for _, item := range conditions(built) {
		if _, absorbed := built.subject.absorbs[item.SourceKind]; absorbed {
			continue
		}
		if len(out) == maxConditions {
			break
		}
		exhibit := recordcase.Exhibit{
			Key:    "condition:" + item.ID.String(),
			Kind:   recordcase.ExhibitFacts,
			Title:  item.SourceKind.Label(),
			Source: joined("Watchtower", strings.ToLower(string(item.Severity))),
			At:     item.OccurredAt,
			Note: recordids.Strip(
				stringutils.FirstNonEmptyTrimmed(item.Summary, item.Title),
			),
			Path:   item.Path,
			Facts:  make([]recordcase.Fact, 0, len(item.Facts)+len(item.Impact)),
			Claims: []string{item.Title, item.Summary},
		}
		for _, stake := range item.Impact {
			exhibit.Facts = append(exhibit.Facts, recordcase.Fact{
				Label: "At stake",
				Value: stake,
				Tone:  recordcase.ToneWarning,
			})
			exhibit.Claims = append(exhibit.Claims, stake)
		}
		for _, fact := range item.Facts {
			row := recordcase.Fact{Label: fact.Label, Value: fact.Value}
			if fact.Value == "" && fact.At > 0 {
				at := fact.At
				row.At = &at
			}
			exhibit.Facts = append(exhibit.Facts, row)
			if fact.Value != "" {
				exhibit.Claims = append(exhibit.Claims, fact.Label+": "+fact.Value)
			}
		}
		out = append(out, exhibit)
	}

	return out
}

func factValue(item *watchtower.Item, label string) string {
	for _, fact := range item.Facts {
		if fact.Label == label {
			return fact.Value
		}
	}

	return ""
}

func lowerWords(value string) string {
	return strings.ToLower(textcase.HumanizeCamelCaseSentence(value))
}
