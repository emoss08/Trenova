package lading

import (
	"github.com/emoss08/trenova/pkg/wasmmodule"
	"github.com/emoss08/trenova/shared/jsonutils"
)

type TestFailure struct {
	Range   Range              `json:"range"`
	Message wasmmodule.Message `json:"message"`
}

type NamedValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TestResult struct {
	Path           string        `json:"path"`
	Name           string        `json:"name"`
	Kind           string        `json:"kind"`
	Range          Range         `json:"range"`
	Passed         bool          `json:"passed"`
	Cases          int           `json:"cases"`
	Failures       []TestFailure `json:"failures"`
	Counterexample []NamedValue  `json:"counterexample"`
}

type Coverage struct {
	Path      string  `json:"path"`
	Covered   []Range `json:"covered"`
	Uncovered []Range `json:"uncovered"`
}

type UnreadData struct {
	Map       string `json:"map"`
	Element   string `json:"element"`
	Qualifier string `json:"qualifier,omitempty"`
	Samples   int    `json:"samples"`
	Total     int    `json:"total"`
	Example   string `json:"example"`
	Read      string `json:"read,omitempty"`
}

type Readiness struct {
	Errors               int  `json:"errors"`
	Failing              int  `json:"failing"`
	Tests                int  `json:"tests"`
	UncoveredFlowClauses int  `json:"uncoveredFlowClauses"`
	Ready                bool `json:"ready"`
}

type TestReport struct {
	Tests     []TestResult `json:"tests"`
	Coverage  []Coverage   `json:"coverage"`
	Unread    []UnreadData `json:"unread"`
	Readiness Readiness    `json:"readiness"`
}

type ReplayInput struct {
	ID      string         `json:"id"`
	Input   RunInput       `json:"input"`
	Lookups []LookupAnswer `json:"lookups,omitempty"`
	Now     string         `json:"now,omitempty"`
}

type ReplayRequest struct {
	Before   jsonutils.RawJSON            `json:"before"`
	After    jsonutils.RawJSON            `json:"after"`
	Context  jsonutils.RawJSON            `json:"context"`
	Map      string                       `json:"map"`
	Inputs   []ReplayInput                `json:"inputs"`
	Settings map[string]jsonutils.RawJSON `json:"settings,omitempty"`
}

type Difference struct {
	Path   string            `json:"path"`
	Before jsonutils.RawJSON `json:"before"`
	After  jsonutils.RawJSON `json:"after"`
}

type ReplayOutcome struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Rule    string `json:"rule"`
}

type ReplayItem struct {
	ID           string              `json:"id"`
	Same         bool                `json:"same"`
	Differences  []Difference        `json:"differences"`
	NewOutcomes  []ReplayOutcome     `json:"newOutcomes"`
	GoneOutcomes []ReplayOutcome     `json:"goneOutcomes"`
	BeforeError  *wasmmodule.Message `json:"beforeError,omitempty"`
	AfterError   *wasmmodule.Message `json:"afterError,omitempty"`
}

type ReplaySummary struct {
	Total         int            `json:"total"`
	Same          int            `json:"same"`
	Differ        int            `json:"differ"`
	NewlyRejected int            `json:"newlyRejected"`
	NewlyHeld     int            `json:"newlyHeld"`
	NewlyDeclined int            `json:"newlyDeclined"`
	Failed        int            `json:"failed"`
	Fields        map[string]int `json:"fields"`
}

type ReplayReport struct {
	Items   []ReplayItem  `json:"items"`
	Summary ReplaySummary `json:"summary"`
}

const (
	RoleCarrier = "carrier"
	RoleBroker  = "broker"
)

const (
	AcknowledgeNone = "none"
)

type StarterRequest struct {
	Name        string     `json:"name"`
	Label       string     `json:"label,omitempty"`
	Role        string     `json:"role"`
	Them        *[2]string `json:"them,omitempty"`
	Us          *[2]string `json:"us,omitempty"`
	Zone        string     `json:"zone,omitempty"`
	Group       *[2]string `json:"group,omitempty"`
	Version     string     `json:"version,omitempty"`
	Separators  []string   `json:"separators,omitempty"`
	Usage       string     `json:"usage,omitempty"`
	Acknowledge string     `json:"acknowledge,omitempty"`
	AckWithin   int        `json:"ackWithin,omitempty"`
	Sets        []string   `json:"sets,omitempty"`
}

type Starter struct {
	Files   []File   `json:"files"`
	Samples []Sample `json:"samples"`
}
