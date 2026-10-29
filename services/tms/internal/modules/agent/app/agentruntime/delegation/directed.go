package delegation

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/worldwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
)

// DirectedTask is the task the person handed another agent, on a turn that
// can run one: the person's own, read as it runs, in a conversation that
// keeps what the other agent does.
func DirectedTask(req *agentruntimeports.RunRequest) *agentruntimeports.DirectedTask {
	if req.Directed == nil || req.Delegation != nil || req.Unattended ||
		req.ThreadID.IsNil() {
		return nil
	}
	task := *req.Directed
	if len(task.Records) == 0 {
		task.Records = recordsInPlay(req)
	}

	return &task
}

// recordsInPlay hands the conversation's subject and the records it has been
// working on to the agent the person turned to. Without them "bill it" reached
// the billing agent as two words, and it went looking for which load.
func recordsInPlay(req *agentruntimeports.RunRequest) []agentrecord.RecordRef {
	watched := worldwatch.WatchedRecords(req)
	records := make([]agentrecord.RecordRef, 0, min(len(watched), runtimetools.MaxDelegateRecords))
	for _, record := range watched {
		if len(records) == runtimetools.MaxDelegateRecords {
			break
		}
		if record.Type == "" || record.ID == "" {
			continue
		}
		records = append(records, agentrecord.RecordRef{EntityType: record.Type, ID: record.ID})
	}
	if len(records) == 0 {
		return nil
	}

	return records
}

func HandedRecords(records []agentrecord.RecordRef) *Context {
	if len(records) == 0 {
		return nil
	}

	return &Context{Records: records}
}

// DirectedReply is what closes a directed turn: the other agent's answer,
// or why there is none, so the conversation never ends on a tool result
// and the person reads the outcome where they read every reply.
func DirectedReply(report *assistantstreamports.AssistantDelegateFinishedEvent) string {
	reply := strings.TrimSpace(report.Reply)
	switch report.Status {
	case assistantstreamports.DelegateStatusCompleted:
		if reply != "" {
			return reply
		}
		return report.AgentName + " finished without saying anything more."
	case assistantstreamports.DelegateStatusExhausted:
		if reply != "" {
			return reply + "\n\n" + report.Reason
		}
		return report.Reason
	case assistantstreamports.DelegateStatusDeclined,
		assistantstreamports.DelegateStatusRefused,
		assistantstreamports.DelegateStatusStopped,
		assistantstreamports.DelegateStatusFailed:
		if report.Reason != "" {
			return report.Reason
		}
	}

	return report.AgentName + " did not report back."
}
