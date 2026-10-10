package aicontrolsummary

import (
	"strconv"
	"strings"
)

// Target is where a linked part of a sentence goes. Only these exist, so a
// model can never invent a link.
type Target string

const (
	TargetWatchtower   = Target("watchtower")
	TargetProviders    = Target("providers")
	TargetRouting      = Target("routing")
	TargetAgentsShadow = Target("agents:shadow")
	TargetAgentsWait   = Target("agents:waiting")
	// TargetProvider is one provider's editor; the segment names it by ID.
	TargetProvider = Target("provider")
)

// Tone colours a linked part of the sentence.
type Tone string

const (
	TonePlain  = Tone("")
	ToneWarn   = Tone("warn")
	ToneDanger = Tone("danger")
)

// Segment is one run of a sentence: text, and where it leads when it leads
// somewhere. Strong marks a figure or name the reader's eye should land on.
type Segment struct {
	Text       string `json:"text"`
	Strong     bool   `json:"strong,omitempty"`
	Target     Target `json:"target,omitempty"`
	ProviderID string `json:"providerId,omitempty"`
	Tone       Tone   `json:"tone,omitempty"`
}

// Summary is a tab's sentence and how it came to be.
type Summary struct {
	Tab      Tab       `json:"tab"`
	Segments []Segment `json:"segments"`
	// Narrated is a sentence a model wrote; false is the plain one.
	Narrated bool `json:"narrated"`
	// Pending is a plain sentence while a model rewords it; read again soon.
	Pending     bool   `json:"pending"`
	FactsHash   string `json:"factsHash"`
	GeneratedAt int64  `json:"generatedAt"`
	Facts       Facts  `json:"facts"`
}

// Plain is the sentence the facts say on their own, without a model. It is
// what a reader sees with no provider, and what a model's version must keep
// to.
func Plain(tab Tab, facts *Facts) []Segment {
	switch tab {
	case TabAgents:
		return agentsSentence(facts)
	case TabProviders:
		return providersSentence(facts)
	default:
		return overviewSentence(facts)
	}
}

func overviewSentence(f *Facts) []Segment {
	if f.NoProvider() {
		return []Segment{
			text("Nothing can answer yet. Your "),
			strong(count(f.Agents.Total, "agent", "agents")),
			text(" " + isAre(f.Agents.Total) + " set up and waiting for a model provider — " +
				"connect one and they start on their own."),
		}
	}

	if f.Paused {
		out := []Segment{
			text("Every agent is "),
			{Text: "paused", Strong: true, Tone: ToneWarn},
			text(". They keep running and recording what they would do, but nothing is offered or executed"),
		}
		if f.Agents.Waiting > 0 {
			out = append(out,
				text(" — "),
				link(count(f.Agents.Waiting, "proposal", "proposals"), TargetWatchtower, TonePlain),
				text(" "+isAre(f.Agents.Waiting)+" held until you resume"),
			)
		}
		return append(out, text("."))
	}

	out := []Segment{strong(count(f.Agents.On, "agent", "agents"))}
	out = append(out, text(" "+isAre(f.Agents.On)+" on"))
	if f.Agents.Working > 0 {
		out = append(out, text(", "), strong(strconv.Itoa(f.Agents.Working)), text(" working right now"))
	}
	out = append(out, text(". "))

	failing := failingLink(f)
	switch {
	case f.Agents.Waiting > 0 && len(failing) > 0:
		out = append(out, waitingLink(f, TonePlain), text(" "+waitWaits(f.Agents.Waiting)+
			" on a person in Watchtower, and "))
		out = append(out, failing...)
		out = append(out, text(" can't connect. "))
	case f.Agents.Waiting > 0:
		out = append(out, waitingLink(f, TonePlain), text(" "+waitWaits(f.Agents.Waiting)+
			" on a person in Watchtower. "))
	case len(failing) > 0:
		out = append(out, failing...)
		out = append(out, text(" can't connect. "))
	}
	out = append(out, uncoveredClause(f, ToneWarn)...)
	return trimEnd(out)
}

func agentsSentence(f *Facts) []Segment {
	out := []Segment{
		strong(strconv.Itoa(f.Agents.On) + " of " + strconv.Itoa(f.Agents.Total)),
		text(" agents " + isAre(f.Agents.On) + " on"),
	}
	if f.Agents.Working > 0 {
		out = append(out, text(", and "), strong(strconv.Itoa(f.Agents.Working)),
			text(" "+isAre(f.Agents.Working)+" working right now"))
	}
	out = append(out, text(". "))
	if f.Agents.Waiting > 0 {
		out = append(out,
			link(count(f.Agents.Waiting, "proposal", "proposals"), TargetAgentsWait, ToneWarn),
			text(" "+waitWaits(f.Agents.Waiting)+" on a person. "),
		)
	}
	if f.Agents.Shadow > 0 {
		one := f.Agents.Shadow == 1
		out = append(out,
			text(count(f.Agents.Shadow, "agent", "agents")+" "+pick(one, "runs", "run")+" in "),
			link("shadow", TargetAgentsShadow, TonePlain),
			text(" and "+pick(one, "has", "have")+" recorded "+
				count(f.Agents.ShadowRecorded, "proposal", "proposals")+
				" nobody has seen — worth a look before you let "+pick(one, "it", "them")+" go live. "),
		)
	}
	return trimEnd(out)
}

func providersSentence(f *Facts) []Segment {
	if f.NoProvider() {
		return []Segment{
			text("No provider is on. "),
			text("Connect one and every AI feature starts working."),
		}
	}
	out := []Segment{
		strong(strconv.Itoa(f.ProvidersOn) + " of " + strconv.Itoa(f.ProvidersTotal)),
		text(" providers " + isAre(f.ProvidersOn) + " taking work — " +
			count(f.WeekCalls, "call", "calls") + " this week. "),
	}
	switch len(f.Failing) {
	case 0:
	case 1:
		out = append(out, failingLink(f)...)
		out = append(out, text(" is failing to connect, so its tasks fall through to the next in line. "))
	default:
		out = append(out, failingLink(f)...)
		out = append(out, text(" are failing to connect, so their tasks fall through to the next in line. "))
	}
	if f.AwaitingKey != nil {
		out = append(out,
			Segment{
				Text:       f.AwaitingKey.Name,
				Target:     TargetProvider,
				ProviderID: f.AwaitingKey.ProviderID.String(),
				Tone:       ToneWarn,
			},
			text(" is waiting for a key. "),
		)
	}
	out = append(out, uncoveredClause(f, TonePlain)...)
	return trimEnd(out)
}

func waitingLink(f *Facts, tone Tone) Segment {
	return link(count(f.Agents.Waiting, "proposal", "proposals"), TargetWatchtower, tone)
}

func failingLink(f *Facts) []Segment {
	switch len(f.Failing) {
	case 0:
		return nil
	case 1:
		failure := f.Failing[0]
		return []Segment{{
			Text:       failure.Name,
			Target:     TargetProvider,
			ProviderID: failure.ProviderID.String(),
			Tone:       ToneDanger,
		}}
	default:
		return []Segment{link(count(len(f.Failing), "provider", "providers"), TargetProviders, ToneDanger)}
	}
}

func uncoveredClause(f *Facts, tone Tone) []Segment {
	if f.Uncovered == 0 {
		return nil
	}
	return []Segment{
		link(count(f.Uncovered, "task", "tasks")+" "+hasHave(f.Uncovered), TargetRouting, tone),
		text(" nowhere to go. "),
	}
}

func pick(one bool, singular, plural string) string {
	if one {
		return singular
	}
	return plural
}

func text(value string) Segment   { return Segment{Text: value} }
func strong(value string) Segment { return Segment{Text: value, Strong: true} }

func link(value string, target Target, tone Tone) Segment {
	return Segment{Text: value, Target: target, Tone: tone}
}

func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return grouped(n) + " " + many
}

func grouped(n int) string {
	digits := strconv.Itoa(n)
	if n < 0 || len(digits) <= 3 {
		return digits
	}
	var builder strings.Builder
	lead := len(digits) % 3
	if lead > 0 {
		builder.WriteString(digits[:lead])
	}
	for idx := lead; idx < len(digits); idx += 3 {
		if builder.Len() > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(digits[idx : idx+3])
	}
	return builder.String()
}

func isAre(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}

func waitWaits(n int) string {
	if n == 1 {
		return "waits"
	}
	return "wait"
}

func hasHave(n int) string {
	if n == 1 {
		return "has"
	}
	return "have"
}

// trimEnd drops the space after the last sentence.
func trimEnd(segments []Segment) []Segment {
	if len(segments) == 0 {
		return segments
	}
	last := &segments[len(segments)-1]
	last.Text = strings.TrimRight(last.Text, " ")
	return segments
}

// Text is the sentence as plain prose.
func Text(segments []Segment) string {
	var builder strings.Builder
	for _, segment := range segments {
		builder.WriteString(segment.Text)
	}
	return strings.TrimSpace(builder.String())
}
