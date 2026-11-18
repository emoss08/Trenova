package redis

import (
	"bufio"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/cacheconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func freeAddress(t *testing.T) *net.TCPAddr {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr, ok := l.Addr().(*net.TCPAddr)
	require.True(t, ok)
	require.NoError(t, l.Close())

	return addr
}

func serveRESPLater(t *testing.T, addr *net.TCPAddr, after time.Duration) {
	t.Helper()

	opened := make(chan net.Listener, 1)
	go func() {
		time.Sleep(after)
		l, err := net.Listen("tcp", addr.String())
		if err != nil {
			close(opened)
			return
		}
		opened <- l
		for {
			conn, acceptErr := l.Accept()
			if acceptErr != nil {
				return
			}
			go answerRESP(conn)
		}
	}()

	t.Cleanup(func() {
		if l, ok := <-opened; ok {
			_ = l.Close()
		}
	})
}

func answerRESP(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		command, err := readCommand(r)
		if err != nil {
			return
		}
		reply := "+OK\r\n"
		switch command {
		case "HELLO":
			reply = "-ERR unknown command 'HELLO'\r\n"
		case "PING":
			reply = "+PONG\r\n"
		}
		if _, err = io.WriteString(conn, reply); err != nil {
			return
		}
	}
}

func readCommand(r *bufio.Reader) (string, error) {
	header, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	count, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(header, "*")))
	if err != nil {
		return "", err
	}
	var name string
	for i := range count {
		if _, err = r.ReadString('\n'); err != nil {
			return "", err
		}
		arg, readErr := r.ReadString('\n')
		if readErr != nil {
			return "", readErr
		}
		if i == 0 {
			name = strings.ToUpper(strings.TrimSpace(arg))
		}
	}
	return name, nil
}

func awaitParams(
	t *testing.T,
	addr *net.TCPAddr,
	wait time.Duration,
) (AwaitParams, *observer.ObservedLogs) {
	t.Helper()

	cache := &cacheconfig.Config{
		Host:        addr.IP.String(),
		Port:        addr.Port,
		DialTimeout: 200 * time.Millisecond,
		ReadTimeout: time.Second,
		MaxRetries:  -1,
	}
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	client, err := NewConnection(ConnectionParams{
		CacheConfig:      cache,
		MonitoringConfig: &monitoringconfig.Config{},
		Logger:           logger,
		LC:               fxtest.NewLifecycle(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	return AwaitParams{
		Client:      client,
		CacheConfig: cache,
		AppConfig:   &appconfig.Config{StartupWait: wait},
		Logger:      logger,
	}, logs
}

func TestAwaitReady_StartsWhenRedisComesUpLate(t *testing.T) {
	addr := freeAddress(t)
	p, logs := awaitParams(t, addr, time.Hour)
	serveRESPLater(t, addr, 700*time.Millisecond)

	require.NoError(t, AwaitReady(p))

	assert.Equal(t, 1, logs.FilterMessage("Waiting for Redis to answer before starting").Len())
	assert.Equal(t, 1, logs.FilterMessage("Redis answered; starting").Len())
	assert.Equal(t, 1, logs.FilterMessage("Redis connection established").Len())
}

func TestAwaitReady_FailsOnceTheStartUpWaitIsSpent(t *testing.T) {
	addr := freeAddress(t)
	p, _ := awaitParams(t, addr, 0)

	err := AwaitReady(p)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Redis did not answer within the start-up wait")
}
