package assistantservice

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
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

func (s *Service) RunTranscript(
	ctx context.Context,
	req repositories.GetAgentRunByIDRequest,
) (*services.TranscriptFile, error) {
	if s.runs == nil {
		return nil, errortypes.NewBusinessError("Agent run transcripts are not available")
	}
	if req.TenantInfo == nil {
		return nil, errortypes.NewAuthorizationError("A tenant is required to read an agent run")
	}
	tenant := *req.TenantInfo

	run, err := s.runs.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	transcript, err := s.runTranscriptOf(ctx, run.ID, tenant)
	if err != nil {
		return nil, err
	}

	proposals, err := s.proposals.ListByRun(ctx, repositories.ListAgentProposalsByRunRequest{
		RunID:      run.ID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	messages := runTranscriptMessages(transcript)
	agentIDs := delegatedAgents(messages)
	if run.AgentDefinitionID.IsNotNil() {
		agentIDs = append(agentIDs, run.AgentDefinitionID)
	}
	names := s.agentNames(ctx, tenant, agentIDs)

	agentName := strings.TrimSpace(names[run.AgentDefinitionID])
	if agentName == "" {
		agentName = runTranscriptAgentFallback
	}

	return &services.TranscriptFile{
		FileName: runTranscriptFileName(agentName, run.ID),
		Body: renderRunTranscript(runTranscriptInput{
			Run:        run,
			Transcript: transcript,
			AgentName:  agentName,
			Delegates:  names,
			Proposals:  proposals,
			ExportedAt: timeutils.NowUnix(),
		}),
	}, nil
}

func (s *Service) runTranscriptOf(
	ctx context.Context,
	runID pulid.ID,
	tenant pagination.TenantInfo,
) (*agent.RunTranscript, error) {
	rows, err := s.runs.ListTranscriptsByIDs(ctx, repositories.ListAgentRunsByIDsRequest{
		IDs:        []pulid.ID{runID},
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if row != nil && row.ID == runID {
			return row.Transcript, nil
		}
	}

	return nil, nil //nolint:nilnil // a run that kept no transcript reads as one with no messages
}

func runTranscriptMessages(transcript *agent.RunTranscript) []conversation.Message {
	if transcript == nil {
		return nil
	}

	messages := make([]conversation.Message, len(transcript.Messages))
	for idx := range transcript.Messages {
		messages[idx] = conversation.MessageOfTranscript(&transcript.Messages[idx])
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
