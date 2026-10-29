package assistanttranscripts

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/attribution"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/chaptertitles"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) RunTranscript(
	ctx context.Context,
	req agentruntimeports.GetAgentRunByIDRequest,
) (*kernelports.TranscriptFile, error) {
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

	proposals, err := s.proposals.ListByRun(ctx, agentproposalports.ListAgentProposalsByRunRequest{
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
	names := s.attribution().AgentNames(ctx, tenant, agentIDs)

	agentName := strings.TrimSpace(names[run.AgentDefinitionID])
	if agentName == "" {
		agentName = runTranscriptAgentFallback
	}

	return &kernelports.TranscriptFile{
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
	rows, err := s.runs.ListTranscriptsByIDs(ctx, &agentruntimeports.ListAgentRunsByIDsRequest{
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

type Service struct {
	logger        *zap.Logger
	conversations conversationports.ConversationRepository
	definitions   agentdefinitionports.AgentDefinitionRepository
	proposals     proposalstores.ProposalStore
	runs          agentruntimeports.AgentRunRepository
	users         tenantports.UserRepository
	chapters      assistantchapterports.ChapterRepository
}

func (s *Service) attribution() *attribution.Attributor {
	return &attribution.Attributor{
		Logger:      s.logger,
		Definitions: s.definitions,
		Runs:        s.runs,
		Users:       s.users,
	}
}

func (s *Service) chapterList() *chaptertitles.Index {
	return &chaptertitles.Index{Logger: s.logger, Chapters: s.chapters}
}

func (s *Service) Transcript(
	ctx context.Context,
	req conversationports.GetThreadRequest,
) (*kernelports.TranscriptFile, error) {
	thread, err := s.conversations.GetThread(ctx, &req)
	if err != nil {
		return nil, err
	}

	messages, err := s.allMessages(ctx, req)
	if err != nil {
		return nil, err
	}

	proposals, err := s.proposals.ListByThread(
		ctx,
		agentproposalports.ListAgentProposalsByThreadRequest{
			ThreadID:   req.ID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	return &kernelports.TranscriptFile{
		FileName: transcriptFileName(thread),
		Body: renderTranscript(&transcriptInput{
			Thread:     thread,
			AgentName:  s.attribution().AgentName(ctx, req, thread),
			Delegates:  s.attribution().AgentNames(ctx, req.TenantInfo, delegatedAgents(messages)),
			Messages:   messages,
			Proposals:  proposals,
			Chapters:   s.chapterList().Titles(ctx, req.ID, req.TenantInfo),
			ExportedAt: timeutils.NowUnix(),
		}),
	}, nil
}

// allMessages reads the thread from its newest page upward until the pages
// run out, and hands it back oldest first.
func (s *Service) allMessages(
	ctx context.Context,
	req conversationports.GetThreadRequest,
) ([]conversation.Message, error) {
	pages := make([][]conversation.Message, 0, 1)
	total := 0
	var before *int

	for page := 0; page < transcriptMaxPages; page++ {
		messages, err := s.conversations.ListMessages(ctx, &conversationports.ListMessagesRequest{
			ThreadID:       req.ID,
			TenantInfo:     req.TenantInfo,
			Limit:          transcriptPageSize,
			BeforeSequence: before,
		})
		if err != nil {
			return nil, err
		}
		if len(messages) == 0 {
			break
		}

		pages = append(pages, messages)
		total += len(messages)

		oldest := messages[0].Sequence
		if len(messages) < transcriptPageSize || (before != nil && oldest >= *before) {
			break
		}
		before = &oldest
	}

	ordered := make([]conversation.Message, 0, total)
	for i := len(pages) - 1; i >= 0; i-- {
		ordered = append(ordered, pages[i]...)
	}

	return ordered, nil
}
