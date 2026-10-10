// Package aicontrolsummary is the one sentence that heads each tab of AI
// control: what is true now, worked out from the organization's own records,
// and reworded by a model only when those records say something new.
package aicontrolsummary

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
)

// Tab is the part of AI control a sentence heads.
type Tab string

const (
	TabOverview  = Tab("Overview")
	TabAgents    = Tab("Agents")
	TabProviders = Tab("Providers")
)

func (t Tab) IsValid() bool {
	switch t {
	case TabOverview, TabAgents, TabProviders:
		return true
	default:
		return false
	}
}

// AgentCounts are the organization's agents and what they are doing.
type AgentCounts struct {
	Total   int `json:"total"`
	On      int `json:"on"`
	Working int `json:"working"`
	// Waiting is proposals still waiting on a person.
	Waiting int `json:"waiting"`
	// Shadow is agents on and in shadow; ShadowRecorded is what they recorded
	// over the shadow window that nobody has been offered.
	Shadow         int `json:"shadow"`
	ShadowRecorded int `json:"shadowRecorded"`
}

// ProviderFailure is an enabled provider whose last call failed and has not
// succeeded since.
type ProviderFailure struct {
	ProviderID    pulid.ID `json:"providerId"`
	Name          string   `json:"name"`
	FailedCalls   int      `json:"failedCalls"`
	LastFailureAt int64    `json:"lastFailureAt"`
}

// ProviderRef names a provider a sentence links to.
type ProviderRef struct {
	ProviderID pulid.ID `json:"providerId"`
	Name       string   `json:"name"`
}

// Facts are what a tab's sentence may say. Nothing else may be said.
type Facts struct {
	Agents         AgentCounts `json:"agents"`
	ProvidersOn    int         `json:"providersOn"`
	ProvidersTotal int         `json:"providersTotal"`
	// WeekCalls is the model calls of the last seven days.
	WeekCalls int `json:"weekCalls"`
	// AwaitingKey is a provider whose protocol needs a key and that has none.
	AwaitingKey *ProviderRef `json:"awaitingKey,omitempty"`
	Failing     []ProviderFailure `json:"failing"`
	// Uncovered is the tasks no enabled provider can take.
	Uncovered int `json:"uncovered"`
	// Paused is every agent held in shadow by the organization's switch.
	Paused bool `json:"paused"`
}

// NoProvider reports an organization with nothing to send AI work to.
func (f *Facts) NoProvider() bool { return f.ProvidersOn == 0 }

// Numbers are every figure a sentence may contain.
func (f *Facts) Numbers() []int {
	numbers := []int{
		f.Agents.Total, f.Agents.On, f.Agents.Working, f.Agents.Waiting,
		f.Agents.Shadow, f.Agents.ShadowRecorded, f.ProvidersOn, f.Uncovered, len(f.Failing),
		f.ProvidersTotal, f.WeekCalls,
	}
	for _, failure := range f.Failing {
		numbers = append(numbers, failure.FailedCalls)
	}
	return numbers
}

// Hash names the facts. A sentence written for one hash is true for every
// read with the same hash, so it is written once.
func (f *Facts) Hash(tab Tab) string {
	var builder strings.Builder
	builder.WriteString(string(tab))
	for _, number := range []int{
		f.Agents.Total, f.Agents.On, f.Agents.Working, f.Agents.Waiting,
		f.Agents.Shadow, f.Agents.ShadowRecorded, f.ProvidersOn, f.Uncovered,
		f.ProvidersTotal, f.WeekCalls,
	} {
		builder.WriteByte('|')
		builder.WriteString(strconv.Itoa(number))
	}
	builder.WriteString("|paused=")
	builder.WriteString(strconv.FormatBool(f.Paused))
	if f.AwaitingKey != nil {
		builder.WriteString("|key=")
		builder.WriteString(f.AwaitingKey.ProviderID.String())
		builder.WriteByte(':')
		builder.WriteString(f.AwaitingKey.Name)
	}
	failing := slices.Clone(f.Failing)
	slices.SortFunc(failing, func(a, b ProviderFailure) int { return strings.Compare(a.Name, b.Name) })
	for _, failure := range failing {
		builder.WriteByte('|')
		builder.WriteString(failure.ProviderID.String())
		builder.WriteByte(':')
		builder.WriteString(failure.Name)
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(failure.FailedCalls))
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:12])
}
