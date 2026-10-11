package sim

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

type profileContextKey struct{}

type tokenPolicy struct {
	RawToken string
	ReadOnly bool
}

type Server struct {
	cfg             *config.Config
	store           *Store
	live            *LiveSimulator
	clock           *SimClock
	scripts         *ScriptEngine
	faults          *FaultEngine
	scenarios       *ScenarioEngine
	dispatcher      *Dispatcher
	logger          *slog.Logger
	mux             *http.ServeMux
	tokenLookup     map[string]struct{}
	tokenPolicies   map[string]tokenPolicy
	requestSeq      atomic.Uint64
	requestSalt     uint64
	eventMu         sync.Mutex
	eventSentAt     map[string]time.Time
	rateLimiter     *rateLimiter
	webhookInboxMu  sync.Mutex
	webhookInbox    []Record
	webhookInboxSeq atomic.Uint64
	geofenceWindow  lazyEmissionWindow
	dvirWindow      lazyEmissionWindow
	formWindow      lazyEmissionWindow
	routeStopWindow lazyEmissionWindow
	documentWindow  lazyEmissionWindow
	geofenceEntries atomic.Int64
	geofenceExits   atomic.Int64
}

type lazyEmissionWindow struct {
	mu     sync.Mutex
	lastAt time.Time
}

func (w *lazyEmissionWindow) nextStart(
	at time.Time,
	fallbackLookback time.Duration,
	maxLookback time.Duration,
) time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()

	windowStart := w.lastAt
	if windowStart.IsZero() ||
		windowStart.After(at) ||
		at.Sub(windowStart) > maxLookback {
		windowStart = at.Add(-fallbackLookback)
	}
	w.lastAt = at
	return windowStart
}

func (w *lazyEmissionWindow) reset() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.lastAt = time.Time{}
}

func NewServer(
	cfg *config.Config,
	store *Store,
	scenarios *ScenarioEngine,
	dispatcher *Dispatcher,
	logger *slog.Logger,
) *Server {
	serverLogger := logger
	if serverLogger == nil {
		serverLogger = slog.Default()
	}

	if cfg == nil {
		defaultCfg := config.Default()
		cfg = &defaultCfg
	}

	srv := &Server{
		cfg:   cfg,
		store: store,
		live: NewLiveSimulator(store, cfg.Seed.DeterministicSeed, LiveSimulationOptions{
			FleetSize:      cfg.Simulation.FleetSize,
			TripHoursMin:   cfg.Simulation.TripHoursMin,
			TripHoursMax:   cfg.Simulation.TripHoursMax,
			EventIntensity: cfg.Simulation.EventIntensity,
			ViolationRate:  cfg.Simulation.ViolationRate,
			SpeedingRate:   cfg.Simulation.SpeedingRate,
			ScriptMode:     cfg.Simulation.ScriptMode,
		}),
		clock: NewSimClock(time.Now().UTC()),
		scripts: NewScriptEngine(
			cfg.Simulation.ScriptPath,
			cfg.Simulation.ScriptMode,
			cfg.Simulation.ScriptTimezone,
		),
		faults:       NewFaultEngine(cfg.Seed.DeterministicSeed),
		scenarios:    scenarios,
		dispatcher:   dispatcher,
		logger:       serverLogger,
		mux:          http.NewServeMux(),
		eventSentAt:  map[string]time.Time{},
		requestSalt:  uint64(time.Now().UnixNano()),
		webhookInbox: []Record{},
	}
	if cfg.RateLimits.Enabled {
		srv.rateLimiter = newRateLimiter(cfg.RateLimits.Multiplier, time.Now)
	}
	if err := srv.scripts.Reload(); err != nil {
		srv.logger.Warn("failed to load scenario scripts", slog.String("error", err.Error()))
	}
	srv.live.SetScriptEngine(srv.scripts)
	srv.live.SetClock(srv.clock)
	if srv.dispatcher != nil {
		srv.dispatcher.SetClock(srv.clock)
		srv.dispatcher.SetFaultEngine(srv.faults)
	}
	srv.tokenLookup = make(map[string]struct{}, len(cfg.Auth.Tokens))
	srv.tokenPolicies = make(map[string]tokenPolicy, len(cfg.Auth.Tokens))
	for _, tokenValue := range cfg.Auth.Tokens {
		policy := parseTokenPolicy(tokenValue)
		clean := strings.TrimSpace(policy.RawToken)
		if clean == "" {
			continue
		}
		srv.tokenLookup[clean] = struct{}{}
		srv.tokenPolicies[clean] = policy
	}
	srv.registerRoutes()
	return srv
}

func (s *Server) HTTPServer() *http.Server {
	address := net.JoinHostPort(s.cfg.Server.Host, strconv.Itoa(s.cfg.Server.Port))
	return &http.Server{
		Addr:              address,
		Handler:           s.withMiddleware(s.mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func (s *Server) registerRoutes() {
	s.registerAdminRoutes()
	s.registerAddressRoutes()
	s.registerAssetRoutes()
	s.registerDriverRoutes()
	s.registerAssignmentRoutes()
	s.registerTagRoutes()
	s.registerTrailerRoutes()
	s.registerEquipmentRoutes()
	s.registerRouteRoutes()
	s.registerDvirRoutes()
	s.registerFormRoutes()
	s.registerMessageRoutes()
	s.registerDocumentRoutes()
	s.registerComplianceRoutes()
	s.registerVehicleRoutes()
	s.registerWebhookRoutes()
	s.registerLiveShareRoutes()
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, pattern := s.mux.Handler(request)
		if isPublicPath(request.URL.Path) || isWebhookInboxDelivery(request) {
			if pattern == "" {
				s.writeRouteError(writer, request)
				return
			}
			next.ServeHTTP(writer, request)
			return
		}

		policy, ok := s.authenticateRequest(writer, request)
		if !ok {
			return
		}
		if !isControlPlanePath(request.URL.Path) &&
			!s.applyRateLimitMiddleware(writer, request, policy.RawToken, pattern) {
			return
		}
		if pattern == "" {
			s.writeRouteError(writer, request)
			return
		}
		s.serveScenarioRequest(writer, request, next)
	})
}

func isPublicPath(path string) bool {
	return path == "/_sim/health" || path == "/_sim/map"
}

func isControlPlanePath(path string) bool {
	return path == "/_sim" || strings.HasPrefix(path, "/_sim/")
}

var routeProbeMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}

func (s *Server) allowedMethods(request *http.Request) []string {
	allowed := make([]string, 0, len(routeProbeMethods))
	probe := request.WithContext(request.Context())
	for _, method := range routeProbeMethods {
		if method == request.Method {
			continue
		}
		probe.Method = method
		if _, pattern := s.mux.Handler(probe); pattern != "" {
			allowed = append(allowed, method)
		}
	}
	return allowed
}

func (s *Server) writeRouteError(writer http.ResponseWriter, request *http.Request) {
	allowed := s.allowedMethods(request)
	if len(allowed) == 0 {
		s.writeAPIError(
			writer,
			http.StatusNotFound,
			fmt.Errorf(
				"%w: no endpoint matches %s %s",
				ErrRouteNotFound,
				request.Method,
				request.URL.Path,
			),
		)
		return
	}
	writer.Header().Set("Allow", strings.Join(allowed, ", "))
	s.writeAPIError(
		writer,
		http.StatusMethodNotAllowed,
		methodNotAllowedError{Method: request.Method, Path: request.URL.Path},
	)
}

func isWebhookInboxDelivery(request *http.Request) bool {
	if request == nil {
		return false
	}
	return request.Method == http.MethodPost && request.URL.Path == "/_sim/webhooks/inbox"
}

func (s *Server) authenticateRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (tokenPolicy, bool) {
	token, err := parseBearerToken(request.Header.Get("Authorization"))
	if err != nil {
		s.writeAPIError(writer, http.StatusUnauthorized, err)
		return tokenPolicy{}, false
	}
	policy, ok := s.tokenPolicyFor(token)
	if !ok {
		s.writeAPIError(writer, http.StatusUnauthorized, ErrUnauthorized)
		return tokenPolicy{}, false
	}
	if policy.ReadOnly && !isSafeHTTPMethod(request.Method) {
		s.writeAPIError(writer, http.StatusForbidden, ErrForbidden)
		return tokenPolicy{}, false
	}
	return policy, true
}

func (s *Server) applyRateLimitMiddleware(
	writer http.ResponseWriter,
	request *http.Request,
	token string,
	pattern string,
) bool {
	if s.rateLimiter == nil {
		return true
	}

	endpointKey := ""
	rule := rateLimitRule{}
	if pattern != "" {
		endpointKey = patternRouteKey(pattern, request.Method)
		rule = endpointRateLimitRule(endpointKey, request.Method)
	}
	decision := s.rateLimiter.allow(token, endpointKey, rule)
	if decision.Allowed {
		return true
	}

	writer.Header().Set("Retry-After", formatRetryAfter(decision.RetryAfter))
	s.writeAPIError(
		writer,
		http.StatusTooManyRequests,
		s.rateLimiter.exceededError(&decision, request.Method, patternPath(pattern)),
	)
	return false
}

func patternPath(pattern string) string {
	method, path, found := strings.Cut(strings.TrimSpace(pattern), " ")
	if !found || strings.HasPrefix(method, "/") {
		return strings.TrimSpace(pattern)
	}
	return strings.TrimSpace(path)
}

func (s *Server) serveScenarioRequest(
	writer http.ResponseWriter,
	request *http.Request,
	next http.Handler,
) {
	profile := s.scenarios.ResolveProfile(request)
	signature := requestSignature(request)
	if !waitForScenarioDelay(request, s.scenarios.Delay(profile, signature)) {
		return
	}
	if s.scenarios.ShouldFail(profile, signature) {
		writer.Header().Set("Retry-After", "1")
		s.writeAPIError(
			writer,
			http.StatusServiceUnavailable,
			errors.New("simulated degraded profile failure"),
		)
		return
	}

	writer.Header().Set(HeaderProfileOverride, profile)
	ctx := context.WithValue(request.Context(), profileContextKey{}, profile)
	faultDecision, hasFault := s.faults.EvaluateEndpoint(
		profile,
		request.Method,
		request.URL.Path,
		signature,
	)
	if !hasFault {
		next.ServeHTTP(writer, request.WithContext(ctx))
		return
	}
	if s.applyEndpointFaultPre(writer, request, &faultDecision.Rule) {
		return
	}
	if faultDecision.Rule.Effect.TruncateJSONBytes <= 0 {
		next.ServeHTTP(writer, request.WithContext(ctx))
		return
	}
	s.serveWithTruncation(
		writer,
		request.WithContext(ctx),
		next,
		faultDecision.Rule.Effect.TruncateJSONBytes,
	)
}

func waitForScenarioDelay(request *http.Request, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	select {
	case <-request.Context().Done():
		timer.Stop()
		return false
	case <-timer.C:
		return true
	}
}

func (s *Server) serveWithTruncation(
	writer http.ResponseWriter,
	request *http.Request,
	next http.Handler,
	truncateLimit int,
) {
	truncatingWriter := newTruncatingResponseWriter(writer, truncateLimit)
	next.ServeHTTP(truncatingWriter, request)
	if flushErr := truncatingWriter.Flush(); flushErr != nil {
		s.logger.Warn(
			"failed to flush truncated response",
			slog.String("error", flushErr.Error()),
		)
	}
}

func (s *Server) tokenPolicyFor(token string) (tokenPolicy, bool) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return tokenPolicy{}, false
	}
	policy, ok := s.tokenPolicies[trimmed]
	return policy, ok
}

func parseBearerToken(headerValue string) (string, error) {
	trimmed := strings.TrimSpace(headerValue)
	if trimmed == "" {
		return "", ErrInvalidAuthorization
	}
	if !strings.HasPrefix(strings.ToLower(trimmed), "bearer ") {
		return "", ErrInvalidAuthorization
	}
	token := strings.TrimSpace(trimmed[len("Bearer "):])
	if token == "" {
		return "", ErrInvalidAuthorization
	}
	return token, nil
}

func parseTokenPolicy(rawToken string) tokenPolicy {
	clean := strings.TrimSpace(rawToken)
	if clean == "" {
		return tokenPolicy{}
	}

	policy := tokenPolicy{RawToken: clean}
	if strings.Contains(clean, "|") {
		parts := strings.Split(clean, "|")
		policy.RawToken = strings.TrimSpace(parts[0])
		for _, part := range parts[1:] {
			switch strings.ToLower(strings.TrimSpace(part)) {
			case "readonly", "read-only", "ro":
				policy.ReadOnly = true
			}
		}
	}

	rawLower := strings.ToLower(policy.RawToken)
	if strings.HasSuffix(rawLower, "-readonly") || strings.HasSuffix(rawLower, "-ro") {
		policy.ReadOnly = true
	}
	return policy
}

func isSafeHTTPMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func (s *Server) profileFromContext(ctx context.Context) string {
	rawProfile := ctx.Value(profileContextKey{})
	profile, ok := rawProfile.(string)
	if !ok {
		return s.scenarios.ActiveProfile()
	}
	return profile
}

func (s *Server) respondJSON(
	writer http.ResponseWriter,
	request *http.Request,
	signature string,
	payload any,
) {
	profile := s.profileFromContext(request.Context())
	applied := s.scenarios.Apply(profile, signature, payload)
	if err := writeJSON(writer, http.StatusOK, applied); err != nil {
		s.logger.Error("failed to write JSON response", slog.String("error", err.Error()))
	}
}

func (s *Server) dispatchEvent(request *http.Request, eventType string, data any) {
	if s.dispatcher == nil {
		return
	}

	profile := s.profileFromContext(request.Context())
	if s.scenarios != nil && s.scenarios.ShouldOmitEvent(profile, eventType, data) {
		s.logger.Debug(
			"simulated event omission",
			slog.String("profile", profile),
			slog.String("eventType", eventType),
		)
		return
	}

	if err := s.dispatcher.Dispatch(profile, eventType, data); err != nil {
		s.logger.Warn(
			"failed to dispatch webhook event",
			slog.String("eventType", eventType),
			slog.String("error", err.Error()),
		)
	}
}

func (s *Server) dispatchEventOnce(
	request *http.Request,
	uniqueKey string,
	eventType string,
	data any,
) bool {
	cleanKey := strings.TrimSpace(uniqueKey)
	if cleanKey == "" {
		s.dispatchEvent(request, eventType, data)
		return true
	}

	now := s.simNow()
	ttl := 72 * time.Hour
	s.eventMu.Lock()
	if sentAt, exists := s.eventSentAt[cleanKey]; exists && now.Sub(sentAt) <= ttl {
		s.eventMu.Unlock()
		return false
	}
	s.eventSentAt[cleanKey] = now
	for key, value := range s.eventSentAt {
		if now.Sub(value) > ttl {
			delete(s.eventSentAt, key)
		}
	}
	s.eventMu.Unlock()

	s.dispatchEvent(request, eventType, data)
	return true
}

func (s *Server) appendWebhookInbox(record Record) {
	if len(record) == 0 {
		return
	}

	s.webhookInboxMu.Lock()
	defer s.webhookInboxMu.Unlock()

	nextID := s.webhookInboxSeq.Add(1)
	entry := cloneRecord(record)
	if strings.TrimSpace(recordID(entry)) == "" {
		entry["id"] = fmt.Sprintf("inbox-%08d", nextID)
	}
	s.webhookInbox = append(s.webhookInbox, entry)

	const maxInboxRecords = 500
	if len(s.webhookInbox) > maxInboxRecords {
		s.webhookInbox = append([]Record{}, s.webhookInbox[len(s.webhookInbox)-maxInboxRecords:]...)
	}
}

func (s *Server) listWebhookInbox(limit int) []Record {
	if limit <= 0 {
		limit = 50
	}

	s.webhookInboxMu.Lock()
	defer s.webhookInboxMu.Unlock()

	size := len(s.webhookInbox)
	if size == 0 {
		return []Record{}
	}
	if limit > size {
		limit = size
	}

	out := make([]Record, 0, limit)
	for idx := size - 1; idx >= 0 && len(out) < limit; idx-- {
		out = append(out, cloneRecord(s.webhookInbox[idx]))
	}
	return out
}

func (s *Server) clearWebhookInbox() {
	s.webhookInboxMu.Lock()
	defer s.webhookInboxMu.Unlock()

	s.webhookInbox = []Record{}
	s.webhookInboxSeq.Store(0)
}

func (s *Server) writeAPIError(writer http.ResponseWriter, status int, err error) {
	body := apiErrorBody{
		Message:   apiErrorMessage(err),
		RequestID: s.nextRequestID(),
	}
	if encodeErr := writeJSON(writer, status, body); encodeErr != nil {
		s.logger.Error(
			"failed to write API error response",
			slog.String("error", encodeErr.Error()),
		)
	}
}

func (s *Server) simNow() time.Time {
	if s.clock == nil {
		return time.Now().UTC()
	}
	return s.clock.Now()
}

func (s *Server) applyEndpointFaultPre(
	writer http.ResponseWriter,
	request *http.Request,
	rule *FaultRule,
) bool {
	if rule == nil {
		return false
	}

	if rule.Effect.LatencyMs > 0 {
		delay := time.Duration(rule.Effect.LatencyMs) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-request.Context().Done():
			timer.Stop()
			return true
		case <-timer.C:
		}
	}

	if rule.Effect.Timeout {
		timer := time.NewTimer(6 * time.Second)
		select {
		case <-request.Context().Done():
			timer.Stop()
		case <-timer.C:
		}
		s.writeAPIError(writer, http.StatusServiceUnavailable, ErrFaultRuleInvalid)
		return true
	}

	if rule.Effect.Drop {
		writer.WriteHeader(http.StatusNoContent)
		return true
	}

	if rule.Effect.StatusCode == http.StatusTooManyRequests {
		s.writeInjectedRateLimit(writer, request)
		return true
	}

	if rule.Effect.StatusCode != 0 {
		s.writeAPIError(
			writer,
			rule.Effect.StatusCode,
			fmt.Errorf("%w: %s", ErrFaultRuleInvalid, rule.ID),
		)
		return true
	}

	return false
}

func (s *Server) writeInjectedRateLimit(writer http.ResponseWriter, request *http.Request) {
	_, pattern := s.mux.Handler(request)
	endpointKey := patternRouteKey(pattern, request.Method)
	rule := endpointRateLimitRule(endpointKey, request.Method)
	limiter := s.rateLimiter
	if limiter == nil {
		limiter = newRateLimiter(s.cfg.RateLimits.Multiplier, time.Now)
	}
	requests := limiter.effectiveRequests(rule)
	decision := rateLimitDecision{
		RetryAfter: time.Duration(float64(rule.Per) / requests),
		Scope:      rateLimitScopeEndpoint,
		Rule:       rule,
	}
	writer.Header().Set("Retry-After", formatRetryAfter(decision.RetryAfter))
	s.writeAPIError(
		writer,
		http.StatusTooManyRequests,
		limiter.exceededError(&decision, request.Method, patternPath(pattern)),
	)
}

type methodNotAllowedError struct {
	Method string
	Path   string
}

func (e methodNotAllowedError) Error() string {
	return e.Method + " not allowed on " + e.Path
}

func (e methodNotAllowedError) Unwrap() error {
	return ErrMethodNotAllowed
}

func apiErrorMessage(err error) string {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "An error has occurred."
	}
	first, size := utf8.DecodeRuneInString(message)
	message = string(unicode.ToUpper(first)) + message[size:]
	switch message[len(message)-1] {
	case '.', '!', '?':
		return message
	default:
		return message + "."
	}
}

type apiErrorBody struct {
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

func (s *Server) nextRequestID() string {
	mixed := splitMix64(s.requestSalt + s.requestSeq.Add(1))
	return fmt.Sprintf("%08x", uint32(mixed>>32))
}

func splitMix64(value uint64) uint64 {
	value += 0x9e3779b97f4a7c15
	value = (value ^ (value >> 30)) * 0xbf58476d1ce4e5b9
	value = (value ^ (value >> 27)) * 0x94d049bb133111eb
	return value ^ (value >> 31)
}

func writeJSON(writer http.ResponseWriter, status int, payload any) error {
	encoded, err := sonic.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, err = writer.Write(encoded)
	if err != nil {
		return fmt.Errorf("write response body: %w", err)
	}
	return nil
}

func readRecordBody(request *http.Request) (Record, error) {
	if request.Body == nil {
		return nil, ErrInvalidBody
	}

	limited := io.LimitReader(request.Body, 2*1024*1024)
	rawBody, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	if len(rawBody) == 0 {
		return nil, ErrInvalidBody
	}

	record := Record{}
	if err = sonic.Unmarshal(rawBody, &record); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidBody, err)
	}
	return record, nil
}

func readBody(request *http.Request) (map[string]any, error) {
	if request.Body == nil {
		return nil, ErrInvalidBody
	}

	limited := io.LimitReader(request.Body, 2*1024*1024)
	rawBody, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	if len(rawBody) == 0 {
		return nil, ErrInvalidBody
	}

	body := map[string]any{}
	if err = sonic.Unmarshal(rawBody, &body); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidBody, err)
	}
	return body, nil
}

func requestSignature(request *http.Request) string {
	return strings.TrimSpace(request.Method) + "|" + request.URL.Path + "|" + request.URL.RawQuery
}

func queryValue(request *http.Request, name string) string {
	return strings.TrimSpace(request.URL.Query().Get(name))
}

func queryID(request *http.Request) (string, error) {
	id := queryValue(request, "id")
	if id == "" {
		return "", ErrQueryIDRequired
	}
	return id, nil
}

func pathID(request *http.Request) (string, error) {
	id := strings.TrimSpace(request.PathValue("id"))
	if id == "" {
		return "", ErrPathIDRequired
	}
	return id, nil
}

func idsFromQuery(values url.Values, names ...string) []string {
	for _, name := range names {
		raw := strings.TrimSpace(values.Get(name))
		if raw == "" {
			continue
		}
		return splitCSV(raw)
	}
	return []string{}
}

func splitCSV(value string) []string {
	items := strings.Split(value, ",")
	out := make([]string, 0, len(items))
	for _, item := range items {
		clean := strings.TrimSpace(item)
		if clean != "" {
			out = append(out, clean)
		}
	}
	return out
}

func parseLimit(values url.Values, maxValue int) int {
	rawLimit := strings.TrimSpace(values.Get("limit"))
	if rawLimit == "" {
		return maxValue
	}

	limit, err := strconv.Atoi(rawLimit)
	if err != nil || limit <= 0 {
		return maxValue
	}
	if limit > maxValue {
		return maxValue
	}
	return limit
}

func filterByIDs(records []Record, ids []string) []Record {
	if len(ids) == 0 {
		return cloneRecords(records)
	}

	lookup := map[string]struct{}{}
	for _, id := range ids {
		lookup[strings.TrimSpace(id)] = struct{}{}
	}

	filtered := make([]Record, 0, len(records))
	for _, record := range records {
		id := recordID(record)
		if _, ok := lookup[id]; ok {
			filtered = append(filtered, cloneRecord(record))
		}
	}
	return filtered
}

func filterByNestedIDs(records []Record, ids []string, keys ...string) []Record {
	if len(ids) == 0 {
		return cloneRecords(records)
	}

	lookup := map[string]struct{}{}
	for _, id := range ids {
		clean := strings.TrimSpace(id)
		if clean != "" {
			lookup[clean] = struct{}{}
		}
	}

	filtered := make([]Record, 0, len(records))
	for _, record := range records {
		value := nestedString(record, keys...)
		if _, ok := lookup[value]; ok {
			filtered = append(filtered, cloneRecord(record))
		}
	}
	return filtered
}

func nestedString(record Record, keys ...string) string {
	if len(keys) == 0 {
		return ""
	}

	var current any = record
	for _, key := range keys {
		mapped, ok := anyAsMap(current)
		if !ok {
			return ""
		}
		next, ok := mapped[key]
		if !ok {
			return ""
		}
		current = next
	}

	value, ok := current.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func anyAsMap(raw any) (map[string]any, bool) {
	switch typed := raw.(type) {
	case map[string]any:
		return typed, true
	case Record:
		return map[string]any(typed), true
	default:
		return nil, false
	}
}

func recordsAsAny(records []Record) []any {
	data := make([]any, 0, len(records))
	for _, record := range records {
		data = append(data, cloneRecord(record))
	}
	return data
}

type truncatingResponseWriter struct {
	underlying http.ResponseWriter
	header     http.Header
	statusCode int
	body       bytes.Buffer
	limit      int
}

func newTruncatingResponseWriter(
	writer http.ResponseWriter,
	limit int,
) *truncatingResponseWriter {
	return &truncatingResponseWriter{
		underlying: writer,
		header:     make(http.Header),
		statusCode: http.StatusOK,
		limit:      limit,
	}
}

func (w *truncatingResponseWriter) Header() http.Header {
	return w.header
}

func (w *truncatingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *truncatingResponseWriter) Write(payload []byte) (int, error) {
	return w.body.Write(payload)
}

func (w *truncatingResponseWriter) Flush() error {
	for key, values := range w.header {
		w.underlying.Header()[key] = append([]string{}, values...)
	}
	w.underlying.WriteHeader(w.statusCode)

	responseBody := w.body.Bytes()
	if w.limit > 0 && len(responseBody) > w.limit {
		responseBody = responseBody[:w.limit]
	}
	_, err := w.underlying.Write(responseBody)
	return err
}
