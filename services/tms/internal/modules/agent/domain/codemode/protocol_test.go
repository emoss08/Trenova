package codemode_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrameReadsOnlyFramesWithTheRunsNonce(t *testing.T) {
	t.Parallel()

	frame, ok := codemode.ParseFrame([]byte("\x1etrenova-rpc:n1:{\"seq\":1}"), "n1")
	require.True(t, ok)
	assert.Equal(t, codemode.FrameRequest, frame.Kind)
	assert.JSONEq(t, `{"seq":1}`, string(frame.Body))

	for _, line := range []string{
		"\x1etrenova-rpc:other:{\"seq\":1}",
		"trenova-rpc:n1:{}",
		"\x1etrenova-write:n1:{}",
		"\x1etrenova-rpc",
		"plain output",
	} {
		_, ok = codemode.ParseFrame([]byte(line), "n1")
		assert.Falsef(t, ok, "%q is ordinary output", line)
	}
}

func TestReplyBodyEndsWithTheTerminatorTheShimWaitsFor(t *testing.T) {
	t.Parallel()

	body, err := codemode.ReplyBody(codemode.Reply{Status: codemode.CallOK}, "n1")
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(string(body), "\n\x1en1\n"))
	assert.Equal(t, "/home/user/.trenova/rpc/7.json", codemode.ReplyPath(7))
}

func TestLineSplitterJoinsChunksAndCutsOverlongLines(t *testing.T) {
	t.Parallel()

	splitter := codemode.NewLineSplitter(8)
	var lines []string
	var cut []bool
	emit := func(line []byte, overflowed bool) {
		lines = append(lines, string(line))
		cut = append(cut, overflowed)
	}
	splitter.Write("ab", emit)
	splitter.Write("c\nde", emit)
	splitter.Write("0123456789\nlast", emit)
	splitter.Flush(emit)

	assert.Equal(t, []string{"abc", "de012345", "last"}, lines)
	assert.Equal(t, []bool{false, true, false}, cut)
}

func TestLimitsForClampsToTheCeilings(t *testing.T) {
	t.Parallel()

	limits := codemode.LimitsFor(60, 60)
	assert.Equal(t, 60*time.Second, limits.Wall)
	assert.Equal(t, 60, limits.Calls)
	assert.Equal(t, 5_000, limits.Rows)
	assert.Equal(t, 8<<20, limits.Bytes)
	assert.Equal(t, codemode.MaxConcurrentReads, limits.Concurrent)

	high := codemode.LimitsFor(10_000, 10_000)
	assert.Equal(t, 180*time.Second, high.Wall)
	assert.Equal(t, 200, high.Calls)
	low := codemode.LimitsFor(0, 0)
	assert.Equal(t, 15*time.Second, low.Wall)
	assert.Equal(t, 1, low.Calls)
	assert.Equal(t, 10*time.Millisecond, codemode.PollInterval)
}

func TestShimNamesNoTool(t *testing.T) {
	t.Parallel()

	shim := string(codemode.ShimSource)
	for _, name := range []string{"shipment_", "invoice_", "customer_", "report_", "_list(", "_get("} {
		assert.NotContainsf(t, shim, name, "the shim is generic; reads come from the index")
	}
	assert.Contains(t, string(codemode.MainSource), "/home/user/.trenova/script.py")
	assert.Contains(t, string(codemode.RunnerSource), "RLIMIT_AS")
}
