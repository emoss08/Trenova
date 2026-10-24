package agentredteam

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/app/scriptrunner"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/scriptports"
	"go.uber.org/zap"
)

const (
	scriptRead   = "read"
	scriptPrint  = "print"
	scriptRepeat = "repeat"

	scriptPrelude = `import json


def _case_read(tool, args):
    reply = json.loads(_trenova_read(json.dumps({"tool": tool, "args": json.loads(args)})))
    print(tool, reply.get("status"), json.dumps(reply.get("data")))

`
)

var caseRunner = sync.OnceValue(func() *scriptrunner.Service {
	return scriptrunner.NewRunner(zap.NewNop())
})

type ScriptRecord struct {
	Seq        int
	Tool       string
	Status     codemode.CallStatus
	Limit      codemode.Limit
	Message    string
	SourceRead bool
}

// scriptDesk runs a case's script in the real interpreter, through the real
// runner and broker. A case writes its script one instruction a line, and each
// becomes Python that calls the SDK's host function straight, as a script that
// names a read the turn does not hold would:
//
//	read <tool> <json arguments>
//	repeat <times> read <tool> <json arguments>
//	print <text>
type scriptDesk struct {
	rec *recorder
}

var _ scriptports.Runner = (*scriptDesk)(nil)

func (d *scriptDesk) Run(
	ctx context.Context,
	req *scriptports.Request,
) (*scriptports.Outcome, error) {
	code, err := caseScript(req.Code)
	if err != nil {
		return nil, err
	}
	scripted := *req
	scripted.Code = code
	scripted.Broker = &recordingBroker{Broker: req.Broker, rec: d.rec}

	return caseRunner().Run(ctx, &scripted)
}

func caseScript(source string) (string, error) {
	var code strings.Builder
	code.WriteString(scriptPrelude)
	for _, raw := range strings.Split(source, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		times := 1
		if rest, ok := strings.CutPrefix(line, scriptRepeat+" "); ok {
			count, after, _ := strings.Cut(rest, " ")
			parsed, err := strconv.Atoi(count)
			if err != nil {
				return "", fmt.Errorf("script line %q: %w", line, err)
			}
			times, line = parsed, strings.TrimSpace(after)
		}
		if text, ok := strings.CutPrefix(line, scriptPrint+" "); ok {
			fmt.Fprintf(&code, "print(%s)\n", strconv.Quote(text))
			continue
		}
		rest, ok := strings.CutPrefix(line, scriptRead+" ")
		if !ok {
			return "", fmt.Errorf("script line %q is not read, repeat or print", line)
		}
		tool, args, _ := strings.Cut(rest, " ")
		if strings.TrimSpace(args) == "" {
			args = "{}"
		}
		fmt.Fprintf(&code, "for _ in range(%d):\n    _case_read(%s, %s)\n",
			times, strconv.Quote(tool), strconv.Quote(args))
	}

	return code.String(), nil
}

type recordingBroker struct {
	codemode.Broker
	rec *recorder
}

func (b *recordingBroker) Read(ctx context.Context, call codemode.Call) codemode.Reply {
	reply := b.Broker.Read(ctx, call)
	b.rec.scripted(&ScriptRecord{
		Seq:     call.Seq,
		Tool:    call.Tool,
		Status:  reply.Status,
		Limit:   reply.Limit,
		Message: reply.Message,
	})

	return reply
}

func (r *recorder) scripted(record *ScriptRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if record.Status == codemode.CallOK && record.Tool == r.sourceTool {
		r.sourceRead = true
	}
	record.SourceRead = r.sourceRead
	r.scripts = append(r.scripts, *record)
}
