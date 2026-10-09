package conversation

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	MaxMessageAttachments  = 5
	MaxQueuedContentRunes  = 16_000
	MaxQueuedPerThread     = 20
	QueuedMessageIDPrefix  = "aqm_"
	queuedMessageTableName = "assistant_queued_messages"
)

var (
	_ bun.BeforeAppendModelHook          = (*QueuedMessage)(nil)
	_ validationframework.TenantedEntity = (*QueuedMessage)(nil)
)

type QueuedRequest struct {
	Page                  *agent.PageContext `json:"page,omitempty"`
	Mentions              []agent.EntityRef  `json:"mentions,omitempty"`
	AttachmentDocumentIDs []pulid.ID         `json:"attachmentDocumentIds,omitempty"`
	ProviderID            pulid.ID           `json:"providerId,omitempty"`
	ProviderChosen        bool               `json:"providerChosen,omitempty"`
}

type QueuedMessage struct {
	bun.BaseModel `bun:"table:assistant_queued_messages,alias:aqm" json:"-"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	ThreadID       pulid.ID       `json:"threadId"       bun:"thread_id,type:VARCHAR(100),notnull"`
	UserID         pulid.ID       `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	Content        string         `json:"content"        bun:"content,type:TEXT,notnull"`
	Request        *QueuedRequest `json:"request"        bun:"request,type:JSONB,notnull"`
	Position       int64          `json:"position"       bun:"position,type:BIGINT,notnull"`
	Steer          bool           `json:"steer"          bun:"steer,type:BOOLEAN,notnull,default:false"`
	Version        int64          `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64          `json:"updatedAt"      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	User         *tenant.User         `json:"-" bun:"rel:belongs-to,join:user_id=id"`
	Thread       *Thread              `json:"-" bun:"rel:belongs-to,join:thread_id=id"`
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

func (m *QueuedMessage) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if m.ID.IsNil() {
			m.ID = pulid.MustNew(QueuedMessageIDPrefix)
		}
		if m.CreatedAt == 0 {
			m.CreatedAt = now
		}
		m.UpdatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}

	return nil
}

func (m *QueuedMessage) GetID() pulid.ID { return m.ID }

func (m *QueuedMessage) GetOrganizationID() pulid.ID { return m.OrganizationID }

func (m *QueuedMessage) GetBusinessUnitID() pulid.ID { return m.BusinessUnitID }

func (m *QueuedMessage) GetTableName() string { return queuedMessageTableName }
