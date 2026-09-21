package assistantservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

// A transcript is the conversation as a document: what was asked, what the
// model answered and thought, each tool it called with what it sent and got
// back, and the proposals it raised. The panel shows the same things one
// disclosure at a time; a person reading a run afterwards, or handing it to
// someone who was not there, wants them in order on one page.

const (
	// transcriptPageSize is how many messages one read carries. A thread is
	// capped at maxThreadMessages, so one page is the usual whole of it.
	transcriptPageSize = maxThreadMessages
	// transcriptMaxPages bounds the read for a thread older than the cap,
	// which existed before it and may exceed it.
	transcriptMaxPages = 25

	transcriptTimeLayout = "Jan 2, 2006 3:04:05 PM MST"
	transcriptSlugLength = 60
)

// transcriptJSON lays JSON out for a reader, keys in order, so the same
// arguments render the same way in every export and a diff between two
// transcripts shows what changed rather than what moved.
var transcriptJSON = sonic.Config{SortMapKeys: true}.Froze()

func (s *Service) Transcript(
	ctx context.Context,
	req repositories.GetThreadRequest,
) (*services.ThreadTranscript, error) {
	thread, err := s.conversations.GetThread(ctx, req)
	if err != nil {
		return nil, err
	}

	messages, err := s.allMessages(ctx, req)
	if err != nil {
		return nil, err
	}

	proposals, err := s.proposals.ListByThread(ctx, repositories.ListAgentProposalsByThreadRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return &services.ThreadTranscript{
		FileName: transcriptFileName(thread),
		Body: renderTranscript(transcriptInput{
			Thread:     thread,
			AgentName:  s.agentName(ctx, req, thread),
			Messages:   messages,
			Proposals:  proposals,
			ExportedAt: timeutils.NowUnix(),
		}),
	}, nil
}

// allMessages reads the thread from its newest page upward until the pages
// run out, and hands it back oldest first.
func (s *Service) allMessages(
	ctx context.Context,
	req repositories.GetThreadRequest,
) ([]conversation.Message, error) {
	pages := make([][]conversation.Message, 0, 1)
	total := 0
	var before *int

	for page := 0; page < transcriptMaxPages; page++ {
		messages, err := s.conversations.ListMessages(ctx, repositories.ListMessagesRequest{
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

// agentName reads the agent's name for the heading. An agent that has since
// been deleted leaves the conversation readable under a plain name.
func (s *Service) agentName(
	ctx context.Context,
	req repositories.GetThreadRequest,
	thread *conversation.Thread,
) string {
	if s.definitions == nil || thread.AgentDefinitionID.IsNil() {
		return "Assistant"
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         thread.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil || definition == nil || definition.Name == "" {
		if err != nil && !errortypes.IsNotFoundError(err) {
			s.logger.Warn("transcript could not name the agent")
		}

		return "Assistant"
	}

	return definition.Name
}

func transcriptFileName(thread *conversation.Thread) string {
	slug := stringutils.Slugify(thread.Title, transcriptSlugLength)
	if slug == "" {
		slug = "conversation"
	}
	id := thread.ID.String()
	if len(id) > 8 {
		id = id[len(id)-8:]
	}

	return slug + "-" + strings.ToLower(id) + ".md"
}

type transcriptInput struct {
	Thread     *conversation.Thread
	AgentName  string
	Messages   []conversation.Message
	Proposals  []*agent.AgentProposal
	ExportedAt int64
}

func renderTranscript(in transcriptInput) string {
	var b strings.Builder
	b.Grow(4096 + len(in.Messages)*512)

	title := strings.TrimSpace(in.Thread.Title)
	if title == "" {
		title = "Conversation"
	}
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "- **Agent:** %s\n", in.AgentName)
	fmt.Fprintf(&b, "- **Started:** %s\n", transcriptTime(in.Thread.CreatedAt))
	if in.Thread.LastMessageAt > 0 {
		fmt.Fprintf(&b, "- **Last message:** %s\n", transcriptTime(in.Thread.LastMessageAt))
	}
	fmt.Fprintf(&b, "- **Messages:** %d\n", len(in.Messages))
	fmt.Fprintf(&b, "- **Exported:** %s\n", transcriptTime(in.ExportedAt))
	fmt.Fprintf(&b, "- **Conversation id:** `%s`\n", in.Thread.ID.String())

	for i := range in.Messages {
		var section strings.Builder
		writeTranscriptMessage(&section, &in.Messages[i], in.AgentName)
		writeSection(&b, section.String())
	}

	if len(in.Proposals) > 0 {
		var section strings.Builder
		section.WriteString("## Proposals\n")
		for _, proposal := range in.Proposals {
			writeTranscriptProposal(&section, proposal)
		}
		writeSection(&b, section.String())
	}

	return b.String()
}

func writeSection(b *strings.Builder, section string) {
	b.WriteString("\n---\n\n")
	b.WriteString(strings.TrimRight(section, "\n"))
	b.WriteString("\n")
}

func writeTranscriptMessage(b *strings.Builder, m *conversation.Message, agentName string) {
	switch m.Role {
	case conversation.RoleUser:
		fmt.Fprintf(b, "## You · %s\n\n", transcriptTime(m.CreatedAt))
		if m.PageContext != nil && (m.PageContext.Title != "" || m.PageContext.Path != "") {
			fmt.Fprintf(b, "_On %s_\n\n", describePage(m.PageContext.Title, m.PageContext.Path))
		}
		writeRefusal(b, m, "Not answered")
		writeText(b, m.Content)
	case conversation.RoleAssistant:
		fmt.Fprintf(b, "## %s · %s", agentName, transcriptTime(m.CreatedAt))
		for _, part := range assistantMeta(m) {
			b.WriteString(" · ")
			b.WriteString(part)
		}
		b.WriteString("\n\n")
		writeRefusal(b, m, "Reply withheld")
		if m.Reasoning.Readable() {
			b.WriteString("<details>\n<summary>Reasoning</summary>\n\n")
			writeQuoted(b, m.Reasoning.Text)
			b.WriteString("\n</details>\n\n")
		}
		writeText(b, m.Content)
		for _, call := range m.ToolCalls {
			fmt.Fprintf(b, "**Called `%s`**\n\n", call.Name)
			writeJSON(b, call.Arguments)
		}
	case conversation.RoleTool:
		name := m.ToolName
		toolName, payload, fenced := agentruntime.UnfenceToolResult(m.Content)
		if fenced && name == "" {
			name = toolName
		}
		if name == "" {
			name = "tool"
		}
		fmt.Fprintf(b, "### Result from `%s`", name)
		if m.ToolFailed {
			b.WriteString(" · failed")
		}
		b.WriteString("\n\n")
		if fenced {
			writeResult(b, payload)
		} else {
			writeText(b, m.Content)
		}
	default:
		fmt.Fprintf(b, "## %s · %s\n\n", m.Role, transcriptTime(m.CreatedAt))
		writeText(b, m.Content)
	}
}

// assistantMeta is what a reader checking a turn wants beside it: the model,
// how long it took, what it cost in tokens.
func assistantMeta(m *conversation.Message) []string {
	parts := make([]string, 0, 3)
	if m.Model != "" {
		parts = append(parts, m.Model)
	}
	if m.LatencyMs > 0 {
		parts = append(parts, fmt.Sprintf("%.1f s", float64(m.LatencyMs)/1000))
	}
	if m.InputTokens > 0 || m.OutputTokens > 0 {
		parts = append(parts, fmt.Sprintf("%d in / %d out tokens", m.InputTokens, m.OutputTokens))
	}

	return parts
}

func writeRefusal(b *strings.Builder, m *conversation.Message, label string) {
	if !m.Refused {
		return
	}

	fmt.Fprintf(b, "> **%s.**", label)
	if m.ScopeCategory != "" {
		fmt.Fprintf(b, " %s", m.ScopeCategory)
	}
	if m.ScopeReason != "" {
		fmt.Fprintf(b, " (%s)", m.ScopeReason)
	}
	b.WriteString("\n\n")
}

func writeTranscriptProposal(b *strings.Builder, p *agent.AgentProposal) {
	fmt.Fprintf(b, "\n### `%s` · %s\n\n", p.ToolName, p.Status)
	fmt.Fprintf(b, "- **Tier:** %s\n", p.AutonomyTier)
	if !p.Confidence.IsZero() {
		fmt.Fprintf(b, "- **Confidence:** %s\n", p.Confidence.String())
	}
	if p.Rationale != "" {
		fmt.Fprintf(b, "- **Rationale:** %s\n", strings.TrimSpace(p.Rationale))
	}
	if p.ExecutedAt != nil && *p.ExecutedAt > 0 {
		fmt.Fprintf(b, "- **Executed:** %s\n", transcriptTime(*p.ExecutedAt))
	}
	if p.ExecutionError != "" {
		fmt.Fprintf(b, "- **Error:** %s\n", p.ExecutionError)
	}
	b.WriteString("\n")
	writeJSON(b, p.ToolParams)
}

func describePage(title, path string) string {
	switch {
	case title != "" && path != "":
		return fmt.Sprintf("%s (`%s`)", title, path)
	case title != "":
		return title
	default:
		return "`" + path + "`"
	}
}

func transcriptTime(ts int64) string {
	if ts <= 0 {
		return "unknown"
	}

	return time.Unix(ts, 0).UTC().Format(transcriptTimeLayout)
}

func writeText(b *strings.Builder, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	b.WriteString(text)
	b.WriteString("\n\n")
}

func writeQuoted(b *strings.Builder, text string) {
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		b.WriteString("> ")
		b.WriteString(line)
		b.WriteString("\n")
	}
}

// writeResult shows a tool result as the tool returned it: JSON laid out
// so a person can read it, or the text as it was. The runtime's own
// truncation notice, when a result was cut, stays in the text where the
// model saw it.
func writeResult(b *strings.Builder, payload string) {
	var decoded any
	if err := sonic.UnmarshalString(payload, &decoded); err == nil {
		writeJSON(b, decoded)

		return
	}

	writeFenced(b, "text", payload)
}

func writeJSON(b *strings.Builder, value any) {
	encoded, err := transcriptJSON.MarshalIndent(value, "", "  ")
	if err != nil {
		writeFenced(b, "text", fmt.Sprint(value))

		return
	}

	writeFenced(b, "json", string(encoded))
}

// writeFenced opens a code fence longer than any run of backticks inside the
// body, so a result that itself contains a fence cannot close this one early.
func writeFenced(b *strings.Builder, language, body string) {
	longest := 0
	run := 0
	for _, r := range body {
		if r == '`' {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	fence := strings.Repeat("`", max(3, longest+1))

	b.WriteString(fence)
	b.WriteString(language)
	b.WriteString("\n")
	b.WriteString(strings.TrimRight(body, "\n"))
	b.WriteString("\n")
	b.WriteString(fence)
	b.WriteString("\n\n")
}
