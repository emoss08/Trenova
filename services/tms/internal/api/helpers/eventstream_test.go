package helpers_test

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serve runs a gin handler behind a real http.Server with the given write
// timeout, because the deadline being tested lives in the server, not the
// handler, and a recorder never enforces one.
func serve(t *testing.T, writeTimeout time.Duration, handler gin.HandlerFunc) string {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/stream", handler)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &http.Server{
		Handler:           router,
		WriteTimeout:      writeTimeout,
		ReadHeaderTimeout: time.Second,
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})

	return "http://" + listener.Addr().String() + "/stream"
}

func readAll(t *testing.T, url string) string {
	t.Helper()

	resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var body strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		body.WriteString(scanner.Text())
		body.WriteString("\n")
	}

	return body.String()
}

// http.Server.WriteTimeout is a deadline for the whole response. On an event
// stream that is a wall-clock budget for the entire answer: the connection is
// severed however healthy it is. A stream has to lift it for itself.
func TestEventStream_OutlivesTheServerWriteTimeout(t *testing.T) {
	url := serve(t, 50*time.Millisecond, func(c *gin.Context) {
		stream, err := helpers.OpenEventStream(c, helpers.EventStreamOptions{})
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		defer stream.Close()

		stream.Emit("delta", map[string]string{"text": "first"})
		time.Sleep(150 * time.Millisecond)
		stream.Emit("done", map[string]string{"text": "last"})
	})

	body := readAll(t, url)

	assert.Contains(t, body, `event: delta`)
	assert.Contains(t, body, `event: done`, "the write after the timeout still arrives")
}

// A reasoning model can sit silent for longer than a proxy's idle timeout
// before its first token. The stream says it is alive while it waits.
func TestEventStream_HeartbeatsWhileNothingIsSaid(t *testing.T) {
	url := serve(t, 0, func(c *gin.Context) {
		stream, err := helpers.OpenEventStream(c, helpers.EventStreamOptions{
			Heartbeat: 20 * time.Millisecond,
		})
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		defer stream.Close()

		time.Sleep(110 * time.Millisecond)
		stream.Emit("done", map[string]string{})
	})

	body := readAll(t, url)

	assert.GreaterOrEqual(t, strings.Count(body, ": keepalive"), 3)
	assert.Contains(t, body, "event: done")
	// A comment line is invisible to the reader's parser but must never split
	// an event in two.
	assert.NotContains(t, body, "event: done\n: keepalive")
}

func TestEventStream_StopsHeartbeatingOnceClosed(t *testing.T) {
	url := serve(t, 0, func(c *gin.Context) {
		stream, err := helpers.OpenEventStream(c, helpers.EventStreamOptions{
			Heartbeat: 10 * time.Millisecond,
		})
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		stream.Emit("done", map[string]string{})
		stream.Close()
		time.Sleep(60 * time.Millisecond)
	})

	body := readAll(t, url)

	assert.Zero(t, strings.Count(body, ": keepalive"), "nothing is written after Close")
}
