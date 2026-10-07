package agentdraftservice

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
)

const defaultIntervalSeconds = 900

var errEmptyReply = errors.New("the model returned nothing")

type draftTool struct {
	Name string `json:"name"`
	Tier string `json:"tier"`
}

type draftReply struct {
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	Icon            string      `json:"icon"`
	Accent          string      `json:"accent"`
	Instructions    string      `json:"instructions"`
	Guardrails      []string    `json:"guardrails"`
	TriggerMode     string      `json:"triggerMode"`
	CronExpression  string      `json:"cronExpression"`
	CronTimezone    string      `json:"cronTimezone"`
	EventKinds      []string    `json:"eventKinds"`
	IntervalSeconds int         `json:"intervalSeconds"`
	Tools           []draftTool `json:"tools"`
	AutonomyCeiling string      `json:"autonomyCeiling"`
	OutputMode      string      `json:"outputMode"`
}

func parseDraftReply(text string) (*draftReply, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, errEmptyReply
	}

	var reply draftReply
	if err := sonic.UnmarshalString(trimmed, &reply); err != nil {
		return nil, fmt.Errorf("the reply was not the requested object: %w", err)
	}

	return &reply, nil
}

type sanitizeInput struct {
	reply       *draftReply
	catalog     []serviceports.ToolCatalogEntry
	timezone    string
	description string
	tenantInfo  pagination.TenantInfo
}

type sanitizer struct {
	in    *sanitizeInput
	notes []serviceports.AgentDraftNote
}

func (s *sanitizer) drop(field, value, reason string) {
	s.notes = append(s.notes, serviceports.AgentDraftNote{
		Field:  field,
		Value:  stringutils.OneLine(value, agentdefinition.MaxGuardrailRunes),
		Reason: reason,
	})
}

func sanitizeDraft(
	in *sanitizeInput,
) (*serviceports.SaveAgentDefinitionRequest, []serviceports.AgentDraftNote, error) {
	s := &sanitizer{in: in, notes: make([]serviceports.AgentDraftNote, 0, 4)}
	reply := in.reply

	instructions := stringutils.TruncateRunes(
		strings.TrimSpace(reply.Instructions),
		agentdefinition.MaxInstructionsRunes,
	)
	ceiling := s.ceiling()
	toolNames, toolTiers := s.tools(ceiling)
	if instructions == "" && len(toolNames) == 0 {
		return nil, nil, errortypes.NewBusinessError(
			"The model could not draft an agent from that description. " +
				"Try saying what it should look at and what it should do.",
		)
	}

	trigger := s.trigger()
	req := &serviceports.SaveAgentDefinitionRequest{
		Name:              s.name(),
		Description:       stringutils.OneLine(reply.Description, agentdefinition.MaxDescriptionLength),
		Instructions:      instructions,
		Guardrails:        s.guardrails(),
		ToolNames:         toolNames,
		ToolTiers:         toolTiers,
		AutonomyCeiling:   ceiling,
		DataAccessCeiling: agentdefinition.DataAccessInternal,
		Enabled:           true,
		ShadowMode:        true,
		TriggerMode:       trigger.mode,
		CronExpression:    trigger.cron,
		CronTimezone:      trigger.timezone,
		EventKinds:        trigger.events,
		IntervalSeconds:   trigger.interval,
		Icon:              s.identity("icon", reply.Icon, agentdefinition.IsKnownIcon),
		Accent:            s.identity("accent", reply.Accent, agentdefinition.IsKnownAccent),
		OutputMode:        outputMode(reply.OutputMode, trigger.mode),
		TenantInfo:        in.tenantInfo,
	}

	return req, s.notes, nil
}

func (s *sanitizer) name() string {
	if name := stringutils.OneLine(s.in.reply.Name, agentdefinition.MaxNameLength); name != "" {
		return name
	}

	s.drop("name", "", "The model gave no name, so the description stands in until you name it.")

	return stringutils.OneLine(s.in.description, agentdefinition.MaxNameLength)
}

func (s *sanitizer) identity(field, value string, known func(string) bool) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || known(trimmed) {
		return trimmed
	}

	s.drop(field, trimmed, fmt.Sprintf("This is not a %s the app offers.", field))

	return ""
}

func (s *sanitizer) guardrails() []string {
	rules := make([]string, 0, len(s.in.reply.Guardrails))
	seen := make(map[string]struct{}, len(s.in.reply.Guardrails))
	for _, raw := range s.in.reply.Guardrails {
		rule := stringutils.CollapseWhitespace(raw)
		if rule == "" {
			continue
		}
		if _, duplicate := seen[rule]; duplicate {
			continue
		}
		seen[rule] = struct{}{}
		switch {
		case utf8.RuneCountInString(rule) > agentdefinition.MaxGuardrailRunes:
			s.drop("guardrails", rule, fmt.Sprintf(
				"A rule can be at most %d characters.", agentdefinition.MaxGuardrailRunes,
			))
		case len(rules) >= agentdefinition.MaxGuardrails:
			s.drop("guardrails", rule, fmt.Sprintf(
				"An agent can have at most %d rules.", agentdefinition.MaxGuardrails,
			))
		default:
			rules = append(rules, rule)
		}
	}

	return rules
}

func (s *sanitizer) ceiling() agent.AutonomyTier {
	value := strings.TrimSpace(s.in.reply.AutonomyCeiling)
	tier := agent.AutonomyTier(value)
	if tier.IsValid() {
		return tier
	}
	if value != "" {
		s.drop("autonomyCeiling", value, "This is not a ceiling; the agent only proposes.")
	}

	return agent.TierPropose
}

func (s *sanitizer) tools(
	ceiling agent.AutonomyTier,
) ([]string, map[string]agent.AutonomyTier) {
	offered := make(map[string]*serviceports.ToolCatalogEntry, len(s.in.catalog))
	for idx := range s.in.catalog {
		offered[s.in.catalog[idx].Name] = &s.in.catalog[idx]
	}

	names := make([]string, 0, len(s.in.reply.Tools))
	tiers := make(map[string]agent.AutonomyTier, len(s.in.reply.Tools))
	seen := make(map[string]struct{}, len(s.in.reply.Tools))
	for _, picked := range s.in.reply.Tools {
		name := strings.TrimSpace(picked.Name)
		if name == "" {
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}

		entry, ok := offered[name]
		switch {
		case !ok:
			s.drop("toolNames", name, "No tool by this name is offered to this organization's agents.")
			continue
		case entry.Core:
			continue
		case len(names) >= agentdefinition.MaxTools:
			s.drop("toolNames", name, fmt.Sprintf(
				"An agent can hold at most %d tools.", agentdefinition.MaxTools,
			))
			continue
		}
		names = append(names, name)

		if entry.Kind != serviceports.ToolCatalogKindAction {
			continue
		}
		if tier, set := s.tier(entry, picked.Tier, ceiling); set {
			tiers[name] = tier
		}
	}

	if len(tiers) == 0 {
		tiers = nil
	}

	return names, tiers
}

func (s *sanitizer) tier(
	entry *serviceports.ToolCatalogEntry,
	value string,
	ceiling agent.AutonomyTier,
) (agent.AutonomyTier, bool) {
	field := "toolTiers." + entry.Name
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}

	requested := agent.AutonomyTier(value)
	if !requested.IsValid() {
		s.drop(field, value, "This is not a tier; the tool keeps its own.")

		return "", false
	}

	limit := maxTierOf(entry).AtMost(ceiling)
	if requested.Above(limit) {
		s.drop(field, value, fmt.Sprintf("%s can be set to %s at most here.", entry.Name, limit))

		return limit, true
	}

	return requested, true
}

type sanitizedTrigger struct {
	mode     agentdefinition.TriggerMode
	cron     string
	timezone string
	events   []agent.EventKind
	interval int
}

func (s *sanitizer) trigger() sanitizedTrigger {
	reply := s.in.reply
	value := strings.TrimSpace(reply.TriggerMode)
	trigger := sanitizedTrigger{
		mode:     agentdefinition.TriggerMode(value),
		timezone: s.defaultTimezone(),
	}
	if value == "" {
		trigger.mode = agentdefinition.TriggerChat
	}
	if !trigger.mode.IsValid() {
		s.drop("triggerMode", value, "This is not a way an agent can start; it answers in chat.")
		trigger.mode = agentdefinition.TriggerChat
	}

	switch trigger.mode {
	case agentdefinition.TriggerScheduled:
		s.schedule(&trigger)
	case agentdefinition.TriggerEvent:
		s.events(&trigger)
	case agentdefinition.TriggerContinuous:
		s.interval(&trigger)
	case agentdefinition.TriggerChat:
	}

	return trigger
}

func (s *sanitizer) defaultTimezone() string {
	if agentdefinition.CronTimezoneValid(s.in.timezone) {
		return strings.TrimSpace(s.in.timezone)
	}

	return agentdefinition.DefaultCronTimezone
}

func (s *sanitizer) schedule(trigger *sanitizedTrigger) {
	reply := s.in.reply
	if zone := strings.TrimSpace(reply.CronTimezone); zone != "" {
		if agentdefinition.CronTimezoneValid(zone) {
			trigger.timezone = zone
		} else {
			s.drop("cronTimezone", zone, fmt.Sprintf(
				"This is not a time zone; the organization's, %s, is used.", trigger.timezone,
			))
		}
	}

	expression := strings.TrimSpace(reply.CronExpression)
	if agentdefinition.CronExpressionValid(expression) {
		trigger.cron = expression

		return
	}

	s.drop("cronExpression", expression,
		"This is not a five-field cron expression, so the schedule was cleared.")
	s.drop("triggerMode", string(agentdefinition.TriggerScheduled),
		"A schedule needs a valid cron expression; it answers in chat until you set one.")
	trigger.mode = agentdefinition.TriggerChat
}

func (s *sanitizer) events(trigger *sanitizedTrigger) {
	seen := make(map[agent.EventKind]struct{}, len(s.in.reply.EventKinds))
	for _, raw := range s.in.reply.EventKinds {
		kind := agent.EventKind(strings.TrimSpace(raw))
		if kind == "" {
			continue
		}
		if _, duplicate := seen[kind]; duplicate {
			continue
		}
		seen[kind] = struct{}{}
		if !kind.IsValid() {
			s.drop("eventKinds", string(kind), "This is not an event the system raises.")
			continue
		}
		trigger.events = append(trigger.events, kind)
	}

	if len(trigger.events) > 0 {
		return
	}

	s.drop("triggerMode", string(agentdefinition.TriggerEvent),
		"An event agent needs at least one event; it answers in chat until you choose one.")
	trigger.mode = agentdefinition.TriggerChat
}

func (s *sanitizer) interval(trigger *sanitizedTrigger) {
	requested := s.in.reply.IntervalSeconds
	switch {
	case requested >= agentdefinition.MinIntervalSeconds:
		trigger.interval = requested
	case requested <= 0:
		trigger.interval = defaultIntervalSeconds
	default:
		s.drop("intervalSeconds", fmt.Sprintf("%d", requested), fmt.Sprintf(
			"An agent waits at least %d seconds between runs.", agentdefinition.MinIntervalSeconds,
		))
		trigger.interval = agentdefinition.MinIntervalSeconds
	}
}

func outputMode(value string, trigger agentdefinition.TriggerMode) agentdefinition.OutputMode {
	if mode := agentdefinition.OutputMode(strings.TrimSpace(value)); mode.IsValid() {
		return mode
	}
	if trigger == agentdefinition.TriggerChat {
		return agentdefinition.OutputConversational
	}

	return agentdefinition.OutputReport
}
