package assistanttranscripts

import (
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationtranscript"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	runTranscriptAgentFallback = "Agent"
	runTranscriptEmpty         = "This run kept no transcript."
)

type runTranscriptInput struct {
	Run        *agent.AgentRun
	Transcript *agent.RunTranscript
	AgentName  string
	Delegates  map[pulid.ID]string
	Proposals  []*agent.AgentProposal
	ExportedAt int64
}

func runTranscriptMessages(transcript *agent.RunTranscript) []conversation.Message {
	if transcript == nil {
		return nil
	}

	messages := make([]conversation.Message, len(transcript.Messages))
	for idx := range transcript.Messages {
		messages[idx] = conversationtranscript.MessageOfTranscript(&transcript.Messages[idx])
	}

	return messages
}

func runTranscriptTrimmed(transcript *agent.RunTranscript) []bool {
	if transcript == nil {
		return nil
	}

	var trimmed []bool
	for idx := range transcript.Messages {
		if !transcript.Messages[idx].Omitted {
			continue
		}
		if trimmed == nil {
			trimmed = make([]bool, len(transcript.Messages))
		}
		trimmed[idx] = true
	}

	return trimmed
}

func renderRunTranscript(in runTranscriptInput) string {
	messages := runTranscriptMessages(in.Transcript)

	var gap transcriptGap
	if in.Transcript != nil && in.Transcript.OmittedMessages > 0 {
		gap = transcriptGap{
			At:    min(max(in.Transcript.OmittedAt, 0), len(messages)),
			Count: in.Transcript.OmittedMessages,
		}
	}

	return renderTranscriptDocument(&transcriptDocument{
		Title:     in.AgentName + " run",
		Facts:     runTranscriptFacts(&in, len(messages), gap.Count),
		AgentName: in.AgentName,
		Delegates: in.Delegates,
		Messages:  messages,
		Trimmed:   runTranscriptTrimmed(in.Transcript),
		Gap:       gap,
		Empty:     runTranscriptEmpty,
		Proposals: in.Proposals,
	})
}

func runTranscriptFacts(in *runTranscriptInput, kept, omitted int) []transcriptFact {
	run := in.Run
	facts := make([]transcriptFact, 0, 9)
	facts = append(facts,
		transcriptFact{Name: "Agent", Value: in.AgentName},
		transcriptFact{Name: "Trigger", Value: string(run.Trigger)},
		transcriptFact{Name: "Status", Value: string(run.Status)},
		transcriptFact{Name: "Started", Value: transcriptTime(runStartedAt(run))},
	)
	if run.CompletedAt != nil && *run.CompletedAt > 0 {
		facts = append(facts, transcriptFact{
			Name:  "Finished",
			Value: transcriptTime(*run.CompletedAt),
		})
	}
	if run.ModelIdentifier != "" {
		facts = append(facts, transcriptFact{Name: "Model", Value: run.ModelIdentifier})
	}

	messages := strconv.Itoa(kept)
	if omitted > 0 {
		messages = fmt.Sprintf("%d kept, %d left out", kept, omitted)
	}
	facts = append(facts,
		transcriptFact{Name: "Messages", Value: messages},
		transcriptFact{Name: "Exported", Value: transcriptTime(in.ExportedAt)},
		transcriptFact{Name: "Run id", Value: "`" + run.ID.String() + "`"},
	)

	return facts
}

func runStartedAt(run *agent.AgentRun) int64 {
	if run.StartedAt > 0 {
		return run.StartedAt
	}

	return run.CreatedAt
}

func runTranscriptFileName(agentName string, runID pulid.ID) string {
	slug := stringutils.Slugify(agentName+" run", transcriptSlugLength)
	if slug == "" {
		slug = "agent-run"
	}

	return slug + "-" + transcriptIDSuffix(runID) + ".md"
}
