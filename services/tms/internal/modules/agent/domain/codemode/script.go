package codemode

import (
	"context"

	"github.com/bytedance/sonic"
)

const (
	ToolRunScript = "agent_run_script"
	// The run-script tool's parameters a batch reads its origin back from.
	ParamScriptTitle   = "title"
	ParamScriptPurpose = "purpose"
	ParamScriptCode    = "code"
)

type ScriptError struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Traceback string `json:"traceback,omitempty"`
}

type CallStatus string

const (
	CallOK      = CallStatus("ok")
	CallRefused = CallStatus("refused")
	CallInvalid = CallStatus("invalid")
	CallLimit   = CallStatus("limit")
	CallFailed  = CallStatus("failed")
)

type Read struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	// Writes marks a tool whose call the script prepares as a change for a
	// person to approve, rather than a read it runs.
	Writes bool `json:"writes,omitempty"`
}

type Call struct {
	Seq       int            `json:"seq"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"args"`
}

type Reply struct {
	Status  CallStatus             `json:"status"`
	Data    sonic.NoCopyRawMessage `json:"data,omitempty"`
	Message string                 `json:"message,omitempty"`
	Limit   Limit                  `json:"limit,omitempty"`
}

type CallSummary struct {
	Seq       int        `json:"seq"`
	Tool      string     `json:"tool"`
	Status    CallStatus `json:"status"`
	Rows      int        `json:"rows"`
	Bytes     int        `json:"bytes"`
	BrokerMs  int64      `json:"brokerMs"`
	ReplyMs   int64      `json:"replyMs,omitempty"`
	Message   string     `json:"message,omitempty"`
	StepKey   string     `json:"-"`
	Withheld  []string   `json:"-"`
	Truncated bool       `json:"-"`
}

type Usage struct {
	Calls int   `json:"calls"`
	Rows  int   `json:"rows"`
	Bytes int   `json:"bytes"`
	Hit   Limit `json:"hit,omitempty"`
}

type Broker interface {
	Index() []Read
	Bound(limits *Limits)
	Read(ctx context.Context, call Call) Reply
	NoteReply(seq int, ms int64)
	Summary() Summary
}

type Summary struct {
	Calls    []CallSummary
	Usage    Usage
	Withheld []string
	Batch    Batch
}

// Batch is what a script prepared to change. Nothing in it has run: when the
// script ends it is proposed as one batch for a person, unless Refusal says
// why it is not.
type Batch struct {
	Changes []PreparedChange `json:"changes,omitempty"`
	Refusal string           `json:"refusal,omitempty"`
}

type PreparedChange struct {
	Intent  int    `json:"intent"`
	Tool    string `json:"tool"`
	Summary string `json:"summary,omitempty"`
}

// Completion is what the tool that ran a script answers with: whether the
// script ran to its end, the only way its prepared changes are proposed.
type Completion interface {
	ScriptCompleted() bool
}
