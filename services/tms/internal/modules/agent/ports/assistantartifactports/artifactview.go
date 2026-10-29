package assistantartifactports

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/pulid"
)

// AssistantArtifact is what a turn produced besides words, as the Desk shows
// it beside the conversation.
type AssistantArtifact struct {
	ID         pulid.ID                 `json:"id"`
	ThreadID   pulid.ID                 `json:"threadId"`
	MessageID  pulid.ID                 `json:"messageId"`
	RunID      pulid.ID                 `json:"runId"`
	ProposalID pulid.ID                 `json:"proposalId"`
	PlanID     pulid.ID                 `json:"planId"`
	Kind       assistantartifact.Kind   `json:"kind"`
	Status     assistantartifact.Status `json:"status"`
	Title      string                   `json:"title"`
	Payload    map[string]any           `json:"payload"`
	// SourceToolCallID ties the artifact to the tool call that produced it,
	// so the transcript can point at it.
	SourceToolCallID string `json:"sourceToolCallId"`
	Pinned           bool   `json:"pinned"`
	// LineageID names the first artifact of the lineage this one is a later
	// version of, and LineageSeq its version number; empty and 1 for the first.
	LineageID  pulid.ID `json:"lineageId"`
	LineageSeq int      `json:"lineageSeq"`
	// Slug names the lineage in a link; the same for every version.
	Slug string `json:"slug"`
	// Turn is the question the person asked in the turn that made the
	// artifact, so an artifact from an earlier day says which turn it was.
	Turn      string `json:"turn,omitempty"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// ListArtifactsOptions is one page of a conversation's artifacts and what
// narrows it, all decided on the server so a long conversation pages rather
// than truncates.
type ListArtifactsOptions struct {
	Limit      int
	Cursor     string
	Query      string
	Family     assistantartifact.Family
	PinnedOnly bool
	Summary    bool
}

// AssistantArtifactPage is one page of lineages, every version of each, how
// many lineages match in all and how they split by family.
type AssistantArtifactPage struct {
	Results    []AssistantArtifact `json:"results"`
	Total      int                 `json:"total"`
	NextCursor string              `json:"nextCursor,omitempty"`
	Counts     ArtifactCounts      `json:"counts"`
}

// DocumentRewriteMode is how a person asked a passage to change.
type DocumentRewriteMode string

const DocumentRewriteShorter DocumentRewriteMode = "shorter"

const DocumentRewritePlainer DocumentRewriteMode = "plain"

const DocumentRewriteAsk DocumentRewriteMode = "ask"

// DocumentRewriteRequest is a passage of a document to rewrite and how. The
// suggestion is not saved; accepting it saves a new version.
type DocumentRewriteRequest struct {
	Text   string              `json:"text"`
	Mode   DocumentRewriteMode `json:"mode"`
	Prompt string              `json:"prompt"`
}

type DocumentRewriteSuggestion struct {
	Text string `json:"text"`
}

// SaveDocumentVersionRequest is a person's edit of a document, kept as its
// next version.
type SaveDocumentVersionRequest struct {
	Body string `json:"body"`
	Note string `json:"note"`
}

// ArtifactFile is an artifact rendered as a file to download.
type ArtifactFile struct {
	FileName    string
	ContentType string
	Body        []byte
}

type ArtifactRowsQuery struct {
	After         string
	Offset        int
	Limit         int
	Query         string
	SortBy        string
	SortDirection string
	IncludeTotal  bool
}

type ArtifactRows struct {
	Rows           []any    `json:"rows"`
	HasMore        bool     `json:"hasMore"`
	NextCursor     string   `json:"nextCursor,omitempty"`
	NextOffset     *int     `json:"nextOffset,omitempty"`
	Total          *int     `json:"total,omitempty"`
	TotalEstimated bool     `json:"totalEstimated,omitempty"`
	Withheld       []string `json:"withheld"`
	Searchable     bool     `json:"searchable"`
	Sortable       []string `json:"sortable"`
}
