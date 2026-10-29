package benchscenario

import (
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
)

const (
	NeverDisclose  = 1_000_000
	AlwaysDisclose = 0
)

type SweepMode string

const (
	SweepAll       = SweepMode("all")
	SweepDisclosed = SweepMode("disclosed")
)

func (m SweepMode) Threshold() int {
	if m == SweepDisclosed {
		return AlwaysDisclose
	}

	return NeverDisclose
}

type Sweep struct {
	Held int
	Mode SweepMode
}

func (s *Sweep) Label() string {
	return "held " + strconv.Itoa(s.Held) + ", " + string(s.Mode)
}

var ErrSweepSkipped = errors.New("skipped by the tool-count sweep")

func SweepTools(base, expected []string, held int) ([]string, error) {
	base = agentdefinition.WithoutCoreTools(base)
	required := make([]string, 0, len(expected))
	for _, spec := range expected {
		options := Alternatives(spec)
		chosen := ""
		for _, option := range options {
			if slices.Contains(base, option) {
				chosen = option
				break
			}
		}
		if chosen == "" {
			if len(options) == 0 || agentdefinition.IsCoreTool(options[0]) || Plumbing(options[0]) {
				continue
			}
			chosen = options[0]
		}
		if !slices.Contains(required, chosen) {
			required = append(required, chosen)
		}
	}
	if len(required) > held {
		return nil, fmt.Errorf("%w: the scenario needs %d tools, more than %d", ErrSweepSkipped,
			len(required), held)
	}

	tools := slices.Clone(required)
	for _, name := range base {
		if len(tools) == held {
			break
		}
		if !slices.Contains(tools, name) {
			tools = append(tools, name)
		}
	}
	if len(tools) < held {
		return nil, fmt.Errorf("%w: the agent holds %d tools, fewer than %d", ErrSweepSkipped,
			len(tools), held)
	}
	slices.Sort(tools)

	return tools, nil
}
