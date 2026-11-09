package auditoverview

import (
	"math"
	"strconv"

	"github.com/emoss08/trenova/shared/i18n"
)

type Target string

const (
	TargetReview = Target("Review")
	TargetView   = Target("View")
)

const (
	ViewAgents   = "agents"
	ViewSecurity = "security"
	ViewDeletes  = "deletes"
)

const (
	tagStrong = "b"
	tagAgents = "v"
	tagReview = "r"

	offHoursWorthSaying = 0.1
	percentScale        = 100
)

var narrativeTags = []string{
	tagStrong,
	tagAgents,
	tagReview + "1",
	tagReview + "2",
	tagReview + "3",
}

type Segment struct {
	Text     string
	Strong   bool
	Tone     string
	Target   Target
	TargetID string
}

type narration struct {
	locale i18n.Locale
	days   int
	result *Overview
}

type parts map[string]Segment

func compose(pieces []i18n.RichPiece, p parts) []Segment {
	out := make([]Segment, 0, len(pieces))
	for _, piece := range pieces {
		segment, ok := p[piece.Tag]
		if piece.Tag == "" || !ok {
			segment = Segment{}
		}
		segment.Text = piece.Text
		if segment.Text != "" {
			out = append(out, segment)
		}
	}

	return out
}

func join(sentences ...[]Segment) []Segment {
	out := make([]Segment, 0, 16)
	for _, sentence := range sentences {
		if len(sentence) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, Segment{Text: " "})
		}
		out = append(out, sentence...)
	}

	return out
}

func percent(share float64) string {
	return strconv.FormatInt(int64(math.Round(share*percentScale)), 10) + "%"
}

func share(part, whole int64) float64 {
	if whole <= 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

func (n *narration) number(value int64) string {
	return i18n.FormatNumber(n.locale, float64(value))
}

func summarize(n *narration) []Segment {
	if n.result.Totals.Entries == 0 {
		return []Segment{{Text: i18n.Translate(
			n.locale,
			"Nothing was written to the audit log {0, plural, one {today} other {in the last # days}}.",
			n.days,
		)}}
	}

	return join(
		volumeSentence(n),
		changeSentence(n),
		reviewSentence(n),
		agentSentence(n),
		offHoursSentence(n),
	)
}

func volumeSentence(n *narration) []Segment {
	t, a := n.result.Totals, n.result.Actors
	p := parts{tagStrong: {Strong: true}}
	switch {
	case a.People > 0 && a.Agents > 0 && a.Integrations > 0:
		return compose(i18n.TranslateRich(
			n.locale,
			"{0, plural, one {# person} other {# people}}, {1, plural, one {# agent} other {# agents}} and {2, plural, one {# API key} other {# API keys}} made <b>{3, plural, one {# change} other {# changes}}</b> {4, plural, one {today} other {in the last # days}}.",
			narrativeTags,
			a.People,
			a.Agents,
			a.Integrations,
			t.Entries,
			n.days,
		), p)
	case a.People > 0 && a.Agents > 0:
		return compose(i18n.TranslateRich(
			n.locale,
			"{0, plural, one {# person} other {# people}} and {1, plural, one {# agent} other {# agents}} made <b>{2, plural, one {# change} other {# changes}}</b> {3, plural, one {today} other {in the last # days}}.",
			narrativeTags,
			a.People,
			a.Agents,
			t.Entries,
			n.days,
		), p)
	case a.People > 0 && a.Integrations > 0:
		return compose(i18n.TranslateRich(
			n.locale,
			"{0, plural, one {# person} other {# people}} and {1, plural, one {# API key} other {# API keys}} made <b>{2, plural, one {# change} other {# changes}}</b> {3, plural, one {today} other {in the last # days}}.",
			narrativeTags,
			a.People,
			a.Integrations,
			t.Entries,
			n.days,
		), p)
	case a.People > 0:
		return compose(i18n.TranslateRich(
			n.locale,
			"{0, plural, one {# person} other {# people}} made <b>{1, plural, one {# change} other {# changes}}</b> {2, plural, one {today} other {in the last # days}}.",
			narrativeTags,
			a.People,
			t.Entries,
			n.days,
		), p)
	default:
		return compose(i18n.TranslateRich(
			n.locale,
			"<b>{0, plural, one {# change was} other {# changes were}}</b> written {1, plural, one {today} other {in the last # days}}.",
			narrativeTags,
			t.Entries,
			n.days,
		), p)
	}
}

func changeSentence(n *narration) []Segment {
	now, before := n.result.Totals.Entries, n.result.PreviousEntries
	if before <= 0 {
		return nil
	}
	delta := share(now-before, before)
	p := parts{tagStrong: {Strong: true}}
	switch {
	case math.Abs(delta) < 0.01:
		return compose(i18n.TranslateRich(
			n.locale,
			"That is about as many as the {0, plural, one {day} other {# days}} before.",
			narrativeTags, n.days,
		), p)
	case delta > 0:
		return compose(i18n.TranslateRich(
			n.locale,
			"That is <b>{0} more</b> than the {1, plural, one {day} other {# days}} before.",
			narrativeTags, percent(delta), n.days,
		), p)
	default:
		return compose(i18n.TranslateRich(
			n.locale,
			"That is <b>{0} fewer</b> than the {1, plural, one {day} other {# days}} before.",
			narrativeTags, percent(-delta), n.days,
		), p)
	}
}

func reviewSentence(n *narration) []Segment {
	r := n.result
	if !r.ReviewsIncluded || r.OpenReviewCount == 0 || len(r.OpenReviews) == 0 {
		return nil
	}

	p := parts{tagStrong: {Strong: true}}
	titles := make([]any, 0, len(r.OpenReviews))
	for i, review := range r.OpenReviews {
		p[tagReview+strconv.Itoa(i+1)] = Segment{
			Tone:     "danger",
			Target:   TargetReview,
			TargetID: review.ID.String(),
		}
		titles = append(titles, review.Title)
	}

	switch {
	case len(titles) == 1 && r.OpenReviewCount == 1:
		return compose(i18n.TranslateRich(n.locale,
			"<b>One thing needs a review</b>: <r1>{0}</r1>.",
			narrativeTags, titles[0]), p)
	case len(titles) == 2 && r.OpenReviewCount == 2:
		return compose(i18n.TranslateRich(n.locale,
			"<b>Two things need a review</b>: <r1>{0}</r1> and <r2>{1}</r2>.",
			narrativeTags, titles[0], titles[1]), p)
	case len(titles) == 3 && r.OpenReviewCount == 3:
		return compose(i18n.TranslateRich(n.locale,
			"<b>Three things need a review</b>: <r1>{0}</r1>, <r2>{1}</r2> and <r3>{2}</r3>.",
			narrativeTags, titles[0], titles[1], titles[2]), p)
	case len(titles) == 1:
		return compose(i18n.TranslateRich(n.locale,
			"<b>{0, plural, other {# things need}} a review</b>, the latest: <r1>{1}</r1>.",
			narrativeTags, r.OpenReviewCount, titles[0]), p)
	case len(titles) == 2:
		return compose(i18n.TranslateRich(
			n.locale,
			"<b>{0, plural, other {# things need}} a review</b>, the latest: <r1>{1}</r1> and <r2>{2}</r2>.",
			narrativeTags,
			r.OpenReviewCount,
			titles[0],
			titles[1],
		), p)
	default:
		return compose(i18n.TranslateRich(
			n.locale,
			"<b>{0, plural, other {# things need}} a review</b>, the latest: <r1>{1}</r1>, <r2>{2}</r2> and <r3>{3}</r3>.",
			narrativeTags,
			r.OpenReviewCount,
			titles[0],
			titles[1],
			titles[2],
		), p)
	}
}

func agentSentence(n *narration) []Segment {
	t := n.result.Totals
	agents := t.Agents()
	if agents == 0 {
		return nil
	}

	p := parts{tagAgents: {Target: TargetView, TargetID: ViewAgents}}
	of := percent(share(agents, t.Entries))
	switch {
	case t.AgentsOwn == 0:
		return compose(i18n.TranslateRich(n.locale,
			"Agents made <v>{0} of the changes</v>, all of them asked for in a conversation.",
			narrativeTags, of), p)
	case t.AgentsAsked == 0:
		return compose(i18n.TranslateRich(n.locale,
			"Agents made <v>{0} of the changes</v>, all of them on their own.",
			narrativeTags, of), p)
	default:
		return compose(i18n.TranslateRich(n.locale,
			"Agents made <v>{0} of the changes</v>, {1} of them asked for in a conversation.",
			narrativeTags, of, percent(share(t.AgentsAsked, agents))), p)
	}
}

func offHoursSentence(n *narration) []Segment {
	t := n.result.Totals
	part := share(t.PeopleOffHours, t.People)
	if t.PeopleOffHours == 0 || part < offHoursWorthSaying {
		return nil
	}

	return compose(i18n.TranslateRich(n.locale,
		"<b>{0}</b> of people's changes were made between 8 PM and 6 AM.",
		narrativeTags, percent(part)), parts{tagStrong: {Strong: true}})
}
