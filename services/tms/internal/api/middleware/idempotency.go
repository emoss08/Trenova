package middleware

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/idempotency"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	idempotencyGraphQLPath       = "/graphql"
	idempotencyLockMargin        = 5 * time.Second
	idempotencyProcessingRetryIn = "1"
)

var idempotencyRetryableCodes = map[string]struct{}{
	string(errortypes.ErrSystemError):     {},
	string(errortypes.ErrTooManyRequests): {},
	string(errortypes.ErrNotImplemented):  {},
}

type IdempotencyParams struct {
	fx.In

	Config       *config.Config
	Store        repositories.IdempotencyStore `optional:"true"`
	ErrorHandler *helpers.ErrorHandler
	Logger       *zap.Logger
}

type IdempotencyMiddleware struct {
	enabled          bool
	store            repositories.IdempotencyStore
	errorHandler     *helpers.ErrorHandler
	logger           *zap.Logger
	recordTTL        time.Duration
	lockTTL          time.Duration
	storeTimeout     time.Duration
	maxRequestBytes  int64
	maxResponseBytes int
}

func NewIdempotencyMiddleware(p IdempotencyParams) *IdempotencyMiddleware {
	cfg := config.IdempotencyConfig{}
	var requestTimeout time.Duration
	if p.Config != nil {
		cfg = p.Config.Security.Idempotency
		requestTimeout = p.Config.Server.RequestTimeout
	}

	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &IdempotencyMiddleware{
		enabled:          !cfg.Disabled && p.Store != nil,
		store:            p.Store,
		errorHandler:     p.ErrorHandler,
		logger:           logger.Named("idempotency"),
		recordTTL:        cfg.GetRecordTTL(),
		lockTTL:          max(cfg.GetLockTTL(), requestTimeout+idempotencyLockMargin),
		storeTimeout:     cfg.GetStoreTimeout(),
		maxRequestBytes:  cfg.GetMaxRequestBytes(),
		maxResponseBytes: cfg.GetMaxResponseBytes(),
	}
}

func (m *IdempotencyMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientKey := c.GetHeader(idempotency.HeaderKey)
		if m == nil || !m.enabled || clientKey == "" || !isIdempotentCandidate(c.Request.Method) {
			c.Next()
			return
		}

		if !idempotency.ValidClientKey(clientKey) {
			m.errorHandler.HandleError(c, errortypes.NewValidationError(
				idempotency.HeaderKey,
				errortypes.ErrInvalid,
				"Idempotency-Key must be 1 to {0} printable characters without spaces",
				idempotency.MaxKeyLength,
			))
			c.Abort()
			return
		}

		authCtx := authctx.GetAuthContext(c)
		if authCtx.OrganizationID.IsNil() || authCtx.PrincipalID.IsNil() {
			c.Next()
			return
		}

		body, ok := m.readBody(c)
		if !ok {
			return
		}

		scopedKey := idempotency.ScopedKey(idempotency.Scope{
			OrganizationID: authCtx.OrganizationID,
			BusinessUnitID: authCtx.BusinessUnitID,
			PrincipalType:  authCtx.PrincipalType,
			PrincipalID:    authCtx.PrincipalID,
		}, clientKey)
		fingerprint := idempotency.Fingerprint(
			c.Request.Method,
			c.Request.URL.RequestURI(),
			body,
		)
		c.Request = c.Request.WithContext(idempotency.WithKey(c.Request.Context(), scopedKey))

		m.serve(c, scopedKey, fingerprint)
	}
}

func (m *IdempotencyMiddleware) readBody(c *gin.Context) ([]byte, bool) {
	if c.Request.Body == nil || c.Request.Body == http.NoBody {
		return nil, true
	}
	if c.Request.ContentLength > m.maxRequestBytes {
		m.rejectTooLarge(c)
		return nil, false
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, m.maxRequestBytes+1))
	_ = c.Request.Body.Close()
	if err != nil {
		m.logger.Debug("failed to read idempotent request body", zap.Error(err))
		m.errorHandler.HandleError(c, errortypes.NewValidationError(
			"body",
			errortypes.ErrInvalid,
			"The request body could not be read",
		))
		c.Abort()
		return nil, false
	}
	if int64(len(body)) > m.maxRequestBytes {
		m.rejectTooLarge(c)
		return nil, false
	}

	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	return body, true
}

func (m *IdempotencyMiddleware) rejectTooLarge(c *gin.Context) {
	m.errorHandler.HandleError(c, errortypes.NewValidationError(
		idempotency.HeaderKey,
		errortypes.ErrInvalid,
		"Idempotency-Key cannot be used on a request body larger than {0} bytes",
		m.maxRequestBytes,
	))
	c.Abort()
}

func (m *IdempotencyMiddleware) serve(c *gin.Context, scopedKey, fingerprint string) {
	owner := rand.Text()
	claimCtx, cancel := context.WithTimeout(c.Request.Context(), m.storeTimeout)
	result, err := m.store.Claim(claimCtx, repositories.IdempotencyClaim{
		Key:         scopedKey,
		Fingerprint: fingerprint,
		Owner:       owner,
		LockTTL:     m.lockTTL,
	})
	cancel()
	if err != nil {
		if c.Request.Context().Err() == nil {
			m.logger.Error("idempotency claim failed; serving without replay", zap.Error(err))
		}
		c.Next()
		return
	}

	if !result.Claimed {
		m.answerExisting(c, result.Existing, fingerprint)
		return
	}

	m.runClaimed(c, scopedKey, owner)
}

func (m *IdempotencyMiddleware) answerExisting(
	c *gin.Context,
	existing *repositories.IdempotencyRecord,
	fingerprint string,
) {
	switch {
	case existing == nil:
		m.rejectInFlight(c)
	case existing.Fingerprint != fingerprint:
		m.errorHandler.HandleError(c, errortypes.NewValidationError(
			idempotency.HeaderKey,
			errortypes.ErrInvalid,
			"This Idempotency-Key was already used for a different request; send a new key for a new request",
		))
		c.Abort()
	case existing.State != repositories.IdempotencyStateCompleted:
		m.rejectInFlight(c)
	case existing.BodyOmitted:
		c.Header(idempotency.HeaderReplayed, "true")
		m.errorHandler.HandleError(c, errortypes.NewConflictError(
			"The request with this Idempotency-Key already completed with status {0}, and its response was too large to keep for replay",
			existing.Status,
		))
		c.Abort()
	default:
		c.Header(idempotency.HeaderReplayed, "true")
		if len(existing.Body) == 0 {
			c.Status(existing.Status)
			c.Writer.WriteHeaderNow()
		} else {
			c.Data(existing.Status, existing.ContentType, existing.Body)
		}
		c.Abort()
	}
}

func (m *IdempotencyMiddleware) rejectInFlight(c *gin.Context) {
	c.Header(HeaderRetryAfter, idempotencyProcessingRetryIn)
	m.errorHandler.HandleError(c, errortypes.NewConflictError(
		"A request with this Idempotency-Key is still being processed; retry once it finishes",
	))
	c.Abort()
}

func (m *IdempotencyMiddleware) runClaimed(c *gin.Context, scopedKey, owner string) {
	capture := &idempotencyCaptureWriter{ResponseWriter: c.Writer, limit: m.maxResponseBytes}
	c.Writer = capture

	finished := false
	defer func() {
		c.Writer = capture.ResponseWriter
		if !finished {
			m.release(c, scopedKey, owner)
		}
	}()

	c.Next()
	finished = true

	if m.isRetryable(c, capture) {
		m.release(c, scopedKey, owner)
		return
	}
	m.complete(c, scopedKey, owner, capture)
}

func (m *IdempotencyMiddleware) isRetryable(
	c *gin.Context,
	capture *idempotencyCaptureWriter,
) bool {
	status := capture.Status()
	if status >= http.StatusInternalServerError ||
		status == http.StatusTooManyRequests ||
		status == http.StatusRequestTimeout {
		return true
	}
	if c.Request.URL.Path == idempotencyGraphQLPath {
		return capture.overflow || graphQLResponseRetryable(capture.body.Bytes())
	}
	return false
}

func (m *IdempotencyMiddleware) complete(
	c *gin.Context,
	scopedKey, owner string,
	capture *idempotencyCaptureWriter,
) {
	completion := &repositories.IdempotencyCompletion{
		Key:         scopedKey,
		Owner:       owner,
		Status:      capture.Status(),
		ContentType: capture.Header().Get("Content-Type"),
		BodyOmitted: capture.overflow,
		TTL:         m.recordTTL,
	}
	if !capture.overflow {
		completion.Body = capture.body.Bytes()
	}

	ctx, cancel := m.detachedContext(c)
	defer cancel()
	if err := m.store.Complete(ctx, completion); err != nil {
		m.logger.Error("failed to record idempotent response", zap.Error(err))
	}
}

func (m *IdempotencyMiddleware) release(c *gin.Context, scopedKey, owner string) {
	ctx, cancel := m.detachedContext(c)
	defer cancel()
	if err := m.store.Release(ctx, scopedKey, owner); err != nil {
		m.logger.Error("failed to release idempotency key", zap.Error(err))
	}
}

func (m *IdempotencyMiddleware) detachedContext(
	c *gin.Context,
) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(c.Request.Context()), m.storeTimeout)
}

func isIdempotentCandidate(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

type graphQLResponseErrors struct {
	Errors []struct {
		Extensions struct {
			Code string `json:"code"`
		} `json:"extensions"`
	} `json:"errors"`
}

func graphQLResponseRetryable(body []byte) bool {
	if len(body) == 0 {
		return true
	}

	var parsed graphQLResponseErrors
	if err := sonic.Unmarshal(body, &parsed); err != nil {
		return true
	}
	for i := range parsed.Errors {
		code := parsed.Errors[i].Extensions.Code
		if code == "" {
			return true
		}
		if _, retryable := idempotencyRetryableCodes[code]; retryable {
			return true
		}
	}
	return false
}

type idempotencyCaptureWriter struct {
	gin.ResponseWriter
	body     bytes.Buffer
	limit    int
	overflow bool
}

func (w *idempotencyCaptureWriter) Write(data []byte) (int, error) {
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *idempotencyCaptureWriter) WriteString(data string) (int, error) {
	w.capture([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

func (w *idempotencyCaptureWriter) capture(data []byte) {
	if w.overflow {
		return
	}
	if w.body.Len()+len(data) > w.limit {
		w.overflow = true
		w.body.Reset()
		return
	}
	w.body.Write(data)
}
