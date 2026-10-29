package benchresult

import (
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	slugLength = 60
	PassMark   = "PASS"
	FailMark   = "FAIL"
	SkipMark   = "SKIP"
)

type CheckResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

type CaseResult struct {
	Scenario     *benchscenario.Scenario   `json:"scenario"`
	Provider     string                    `json:"provider"`
	Cell         string                    `json:"cell,omitempty"`
	ProviderKind string                    `json:"providerKind,omitempty"`
	Model        string                    `json:"model,omitempty"`
	Effort       string                    `json:"effort,omitempty"`
	Metrics      CaseMetrics               `json:"metrics"`
	Repeat       int                       `json:"repeat"`
	Agent        string                    `json:"agent,omitempty"`
	ThreadID     string                    `json:"threadId,omitempty"`
	Steps        []*StepResult             `json:"steps"`
	Cleanup      []*benchrecord.TurnRecord `json:"cleanup,omitempty"`
	Passed       bool                      `json:"passed"`
	Skipped      string                    `json:"skipped,omitempty"`
	Error        string                    `json:"error,omitempty"`
	StartedAt    time.Time                 `json:"startedAt"`
	FinishedAt   time.Time                 `json:"finishedAt"`
	Usage        benchrecord.Usage         `json:"usage"`

	Unauthorized []Unauthorized `json:"unauthorizedEffects,omitempty"`
	AuditError   string         `json:"auditError,omitempty"`
}

func (c *CaseResult) Key() string {
	return c.Scenario.Name + " @ " + c.Provider
}

func (c *CaseResult) Errors() []string {
	errs := make([]string, 0, len(c.Steps)+1)
	if c.Error != "" {
		errs = append(errs, c.Error)
	}
	for _, step := range c.Steps {
		if step.Error != "" {
			errs = append(errs, "step "+strconv.Itoa(step.Index)+": "+step.Error)
		}
	}

	return errs
}

func (c *CaseResult) Verdict() string {
	switch {
	case c.Skipped != "":
		return SkipMark
	case c.Passed:
		return PassMark
	default:
		return FailMark
	}
}

func (c *CaseResult) ID() string {
	return stringutils.SlugifyASCII(c.Scenario.Name, slugLength) + "--" +
		stringutils.SlugifyASCII(c.Provider, slugLength) + "--" + strconv.Itoa(c.Repeat)
}

type StepResult struct {
	Index     int                           `json:"index"`
	Kind      string                        `json:"kind"`
	Label     string                        `json:"label"`
	Turns     []*benchrecord.TurnRecord     `json:"turns"`
	Decisions []*benchrecord.DecisionRecord `json:"decisions,omitempty"`
	Checks    []CheckResult                 `json:"checks"`
	Rubric    string                        `json:"rubric,omitempty"`
	Formula   string                        `json:"formula,omitempty"`
	Error     string                        `json:"error,omitempty"`
	Passed    bool                          `json:"passed"`
}
