package writecoverage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type Category string

const (
	CategorySecurity            = Category("security")
	CategoryConfiguration       = Category("configuration")
	CategoryUserPreference      = Category("user-preference")
	CategoryInfrastructure      = Category("infrastructure")
	CategoryAgentAdministration = Category("agent-administration")
	CategoryCounterparty        = Category("counterparty")
	CategoryReadOnly            = Category("read-only")
	CategoryAttestation         = Category("attestation")
	CategoryDuplicate           = Category("duplicate")
)

type CategoryInfo struct {
	Category Category
	Title    string
	Meaning  string
}

var categories = []CategoryInfo{
	{
		Category: CategorySecurity,
		Title:    "Security",
		Meaning: "Sign-in, sessions, passwords, API keys, SSO, identity providers, roles, " +
			"permissions and every other grant of access. Never agent-operated: an agent " +
			"that could widen access could widen its own.",
	},
	{
		Category: CategoryConfiguration,
		Title:    "Configuration",
		Meaning: "Organization-wide settings, controls, lookup tables, templates and " +
			"integration connections an administrator sets once and every later write " +
			"depends on.",
	},
	{
		Category: CategoryUserPreference,
		Title:    "User preference",
		Meaning: "A person's own interface state: saved table views, the sidebar, " +
			"favorites, notification read state, a profile picture.",
	},
	{
		Category: CategoryInfrastructure,
		Title:    "Infrastructure",
		Meaning: "Plumbing a client, a provider or the platform drives rather than a " +
			"decision a person makes: upload sessions, inbound webhooks, presence " +
			"signals, the GraphQL transport, repair operations.",
	},
	{
		Category: CategoryAgentAdministration,
		Title:    "Agent administration",
		Meaning: "Defining, configuring, evaluating and overseeing agents, including " +
			"deciding what they propose. An agent that did this would be grading its " +
			"own work.",
	},
	{
		Category: CategoryCounterparty,
		Title:    "Counterparty",
		Meaning: "Done by someone other than the organization's staff acting for " +
			"themselves: a driver in their own portal, a customer or carrier through a " +
			"public link. An agent acts for the organization and must not act as them.",
	},
	{
		Category: CategoryReadOnly,
		Title:    "Read-only",
		Meaning: "Sent as a POST or a mutation but only computes, previews, validates or " +
			"tests, and changes nothing.",
	},
	{
		Category: CategoryAttestation,
		Title:    "Attestation",
		Meaning: "A sign-off a named, accountable person must make: certifying a " +
			"regulatory summary, filing a return, overriding a failed vetting.",
	},
	{
		Category: CategoryDuplicate,
		Title:    "Duplicate",
		Meaning: "Another surface for a write listed elsewhere that the analysis could " +
			"not merge on its own. The reason names the write it duplicates.",
	},
}

func Categories() []CategoryInfo {
	return categories
}

func categoryInfo(category Category) (CategoryInfo, bool) {
	for _, info := range categories {
		if info.Category == category {
			return info, true
		}
	}

	return CategoryInfo{}, false
}

type Decision struct {
	Tools   []string `yaml:"tools"`
	Exempt  Category `yaml:"exempt"`
	Reason  string   `yaml:"reason"`
	Pending string   `yaml:"pending"`
}

type State string

const (
	StateCovered = State("covered")
	StateExempt  = State("exempt")
	StatePending = State("pending")
)

func (d Decision) State() State {
	switch {
	case len(d.Tools) > 0:
		return StateCovered
	case d.Exempt != "":
		return StateExempt
	default:
		return StatePending
	}
}

type Mapping struct {
	Writes map[string]Decision `yaml:"writes"`
}

func LoadMapping(path string) (Mapping, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Mapping{}, fmt.Errorf("read %s: %w", path, err)
	}

	return ParseMapping(content)
}

func ParseMapping(content []byte) (Mapping, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)

	var mapping Mapping
	if err := decoder.Decode(&mapping); err != nil && !errors.Is(err, io.EOF) {
		return Mapping{}, fmt.Errorf("parse %s: %w", MappingDisplayPath, err)
	}
	if mapping.Writes == nil {
		mapping.Writes = make(map[string]Decision)
	}

	return mapping, nil
}
