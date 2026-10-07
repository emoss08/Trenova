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
			text("No model provider is on, so "),
			strong(count(f.Agents.On, "agent", "agents")),
			text(" can't run yet. "),
			link("Connect a provider", TargetProviders, TonePlain),
			text(" to start."),
		}
	}

	out := []Segment{strong(count(f.Agents.On, "agent", "agents"))}
	out = append(out, text(" "+isAre(f.Agents.On)+" on"))
	switch {
	case f.Paused:
		out = append(out, text(", all paused in shadow"))
	case f.Agents.Working > 0:
		out = append(out, text(", "), strong(strconv.Itoa(f.Agents.Working)), text(" working right now"))
	}
	out = append(out, text(". "))
	out = append(out, waitingClause(f)...)
	out = append(out, failingClause(f)...)
	out = append(out, uncoveredClause(f)...)
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
	out = append(out, waitingClause(f)...)
	if f.Agents.Shadow > 0 {
		out = append(out,
			strong(count(f.Agents.Shadow, "agent", "agents")),
			text(" run in "),
			link("shadow", TargetAgentsShadow, TonePlain),
			text(" and have recorded "),
			strong(strconv.Itoa(f.Agents.ShadowRecorded)),
			text(" proposals nobody has seen. "),
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
		strong(count(f.ProvidersOn, "provider", "providers")),
		text(" " + isAre(f.ProvidersOn) + " on. "),
	}
	out = append(out, failingClause(f)...)
	out = append(out, uncoveredClause(f)...)
	if len(f.Failing) == 0 && f.Uncovered == 0 {
		out = append(out, text("Every task has a provider, and none is failing."))
	}
	return trimEnd(out)
}

func waitingClause(f *Facts) []Segment {
	if f.Agents.Waiting == 0 {
		return nil
	}
	return []Segment{
		link(count(f.Agents.Waiting, "proposal", "proposals"), TargetWatchtower, ToneWarn),
		text(" " + waitWaits(f.Agents.Waiting) + " on a person in Watchtower. "),
	}
}

func failingClause(f *Facts) []Segment {
	switch len(f.Failing) {
	case 0:
		return nil
	case 1:
		failure := f.Failing[0]
		return []Segment{
			{
				Text:       failure.Name,
				Target:     TargetProvider,
				ProviderID: failure.ProviderID.String(),
				Tone:       ToneDanger,
			},
			text(" can't connect. "),
		}
	default:
		return []Segment{
			link(count(len(f.Failing), "provider", "providers"), TargetProviders, ToneDanger),
			text(" can't connect. "),
		}
	}
}

func uncoveredClause(f *Facts) []Segment {
	if f.Uncovered == 0 {
		return nil
	}
	return []Segment{
		link(count(f.Uncovered, "task", "tasks"), TargetRouting, ToneWarn),
		text(" " + hasHave(f.Uncovered) + " nowhere to go. "),
	}
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
	return strconv.Itoa(n) + " " + many
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
