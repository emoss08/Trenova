package agentredteam

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
)

const (
	caseScriptSeconds = 60
	caseScriptReads   = 60

	scriptRead   = "read"
	scriptPrint  = "print"
	scriptRepeat = "repeat"
)

type ScriptRecord struct {
	Seq        int
	Tool       string
	Status     codemode.CallStatus
	Limit      codemode.Limit
	Message    string
	SourceRead bool
}

// scriptDesk stands in for the sandbox: a case's script is one instruction a
// line, and each read goes through the real broker, as a script's frames do.
//
//	read <tool> <json arguments>
//	repeat <times> read <tool> <json arguments>
//	print <text>
type scriptDesk struct {
	rec *recorder
}

var _ agentextensionports.ScriptRunner = (*scriptDesk)(nil)

func (d *scriptDesk) RunScript(
	ctx context.Context,
	req *agentextensionports.ScriptRequest,
) (*agentextensionports.ScriptOutcome, error) {
	limits := codemode.LimitsFor(caseScriptSeconds, caseScriptReads)
	req.Broker.Bound(limits)

	var output strings.Builder
	seq := 0
	for _, raw := range strings.Split(req.Code, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		times := 1
		if rest, ok := strings.CutPrefix(line, scriptRepeat+" "); ok {
			count, after, _ := strings.Cut(rest, " ")
			parsed, err := strconv.Atoi(count)
			if err != nil {
				return nil, fmt.Errorf("script line %q: %w", line, err)
			}
			times, line = parsed, strings.TrimSpace(after)
		}
		if text, ok := strings.CutPrefix(line, scriptPrint+" "); ok {
			output.WriteString(text + "\n")
			continue
		}
		rest, ok := strings.CutPrefix(line, scriptRead+" ")
		if !ok {
			return nil, fmt.Errorf("script line %q is not read, repeat or print", line)
		}
		tool, rawArgs, _ := strings.Cut(rest, " ")
		args := map[string]any{}
		if strings.TrimSpace(rawArgs) != "" {
			if err := sonic.UnmarshalString(rawArgs, &args); err != nil {
				return nil, fmt.Errorf("script line %q: %w", line, err)
			}
		}
		for range times {
			seq++
			reply := req.Broker.Read(ctx, codemode.Call{Seq: seq, Tool: tool, Arguments: args})
			d.rec.scripted(ScriptRecord{
				Seq:     seq,
				Tool:    tool,
				Status:  reply.Status,
				Limit:   reply.Limit,
				Message: reply.Message,
			})
			fmt.Fprintf(&output, "%d %s %s %s\n", seq, tool, reply.Status, string(reply.Data))
		}
	}

	return &agentextensionports.ScriptOutcome{
		Status: agentextensionports.ScriptSucceeded,
		Output: output.String(),
		Limits: limits,
	}, nil
}

func (r *recorder) scripted(record ScriptRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if record.Status == codemode.CallOK && record.Tool == r.sourceTool {
		r.sourceRead = true
	}
	record.SourceRead = r.sourceRead
	r.scripts = append(r.scripts, record)
}
