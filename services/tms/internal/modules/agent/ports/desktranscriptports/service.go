package desktranscriptports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	ExportWorkflowName = "DeskTranscriptExportWorkflow"
	ExportReadyEvent   = "desk_transcript_export_ready"
	ExportFailedEvent  = "desk_transcript_export_failed"
	ExportResource     = "desk-transcript-export"
	ExportsPath        = "/admin/desk-conversations?view=exports"
)

type ExportPayload struct {
	ExportID       pulid.ID `json:"exportId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

func (p *ExportPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type RenderRequest struct {
	TenantInfo pagination.TenantInfo
	Thread     *conversation.Thread
	Messages   []conversation.Message
	OwnerName  string
	OwnerEmail string
	AgentName  string
	Location   *time.Location
	ExportedAt int64
}

type RecordOwner struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type RecordToolCall struct {
	Name string `json:"name"`
}

type RecordTool struct {
	Name    string `json:"name"`
	Failed  bool   `json:"failed,omitempty"`
	Verdict string `json:"verdict,omitempty"`
	Summary string `json:"summary,omitempty"`
	Result  string `json:"result,omitempty"`
}

type RecordMessage struct {
	Role      string           `json:"role"`
	Kind      string           `json:"kind"`
	Agent     string           `json:"agent,omitempty"`
	At        string           `json:"at"`
	Text      string           `json:"text,omitempty"`
	ToolCalls []RecordToolCall `json:"toolCalls,omitempty"`
	Tool      *RecordTool      `json:"tool,omitempty"`
	Refused   bool             `json:"refused,omitempty"`
}

type ConversationRecord struct {
	Owner         RecordOwner     `json:"owner"`
	Agent         string          `json:"agent"`
	Title         string          `json:"title"`
	Origin        string          `json:"origin"`
	Archived      bool            `json:"archived"`
	StartedAt     string          `json:"startedAt"`
	LastMessageAt string          `json:"lastMessageAt,omitempty"`
	MessageCount  int             `json:"messageCount"`
	Messages      []RecordMessage `json:"messages"`
}

type RenderedConversation struct {
	FileName      string
	ArchiveFolder string
	ArchiveName   string
	Markdown      string
	Record        ConversationRecord
}

type TranscriptRenderer interface {
	RenderConversation(ctx context.Context, req *RenderRequest) (*RenderedConversation, error)
}

type ListRequest struct {
	Actor  *permissionports.RequestActor
	Filter desktranscript.ConversationFilter
	Query  string
	Cursor pagination.CursorInfo
}

type ConversationRequest struct {
	Actor *permissionports.RequestActor
	ID    pulid.ID
}

type OpenedConversation struct {
	Conversation *desktranscript.Conversation
	Transcript   *agent.RunTranscript
}

type PreviewRequest struct {
	Actor  *permissionports.RequestActor
	Filter desktranscript.ConversationFilter
}

type ExportPreview struct {
	Count       int  `json:"count"`
	Exact       bool `json:"exact"`
	Limit       int  `json:"limit"`
	WithinLimit bool `json:"withinLimit"`
}

type RequestExportRequest struct {
	Actor           *permissionports.RequestActor
	Filter          *desktranscript.ConversationFilter
	ConversationIDs []pulid.ID
}

type ListExportsServiceRequest struct {
	Actor  *permissionports.RequestActor
	Filter *pagination.QueryOptions
	Cursor pagination.CursorInfo
}

type ExportDownloadRequest struct {
	Actor    *permissionports.RequestActor
	ExportID pulid.ID
}

type ExportDownload struct {
	URL       string `json:"url"`
	FileName  string `json:"fileName"`
	ExpiresAt int64  `json:"expiresAt"`
}

type Service interface {
	ListConversations(
		ctx context.Context,
		req *ListRequest,
	) (*pagination.CursorListResult[*desktranscript.Conversation], error)
	OpenConversation(ctx context.Context, req ConversationRequest) (*OpenedConversation, error)
	DownloadConversation(
		ctx context.Context,
		req ConversationRequest,
	) (*kernelports.TranscriptFile, error)
	PreviewExport(ctx context.Context, req *PreviewRequest) (*ExportPreview, error)
	RequestExport(ctx context.Context, req *RequestExportRequest) (*desktranscript.Export, error)
	ListExports(
		ctx context.Context,
		req *ListExportsServiceRequest,
	) (*pagination.CursorListResult[*desktranscript.Export], error)
	ExportDownload(ctx context.Context, req ExportDownloadRequest) (*ExportDownload, error)
}
