package sim

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	faultKindEndpoint = "endpoint"
	faultKindWebhook  = "webhook"
)

type FaultTarget struct {
	Kind             string `json:"kind"`
	Method           string `json:"method,omitempty"`
	PathPattern      string `json:"pathPattern,omitempty"`
	WebhookEventType string `json:"webhookEventType,omitempty"`
}

type FaultMatch struct {
	Profile string `json:"profile,omitempty"`
}

type FaultEffect struct {
	StatusCode        int  `json:"statusCode,omitempty"`
	LatencyMs         int  `json:"latencyMs,omitempty"`
	Drop              bool `json:"drop,omitempty"`
	Timeout           bool `json:"timeout,omitempty"`
	TruncateJSONBytes int  `json:"truncateJsonBytes,omitempty"`
}

type FaultRule struct {
	ID      string      `json:"id"`
	Enabled bool        `json:"enabled"`
	Target  FaultTarget `json:"target"`
	Match   FaultMatch  `json:"match"`
	Effect  FaultEffect `json:"effect"`
	Rate    float64     `json:"rate"`
	SeedKey string      `json:"seedKey,omitempty"`
}

type FaultDecision struct {
	Rule FaultRule
}

type FaultEngine struct {
	mu    sync.RWMutex
	seed  string
	rules map[string]FaultRule
	seq   atomic.Uint64
}

func NewFaultEngine(seed string) *FaultEngine {
	return &FaultEngine{
		seed:  strings.TrimSpace(seed),
		rules: map[string]FaultRule{},
	}
}

func (f *FaultEngine) Snapshot() []FaultRule {
	f.mu.RLock()
	defer f.mu.RUnlock()

	rules := make([]FaultRule, 0, len(f.rules))
	for id := range f.rules {
		rule := f.rules[id]
		rules = append(rules, cloneFaultRule(&rule))
	}
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ID < rules[j].ID
	})
	return rules
}

func (f *FaultEngine) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rules = map[string]FaultRule{}
}

func (f *FaultEngine) Replace(rules []FaultRule) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	next := map[string]FaultRule{}
	for idx := range rules {
		candidate := cloneFaultRule(&rules[idx])
		if err := f.normalizeAndValidateRule(&candidate); err != nil {
			return err
		}
		if strings.TrimSpace(candidate.ID) == "" {
			candidate.ID = f.nextRuleIDLocked()
		}
		next[candidate.ID] = candidate
	}
	f.rules = next
	return nil
}

func (f *FaultEngine) Add(rule *FaultRule) (FaultRule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	candidate := cloneFaultRule(rule)
	if err := f.normalizeAndValidateRule(&candidate); err != nil {
		return FaultRule{}, err
	}
	if strings.TrimSpace(candidate.ID) == "" {
		candidate.ID = f.nextRuleIDLocked()
	}
	f.rules[candidate.ID] = candidate
	return cloneFaultRule(&candidate), nil
}

func (f *FaultEngine) Delete(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	clean := strings.TrimSpace(id)
	if clean == "" {
		return false
	}
	if _, ok := f.rules[clean]; !ok {
		return false
	}
	delete(f.rules, clean)
	return true
}

func (f *FaultEngine) EvaluateEndpoint(
	profile string,
	method string,
	path string,
	signature string,
) (FaultDecision, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	candidates := make([]faultCandidate, 0, len(f.rules))
	for id := range f.rules {
		rule := f.rules[id]
		if !rule.Enabled {
			continue
		}
		if !strings.EqualFold(rule.Target.Kind, faultKindEndpoint) {
			continue
		}
		match, targetSpecificity := endpointTargetMatch(rule.Target, method, path)
		if !match {
			continue
		}
		pm, profileSpecificity := profileMatch(rule.Match.Profile, profile)
		if !pm {
			continue
		}
		candidates = append(candidates, faultCandidate{
			rule:               rule,
			targetSpecificity:  targetSpecificity,
			profileSpecificity: profileSpecificity,
		})
	}
	return f.resolveDecision(candidates, signature)
}

func (f *FaultEngine) EvaluateWebhook(
	profile string,
	eventType string,
	signature string,
) (FaultDecision, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	candidates := make([]faultCandidate, 0, len(f.rules))
	for id := range f.rules {
		rule := f.rules[id]
		if !rule.Enabled {
			continue
		}
		if !strings.EqualFold(rule.Target.Kind, faultKindWebhook) {
			continue
		}
		match, targetSpecificity := webhookTargetMatch(rule.Target, eventType)
		if !match {
			continue
		}
		pm, profileSpecificity := profileMatch(rule.Match.Profile, profile)
		if !pm {
			continue
		}
		candidates = append(candidates, faultCandidate{
			rule:               rule,
			targetSpecificity:  targetSpecificity,
			profileSpecificity: profileSpecificity,
		})
	}
	return f.resolveDecision(candidates, signature)
}

type faultCandidate struct {
	rule               FaultRule
	targetSpecificity  int
	profileSpecificity int
}

func (f *FaultEngine) resolveDecision(
	candidates []faultCandidate,
	signature string,
) (FaultDecision, bool) {
	if len(candidates) == 0 {
		return FaultDecision{}, false
	}

	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.targetSpecificity != right.targetSpecificity {
			return left.targetSpecificity > right.targetSpecificity
		}
		if left.profileSpecificity != right.profileSpecificity {
			return left.profileSpecificity > right.profileSpecificity
		}
		return left.rule.ID < right.rule.ID
	})

	choice := candidates[0].rule
	rate := choice.Rate
	if rate >= 1 {
		return FaultDecision{Rule: cloneFaultRule(&choice)}, true
	}
	if rate <= 0 {
		return FaultDecision{}, false
	}

	fingerprint := strings.Join([]string{
		f.seed,
		choice.ID,
		choice.SeedKey,
		signature,
	}, "|")
	if deterministicScore(fingerprint) < rate {
		return FaultDecision{Rule: cloneFaultRule(&choice)}, true
	}
	return FaultDecision{}, false
}

func (f *FaultEngine) normalizeAndValidateRule(rule *FaultRule) error {
	if rule == nil {
		return ErrFaultRuleInvalid
	}

	rule.ID = strings.TrimSpace(rule.ID)
	rule.Target.Kind = strings.ToLower(strings.TrimSpace(rule.Target.Kind))
	rule.Target.Method = strings.ToUpper(strings.TrimSpace(rule.Target.Method))
	rule.Target.PathPattern = strings.TrimSpace(rule.Target.PathPattern)
	rule.Target.WebhookEventType = strings.TrimSpace(rule.Target.WebhookEventType)
	rule.Match.Profile = strings.TrimSpace(rule.Match.Profile)
	rule.SeedKey = strings.TrimSpace(rule.SeedKey)

	if !rule.Enabled {
		rule.Enabled = false
	} else {
		rule.Enabled = true
	}
	if rule.Rate == 0 {
		rule.Rate = 1
	}
	if rule.Rate < 0 || rule.Rate > 1 {
		return ErrFaultRateOutOfRange
	}
	if rule.Match.Profile == "" {
		rule.Match.Profile = "*"
	}

	switch rule.Target.Kind {
	case faultKindEndpoint:
		if rule.Target.Method == "" {
			rule.Target.Method = "*"
		}
		if rule.Target.PathPattern == "" {
			return ErrFaultTargetPathRequired
		}
	case faultKindWebhook:
		if rule.Target.WebhookEventType == "" {
			return ErrFaultTargetEventRequired
		}
	default:
		return ErrFaultTargetKindInvalid
	}

	if rule.Effect.StatusCode != 0 &&
		(rule.Effect.StatusCode < 100 || rule.Effect.StatusCode > 599) {
		return ErrFaultStatusCodeInvalid
	}
	if rule.Effect.LatencyMs < 0 {
		return ErrFaultLatencyInvalid
	}
	if rule.Effect.TruncateJSONBytes < 0 {
		return ErrFaultTruncateInvalid
	}
	return nil
}

func (f *FaultEngine) nextRuleIDLocked() string {
	next := f.seq.Add(1)
	return fmt.Sprintf("fault-%04d", next)
}

func cloneFaultRule(rule *FaultRule) FaultRule {
	if rule == nil {
		return FaultRule{}
	}
	return *rule
}

func endpointTargetMatch(
	target FaultTarget,
	method, path string,
) (matched bool, specificity int) {
	cleanMethod := strings.ToUpper(strings.TrimSpace(method))
	targetMethod := strings.ToUpper(strings.TrimSpace(target.Method))
	methodScore, methodMatched := endpointMethodScore(targetMethod, cleanMethod)
	if !methodMatched {
		return false, 0
	}

	pattern := strings.TrimSpace(target.PathPattern)
	pathScore, pathMatched := endpointPathScore(pattern, path)
	if !pathMatched {
		return false, 0
	}

	return true, methodScore*10 + pathScore
}

func webhookTargetMatch(target FaultTarget, eventType string) (matched bool, specificity int) {
	candidate := strings.TrimSpace(target.WebhookEventType)
	if candidate == "" || candidate == "*" {
		return true, 1
	}
	if strings.EqualFold(candidate, strings.TrimSpace(eventType)) {
		return true, 2
	}
	return false, 0
}

func profileMatch(candidate, profile string) (matched bool, specificity int) {
	cleanCandidate := strings.TrimSpace(candidate)
	cleanProfile := strings.TrimSpace(profile)
	if cleanCandidate == "" || cleanCandidate == "*" {
		return true, 0
	}
	if strings.EqualFold(cleanCandidate, cleanProfile) {
		return true, 1
	}
	return false, 0
}

func endpointMethodScore(targetMethod, method string) (score int, matched bool) {
	switch targetMethod {
	case "", "*":
		return 0, true
	default:
		if targetMethod == method {
			return 1, true
		}
		return 0, false
	}
}

func endpointPathScore(pattern, path string) (score int, matched bool) {
	switch {
	case pattern == "", pattern == "*":
		return 0, true
	case strings.HasSuffix(pattern, "*"):
		prefix := strings.TrimSuffix(pattern, "*")
		if !strings.HasPrefix(path, prefix) {
			return 0, false
		}
		return 1, true
	default:
		if path != pattern {
			return 0, false
		}
		return 2, true
	}
}

func deterministicScore(signature string) float64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(signature))
	value := hash.Sum64() % 10000
	return float64(value) / 10000
}
