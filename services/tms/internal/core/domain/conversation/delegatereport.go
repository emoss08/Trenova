package conversation

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	// maxReportReplyRunes bounds the delegate's answer kept on the message.
	// The delegating agent read the whole answer when the turn ran; the saved
	// account is what the thread draws, and a long answer is cut with an
	// ellipsis rather than making the row grow with it.
	maxReportReplyRunes  = 2000
	maxReportReasonRunes = 500
	maxReportNameRunes   = 200
	maxReportWordRunes   = 40
	maxReportIDRunes     = 200
	maxReportWrites      = 20
	maxReportDocuments   = 10
	maxWriteSummaryRunes = 200
	maxWriteErrorRunes   = 500
)

// DelegateStatus is how a task handed to another agent ended.
type DelegateStatus string

const (
	// DelegateStatusCompleted is a task the other agent finished and
	// answered.
	DelegateStatusCompleted = DelegateStatus("completed")
	// DelegateStatusExhausted is one it spent its tool budget on.
	DelegateStatusExhausted = DelegateStatus("exhausted")
	// DelegateStatusRefused is one whose answer the output guard withheld.
	DelegateStatusRefused = DelegateStatus("refused")
	// DelegateStatusDeclined is one that never started: the agent could not
	// be asked.
	DelegateStatusDeclined = DelegateStatus("declined")
	// DelegateStatusFailed is one that ended partway, for the reason given.
	DelegateStatusFailed = DelegateStatus("failed")
	// DelegateStatusStopped is one the person stopped.
	DelegateStatusStopped = DelegateStatus("stopped")
)

// DelegateReport is the account of a task handed to another agent: how it
// ended, what it answered, and every write it made or proposed. The reader is
// shown it live as delegate_finished, the delegating model reads it as the
// call's result, and the delegate_task result message keeps it, bounded, so a
// saved conversation draws the hand-off from it rather than from the text.
type DelegateReport struct {
	DelegateCallID string   `json:"delegateCallId"`
	AgentID        pulid.ID `json:"agentId"`
	AgentName      string   `json:"agentName"`
	// Icon and Accent are the agent's mark as the turn knew it.
	Icon   string         `json:"icon,omitempty"`
	Accent string         `json:"accent,omitempty"`
	Status DelegateStatus `json:"status"`
	// Reply is the other agent's answer; Reason is why it did not finish.
	Reply  string `json:"reply,omitempty"`
	Reason string `json:"reason,omitempty"`
	// Made are the writes it made, Awaiting those waiting on a person's
	// decision, and Published the documents it kept beside the conversation.
	Made          []DelegateWrite    `json:"made"`
	Awaiting      []DelegateWrite    `json:"awaiting"`
	Published     []DelegateDocument `json:"published"`
	ToolCallsUsed int                `json:"toolCallsUsed"`
	// MoreMade, MoreAwaiting and MorePublished count what a bounded account
	// left out of each list, so a reader can say there was more rather than
	// imply the list is whole.
	MoreMade      int `json:"moreMade,omitempty"`
	MoreAwaiting  int `json:"moreAwaiting,omitempty"`
	MorePublished int `json:"morePublished,omitempty"`
}

// DelegateWrite is one write another agent made or proposed on a task.
type DelegateWrite struct {
	ToolName string             `json:"toolName"`
	CallID   string             `json:"callId"`
	Tier     agent.AutonomyTier `json:"tier"`
	// Summary names what the write is about, from its arguments.
	Summary string `json:"summary,omitempty"`
	// Result is what an executed write made, when its tool says.
	Result *agent.ToolExecutionResult `json:"result,omitempty"`
	// Error is why an executed write failed.
	Error string `json:"error,omitempty"`
	// Simulated says the agent was in simulation, so the write was
	// previewed rather than made.
	Simulated bool `json:"simulated,omitempty"`
}

// DelegateDocument is something another agent kept beside the conversation.
type DelegateDocument struct {
	ID    pulid.ID `json:"id"`
	Kind  string   `json:"kind"`
	Title string   `json:"title"`
}

// Bounded is the account cut to what a saved message keeps: the answer and
// the reason ellipsized, labels to one line, each list to its first few
// entries with the rest counted, and each write's result bounded as a
// proposal's is. Nil stays nil. The lists are never nil, so the saved account
// has the shape the live one has.
func (r *DelegateReport) Bounded() *DelegateReport {
	if r == nil {
		return nil
	}

	bounded := &DelegateReport{
		DelegateCallID: stringutils.TruncateRunes(r.DelegateCallID, maxReportIDRunes),
		AgentID:        r.AgentID,
		AgentName:      stringutils.OneLine(r.AgentName, maxReportNameRunes),
		Icon:           stringutils.OneLine(r.Icon, maxReportWordRunes),
		Accent:         stringutils.OneLine(r.Accent, maxReportWordRunes),
		Status:         r.Status,
		Reply:          stringutils.Ellipsize(r.Reply, maxReportReplyRunes),
		Reason:         stringutils.Ellipsize(r.Reason, maxReportReasonRunes),
		ToolCallsUsed:  max(r.ToolCallsUsed, 0),
	}

	bounded.Made, bounded.MoreMade = boundedWrites(r.Made, r.MoreMade)
	bounded.Awaiting, bounded.MoreAwaiting = boundedWrites(r.Awaiting, r.MoreAwaiting)
	bounded.Published, bounded.MorePublished = boundedDocuments(r.Published, r.MorePublished)

	return bounded
}

func boundedWrites(writes []DelegateWrite, more int) ([]DelegateWrite, int) {
	kept := min(len(writes), maxReportWrites)
	out := make([]DelegateWrite, 0, kept)
	for idx := range kept {
		write := &writes[idx]
		out = append(out, DelegateWrite{
			ToolName: stringutils.OneLine(write.ToolName, maxReportNameRunes),
			CallID:   stringutils.TruncateRunes(write.CallID, maxReportIDRunes),
			Tier:     write.Tier,
			Summary: stringutils.Ellipsize(
				stringutils.CollapseWhitespace(write.Summary),
				maxWriteSummaryRunes,
			),
			Result:    write.Result.Bounded(),
			Error:     stringutils.Ellipsize(write.Error, maxWriteErrorRunes),
			Simulated: write.Simulated,
		})
	}

	return out, max(more, 0) + len(writes) - kept
}

func boundedDocuments(documents []DelegateDocument, more int) ([]DelegateDocument, int) {
	kept := min(len(documents), maxReportDocuments)
	out := make([]DelegateDocument, 0, kept)
	for idx := range kept {
		document := &documents[idx]
		out = append(out, DelegateDocument{
			ID:    document.ID,
			Kind:  stringutils.OneLine(document.Kind, maxReportWordRunes),
			Title: stringutils.OneLine(document.Title, maxReportNameRunes),
		})
	}

	return out, max(more, 0) + len(documents) - kept
}
