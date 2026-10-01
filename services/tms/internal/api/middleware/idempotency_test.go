package middleware

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/idempotency"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type memoryIdempotencyEntry struct {
	record repositories.IdempotencyRecord
	owner  string
}

type memoryIdempotencyStore struct {
	mu       sync.Mutex
	entries  map[string]*memoryIdempotencyEntry
	claimErr error
	lockTTLs []time.Duration
}

func newMemoryIdempotencyStore() *memoryIdempotencyStore {
	return &memoryIdempotencyStore{entries: make(map[string]*memoryIdempotencyEntry)}
}

func (s *memoryIdempotencyStore) Claim(
	_ context.Context,
	claim repositories.IdempotencyClaim,
) (repositories.IdempotencyClaimResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.claimErr != nil {
		return repositories.IdempotencyClaimResult{}, s.claimErr
	}
	s.lockTTLs = append(s.lockTTLs, claim.LockTTL)
	if entry, ok := s.entries[claim.Key]; ok {
		record := entry.record
		return repositories.IdempotencyClaimResult{Existing: &record}, nil
	}
	s.entries[claim.Key] = &memoryIdempotencyEntry{
		record: repositories.IdempotencyRecord{
			State:       repositories.IdempotencyStateProcessing,
			Fingerprint: claim.Fingerprint,
		},
		owner: claim.Owner,
	}
	return repositories.IdempotencyClaimResult{Claimed: true}, nil
}

func (s *memoryIdempotencyStore) Complete(
	_ context.Context,
	completion *repositories.IdempotencyCompletion,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[completion.Key]
	if !ok || entry.owner != completion.Owner {
		return repositories.ErrIdempotencyClaimLost
	}
	entry.owner = ""
	entry.record.State = repositories.IdempotencyStateCompleted
	entry.record.Status = completion.Status
	entry.record.ContentType = completion.ContentType
	entry.record.Body = append([]byte(nil), completion.Body...)
	entry.record.BodyOmitted = completion.BodyOmitted
	return nil
}

func (s *memoryIdempotencyStore) Release(_ context.Context, key, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry, ok := s.entries[key]; ok && entry.owner == owner {
		delete(s.entries, key)
	}
	return nil
}

func (s *memoryIdempotencyStore) size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

type idempotencyHarness struct {
	router *gin.Engine
	store  *memoryIdempotencyStore
	calls  int
	keys   []string
}

type idempotencyHarnessOptions struct {
	cfg     config.IdempotencyConfig
	server  config.ServerConfig
	handler func(c *gin.Context, calls int)
	userID  pulid.ID
}

func newIdempotencyHarness(t *testing.T, opts idempotencyHarnessOptions) *idempotencyHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		App:      config.AppConfig{Debug: true},
		Server:   opts.server,
		Security: config.SecurityConfig{Idempotency: opts.cfg},
	}
	h := &idempotencyHarness{store: newMemoryIdempotencyStore()}
	mw := NewIdempotencyMiddleware(IdempotencyParams{
		Config: cfg,
		Store:  h.store,
		ErrorHandler: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: cfg,
		}),
		Logger: zap.NewNop(),
	})

	userID := opts.userID
	if userID.IsNil() {
		userID = pulid.MustNew("usr_")
	}
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	handler := opts.handler
	if handler == nil {
		handler = func(c *gin.Context, calls int) {
			c.JSON(http.StatusCreated, gin.H{"call": calls})
		}
	}

	h.router = gin.New()
	h.router.Use(func(c *gin.Context) {
		authctx.SetAuthContext(c, userID, buID, orgID)
		c.Next()
	})
	h.router.Use(mw.Handle())
	serve := func(c *gin.Context) {
		h.calls++
		key, _ := idempotency.KeyFrom(c.Request.Context())
		h.keys = append(h.keys, key)
		handler(c, h.calls)
	}
	h.router.POST("/api/v1/shipments/", serve)
	h.router.GET("/api/v1/shipments/", serve)
	h.router.POST("/graphql", serve)
	return h
}

func (h *idempotencyHarness) do(method, path, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set(idempotency.HeaderKey, key)
	}
	rec := httptest.NewRecorder()
	h.router.ServeHTTP(rec, req)
	return rec
}

func TestIdempotencyReplaysTheFirstResponse(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{})

	first := h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{"bol":"A"}`)
	second := h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{"bol":"A"}`)

	require.Equal(t, http.StatusCreated, first.Code)
	assert.Equal(t, http.StatusCreated, second.Code)
	assert.JSONEq(t, `{"call":1}`, second.Body.String())
	assert.Equal(t, "true", second.Header().Get(idempotency.HeaderReplayed))
	assert.Empty(t, first.Header().Get(idempotency.HeaderReplayed))
	assert.Contains(t, second.Header().Get("Content-Type"), "application/json")
	assert.Equal(t, 1, h.calls)
	require.Len(t, h.keys, 1)
	assert.Len(t, h.keys[0], idempotency.ScopedKeyLen)
}

func TestIdempotencyPassesRequestsWithoutAKey(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{})

	h.do(http.MethodPost, "/api/v1/shipments/", "", `{}`)
	h.do(http.MethodPost, "/api/v1/shipments/", "", `{}`)

	assert.Equal(t, 2, h.calls)
	assert.Equal(t, []string{"", ""}, h.keys)
	assert.Zero(t, h.store.size())
}

func TestIdempotencyIgnoresSafeMethods(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{})

	h.do(http.MethodGet, "/api/v1/shipments/", "key-1", "")
	h.do(http.MethodGet, "/api/v1/shipments/", "key-1", "")

	assert.Equal(t, 2, h.calls)
	assert.Zero(t, h.store.size())
}

func TestIdempotencyRejectsAKeyReusedForADifferentRequest(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{})

	h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{"bol":"A"}`)
	rec := h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{"bol":"B"}`)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "already used for a different request")
	assert.Equal(t, 1, h.calls)
}

func TestIdempotencyRejectsAnInvalidKey(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{})

	for _, key := range []string{"has space", strings.Repeat("k", idempotency.MaxKeyLength+1)} {
		rec := h.do(http.MethodPost, "/api/v1/shipments/", key, `{}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code, key)
	}
	assert.Zero(t, h.calls)
}

func TestIdempotencyAnswersConflictWhileTheFirstRequestRuns(t *testing.T) {
	var h *idempotencyHarness
	var inner *httptest.ResponseRecorder
	h = newIdempotencyHarness(t, idempotencyHarnessOptions{
		handler: func(c *gin.Context, calls int) {
			if calls == 1 {
				inner = h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)
			}
			c.JSON(http.StatusCreated, gin.H{"call": calls})
		},
	})

	first := h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)

	require.Equal(t, http.StatusCreated, first.Code)
	require.NotNil(t, inner)
	assert.Equal(t, http.StatusConflict, inner.Code)
	assert.Equal(t, idempotencyProcessingRetryIn, inner.Header().Get(HeaderRetryAfter))
	assert.Equal(t, 1, h.calls)
}

func TestIdempotencyReleasesTheKeyAfterAServerError(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{
		handler: func(c *gin.Context, calls int) {
			if calls == 1 {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "down"})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"call": calls})
		},
	})

	first := h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)
	second := h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)

	assert.Equal(t, http.StatusServiceUnavailable, first.Code)
	assert.Equal(t, http.StatusCreated, second.Code)
	assert.Empty(t, second.Header().Get(idempotency.HeaderReplayed))
	assert.Equal(t, 2, h.calls)
}

func TestIdempotencyKeepsAClientError(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{
		handler: func(c *gin.Context, calls int) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"call": calls})
		},
	})

	h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)
	second := h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)

	assert.Equal(t, http.StatusUnprocessableEntity, second.Code)
	assert.Equal(t, "true", second.Header().Get(idempotency.HeaderReplayed))
	assert.Equal(t, 1, h.calls)
}

func TestIdempotencyReleasesTheKeyWhenTheHandlerPanics(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{
		handler: func(c *gin.Context, calls int) {
			if calls == 1 {
				panic("boom")
			}
			c.JSON(http.StatusCreated, gin.H{"call": calls})
		},
	})
	h.router.Use(gin.Recovery())

	assert.Panics(t, func() {
		h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)
	})
	second := h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)

	assert.Equal(t, http.StatusCreated, second.Code)
	assert.Equal(t, 2, h.calls)
}

func TestIdempotencyGraphQLSystemErrorIsRetryable(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{
		handler: func(c *gin.Context, calls int) {
			if calls == 1 {
				c.Data(http.StatusOK, "application/json",
					[]byte(`{"errors":[{"message":"x","extensions":{"code":"SYSTEM_ERROR"}}],"data":null}`))
				return
			}
			c.Data(http.StatusOK, "application/json", []byte(`{"data":{"createShipment":{"id":"shp_1"}}}`))
		},
	})

	h.do(http.MethodPost, "/graphql", "key-1", `{"extensions":{}}`)
	second := h.do(http.MethodPost, "/graphql", "key-1", `{"extensions":{}}`)
	third := h.do(http.MethodPost, "/graphql", "key-1", `{"extensions":{}}`)

	assert.Equal(t, 2, h.calls)
	assert.Empty(t, second.Header().Get(idempotency.HeaderReplayed))
	assert.Equal(t, "true", third.Header().Get(idempotency.HeaderReplayed))
	assert.JSONEq(t, `{"data":{"createShipment":{"id":"shp_1"}}}`, third.Body.String())
}

func TestIdempotencyGraphQLValidationErrorIsKept(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{
		handler: func(c *gin.Context, _ int) {
			c.Data(http.StatusOK, "application/json",
				[]byte(`{"errors":[{"message":"x","extensions":{"code":"INVALID"}}],"data":null}`))
		},
	})

	h.do(http.MethodPost, "/graphql", "key-1", `{}`)
	second := h.do(http.MethodPost, "/graphql", "key-1", `{}`)

	assert.Equal(t, 1, h.calls)
	assert.Equal(t, "true", second.Header().Get(idempotency.HeaderReplayed))
}

func TestIdempotencyOversizedResponseIsNotReplayed(t *testing.T) {
	large := strings.Repeat("x", 2048)
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{
		cfg: config.IdempotencyConfig{MaxResponseBytes: 1024},
		handler: func(c *gin.Context, _ int) {
			c.String(http.StatusCreated, large)
		},
	})

	first := h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)
	second := h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)

	assert.Equal(t, large, first.Body.String())
	assert.Equal(t, http.StatusConflict, second.Code)
	assert.Equal(t, "true", second.Header().Get(idempotency.HeaderReplayed))
	assert.Equal(t, 1, h.calls)
}

func TestIdempotencyRejectsAnOversizedRequestBody(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{
		cfg: config.IdempotencyConfig{MaxRequestBytes: 1024},
	})

	rec := h.do(http.MethodPost, "/api/v1/shipments/", "key-1", strings.Repeat("x", 2048))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Zero(t, h.calls)
}

func TestIdempotencyHandlerStillReadsTheBody(t *testing.T) {
	var seen string
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{
		handler: func(c *gin.Context, _ int) {
			raw, err := io.ReadAll(c.Request.Body)
			require.NoError(t, err)
			seen = string(raw)
			c.Status(http.StatusNoContent)
		},
	})

	h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{"bol":"A"}`)
	second := h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{"bol":"A"}`)

	assert.JSONEq(t, `{"bol":"A"}`, seen)
	assert.Equal(t, http.StatusNoContent, second.Code)
	assert.Equal(t, 1, h.calls)
}

func TestIdempotencyFailsOpenWhenTheStoreIsDown(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{})
	h.store.claimErr = errors.New("redis down")

	h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)
	h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)

	assert.Equal(t, 2, h.calls)
	require.Len(t, h.keys, 2)
	assert.Equal(t, h.keys[0], h.keys[1])
	assert.NotEmpty(t, h.keys[0])
}

func TestIdempotencyLockOutlastsTheRequestTimeout(t *testing.T) {
	h := newIdempotencyHarness(t, idempotencyHarnessOptions{
		cfg:    config.IdempotencyConfig{LockTTL: time.Second},
		server: config.ServerConfig{RequestTimeout: time.Minute},
	})

	h.do(http.MethodPost, "/api/v1/shipments/", "key-1", `{}`)

	require.Len(t, h.store.lockTTLs, 1)
	assert.Equal(t, time.Minute+idempotencyLockMargin, h.store.lockTTLs[0])
}

func TestIdempotencyScopesKeysToThePrincipal(t *testing.T) {
	scope := idempotency.Scope{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
	}
	other := scope
	other.PrincipalID = pulid.MustNew("usr_")

	assert.NotEqual(t, idempotency.ScopedKey(scope, "k"), idempotency.ScopedKey(other, "k"))
	assert.Equal(t, idempotency.ScopedKey(scope, "k"), idempotency.ScopedKey(scope, "k"))
}
