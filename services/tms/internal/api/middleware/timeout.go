package middleware

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
)

func NewRequestTimeoutMiddleware(
	cfg *config.Config,
	errorHandler *helpers.ErrorHandler,
) gin.HandlerFunc {
	timeout := cfg.Server.RequestTimeout
	if timeout <= 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		if skipRequestTimeout(c) {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		timeoutResponseContext := helpers.TimeoutResponseContext{
			RequestID: c.GetHeader("X-Request-ID"),
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			IP:        c.ClientIP(),
		}
		if timeoutResponseContext.RequestID == "" {
			timeoutResponseContext.RequestID = c.GetString("request_id")
		}

		c.Request = c.Request.WithContext(ctx)
		writer := newTimeoutWriter(c.Writer)
		c.Writer = writer

		done := make(chan struct{})
		panicCh := make(chan any, 1)

		go func() {
			defer close(done)
			defer func() {
				if r := recover(); r != nil {
					panicCh <- r
				}
			}()
			c.Next()
		}()

		select {
		case <-done:
			select {
			case r := <-panicCh:
				panic(r)
			default:
			}
			writer.flush()
		case r := <-panicCh:
			panic(r)
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				writer.timeout()
				errorHandler.WriteRequestTimeout(
					writer.ResponseWriter,
					timeoutResponseContext,
					helpers.NewRequestTimeoutError(timeout),
				)
				writer.flushUnderlying()
				<-done
				select {
				case r := <-panicCh:
					panic(r)
				default:
				}
				return
			}
			writer.flush()
		}
	}
}

func skipRequestTimeout(c *gin.Context) bool {
	if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
		return true
	}

	path := c.Request.URL.Path
	if strings.Contains(path, "/ws/") ||
		strings.Contains(path, "/websocket") ||
		strings.Contains(path, "/live") ||
		strings.Contains(path, "/stream") {
		return true
	}

	accept := c.GetHeader("Accept")
	return strings.Contains(accept, "text/event-stream")
}

type timeoutWriter struct {
	gin.ResponseWriter

	mu          sync.Mutex
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
	timedOut    bool
}

func newTimeoutWriter(writer gin.ResponseWriter) *timeoutWriter {
	header := make(http.Header, len(writer.Header()))
	for key, values := range writer.Header() {
		header[key] = append([]string(nil), values...)
	}

	return &timeoutWriter{
		ResponseWriter: writer,
		header:         header,
		status:         http.StatusOK,
	}
}

func (w *timeoutWriter) Header() http.Header {
	return w.header
}

func (w *timeoutWriter) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut || w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
}

func (w *timeoutWriter) WriteHeaderNow() {
	w.WriteHeader(w.status)
}

func (w *timeoutWriter) Write(data []byte) (int, error) {
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

func (w *timeoutWriter) WriteString(data string) (int, error) {
	return w.Write([]byte(data))
}

func (w *timeoutWriter) Status() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status
}

func (w *timeoutWriter) Size() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.body.Len() == 0 && !w.wroteHeader {
		return -1
	}
	return w.body.Len()
}

func (w *timeoutWriter) Written() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.wroteHeader || w.body.Len() > 0
}

func (w *timeoutWriter) Flush() {}

func (w *timeoutWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.Hijack()
}

func (w *timeoutWriter) CloseNotify() <-chan bool {
	return w.ResponseWriter.CloseNotify()
}

func (w *timeoutWriter) Pusher() http.Pusher {
	return w.ResponseWriter.Pusher()
}

func (w *timeoutWriter) timeout() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.timedOut = true
}

func (w *timeoutWriter) flushUnderlying() {
	flusher, ok := w.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

func (w *timeoutWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return
	}

	dst := w.ResponseWriter.Header()
	for key, values := range w.header {
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}

	if w.wroteHeader || w.body.Len() > 0 || w.status != http.StatusOK {
		w.ResponseWriter.WriteHeader(w.status)
	}
	if w.body.Len() > 0 {
		_, _ = w.ResponseWriter.Write(w.body.Bytes())
	}
}

var _ gin.ResponseWriter = (*timeoutWriter)(nil)
