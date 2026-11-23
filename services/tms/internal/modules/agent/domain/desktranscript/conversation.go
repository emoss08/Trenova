package desktranscript

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MaxExportConversations   = 5000
	MaxSelectedConversations = 200
	MaxExportBytes           = int64(1) << 30
	MaxOwnersNamed           = 20
)

type Conversation struct {
	ID                pulid.ID                  `json:"id"`
	UserID            pulid.ID                  `json:"-"`
	AgentDefinitionID pulid.ID                  `json:"-"`
	OwnerName         string                    `json:"ownerName"`
	AgentName         string                    `json:"agentName"`
	Title             string                    `json:"title"`
	Origin            conversation.ThreadOrigin `json:"origin"`
	Archived          bool                      `json:"archived"`
	MessageCount      int                       `json:"messageCount"`
	StartedAt         int64                     `json:"startedAt"`
	LastMessageAt     *int64                    `json:"lastMessageAt"`
}

func (c *Conversation) GetID() pulid.ID {
	return c.ID
}

func (c *Conversation) GetCreatedAt() int64 {
	return c.StartedAt
}

func ConversationOf(thread *conversation.Thread) *Conversation {
	out := &Conversation{
		ID:                thread.ID,
		UserID:            thread.UserID,
		AgentDefinitionID: thread.AgentDefinitionID,
		Title:             thread.Title,
		Origin:            thread.Origin,
		Archived:          thread.Status == conversation.ThreadStatusArchived,
		StartedAt:         thread.CreatedAt,
	}
	if thread.LastMessageAt > 0 {
		last := thread.LastMessageAt
		out.LastMessageAt = &last
	}

	return out
}
