package lading

import (
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/shared/jsonutils"
)

const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
)

const (
	ClauseReceive = "receive"
	ClauseRespond = "respond"
	ClauseSend    = "send"
	ClauseExpect  = "expect"
)

type Range struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity string             `json:"severity"`
	Code     string             `json:"code"`
	Message  wasmmodule.Message `json:"message"`
	Fixes    jsonutils.RawJSON  `json:"fixes,omitempty"`
	Related  jsonutils.RawJSON  `json:"related,omitempty"`
}

type CheckedFile struct {
	Path        string       `json:"path"`
	Kind        string       `json:"kind"`
	Name        string       `json:"name"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type Party struct {
	Qualifier string `json:"qualifier"`
	ID        string `json:"id"`
}

type GroupCodes struct {
	Theirs string `json:"theirs"`
	Ours   string `json:"ours"`
}

type Interchange struct {
	Them       *Party      `json:"them"`
	Us         *Party      `json:"us"`
	Group      *GroupCodes `json:"group"`
	Version    string      `json:"version"`
	Usage      string      `json:"usage"`
	Separators []string    `json:"separators"`
}

type Setting struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Default string            `json:"default"`
	Range   []string          `json:"range"`
	Doc     string            `json:"doc"`
	Value   jsonutils.RawJSON `json:"value"`
}

type Partner struct {
	Name        string       `json:"name"`
	Label       string       `json:"label"`
	Zone        string       `json:"zone"`
	Currency    string       `json:"currency"`
	Settings    []Setting    `json:"settings"`
	Interchange *Interchange `json:"interchange"`
}

type Map struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Direction string `json:"direction"`
	From      string `json:"from"`
	To        string `json:"to"`
	Document  string `json:"document"`
	Set       string `json:"set"`
	Source    string `json:"source"`
}

type Clause struct {
	Kind        string   `json:"kind"`
	Set         string   `json:"set"`
	Binder      string   `json:"binder,omitempty"`
	Map         string   `json:"map,omitempty"`
	Rules       string   `json:"rules,omitempty"`
	Acknowledge string   `json:"acknowledge,omitempty"`
	Events      []string `json:"events,omitempty"`
	Until       string   `json:"until,omitempty"`
	Batched     bool     `json:"batched,omitempty"`
	Within      int64    `json:"within,omitempty"`
}

type Flow struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Clauses []Clause `json:"clauses"`
}

type RulesFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Map  string `json:"map"`
}

type Guide struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Set     string `json:"set"`
	Version string `json:"version"`
	Base    string `json:"base"`
}

type Check struct {
	Files     []CheckedFile `json:"files"`
	Partner   *Partner      `json:"partner"`
	Maps      []Map         `json:"maps"`
	Flows     []Flow        `json:"flows"`
	Rules     []RulesFile   `json:"rules"`
	Libraries []string      `json:"libraries"`
	Guides    []Guide       `json:"guides"`
	Errors    int           `json:"errors"`
	Warnings  int           `json:"warnings"`
}

func (c *Check) Map(name string) *Map {
	for index := range c.Maps {
		if c.Maps[index].Name == name {
			return &c.Maps[index]
		}
	}
	return nil
}

// Receiving is the flow and its receive clause that handles a set from the partner.
func (c *Check) Receiving(set string) (*Flow, *Clause) {
	return c.clause(ClauseReceive, func(clause *Clause) bool { return clause.Set == set })
}

// Sending is the flow and its send clause that writes a set on one of Trenova's events.
func (c *Check) Sending(set, event string) (*Flow, *Clause) {
	return c.clause(ClauseSend, func(clause *Clause) bool {
		if clause.Set != set {
			return false
		}
		for _, candidate := range clause.Events {
			if candidate == event {
				return true
			}
		}
		return false
	})
}

// Handling is the flow and its send clause that answers one of Trenova's events.
func (c *Check) Handling(event string) (*Flow, *Clause) {
	return c.clause(ClauseSend, func(clause *Clause) bool {
		for _, candidate := range clause.Events {
			if candidate == event {
				return true
			}
		}
		return false
	})
}

// Responding is the flow and its respond clause that answers a set.
func (c *Check) Responding(set string) (*Flow, *Clause) {
	return c.clause(ClauseRespond, func(clause *Clause) bool { return clause.Set == set })
}

// Expecting is the flow and its expect clause that watches acknowledgments.
func (c *Check) Expecting(flow string) *Clause {
	for index := range c.Flows {
		if c.Flows[index].Name != flow {
			continue
		}
		for clause := range c.Flows[index].Clauses {
			if c.Flows[index].Clauses[clause].Kind == ClauseExpect {
				return &c.Flows[index].Clauses[clause]
			}
		}
	}
	return nil
}

func (c *Check) clause(kind string, match func(*Clause) bool) (*Flow, *Clause) {
	for index := range c.Flows {
		flow := &c.Flows[index]
		for clause := range flow.Clauses {
			candidate := &flow.Clauses[clause]
			if candidate.Kind == kind && match(candidate) {
				return flow, candidate
			}
		}
	}
	return nil, nil
}

func (c *Check) Setting(name string) *Setting {
	if c.Partner == nil {
		return nil
	}
	for index := range c.Partner.Settings {
		if c.Partner.Settings[index].Name == name {
			return &c.Partner.Settings[index]
		}
	}
	return nil
}

type Formatted struct {
	Text  string              `json:"text"`
	Error *wasmmodule.Message `json:"error"`
}
