package helpers

import (
	"errors"
	"fmt"

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

	if info.StatusCode >= 500 {
		h.logger.Error("Server error", fields...)
	} else {
		h.logger.Warn("Client error", fields...)
	}
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
