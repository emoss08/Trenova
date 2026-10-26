package scriptvm

import (
	"fmt"

	"github.com/bytedance/sonic"
)

const (
	opStart    = "start"
	opResume   = "resume"
	stepCall   = "call"
	stepDone   = "done"
	emptyIndex = "[]"
	emptyInput = "{}"
	emptyList  = "[]"
)

type wireClock struct {
	UnixSeconds   int64  `json:"unixSeconds"`
	OffsetSeconds int    `json:"offsetSeconds"`
	Zone          string `json:"zone,omitempty"`
}

type wireLimits struct {
	OutputChars int   `json:"outputChars"`
	MemoryBytes int   `json:"memoryBytes"`
	CPUMs       int64 `json:"cpuMs"`
	Recursion   int   `json:"recursion"`
	Calls       int   `json:"calls"`
}

type wireStart struct {
	Op       string     `json:"op"`
	Code     string     `json:"code"`
	Index    string     `json:"index"`
	Inputs   string     `json:"inputs"`
	Datasets string     `json:"datasets"`
	Clock    wireClock  `json:"clock"`
	Limits   wireLimits `json:"limits"`
}

type wireResume struct {
	Op    string `json:"op"`
	Reply string `json:"reply"`
}

type wireStep struct {
	Step            string       `json:"step"`
	Function        Function     `json:"function"`
	Payload         string       `json:"payload"`
	Status          Status       `json:"status"`
	Limit           Limit        `json:"limit"`
	Error           *ScriptError `json:"error"`
	Output          string       `json:"output"`
	OutputTruncated bool         `json:"outputTruncated"`
}

func (s *wireStep) outcome() *Outcome {
	return &Outcome{
		Status:          s.Status,
		Limit:           s.Limit,
		Error:           s.Error,
		Output:          s.Output,
		OutputTruncated: s.OutputTruncated,
	}
}

func startRequest(program *Program) ([]byte, error) {
	limits := program.Limits.withDefaults()
	request := wireStart{
		Op:       opStart,
		Code:     program.Code,
		Index:    rawOr(program.Index, emptyIndex),
		Inputs:   rawOr(program.Inputs, emptyInput),
		Datasets: rawOr(program.Datasets, emptyList),
		Clock:    clockOf(program.Clock),
		Limits: wireLimits{
			OutputChars: limits.OutputChars,
			MemoryBytes: limits.MemoryBytes,
			CPUMs:       limits.CPU.Milliseconds(),
			Recursion:   limits.Recursion,
			Calls:       limits.Calls,
		},
	}
	encoded, err := sonic.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("scriptvm: encode the script: %w", err)
	}
	if len(encoded) > MaxRequestBytes {
		return nil, fmt.Errorf("scriptvm: the script's request is %d bytes", len(encoded))
	}
	return encoded, nil
}

func resumeRequest(reply []byte) ([]byte, error) {
	encoded, err := sonic.Marshal(wireResume{Op: opResume, Reply: string(reply)})
	if err != nil {
		return nil, fmt.Errorf("scriptvm: encode the host's answer: %w", err)
	}
	if len(encoded) > MaxRequestBytes {
		return nil, fmt.Errorf("scriptvm: the host's answer is %d bytes", len(encoded))
	}
	return encoded, nil
}

func clockOf(clock Clock) wireClock {
	now := clock.Now
	if now.IsZero() {
		now = timeNow()
	}
	if clock.Location != nil {
		now = now.In(clock.Location)
	} else {
		now = now.UTC()
	}
	zone, offset := now.Zone()
	return wireClock{UnixSeconds: now.Unix(), OffsetSeconds: offset, Zone: zone}
}

func rawOr(raw []byte, fallback string) string {
	if len(raw) == 0 {
		return fallback
	}
	return string(raw)
}

func decodeStep(raw []byte) (*wireStep, error) {
	var step wireStep
	if err := sonic.Unmarshal(raw, &step); err != nil {
		return nil, fmt.Errorf("scriptvm: decode the module's step: %w", err)
	}
	if step.Step != stepCall && step.Step != stepDone {
		return nil, fmt.Errorf("scriptvm: the module answered an unknown step %q", step.Step)
	}
	return &step, nil
}
