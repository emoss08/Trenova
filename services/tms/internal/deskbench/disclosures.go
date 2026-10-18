package deskbench

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/shared/pulid"
)

type DisclosureKind string

const (
	DisclosurePreselected = DisclosureKind("preselected")
	DisclosureFound       = DisclosureKind("found")
)

type Disclosure struct {
	At             time.Time      `json:"at"`
	Kind           DisclosureKind `json:"kind"`
	Disclosed      bool           `json:"disclosed"`
	Held           int            `json:"held,omitempty"`
	Need           string         `json:"need,omitempty"`
	Ranked         []string       `json:"ranked"`
	DelegateCallID string         `json:"delegateCallId,omitempty"`

	attribution aiusageports.AIUsageAttribution
}

type DisclosureLog struct {
	mu      sync.Mutex
	entries []*Disclosure
}

func NewDisclosureLog() *DisclosureLog {
	return &DisclosureLog{entries: make([]*Disclosure, 0, 64)}
}

func (l *DisclosureLog) Preselected(_ context.Context, disclosure *agentports.ToolDisclosure) {
	l.add(DisclosurePreselected, disclosure)
}

func (l *DisclosureLog) Found(_ context.Context, disclosure *agentports.ToolDisclosure) {
	l.add(DisclosureFound, disclosure)
}

func (l *DisclosureLog) add(kind DisclosureKind, disclosure *agentports.ToolDisclosure) {
	if disclosure == nil {
		return
	}

	entry := &Disclosure{
		At:             time.Now(),
		Kind:           kind,
		Disclosed:      disclosure.Disclosed,
		Held:           disclosure.Held,
		Need:           disclosure.Need,
		Ranked:         slices.Clone(disclosure.Ranked),
		DelegateCallID: disclosure.Attribution.DelegateCallID,
		attribution:    disclosure.Attribution,
	}

	l.mu.Lock()
	l.entries = append(l.entries, entry)
	l.mu.Unlock()
}

func (l *DisclosureLog) For(threadID, ownerID pulid.ID, start, end time.Time) []Disclosure {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]Disclosure, 0, 4)
	for _, entry := range l.entries {
		if entry.At.Before(start) || entry.At.After(end) {
			continue
		}
		call := ModelCall{Attribution: entry.attribution}
		if call.belongsTo(threadID, ownerID) {
			out = append(out, *entry)
		}
	}

	return out
}
