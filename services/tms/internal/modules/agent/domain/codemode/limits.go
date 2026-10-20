package codemode

import "time"

const (
	PollInterval       = 10 * time.Millisecond
	MaxConcurrentReads = 8
	ScriptsPerTurn     = 3

	MaxCodeChars  = 20_000
	MaxInputKeys  = 32
	MaxInputBytes = 16 << 10

	DefaultRows       = 5_000
	DefaultBytes      = 8 << 20
	DefaultCallBytes  = 1 << 20
	DefaultOutput     = 8_000
	DefaultResult     = 64 << 10
	DefaultMemory     = 1 << 30
	DefaultFrameBytes = 64 << 10

	MinSeconds = 15
	MaxSeconds = 180
	MinCalls   = 1
	MaxCalls   = 200
)

type Limits struct {
	Wall        time.Duration
	Calls       int
	Rows        int
	Bytes       int
	CallBytes   int
	OutputChars int
	ResultBytes int
	MemoryBytes int64
	FrameBytes  int
	Concurrent  int
}

func LimitsFor(seconds, calls int) Limits {
	seconds = min(max(seconds, MinSeconds), MaxSeconds)
	calls = min(max(calls, MinCalls), MaxCalls)

	return Limits{
		Wall:        time.Duration(seconds) * time.Second,
		Calls:       calls,
		Rows:        DefaultRows,
		Bytes:       DefaultBytes,
		CallBytes:   DefaultCallBytes,
		OutputChars: DefaultOutput,
		ResultBytes: DefaultResult,
		MemoryBytes: DefaultMemory,
		FrameBytes:  DefaultFrameBytes,
		Concurrent:  MaxConcurrentReads,
	}
}

type Limit string

const (
	LimitNone   = Limit("")
	LimitWall   = Limit("wall")
	LimitCalls  = Limit("calls")
	LimitRows   = Limit("rows")
	LimitBytes  = Limit("bytes")
	LimitCall   = Limit("call_bytes")
	LimitOutput = Limit("output")
	LimitResult = Limit("result")
	LimitMemory = Limit("memory")
	LimitCPU    = Limit("cpu")
	LimitFrame  = Limit("frame")
)
