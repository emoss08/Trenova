package sim

import (
	"fmt"
	"hash/fnv"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	HeaderProfileOverride = "X-Samsara-Sim-Profile"
)

type ScenarioProfile struct {
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	FailureRate   float64       `json:"failureRate"`
	OmitRate      float64       `json:"omitRate"`
	DataRatio     float64       `json:"dataRatio"`
	Delay         time.Duration `json:"delay"`
	EventOmitRate float64       `json:"eventOmitRate"`
	EventTypes    []string      `json:"eventTypes,omitempty"`
}

type ScenarioEngine struct {
	mu            sync.RWMutex
	seed          string
	activeProfile string
	profiles      map[string]ScenarioProfile
}

func NewScenarioEngine(
	seed string,
	defaultProfile string,
) (*ScenarioEngine, error) {
	profiles := defaultProfiles()
	active := strings.TrimSpace(defaultProfile)
	if active == "" {
		active = "default"
	}
	if _, ok := profiles[active]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrProfileNotFound, active)
	}

	return &ScenarioEngine{
		seed:          strings.TrimSpace(seed),
		activeProfile: active,
		profiles:      profiles,
	}, nil
}

func (s *ScenarioEngine) Profiles() []ScenarioProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ordered := []string{"default", "sparse", "partial", "degraded"}
	out := make([]ScenarioProfile, 0, len(s.profiles))
	for _, name := range ordered {
		profile, ok := s.profiles[name]
		if ok {
			out = append(out, profile)
		}
	}
	return out
}

func (s *ScenarioEngine) ActiveProfile() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeProfile
}

func (s *ScenarioEngine) SetActiveProfile(profile string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	candidate := strings.TrimSpace(profile)
	if _, ok := s.profiles[candidate]; !ok {
		return fmt.Errorf("%w: %s", ErrProfileNotFound, candidate)
	}
	s.activeProfile = candidate
	return nil
}

func (s *ScenarioEngine) ResolveProfile(request *http.Request) string {
	override := strings.TrimSpace(request.Header.Get(HeaderProfileOverride))
	if override != "" && s.exists(override) {
		return override
	}
	return s.ActiveProfile()
}

func (s *ScenarioEngine) Delay(profile, signature string) time.Duration {
	cfg, ok := s.profileConfig(profile)
	if !ok {
		return 0
	}
	if cfg.Delay <= 0 {
		return 0
	}
	factor := 0.5 + s.score(profile, signature)
	return time.Duration(float64(cfg.Delay) * factor)
}

func (s *ScenarioEngine) ShouldFail(profile, signature string) bool {
	cfg, ok := s.profileConfig(profile)
	if !ok {
		return false
	}
	if cfg.FailureRate <= 0 {
		return false
	}
	return s.score(profile, signature) < cfg.FailureRate
}

func (s *ScenarioEngine) ShouldOmitEvent(profile, eventType string, data any) bool {
	cfg, ok := s.profileConfig(profile)
	if !ok || cfg.EventOmitRate <= 0 {
		return false
	}

	cleanEventType := strings.TrimSpace(eventType)
	if cleanEventType == "" {
		return false
	}
	if !matchesEventType(cleanEventType, cfg.EventTypes) {
		return false
	}

	signature := "event|" + cleanEventType + "|" + eventIdentity(data)
	return s.score(profile, signature) < cfg.EventOmitRate
}

func (s *ScenarioEngine) Apply(profile, signature string, payload any) any {
	cfg, ok := s.profileConfig(profile)
	if !ok {
		return payload
	}
	cloned := cloneAny(payload)
	if cfg.DataRatio < 1 {
		cloned = s.applyDataRatio(cloned, cfg.DataRatio, profile, signature)
	}
	if cfg.OmitRate > 0 {
		cloned = s.applySparseOmissions(cloned, cfg.OmitRate, profile, signature)
	}
	return cloned
}

func (s *ScenarioEngine) exists(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.profiles[name]
	return ok
}

func (s *ScenarioEngine) profileConfig(name string) (ScenarioProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.profiles[name]
	return cfg, ok
}

func (s *ScenarioEngine) score(profile, signature string) float64 {
	fingerprint := strings.TrimSpace(s.seed) + "|" + profile + "|" + signature

	hash := fnv.New64a()
	_, _ = hash.Write([]byte(fingerprint))
	value := hash.Sum64() % 10000
	return float64(value) / 10000.0
}

func (s *ScenarioEngine) applyDataRatio(
	payload any,
	ratio float64,
	profile string,
	signature string,
) any {
	switch typed := payload.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, value := range typed {
			if key == "data" {
				out[key] = s.reduceDataSlice(value, ratio, profile, signature)
				continue
			}
			out[key] = s.applyDataRatio(value, ratio, profile, signature+"|"+key)
		}
		return out
	case []any:
		reduced := make([]any, 0, len(typed))
		for idx, value := range typed {
			score := s.score(profile, signature+fmt.Sprintf("|%d", idx))
			if score <= ratio {
				reduced = append(reduced, s.applyDataRatio(value, ratio, profile, signature))
			}
		}
		if len(reduced) == 0 && len(typed) > 0 {
			reduced = append(reduced, s.applyDataRatio(typed[0], ratio, profile, signature))
		}
		return reduced
	default:
		return payload
	}
}

func (s *ScenarioEngine) reduceDataSlice(
	payload any,
	ratio float64,
	profile string,
	signature string,
) any {
	records, ok := payload.([]any)
	if !ok {
		return s.applyDataRatio(payload, ratio, profile, signature)
	}

	filtered := make([]any, 0, len(records))
	for idx, item := range records {
		recordSignature := signature + "|" + strconv.Itoa(idx)
		if mapped, isMap := item.(map[string]any); isMap {
			recordSignature = signature + "|" + recordIdentity(mapped, idx)
		}
		if s.score(profile, recordSignature) <= ratio {
			filtered = append(filtered, s.applyDataRatio(item, ratio, profile, recordSignature))
		}
	}
	if len(filtered) == 0 && len(records) > 0 {
		filtered = append(filtered, s.applyDataRatio(records[0], ratio, profile, signature))
	}
	return filtered
}

func (s *ScenarioEngine) applySparseOmissions(
	payload any,
	rate float64,
	profile string,
	signature string,
) any {
	switch typed := payload.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, value := range typed {
			omitScore := s.score(profile, signature+"|omit|"+key)
			if isSparseCandidate(key) && omitScore < rate {
				continue
			}
			out[key] = s.applySparseOmissions(value, rate, profile, signature+"|"+key)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for idx, value := range typed {
			entry := s.applySparseOmissions(
				value,
				rate,
				profile,
				signature+fmt.Sprintf("|%d", idx),
			)
			out = append(out, entry)
		}
		return out
	default:
		return payload
	}
}

func recordIdentity(record map[string]any, fallback int) string {
	raw, ok := record["id"]
	if !ok {
		return fmt.Sprintf("item-%d", fallback)
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fmt.Sprintf("item-%d", fallback)
	}
	return value
}

func defaultProfiles() map[string]ScenarioProfile {
	return map[string]ScenarioProfile{
		"default": {
			Name:          "default",
			Description:   "full fidelity responses and no simulated failures",
			FailureRate:   0,
			OmitRate:      0,
			DataRatio:     1,
			Delay:         0,
			EventOmitRate: 0,
			EventTypes:    []string{},
		},
		"sparse": {
			Name:          "sparse",
			Description:   "drops optional fields and omits a meaningful slice of webhook events",
			FailureRate:   0,
			OmitRate:      0.65,
			DataRatio:     1,
			Delay:         0,
			EventOmitRate: 0.35,
			EventTypes:    []string{"*"},
		},
		"partial": {
			Name:          "partial",
			Description:   "returns partial collections, omits optional fields, and drops many events",
			FailureRate:   0,
			OmitRate:      0.25,
			DataRatio:     0.55,
			Delay:         0,
			EventOmitRate: 0.6,
			EventTypes:    []string{"*"},
		},
		"degraded": {
			Name:          "degraded",
			Description:   "partial data, omitted optional fields, frequent event drops, and deterministic 503 faults",
			FailureRate:   0.2,
			OmitRate:      0.55,
			DataRatio:     0.4,
			Delay:         220 * time.Millisecond,
			EventOmitRate: 0.8,
			EventTypes:    []string{"*"},
		},
	}
}

func matchesEventType(eventType string, candidates []string) bool {
	if len(candidates) == 0 {
		return true
	}
	for _, candidate := range candidates {
		clean := strings.TrimSpace(candidate)
		if clean == "" {
			continue
		}
		if clean == "*" || strings.EqualFold(clean, eventType) {
			return true
		}
	}
	return false
}

func eventIdentity(data any) string {
	mapped, ok := anyAsMap(data)
	if !ok {
		return strings.TrimSpace(fmt.Sprintf("%T", data))
	}

	for _, key := range []string{"id", "eventId", "requestId"} {
		if value, exists := mapped[key]; exists {
			id := strings.TrimSpace(fmt.Sprintf("%v", value))
			if id != "" {
				return key + ":" + id
			}
		}
	}

	for _, key := range []string{"asset", "driver", "vehicle", "route", "address"} {
		nested, exists := mapped[key]
		if !exists {
			continue
		}
		nestedMap, nestedOK := anyAsMap(nested)
		if !nestedOK {
			continue
		}
		id := strings.TrimSpace(fmt.Sprintf("%v", nestedMap["id"]))
		if id != "" {
			return key + ":" + id
		}
	}

	if rawData, exists := mapped["data"]; exists {
		nestedIdentity := eventIdentity(rawData)
		if nestedIdentity != "" {
			return "data:" + nestedIdentity
		}
	}

	return recordIdentity(mapped, 0)
}

func isSparseCandidate(field string) bool {
	switch field {
	case
		"notes",
		"description",
		"tags",
		"attributes",
		"externalIds",
		"contacts",
		"customHeaders",
		"eventTypes",
		"location",
		"speed",
		"stops",
		"sections",
		"fields",
		"asset",
		"driver",
		"vehicle",
		"approvalDetails",
		"assetsLocationLinkConfig",
		"assetsNearLocationLinkConfig",
		"assetsOnRouteLinkConfig",
		"auxInput1",
		"auxInput2",
		"auxInput3",
		"auxInput4",
		"auxInput5",
		"auxInput6",
		"auxInput7",
		"auxInput8",
		"auxInput9":
		return true
	default:
		return false
	}
}
