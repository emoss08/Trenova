package codemode

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

const (
	frameMark      = "\x1etrenova-"
	replyMark      = "\x1e"
	FrameRequest   = "rpc"
	FrameResult    = "result"
	FrameEnd       = "end"
	EndOK          = "ok"
	EndRaised      = "raised"
	EndLimit       = "limit"
	EndKilled      = "killed"
	SandboxRoot    = "/home/user/.trenova"
	SandboxRPCDir  = SandboxRoot + "/rpc"
	SandboxSDKDir  = SandboxRoot + "/sdk"
	SandboxWorkDir = "/home/user/work"
)

type Frame struct {
	Kind string
	Body []byte
}

type ScriptError struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Traceback string `json:"traceback,omitempty"`
}

type End struct {
	Status       string       `json:"status"`
	Limit        Limit        `json:"limit,omitempty"`
	Error        *ScriptError `json:"error,omitempty"`
	RoundTripsMs []int64      `json:"roundTripsMs,omitempty"`
}

func ParseFrame(line []byte, nonce string) (Frame, bool) {
	if !bytes.HasPrefix(line, []byte(frameMark)) {
		return Frame{}, false
	}
	rest := line[len(frameMark):]
	kind, after, found := bytes.Cut(rest, []byte(":"))
	if !found {
		return Frame{}, false
	}
	stamp, body, found := bytes.Cut(after, []byte(":"))
	if !found || string(stamp) != nonce {
		return Frame{}, false
	}
	switch string(kind) {
	case FrameRequest, FrameResult, FrameEnd:
		return Frame{Kind: string(kind), Body: body}, true
	default:
		return Frame{}, false
	}
}

func ReplyPath(seq int) string {
	return SandboxRPCDir + "/" + strconv.Itoa(seq) + ".json"
}

func ReplyBody(reply Reply, nonce string) ([]byte, error) {
	encoded, err := sonic.Marshal(reply)
	if err != nil {
		return nil, err
	}
	body := make([]byte, 0, len(encoded)+len(nonce)+3)
	body = append(body, encoded...)
	body = append(body, '\n')
	body = append(body, replyMark...)
	body = append(body, nonce...)
	body = append(body, '\n')

	return body, nil
}

type LineSplitter struct {
	max      int
	pending  []byte
	dropping bool
}

func NewLineSplitter(maxLine int) *LineSplitter {
	return &LineSplitter{max: maxLine}
}

func (s *LineSplitter) Write(text string, emit func(line []byte, overflowed bool)) {
	for text != "" {
		idx := strings.IndexByte(text, '\n')
		if idx < 0 {
			s.buffer(text)
			return
		}
		s.buffer(text[:idx])
		line, overflowed := s.pending, s.dropping
		s.pending, s.dropping = nil, false
		emit(line, overflowed)
		text = text[idx+1:]
	}
}

func (s *LineSplitter) Flush(emit func(line []byte, overflowed bool)) {
	if len(s.pending) == 0 && !s.dropping {
		return
	}
	line, overflowed := s.pending, s.dropping
	s.pending, s.dropping = nil, false
	emit(line, overflowed)
}

func (s *LineSplitter) buffer(text string) {
	if s.dropping {
		return
	}
	if len(s.pending)+len(text) > s.max {
		room := max(s.max-len(s.pending), 0)
		s.pending = append(s.pending, text[:room]...)
		s.dropping = true
		return
	}
	s.pending = append(s.pending, text...)
}
