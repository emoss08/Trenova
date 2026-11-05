package apichain

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type Probe string

const (
	ProbeUnauthenticated Probe = "unauthenticated"
	ProbeOtherTenant     Probe = "other-tenant"
	ProbeNoPermission    Probe = "no-permission"
)

var knownProbes = []Probe{ProbeUnauthenticated, ProbeOtherTenant, ProbeNoPermission}

type Chain struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Actor       string            `yaml:"actor"`
	Fixtures    map[string]string `yaml:"fixtures"`
	Steps       []Step            `yaml:"steps"`
	Probes      []Probe           `yaml:"probes"`
	Volatile    []string          `yaml:"volatile"`

	Path string `yaml:"-"`
}

type Step struct {
	ID          string            `yaml:"id"`
	GraphQL     string            `yaml:"graphql"`
	Document    string            `yaml:"document"`
	Vars        map[string]any    `yaml:"vars"`
	REST        string            `yaml:"rest"`
	Body        any               `yaml:"body"`
	Export      map[string]string `yaml:"export"`
	Expect      map[string]any    `yaml:"expect"`
	ExpectError string            `yaml:"expectError"`
	Effects     []Effect          `yaml:"effects"`
	OpenTo      []Probe           `yaml:"openTo"`
}

func (s *Step) IsOpenTo(probe Probe) bool {
	return slices.Contains(s.OpenTo, probe)
}

type Effect struct {
	Read      string `yaml:"read"`
	Path      string `yaml:"path"`
	Equals    any    `yaml:"equals"`
	Increase  any    `yaml:"increase"`
	Decrease  any    `yaml:"decrease"`
	Unchanged bool   `yaml:"unchanged"`
}

func (e Effect) NeedsBefore() bool {
	return e.Increase != nil || e.Decrease != nil || e.Unchanged
}

func (s *Step) IsGraphQL() bool {
	return s.GraphQL != ""
}

func (s *Step) Method() string {
	method, _, _ := strings.Cut(s.REST, " ")
	return method
}

func (s *Step) Route() string {
	_, route, _ := strings.Cut(s.REST, " ")
	return strings.TrimSpace(route)
}

func (s *Step) Label() string {
	if s.IsGraphQL() {
		return s.GraphQL
	}

	return s.REST
}

func (c *Chain) Step(id string) (*Step, bool) {
	for i := range c.Steps {
		if c.Steps[i].ID == id {
			return &c.Steps[i], true
		}
	}

	return nil, false
}

func (c *Chain) GoldenPath() string {
	return strings.TrimSuffix(c.Path, filepath.Ext(c.Path)) + ".golden.json"
}

func Load(path string) (*Chain, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	var chain Chain
	if err = decoder.Decode(&chain); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	chain.Path = path
	if err = chain.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return &chain, nil
}

func LoadDir(dir string) ([]*Chain, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	slices.Sort(paths)

	chains := make([]*Chain, 0, len(paths))
	for _, path := range paths {
		chain, loadErr := Load(path)
		if loadErr != nil {
			return nil, loadErr
		}
		chains = append(chains, chain)
	}

	return chains, nil
}

var (
	namePattern   = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	idPattern     = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
	restMethods   = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	reservedScope = []string{scopeFixture, scopeRun}
)

func (c *Chain) validate() error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	if !namePattern.MatchString(c.Name) {
		fail("name %q must be lowercase letters, digits and hyphens", c.Name)
	}
	if want := strings.TrimSuffix(filepath.Base(c.Path), filepath.Ext(c.Path)); c.Name != want {
		fail("name %q must match the file name %q", c.Name, want)
	}
	if c.Actor == "" {
		fail("actor is required: the email of the fixture user the chain logs in as")
	}
	if len(c.Steps) == 0 {
		fail("a chain needs at least one step")
	}
	for name := range c.Fixtures {
		if !idPattern.MatchString(name) {
			fail("fixture %q must be a camelCase name", name)
		}
	}
	for _, probe := range c.Probes {
		if !slices.Contains(knownProbes, probe) {
			fail("unknown probe %q (known: %v)", probe, knownProbes)
		}
	}

	seen := map[string]bool{}
	for i := range c.Steps {
		step := &c.Steps[i]
		for _, err := range c.validateStep(step, seen) {
			fail("step %d (%s): %w", i+1, step.ID, err)
		}
		seen[step.ID] = true
	}

	return errors.Join(errs...)
}

func (c *Chain) validateStep(step *Step, earlier map[string]bool) []error {
	var errs []error
	fail := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	if !idPattern.MatchString(step.ID) {
		fail("id %q must be a camelCase name", step.ID)
	}
	if slices.Contains(reservedScope, step.ID) {
		fail("id %q is reserved", step.ID)
	}
	if earlier[step.ID] {
		fail("id %q is used twice", step.ID)
	}
	for _, probe := range step.OpenTo {
		if !slices.Contains(c.Probes, probe) {
			fail("openTo names %q, which the chain does not probe", probe)
		}
	}

	switch {
	case step.GraphQL != "" && step.REST != "":
		fail("a step is graphql or rest, not both")
	case step.GraphQL == "" && step.REST == "":
		fail("a step needs graphql (an operation name) or rest (\"METHOD /api/v1/...\")")
	case step.REST != "":
		if !slices.Contains(restMethods, step.Method()) {
			fail("rest %q must start with one of %v", step.REST, restMethods)
		}
		if !strings.HasPrefix(step.Route(), "/") {
			fail("rest %q needs a path starting with /", step.REST)
		}
		if step.Vars != nil || step.Document != "" {
			fail("vars and document belong to graphql steps; a rest step sends body")
		}
	default:
		if step.Body != nil {
			fail("body belongs to rest steps; a graphql step sends vars")
		}
	}

	refs := References(step.Vars)
	refs = append(refs, References(step.Body)...)
	refs = append(refs, References(step.REST)...)
	refs = append(refs, References(step.Expect)...)
	for _, effect := range step.Effects {
		refs = append(refs, References(effect.Equals)...)
		refs = append(refs, References(effect.Increase)...)
		refs = append(refs, References(effect.Decrease)...)
	}
	for _, ref := range refs {
		if err := c.checkReference(ref, earlier); err != nil {
			errs = append(errs, err)
		}
	}

	for i, effect := range step.Effects {
		if _, ok := c.Step(effect.Read); !ok {
			fail("effect %d reads step %q, which the chain does not have", i+1, effect.Read)
		}
		if effect.Path == "" {
			fail("effect %d needs a path into the read's response", i+1)
		}
		set := 0
		for _, given := range []bool{effect.Equals != nil, effect.Increase != nil, effect.Decrease != nil, effect.Unchanged} {
			if given {
				set++
			}
		}
		if set != 1 {
			fail("effect %d needs exactly one of equals, increase, decrease or unchanged", i+1)
		}
	}

	return errs
}

func (c *Chain) checkReference(ref string, earlier map[string]bool) error {
	scope, rest, _ := strings.Cut(ref, ".")
	switch scope {
	case scopeFixture:
		if _, ok := c.Fixtures[rest]; !ok {
			return fmt.Errorf("${%s} names a fixture the chain does not declare", ref)
		}
	case scopeRun:
		if !slices.Contains(runValues, rest) {
			return fmt.Errorf("${%s}: run offers %v", ref, runValues)
		}
	default:
		if !earlier[scope] {
			return fmt.Errorf("${%s} refers to step %q, which has not run yet", ref, scope)
		}
	}

	return nil
}
