package assistanttranscripts

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/decisionnotes"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	oversightNameRunes     = 80
	oversightFallbackOwner = "Someone"
	oversightFallbackTitle = "Conversation"
	oversightFallbackAgent = "Assistant"
)

func (s *Service) RenderConversation(
	ctx context.Context,
	req *desktranscriptports.RenderRequest,
) (*desktranscriptports.RenderedConversation, error) {
	if req == nil || req.Thread == nil {
		return nil, errortypes.NewValidationError(
			"conversation",
			errortypes.ErrRequired,
			"A conversation is required",
		)
	}
	thread := req.Thread

	proposals, err := s.proposals.ListByThread(
		ctx,
		agentproposalports.ListAgentProposalsByThreadRequest{
			ThreadID:   thread.ID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	agentName := strings.TrimSpace(req.AgentName)
	if agentName == "" {
		agentName = oversightFallbackAgent
	}
	ownerName := strings.TrimSpace(req.OwnerName)
	if ownerName == "" {
		ownerName = oversightFallbackOwner
	}
	exportedAt := req.ExportedAt
	if exportedAt <= 0 {
		exportedAt = timeutils.NowUnix()
	}
	delegates := s.attribution().AgentNames(ctx, req.TenantInfo, delegatedAgents(req.Messages))

	markdown := renderTranscript(&transcriptInput{
		Thread:     thread,
		AgentName:  agentName,
		PersonName: ownerName,
		Delegates:  delegates,
		Messages:   req.Messages,
		Proposals:  proposals,
		Chapters:   s.chapterList().Titles(ctx, thread.ID, req.TenantInfo),
		ExportedAt: exportedAt,
	})

	folder := oversightName(ownerName, oversightFallbackOwner)
	stem := oversightDate(thread.CreatedAt, req.Location) + " - " +
		oversightName(thread.Title, oversightFallbackTitle)

	return &desktranscriptports.RenderedConversation{
		FileName:      folder + " - " + stem + ".md",
		ArchiveFolder: folder,
		ArchiveName:   stem + ".md",
		Markdown:      markdown,
		Record: conversationRecord(&recordInput{
			Thread:     thread,
			OwnerName:  ownerName,
			OwnerEmail: req.OwnerEmail,
			AgentName:  agentName,
			Delegates:  delegates,
			Messages:   req.Messages,
		}),
	}, nil
}

func oversightDate(ts int64, loc *time.Location) string {
	if ts <= 0 {
		return "undated"
	}
	if loc == nil {
		loc = time.UTC
	}

	return time.Unix(ts, 0).In(loc).Format("2006-01-02")
}

func oversightName(value, fallback string) string {
	var b strings.Builder
	space := false
	runes := 0
	for _, r := range strings.TrimSpace(value) {
		if runes >= oversightNameRunes {
			break
		}
		keep := unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("-_.,()'&+", r)
		if !keep || unicode.IsSpace(r) {
			space = b.Len() > 0
			continue
		}
		if space {
			b.WriteByte(' ')
			runes++
			space = false
		}
		b.WriteRune(r)
		runes++
	}

	name := strings.Trim(b.String(), " .")
	if name == "" {
		return fallback
	}

	return name
}

type recordInput struct {
	Thread     *conversation.Thread
	OwnerName  string
	OwnerEmail string
	AgentName  string
	Delegates  map[pulid.ID]string
	Messages   []conversation.Message
}

func recordTime(ts int64) string {
	if ts <= 0 {
		return ""
	}

	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func conversationRecord(in *recordInput) desktranscriptports.ConversationRecord {
	messages := withoutSilentReplies(in.Messages)
	record := desktranscriptports.ConversationRecord{
		Owner:         desktranscriptports.RecordOwner{Name: in.OwnerName, Email: in.OwnerEmail},
		Agent:         in.AgentName,
		Title:         in.Thread.Title,
		Origin:        string(in.Thread.Origin),
		Archived:      in.Thread.Status == conversation.ThreadStatusArchived,
		StartedAt:     recordTime(in.Thread.CreatedAt),
		LastMessageAt: recordTime(in.Thread.LastMessageAt),
		MessageCount:  len(messages),
		Messages:      make([]desktranscriptports.RecordMessage, 0, len(messages)),
	}

	for idx := range messages {
		record.Messages = append(record.Messages, recordMessage(&messages[idx], in))
	}

	return record
}

func recordMessage(m *conversation.Message, in *recordInput) desktranscriptports.RecordMessage {
	out := desktranscriptports.RecordMessage{
		Role:    string(m.Role),
		Kind:    string(m.Kind),
		At:      recordTime(m.CreatedAt),
		Refused: m.Refused,
	}

	switch m.Role {
	case conversation.RoleUser:
		out.Text = recordUserText(m)
	case conversation.RoleAssistant:
		out.Agent = recordAgent(m, in)
		out.Text = strings.TrimSpace(replycheck.StripArtifactLinks(m.Content))
		for _, call := range m.ToolCalls {
			out.ToolCalls = append(
				out.ToolCalls,
				desktranscriptports.RecordToolCall{Name: call.Name},
			)
		}
	case conversation.RoleTool:
		out.Agent = recordAgent(m, in)
		out.Tool = recordTool(m)
	default:
		out.Text = strings.TrimSpace(m.Content)
	}

	return out
}

func recordAgent(m *conversation.Message, in *recordInput) string {
	if m.Delegated() && m.AgentDefinitionID.IsNotNil() {
		if name := strings.TrimSpace(in.Delegates[m.AgentDefinitionID]); name != "" {
			return name
		}
	}

	return in.AgentName
}

func recordUserText(m *conversation.Message) string {
	switch {
	case m.Kind == conversation.MessageKindDecisionNote:
		return decisionnotes.DecisionHeadline(m.Content)
	case m.Kind == conversation.MessageKindHandoffBrief && m.Handoff != nil:
		return strings.TrimSpace(m.Handoff.Summary)
	default:
		return strings.TrimSpace(m.Content)
	}
}

func recordTool(m *conversation.Message) *desktranscriptports.RecordTool {
	tool := &desktranscriptports.RecordTool{
		Name:    m.ToolName,
		Failed:  m.ToolFailed,
		Verdict: m.ToolVerdict,
		Summary: m.ToolSummary,
	}
	name, payload, fenced := toolresult.Unfence(m.Content)
	if fenced && tool.Name == "" {
		tool.Name = name
	}
	switch {
	case fenced:
		tool.Result = strings.TrimSpace(payload)
	default:
		if message, failed := toolresult.UnfenceFailure(m.Content); failed {
			tool.Result = strings.TrimSpace(message)
		} else {
			tool.Result = strings.TrimSpace(toolresult.UnfenceInline(m.Content))
		}
	}
	if tool.Name == "" {
		tool.Name = "tool"
	}

	return tool
}
