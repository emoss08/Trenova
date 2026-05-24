package helpers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Logger interface {
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type ErrorHandlerParams struct {
	fx.In

	Logger *zap.Logger
	Config *config.Config
}

type ErrorHandler struct {
	logger     Logger
	baseURI    string
	classifier *ChainClassifier
	sanitizer  *Sanitizer
}

type TimeoutResponseContext struct {
	RequestID string
	Method    string
	Path      string
	IP        string
}

func NewErrorHandler(params ErrorHandlerParams) *ErrorHandler {
	return &ErrorHandler{
		logger:     params.Logger.Named("error-handler"),
		baseURI:    params.Config.App.GetProblemTypeBaseURI(),
		classifier: NewDefaultClassifier(),
		sanitizer:  NewSanitizer(params.Config.App.Debug),
	}
}

func (h *ErrorHandler) HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if h.isClientCancellation(c, err) {
		requestID := extractRequestID(c)
		h.logger.Warn("Request canceled by client",
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("ip", c.ClientIP()),
			zap.Error(err),
		)
		c.Abort()
		return
	}

	problemType := h.classifier.Classify(err)
	validationErrors := h.sanitizer.ExtractErrors(err)
	usageStats := h.sanitizer.ExtractUsageStats(err)
	params := h.sanitizer.ExtractParams(err)
	detail := h.sanitizer.SanitizeMessage(err, problemType)
	requestID := extractRequestID(c)

	h.logError(c, err, problemType, requestID)

	problem := NewProblemBuilder(h.baseURI).
		WithType(problemType).
		WithDetail(detail).
		WithInstance(c.Request.URL.Path, requestID).
		WithTraceID(requestID).
		WithErrors(validationErrors).
		WithUsageStats(usageStats).
		WithParams(params).
		Build()

	c.Header("Content-Type", ProblemJSONContentType)
	c.JSON(problem.Status, problem)
	c.Abort()
}

func (h *ErrorHandler) WriteRequestTimeout(
	w http.ResponseWriter,
	ctx TimeoutResponseContext,
	err error,
) {
	problemType := h.classifier.Classify(err)
	h.logTimeoutError(ctx, err, problemType)

	problem := NewProblemBuilder(h.baseURI).
		WithType(problemType).
		WithDetail(h.sanitizer.SanitizeMessage(err, problemType)).
		WithInstance(ctx.Path, ctx.RequestID).
		WithTraceID(ctx.RequestID).
		Build()

	w.Header().Set("Content-Type", ProblemJSONContentType)
	w.WriteHeader(problem.Status)
	if bytes, marshalErr := sonic.Marshal(problem); marshalErr == nil {
		_, _ = w.Write(bytes)
	}
}

func (h *ErrorHandler) logError(
	c *gin.Context,
	err error,
	problemType ProblemType,
	requestID string,
) {
	info := problemType.Info()
	if !info.ShouldLog {
		return
	}

	fields := []zap.Field{
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("ip", c.ClientIP()),
		zap.Int("status", info.StatusCode),
		zap.String("problem_type", string(problemType)),
		zap.Error(err),
	}

	if c.Request.URL.RawQuery != "" {
		fields = append(fields, zap.String("query", c.Request.URL.RawQuery))
	}

	if problemType == ProblemTypeTimeout {
		h.logger.Warn("Request timed out", fields...)
	} else if info.StatusCode >= 500 {
		h.logger.Error("Server error", fields...)
	} else {
		h.logger.Warn("Client error", fields...)
	}
}

func (h *ErrorHandler) logTimeoutError(
	ctx TimeoutResponseContext,
	err error,
	problemType ProblemType,
) {
	info := problemType.Info()
	if !info.ShouldLog {
		return
	}

	fields := []zap.Field{
		zap.String("request_id", ctx.RequestID),
		zap.String("method", ctx.Method),
		zap.String("path", ctx.Path),
		zap.String("ip", ctx.IP),
		zap.Int("status", info.StatusCode),
		zap.String("problem_type", string(problemType)),
		zap.Error(err),
	}

	h.logger.Warn("Request timed out", fields...)
}

func (h *ErrorHandler) isClientCancellation(c *gin.Context, err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if !errors.Is(err, context.Canceled) {
		return false
	}
	if c.Request.Context().Err() == context.DeadlineExceeded {
		return false
	}
	return true
}

func (h *ErrorHandler) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				err := recoverToError(r)
				h.logger.Error("Panic recovered",
					zap.Any("panic", r),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)
				h.handlePanic(c, err)
			}
		}()

		c.Next()

		if len(c.Errors) > 0 {
			h.HandleError(c, c.Errors.Last().Err)
		}
	}
}

func (h *ErrorHandler) handlePanic(c *gin.Context, err error) {
	requestID := extractRequestID(c)
	problem := NewProblemBuilder(h.baseURI).
		WithType(ProblemTypeInternal).
		WithDetail(h.sanitizer.SanitizeMessage(err, ProblemTypeInternal)).
		WithInstance(c.Request.URL.Path, requestID).
		WithTraceID(requestID).
		Build()

	c.Header("Content-Type", ProblemJSONContentType)
	c.JSON(problem.Status, problem)
	c.Abort()
}

func extractRequestID(c *gin.Context) string {
	if id := c.GetHeader("X-Request-ID"); id != "" {
		return id
	}
	return c.GetString("request_id")
}

func recoverToError(r any) error {
	switch x := r.(type) {
	case string:
		return errors.New(x)
	case error:
		return x
	default:
		return fmt.Errorf("unknown panic: %v", r)
	}
}
