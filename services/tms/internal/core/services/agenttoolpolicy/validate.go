package agenttoolpolicy

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/productguide"
)

const artifactProbeID = "artifact-probe"

var permissionRegistry = sync.OnceValue(permission.NewRegistry)

type Spec struct {
	ToolName string
	Policy   serviceports.ToolPolicy
	Reports  bool
}

func Validate(specs []Spec) error {
	problems := make([]error, 0)
	seen := make(map[string]struct{}, len(specs))

	for idx := range specs {
		spec := &specs[idx]
		name := spec.Policy.Name
		if strings.TrimSpace(name) == "" {
			problems = append(problems, fmt.Errorf("tool %q: policy has no name", spec.ToolName))
			name = spec.ToolName
		}
		if _, dup := seen[name]; dup {
			problems = append(problems, fmt.Errorf("%s: declared twice", name))
		}
		seen[name] = struct{}{}

		for _, problem := range specProblems(spec) {
			problems = append(problems, fmt.Errorf("%s: %s", name, problem))
		}
	}

	return errors.Join(problems...)
}

func specProblems(spec *Spec) []string {
	policy := spec.Policy
	problems := make([]string, 0)
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}

	if spec.ToolName != policy.Name {
		add("policy is named %q but the tool is %q", policy.Name, spec.ToolName)
	}
	if !policy.Kind.IsValid() {
		add("kind %q is not valid", policy.Kind)
	}
	if !policy.Scope.IsValid() {
		add("scope %q is not valid", policy.Scope)
	}
	if !policy.Effect.IsValid() {
		add("effect %q is not valid", policy.Effect)
	}
	if !policy.DefaultTier.IsValid() {
		add("default tier %q is not valid", policy.DefaultTier)
	}
	if !policy.MaxTier.IsValid() {
		add("max tier %q is not valid", policy.MaxTier)
	}
	if !policy.Operation.IsValid() {
		add("operation %q is not valid", policy.Operation)
	}
	if !permissionRegistry().HasResource(policy.Resource.String()) {
		add("resource %q is not a registered permission resource", policy.Resource)
	}
	if strings.TrimSpace(policy.Rationale) == "" {
		add("has no rationale")
	}

	problems = append(problems, egressProblems(policy)...)
	problems = append(problems, tierProblems(policy)...)
	problems = append(problems, externalReadProblems(policy)...)

	if spec.Reports && !artifactResolves(policy.Artifact) {
		add("reports what it made, so its artifact %q must be a record-link key", policy.Artifact)
	}
	if !spec.Reports && policy.Artifact != "" && !artifactResolves(policy.Artifact) {
		add("artifact %q is not a record-link key", policy.Artifact)
	}
	if policy.Condition != nil &&
		(policy.Condition.Limit == nil || strings.TrimSpace(policy.Condition.Description) == "") {
		add("condition needs both a description and a limit")
	}

	return problems
}

func egressProblems(policy serviceports.ToolPolicy) []string {
	problems := make([]string, 0)
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}

	if len(policy.Egress) == 0 {
		add("declares no egress class")
	}
	seen := make(map[agent.EgressClass]struct{}, len(policy.Egress))
	for _, class := range policy.Egress {
		if !class.IsValid() {
			add("egress class %q is not valid", class)
		}
		if _, dup := seen[class]; dup {
			add("egress class %q is listed twice", class)
		}
		seen[class] = struct{}{}
	}

	switch policy.Kind {
	case agent.ToolKindQuery:
		if len(policy.Egress) != 1 || policy.Egress[0] != agent.EgressNone {
			add("a query tool's only egress class is none")
		}
		if policy.Operation != permission.OpRead {
			add("a query tool is used under the read operation, not %q", policy.Operation)
		}
	case agent.ToolKindAction:
		if policy.HasEgress(agent.EgressNone) {
			add("an action tool changes something, so none is not one of its classes")
		}
	case agent.ToolKindRuntime:
		for _, class := range policy.Egress {
			if class.Leaves() {
				add("a runtime tool never leaves the organization, but declares %q", class)
			}
		}
	}

	if len(policy.Egress) > 1 && policy.Classify == nil {
		add("declares %d egress classes, so it must classify each call", len(policy.Egress))
	}
	if policy.Scope == agent.ToolScopeSelf && policy.Kind != agent.ToolKindQuery &&
		!policy.HasEgress(agent.EgressPersonal) {
		add("acts only on the caller's own records, so personal is one of its classes")
	}
	if policy.PersonalRunsUnasked && !policy.HasEgress(agent.EgressPersonal) {
		add("runs a personal call unasked, so personal is one of its classes")
	}

	return problems
}

func tierProblems(policy serviceports.ToolPolicy) []string {
	if !policy.DefaultTier.IsValid() || !policy.MaxTier.IsValid() || len(policy.Egress) == 0 {
		return nil
	}

	problems := make([]string, 0)
	if policy.DefaultTier.Above(policy.MaxTier) {
		problems = append(problems, fmt.Sprintf(
			"default tier %s is above its max tier %s", policy.DefaultTier, policy.MaxTier))
	}
	if ceiling := policy.EgressCeiling(); policy.MaxTier.Above(ceiling) {
		problems = append(problems, fmt.Sprintf(
			"max tier %s is above %s, the most its egress classes allow",
			policy.MaxTier, ceiling))
	}

	return problems
}

func externalReadProblems(policy serviceports.ToolPolicy) []string {
	problems := make([]string, 0)
	if !policy.ReadsExternal.IsValid() {
		problems = append(problems,
			fmt.Sprintf("external read %q is not valid", policy.ReadsExternal))
	}
	if policy.ReadsExternal.IsValid() && policy.ReadsExternal != agent.ExternalReadNever &&
		!policy.Source.IsValid() {
		problems = append(problems, "reads outside text, so it names the taint source")
	}
	if policy.Source != "" && !policy.Source.IsValid() {
		problems = append(problems, fmt.Sprintf("taint source %q is not valid", policy.Source))
	}
	if policy.CarriesTaint && policy.Kind != agent.ToolKindAction {
		problems = append(problems, "only a write can carry a run's taint into what it saves")
	}

	return problems
}

func artifactResolves(entity string) bool {
	if strings.TrimSpace(entity) == "" {
		return false
	}
	_, ok := productguide.RecordPath(entity, artifactProbeID)

	return ok
}
