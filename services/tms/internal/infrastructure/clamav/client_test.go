package clamav

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const eicar = `X5O!P%@AP[4\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*`

type fakeClamd struct {
	listener net.Listener
	reply    string
	received chan []byte
}

func startFakeClamd(t *testing.T, network, address string) *fakeClamd {
	t.Helper()
	listener, err := net.Listen(network, address)
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	fake := &fakeClamd{listener: listener, received: make(chan []byte, 8)}
	go fake.serve()
	return fake
}

func (f *fakeClamd) serve() {
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			return
		}
		go f.handle(conn)
	}
}

func (f *fakeClamd) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	reader := bufio.NewReader(conn)
	command, err := reader.ReadString(0)
	if err != nil {
		return
	}
	switch command {
	case "zPING\x00":
		_, _ = io.WriteString(conn, "PONG\x00")
	case "zINSTREAM\x00":
		var body bytes.Buffer
		var size [4]byte
		for {
			if _, err = io.ReadFull(reader, size[:]); err != nil {
				return
			}
			n := binary.BigEndian.Uint32(size[:])
			if n == 0 {
				break
			}
			if _, err = io.CopyN(&body, reader, int64(n)); err != nil {
				return
			}
		}
		f.received <- body.Bytes()
		reply := f.reply
		if reply == "" {
			reply = "stream: OK"
			if bytes.Contains(body.Bytes(), []byte("EICAR-STANDARD-ANTIVIRUS-TEST-FILE")) {
				reply = "stream: Win.Test.EICAR_HDB-1 FOUND"
			}
		}
		_, _ = io.WriteString(conn, reply+"\x00")
	}
}

func TestScan_StreamsTheFileAndReadsTheVerdict(t *testing.T) {
	t.Parallel()

	fake := startFakeClamd(t, "tcp", "127.0.0.1:0")
	client := New(
		config.ClamAVConfig{Address: fake.listener.Addr().String(), Timeout: 5 * time.Second},
	)

	large := bytes.Repeat([]byte("rate confirmation "), 10_000)
	result, err := client.Scan(t.Context(), large)
	require.NoError(t, err)
	assert.Equal(t, contentscanports.VerdictClean, result.Verdict)
	assert.Equal(t, large, <-fake.received)

	result, err = client.Scan(t.Context(), []byte(eicar))
	require.NoError(t, err)
	assert.Equal(t, contentscanports.VerdictInfected, result.Verdict)
	assert.Equal(t, "Win.Test.EICAR_HDB-1", result.Signature)

	require.NoError(t, client.Ping(t.Context()))
}

func TestScan_ReachesClamdOverAUnixSocket(t *testing.T) {
	t.Parallel()

	socket := filepath.Join(t.TempDir(), "clamd.sock")
	startFakeClamd(t, "unix", socket)
	client := New(config.ClamAVConfig{Address: "unix:" + socket})

	result, err := client.Scan(t.Context(), []byte("clean"))
	require.NoError(t, err)
	assert.Equal(t, contentscanports.VerdictClean, result.Verdict)
}

func TestScan_ReportsAFileItCouldNotCheckAsUnavailable(t *testing.T) {
	t.Parallel()

	unreachable := New(config.ClamAVConfig{Address: "127.0.0.1:1", Timeout: time.Second})
	_, err := unreachable.Scan(t.Context(), []byte("x"))
	require.ErrorIs(t, err, contentscanports.ErrScannerUnavailable)
	require.ErrorIs(t, unreachable.Ping(t.Context()), contentscanports.ErrScannerUnavailable)

	fake := startFakeClamd(t, "tcp", "127.0.0.1:0")
	fake.reply = "Can't allocate memory ERROR"
	client := New(config.ClamAVConfig{Address: fake.listener.Addr().String()})
	_, err = client.Scan(t.Context(), []byte("x"))
	require.ErrorIs(t, err, contentscanports.ErrScannerUnavailable)
	assert.Contains(t, err.Error(), "allocate memory")
}

func TestScan_ReportsAFileTooLargeToCheckAsTooLargeNotUnavailable(t *testing.T) {
	t.Parallel()

	fake := startFakeClamd(t, "tcp", "127.0.0.1:0")
	small := New(config.ClamAVConfig{Address: fake.listener.Addr().String(), MaxStreamBytes: 4})
	_, err := small.Scan(t.Context(), []byte(strings.Repeat("x", 5)))
	var tooLarge *contentscanports.TooLargeError
	require.ErrorAs(t, err, &tooLarge)
	assert.Equal(t, int64(5), tooLarge.Size)
	assert.Equal(t, int64(4), tooLarge.Limit)
	assert.NotErrorIs(t, err, contentscanports.ErrScannerUnavailable)

	fake.reply = "INSTREAM size limit exceeded. ERROR"
	client := New(config.ClamAVConfig{Address: fake.listener.Addr().String()})
	_, err = client.Scan(t.Context(), []byte("xyz"))
	require.ErrorAs(t, err, &tooLarge)
	assert.Equal(t, int64(3), tooLarge.Size)
	assert.Zero(t, tooLarge.Limit, "clamd does not say what its own limit is")
	assert.NotErrorIs(t, err, contentscanports.ErrScannerUnavailable)
}
