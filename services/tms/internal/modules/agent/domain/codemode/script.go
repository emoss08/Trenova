package codemode

import (
	"context"

	"github.com/bytedance/sonic"
)

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
}

type Call struct {
	Seq       int            `json:"seq"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"args"`
}

type Reply struct {
	Status  CallStatus      `json:"status"`
	Data    sonic.NoCopyRawMessage `json:"data,omitempty"`
	Message string          `json:"message,omitempty"`
	Limit   Limit           `json:"limit,omitempty"`
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
	Bound(limits Limits)
	Read(ctx context.Context, call Call) Reply
	NoteReply(seq int, ms int64)
	Summary() Summary
}

type Summary struct {
	Calls    []CallSummary
	Usage    Usage
	Withheld []string
}
