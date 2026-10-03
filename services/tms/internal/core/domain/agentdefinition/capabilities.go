package agentdefinition

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	// DefaultMaxChangeItems is the largest change an agent may make when
	// nobody has said otherwise: well past any one tool's own batch cap, so
	// the default never splits a change a tool accepts.
	DefaultMaxChangeItems = 500
	maxChangeItemsCeiling = 10000

	// DefaultBusinessHoursStart and DefaultBusinessHoursEnd are 7 AM and
	// 6 PM, in minutes after midnight.
	DefaultBusinessHoursStart = 7 * 60
	DefaultBusinessHoursEnd   = 18 * 60
	minutesInDay              = 24 * 60

	maxDelegateTopicRunes = 80
)

func (d *Definition) applyCapabilityDefaults() {
	if d.MaxChangeItems <= 0 {
		d.MaxChangeItems = DefaultMaxChangeItems
	}
	if d.BusinessHoursStart == 0 && d.BusinessHoursEnd == 0 {
		d.BusinessHoursStart = DefaultBusinessHoursStart
		d.BusinessHoursEnd = DefaultBusinessHoursEnd
	}
}

// ChangeLimit is the most records one change may touch.
func (d *Definition) ChangeLimit() int {
	if d.MaxChangeItems <= 0 {
		return DefaultMaxChangeItems
	}

	return d.MaxChangeItems
}

// BusinessHoursZone is the zone the business-hours window is read in: the
// agent's own, or fallback (the organization's) when it names none.
func (d *Definition) BusinessHoursZone(fallback string) string {
	if zone := strings.TrimSpace(d.BusinessHoursTimezone); zone != "" {
		return zone
	}

	return fallback
}

// WithinBusinessHours reports whether the agent may make a change on its own
// at now. It always may when the rule is off. The window is half open, so a
// change at the closing minute is outside it.
func (d *Definition) WithinBusinessHours(now time.Time, fallbackZone string) bool {
	if !d.BusinessHoursOnly {
		return true
	}

	local := now.In(timeutils.LoadLocation(d.BusinessHoursZone(fallbackZone)))
	minute := local.Hour()*60 + local.Minute()

	return minute >= d.BusinessHoursStart && minute < d.BusinessHoursEnd
}

// DelegateTopic is the label naming what goes to a delegate, or "" when
// none was given.
func (d *Definition) DelegateTopic(id pulid.ID) string {
	if d.DelegateTopics == nil {
		return ""
	}

	return strings.TrimSpace(d.DelegateTopics[id.String()])
}

// TurnToolOff removes a tool from the agent's grant and remembers it as
// switched off, so it can be offered back. Its tier and daily cap go with it,
// since both are only valid for a tool the agent holds.
func (d *Definition) TurnToolOff(name string) {
	d.ToolNames = slices.DeleteFunc(slices.Clone(d.ToolNames), func(held string) bool {
		return held == name
	})
	if _, ok := d.ToolTiers[name]; ok {
		d.ToolTiers = maps.Clone(d.ToolTiers)
		delete(d.ToolTiers, name)
	}
	if _, ok := d.ToolDailyLimits[name]; ok {
		d.ToolDailyLimits = maps.Clone(d.ToolDailyLimits)
		delete(d.ToolDailyLimits, name)
	}
	if !slices.Contains(d.DisabledToolNames, name) {
		d.DisabledToolNames = append(slices.Clone(d.DisabledToolNames), name)
	}
}

// TurnToolOn gives a switched-off tool back to the agent.
func (d *Definition) TurnToolOn(name string) {
	d.DisabledToolNames = slices.DeleteFunc(slices.Clone(d.DisabledToolNames),
		func(off string) bool { return off == name })
	if !slices.Contains(d.ToolNames, name) {
		d.ToolNames = append(slices.Clone(d.ToolNames), name)
	}
}

// ForgetReheldTools drops from the switched-off list any tool the agent
// holds again, which happens when an administrator picks it in AI Control.
func (d *Definition) ForgetReheldTools() {
	if len(d.DisabledToolNames) == 0 {
		return
	}
	d.DisabledToolNames = slices.DeleteFunc(slices.Clone(d.DisabledToolNames),
		func(off string) bool { return slices.Contains(d.ToolNames, off) })
	if len(d.DisabledToolNames) == 0 {
		d.DisabledToolNames = nil
	}
}

// FormatMinute is a minute after midnight as a clock time such as "7 AM"
// or "5:30 PM".
func FormatMinute(minute int) string {
	hour := (minute / 60) % 24
	mins := minute % 60
	suffix := "AM"
	if hour >= 12 {
		suffix = "PM"
	}
	display := hour % 12
	if display == 0 {
		display = 12
	}
	if mins == 0 {
		return fmt.Sprintf("%d %s", display, suffix)
	}

	return fmt.Sprintf("%d:%02d %s", display, mins, suffix)
}

func (d *Definition) validateCapabilities(multiErr *errortypes.MultiError) {
	if d.MaxChangeItems < 1 || d.MaxChangeItems > maxChangeItemsCeiling {
		multiErr.Add(
			"maxChangeItems",
			errortypes.ErrInvalid,
			"The largest single change must be between 1 and 10000 items",
		)
	}
	d.validateBusinessHours(multiErr)

	for id, topic := range d.DelegateTopics {
		field := "delegateTopics." + id
		parsed, err := pulid.Parse(id)
		if err != nil || !d.MayDelegateTo(parsed) {
			multiErr.Add(field, errortypes.ErrInvalid,
				"A topic can only name an agent this one hands work to")
			continue
		}
		if utf8.RuneCountInString(strings.TrimSpace(topic)) > maxDelegateTopicRunes {
			multiErr.Add(field, errortypes.ErrInvalid,
				"A hand-off topic cannot be longer than 80 characters")
		}
	}

	seen := make(map[string]struct{}, len(d.DisabledToolNames))
	for idx, name := range d.DisabledToolNames {
		field := fmt.Sprintf("disabledToolNames[%d]", idx)
		switch {
		case strings.TrimSpace(name) == "":
			multiErr.Add(field, errortypes.ErrInvalid, "Tool name cannot be empty")
		case IsCoreTool(name):
			multiErr.Add(field, errortypes.ErrInvalid, "Every agent holds this tool")
		case slices.Contains(d.ToolNames, name):
			multiErr.Add(field, errortypes.ErrInvalid, "A tool cannot be both on and off")
		}
		if _, duplicate := seen[name]; duplicate {
			multiErr.Add(field, errortypes.ErrDuplicate, "Tool is listed more than once")
		}
		seen[name] = struct{}{}
	}
	if len(d.DisabledToolNames) > MaxTools {
		multiErr.Add("disabledToolNames", errortypes.ErrInvalid,
			"An agent cannot have more than 64 tools switched off")
	}
}

func (d *Definition) validateBusinessHours(multiErr *errortypes.MultiError) {
	if d.BusinessHoursStart < 0 || d.BusinessHoursStart >= minutesInDay {
		multiErr.Add("businessHoursStart", errortypes.ErrInvalid,
			"Business hours must start within the day")
	}
	if d.BusinessHoursEnd <= 0 || d.BusinessHoursEnd > minutesInDay {
		multiErr.Add("businessHoursEnd", errortypes.ErrInvalid,
			"Business hours must end within the day")
	}
	if d.BusinessHoursEnd <= d.BusinessHoursStart {
		multiErr.Add("businessHoursEnd", errortypes.ErrInvalid,
			"Business hours must end after they start")
	}
	if zone := strings.TrimSpace(d.BusinessHoursTimezone); zone != "" {
		if _, err := time.LoadLocation(zone); err != nil {
			multiErr.Add("businessHoursTimezone", errortypes.ErrInvalid,
				"Time zone is not one this system knows")
		}
	}
}

// PruneDelegateTopics drops the topics of agents this one no longer hands
// work to.
// The map is rebuilt rather than edited, since a copy of the definition
// shares it.
func (d *Definition) PruneDelegateTopics() {
	kept := make(map[string]string, len(d.DelegateTopics))
	for id, topic := range d.DelegateTopics {
		parsed, err := pulid.Parse(id)
		if err == nil && d.MayDelegateTo(parsed) && strings.TrimSpace(topic) != "" {
			kept[id] = strings.TrimSpace(topic)
		}
	}
	if len(kept) == 0 {
		kept = nil
	}
	d.DelegateTopics = kept
}
