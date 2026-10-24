package conversation

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	MaxMessageAttachments  = 5
	MaxQueuedContentRunes  = 16_000
	MaxQueuedPerThread     = 20
	QueuedMessageIDPrefix  = "aqm_"
	queuedMessageTableName = "assistant_queued_messages"
)

var (
	_ validationframework.TenantedEntity = (*QueuedMessage)(nil)
)

type QueuedRequest struct {
	Page                  *agent.PageContext `json:"page,omitempty"`
	Surface               agent.Surface      `json:"surface,omitempty"`
	Mentions              []agent.EntityRef  `json:"mentions,omitempty"`
	AttachmentDocumentIDs []pulid.ID         `json:"attachmentDocumentIds,omitempty"`
	ProviderID            pulid.ID           `json:"providerId,omitempty"`
	ProviderChosen        bool               `json:"providerChosen,omitempty"`
	// DirectedAgentID hands the message to another agent the person chose,
	// once the conversation is free. It is a task for that agent, never a
	// word to the reply under way, so it cannot steer.
	DirectedAgentID pulid.ID `json:"directedAgentId,omitempty"`
}

// Directed reports a message handed to another agent.
func (r *QueuedRequest) Directed() bool {
	return r != nil && r.DirectedAgentID.IsNotNil()
}

type QueuedMessage struct {
	keel.Table `keel:"assistant_queued_messages"`

	ID             pulid.ID       `json:"id" keel:"id,pk,autoid:aqm_"`
	OrganizationID pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	ThreadID       pulid.ID       `json:"threadId" keel:"thread_id"`
	UserID         pulid.ID       `json:"userId" keel:"user_id"`
	Content        string         `json:"content" keel:"content"`
	Request        *QueuedRequest `json:"request" keel:"request,jsonb"`
	Position       int64          `json:"position" keel:"position"`
	Steer          bool           `json:"steer" keel:"steer"`
	Version        int64          `json:"version" keel:"version,version"`
	CreatedAt      int64          `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64          `json:"updatedAt" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"-"`
	BusinessUnit *tenant.BusinessUnit `json:"-"`
	User         *tenant.User         `json:"-" keel:"belongs:user_id"`
	Thread       *Thread              `json:"-" keel:"belongs:thread_id"`
}

func (m *QueuedMessage) Normalize() {
	m.Content = strings.TrimSpace(m.Content)
	if m.Request == nil {
		m.Request = &QueuedRequest{}
	}
	m.Request.Page = m.Request.Page.Normalized()
	m.Request.Mentions = agent.NormalizeEntityRefs(m.Request.Mentions)
}

func (m *QueuedMessage) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&m.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&m.ThreadID, validation.Required.Error("Conversation is required")),
		validation.Field(&m.UserID, validation.Required.Error("User is required")),
	))
	ValidateQueuedContent("content", m.Content, multiErr)
	if m.Request == nil {
		return
	}
	if m.Request.Page != nil {
		m.Request.Page.Validate("context", multiErr)
	}
	m.Request.Surface.Validate("surface", multiErr)
	agent.ValidateEntityRefs("mentions", m.Request.Mentions, multiErr)
	if len(m.Request.AttachmentDocumentIDs) > MaxMessageAttachments {
		multiErr.Add("attachmentDocumentIds", errortypes.ErrInvalid,
			fmt.Sprintf("At most %d files can be attached to one message", MaxMessageAttachments))
	}
	if m.Steer && len(m.Request.AttachmentDocumentIDs) > 0 {
		multiErr.Add("attachmentDocumentIds", errortypes.ErrInvalid,
			"Files go with a queued message; a message that steers the reply carries words only")
	}
}

func ValidateQueuedContent(field, content string, multiErr *errortypes.MultiError) {
	switch {
	case strings.TrimSpace(content) == "":
		multiErr.Add(field, errortypes.ErrRequired, "Message cannot be empty")
	case utf8.RuneCountInString(content) > MaxQueuedContentRunes:
		multiErr.Add(field, errortypes.ErrInvalid,
			fmt.Sprintf("A message can be at most %d characters", MaxQueuedContentRunes))
	}
}

func (m *QueuedMessage) GetID() pulid.ID { return m.ID }

func (m *QueuedMessage) GetOrganizationID() pulid.ID { return m.OrganizationID }

func (m *QueuedMessage) GetBusinessUnitID() pulid.ID { return m.BusinessUnitID }

func (m *QueuedMessage) GetTableName() string { return queuedMessageTableName }
