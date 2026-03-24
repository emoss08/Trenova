package sim

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	scriptModeMerge    = "merge"
	scriptModeOverride = "override"
)

type scriptFile struct {
	Version   int               `yaml:"version"`
	Timezone  string            `yaml:"timezone"`
	Scenarios []scriptScenarioY `yaml:"scenarios"`
}

type scriptScenarioY struct {
	ID       string         `yaml:"id"`
	DriverID string         `yaml:"driverId"`
	Vehicle  string         `yaml:"vehicleId"`
	BaseDate string         `yaml:"baseDate"`
	Events   []scriptEventY `yaml:"events"`
}

type scriptEventY struct {
	Type       string         `yaml:"type"`
	Start      string         `yaml:"start"`
	DurationMs int64          `yaml:"durationMs"`
	Severity   string         `yaml:"severity"`
	Metadata   map[string]any `yaml:"metadata"`
}

type compiledScriptScenario struct {
	ID       string
	DriverID string
	Vehicle  string
	BaseDate time.Time
	Events   []compiledScriptEvent
}

type compiledScriptEvent struct {
	EventType string
	Offset    time.Duration
	Duration  time.Duration
	Severity  string
	Metadata  map[string]any
	Index     int
}

type ScriptStatus struct {
	Loaded        bool     `json:"loaded"`
	Path          string   `json:"path"`
	Mode          string   `json:"mode"`
	ScenarioCount int      `json:"scenarioCount"`
	EventCount    int      `json:"eventCount"`
	Warnings      []string `json:"warnings"`
}

type ScriptEngine struct {
	mu        sync.RWMutex
	path      string
	mode      string
	timezone  string
	scenarios []compiledScriptScenario
	warnings  []string
	loaded    bool
}

func NewScriptEngine(path, mode, timezone string) *ScriptEngine {
	engine := &ScriptEngine{
		path:     strings.TrimSpace(path),
		mode:     normalizeScriptMode(mode),
		timezone: firstNonEmpty(strings.TrimSpace(timezone), "UTC"),
	}
	return engine
}

func (s *ScriptEngine) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(s.path) == "" {
		s.loaded = false
		s.scenarios = []compiledScriptScenario{}
		s.warnings = []string{"script path is empty"}
		return ErrScriptConfigInvalid
	}

	content, err := os.ReadFile(s.path)
	if err != nil {
		s.loaded = false
		s.scenarios = []compiledScriptScenario{}
		s.warnings = []string{fmt.Sprintf("read script file: %v", err)}
		return fmt.Errorf("%w: %w", ErrScriptParseFailed, err)
	}

	payload := scriptFile{}
	if err = yaml.Unmarshal(content, &payload); err != nil {
		s.loaded = false
		s.scenarios = []compiledScriptScenario{}
		s.warnings = []string{fmt.Sprintf("parse yaml: %v", err)}
		return fmt.Errorf("%w: %w", ErrScriptParseFailed, err)
	}

	if payload.Version != 1 {
		s.loaded = false
		s.scenarios = []compiledScriptScenario{}
		s.warnings = []string{"script version must be 1"}
		return ErrScriptConfigInvalid
	}

	locationName := firstNonEmpty(strings.TrimSpace(payload.Timezone), s.timezone)
	location, err := time.LoadLocation(locationName)
	if err != nil {
		s.loaded = false
		s.scenarios = []compiledScriptScenario{}
		s.warnings = []string{fmt.Sprintf("invalid timezone %q: %v", locationName, err)}
		return ErrScriptConfigInvalid
	}

	compiled := make([]compiledScriptScenario, 0, len(payload.Scenarios))
	for idx := range payload.Scenarios {
		scenario, compileErr := compileScenario(&payload.Scenarios[idx], location)
		if compileErr != nil {
			s.loaded = false
			s.scenarios = []compiledScriptScenario{}
			s.warnings = []string{compileErr.Error()}
			return compileErr
		}
		if overlapErr := validateDutyEventOverlaps(&scenario); overlapErr != nil {
			s.loaded = false
			s.scenarios = []compiledScriptScenario{}
			s.warnings = []string{overlapErr.Error()}
			return overlapErr
		}
		compiled = append(compiled, scenario)
	}

	s.scenarios = compiled
	s.loaded = true
	s.warnings = []string{}
	return nil
}

func compileScenario(
	raw *scriptScenarioY,
	location *time.Location,
) (compiledScriptScenario, error) {
	if raw == nil {
		return compiledScriptScenario{}, ErrScriptConfigInvalid
	}
	id := strings.TrimSpace(raw.ID)
	if id == "" {
		return compiledScriptScenario{}, ErrScriptConfigInvalid
	}
	driverID := strings.TrimSpace(raw.DriverID)
	if driverID == "" {
		return compiledScriptScenario{}, ErrScriptConfigInvalid
	}
	vehicleID := strings.TrimSpace(raw.Vehicle)
	if vehicleID == "" {
		return compiledScriptScenario{}, ErrScriptConfigInvalid
	}

	baseDate, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(raw.BaseDate), location)
	if err != nil {
		return compiledScriptScenario{}, ErrScriptConfigInvalid
	}

	events := make([]compiledScriptEvent, 0, len(raw.Events))
	for idx := range raw.Events {
		item := raw.Events[idx]
		eventType := strings.TrimSpace(item.Type)
		if _, ok := allowedScriptEventTypes[eventType]; !ok {
			return compiledScriptScenario{}, ErrScriptConfigInvalid
		}

		offset, offsetErr := parseScriptStartOffset(item.Start)
		if offsetErr != nil {
			return compiledScriptScenario{}, ErrScriptConfigInvalid
		}
		if item.DurationMs <= 0 {
			return compiledScriptScenario{}, ErrScriptConfigInvalid
		}

		events = append(events, compiledScriptEvent{
			EventType: eventType,
			Offset:    offset,
			Duration:  time.Duration(item.DurationMs) * time.Millisecond,
			Severity:  firstNonEmpty(strings.TrimSpace(item.Severity), "info"),
			Metadata:  item.Metadata,
			Index:     idx,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].Offset == events[j].Offset {
			return events[i].Index < events[j].Index
		}
		return events[i].Offset < events[j].Offset
	})

	return compiledScriptScenario{
		ID:       id,
		DriverID: driverID,
		Vehicle:  vehicleID,
		BaseDate: baseDate,
		Events:   events,
	}, nil
}

var allowedScriptEventTypes = map[string]struct{}{
	simEventStopTrafficDelay: {},
	simEventStopFuelBreak:    {},
	simEventDutyOffDutyPause: {},
	simEventDutySleeperBlock: {},
	simEventSpeedMinor:       {},
	simEventSpeedMajor:       {},
	simEventViolationBreak:   {},
	simEventViolationShift:   {},
	simEventViolationDrive:   {},
	simEventViolationCycle:   {},
}

func parseScriptStartOffset(value string) (time.Duration, error) {
	clean := strings.TrimSpace(value)
	layouts := []string{"15:04:05", "15:04"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, clean)
		if err != nil {
			continue
		}
		h, m, sec := parsed.Clock()
		return time.Duration(h)*time.Hour +
			time.Duration(m)*time.Minute +
			time.Duration(sec)*time.Second, nil
	}
	return 0, ErrScriptConfigInvalid
}

func validateDutyEventOverlaps(scenario *compiledScriptScenario) error {
	if scenario == nil {
		return ErrScriptConfigInvalid
	}

	type interval struct {
		start time.Duration
		end   time.Duration
	}
	dutyIntervals := make([]interval, 0, len(scenario.Events))
	for _, event := range scenario.Events {
		if event.EventType != simEventDutyOffDutyPause &&
			event.EventType != simEventDutySleeperBlock {
			continue
		}
		dutyIntervals = append(dutyIntervals, interval{
			start: event.Offset,
			end:   event.Offset + event.Duration,
		})
	}
	if len(dutyIntervals) <= 1 {
		return nil
	}
	sort.Slice(dutyIntervals, func(i, j int) bool {
		return dutyIntervals[i].start < dutyIntervals[j].start
	})
	for idx := 1; idx < len(dutyIntervals); idx++ {
		prev := dutyIntervals[idx-1]
		curr := dutyIntervals[idx]
		if curr.start < prev.end {
			return ErrScriptConfigInvalid
		}
	}
	return nil
}

func (s *ScriptEngine) Status() ScriptStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	eventCount := 0
	for _, scenario := range s.scenarios {
		eventCount += len(scenario.Events)
	}
	return ScriptStatus{
		Loaded:        s.loaded,
		Path:          s.path,
		Mode:          s.mode,
		ScenarioCount: len(s.scenarios),
		EventCount:    eventCount,
		Warnings:      append([]string{}, s.warnings...),
	}
}

func (s *ScriptEngine) Mode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

func (s *ScriptEngine) EventsWindow(
	start time.Time,
	end time.Time,
	driverFilter map[string]struct{},
	vehicleFilter map[string]struct{},
) []SimEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.loaded || len(s.scenarios) == 0 {
		return []SimEvent{}
	}

	out := make([]SimEvent, 0, len(s.scenarios)*4)
	start = start.UTC()
	end = end.UTC()
	for _, scenario := range s.scenarios {
		if len(driverFilter) > 0 {
			if _, ok := driverFilter[scenario.DriverID]; !ok {
				continue
			}
		}
		if len(vehicleFilter) > 0 {
			if _, ok := vehicleFilter[scenario.Vehicle]; !ok {
				continue
			}
		}

		for _, event := range scenario.Events {
			startsAt := scenario.BaseDate.Add(event.Offset).UTC()
			endsAt := startsAt.Add(event.Duration)
			if endsAt.Before(start) || !startsAt.Before(end) {
				continue
			}
			out = append(out, SimEvent{
				ID:        buildScriptEventID(scenario.ID, scenario.BaseDate, event.Index),
				Type:      event.EventType,
				DriverID:  scenario.DriverID,
				VehicleID: scenario.Vehicle,
				StartsAt:  startsAt,
				EndsAt:    endsAt,
				Severity:  event.Severity,
				Metadata:  cloneAnyAsMap(event.Metadata),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].StartsAt.Equal(out[j].StartsAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].StartsAt.Before(out[j].StartsAt)
	})
	return out
}

func normalizeScriptMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case scriptModeOverride:
		return scriptModeOverride
	default:
		return scriptModeMerge
	}
}

func buildScriptEventID(scenarioID string, baseDate time.Time, index int) string {
	cleanScenario := strings.ToLower(strings.TrimSpace(scenarioID))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "_", "-", ".", "-")
	cleanScenario = replacer.Replace(cleanScenario)
	return fmt.Sprintf(
		"evt-script-%s-%s-%d",
		cleanScenario,
		baseDate.UTC().Format("20060102"),
		index,
	)
}

func cloneAnyAsMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	cloned := cloneAny(raw)
	value, _ := anyAsMap(cloned)
	if value == nil {
		return map[string]any{}
	}
	return value
}
