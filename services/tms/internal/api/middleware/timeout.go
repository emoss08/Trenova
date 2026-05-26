package middleware

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
)

func NewRequestTimeoutHandler(
	next http.Handler,
	cfg *config.Config,
	errorHandler *helpers.ErrorHandler,
) http.Handler {
	timeout := cfg.Server.RequestTimeout
	if timeout <= 0 {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if skipRequestTimeout(r) {
			next.ServeHTTP(w, r)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		writer := newTimeoutResponseWriter()
		done := make(chan struct{})
		panicCh := make(chan any, 1)

		go func() {
			defer close(done)
			defer func() {
				if recovered := recover(); recovered != nil {
					panicCh <- recovered
				}
			}()
			next.ServeHTTP(writer, r.WithContext(ctx))
		}()

		select {
		case <-done:
			repanicIfNeeded(panicCh)
			writer.flush(w)
		case recovered := <-panicCh:
			panic(recovered)
		case <-ctx.Done():
			writer.timeout()
			if ctx.Err() != context.DeadlineExceeded {
				return
			}

			applyTimeoutFallbackHeaders(w.Header(), cfg, r)
			errorHandler.WriteRequestTimeout(
				w,
				timeoutResponseContext(r),
				helpers.NewRequestTimeoutError(timeout),
			)
		}
	})
}

func repanicIfNeeded(panicCh <-chan any) {
	select {
	case recovered := <-panicCh:
		panic(recovered)
	default:
	}
}

func skipRequestTimeout(r *http.Request) bool {
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return true
	}

	path := r.URL.Path
	if strings.Contains(path, "/ws/") ||
		strings.Contains(path, "/websocket") ||
		strings.Contains(path, "/live") ||
		strings.Contains(path, "/stream") {
		return true
	}

	return strings.Contains(r.Header.Get("Accept"), "text/event-stream")
}

func timeoutResponseContext(r *http.Request) helpers.TimeoutResponseContext {
	return helpers.TimeoutResponseContext{
		RequestID: requestID(r),
		Method:    r.Method,
		Path:      r.URL.Path,
		IP:        clientIP(r),
	}
}

func requestID(r *http.Request) string {
	return r.Header.Get("X-Request-ID")
}

func clientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		if ip, _, ok := strings.Cut(forwardedFor, ","); ok {
			return strings.TrimSpace(ip)
		}
		return strings.TrimSpace(forwardedFor)
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func applyTimeoutFallbackHeaders(header http.Header, cfg *config.Config, r *http.Request) {
	ApplySecurityHeaders(header, cfg)
	applyTimeoutFallbackCORS(header, cfg, r)
}

func applyTimeoutFallbackCORS(header http.Header, cfg *config.Config, r *http.Request) {
	if !cfg.CorsEnabled() || header.Get("Access-Control-Allow-Origin") != "" {
		return
	}

	origin := r.Header.Get("Origin")
	if origin == "" || !originAllowed(origin, cfg.Server.CORS.AllowedOrigins) {
		return
	}

	header.Set("Access-Control-Allow-Origin", origin)
	header.Add("Vary", "Origin")
	if cfg.Server.CORS.Credentials {
		header.Set("Access-Control-Allow-Credentials", "true")
	}
	if len(cfg.Server.CORS.ExposeHeaders) > 0 {
		header.Set(
			"Access-Control-Expose-Headers",
			strings.Join(cfg.Server.CORS.ExposeHeaders, ", "),
		)
	}
}

func originAllowed(origin string, allowed []string) bool {
	for _, value := range allowed {
		if value == "*" || value == origin {
			return true
		}
	}
	return false
}

type timeoutResponseWriter struct {
	mu          sync.Mutex
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
	timedOut    bool
}

func newTimeoutResponseWriter() *timeoutResponseWriter {
	return &timeoutResponseWriter{
		header: make(http.Header),
		status: http.StatusOK,
	}
}

func (w *timeoutResponseWriter) Header() http.Header {
	return w.header
}

func (w *timeoutResponseWriter) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut || w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
}

func (w *timeoutResponseWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	if !w.wroteHeader {
		w.status = http.StatusOK
		w.wroteHeader = true
	}
	return w.body.Write(data)
}

func (w *timeoutResponseWriter) Flush() {}

func (w *timeoutResponseWriter) timeout() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.timedOut = true
}

func (w *timeoutResponseWriter) flush(dst http.ResponseWriter) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return
	}

	copyHeader(dst.Header(), w.header)
	if w.wroteHeader || w.body.Len() > 0 || w.status != http.StatusOK {
		dst.WriteHeader(w.status)
	}
	if w.body.Len() > 0 {
		_, _ = dst.Write(w.body.Bytes())
	}
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

var _ http.ResponseWriter = (*timeoutResponseWriter)(nil)
var _ http.Flusher = (*timeoutResponseWriter)(nil)
