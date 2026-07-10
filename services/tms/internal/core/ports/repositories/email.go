package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEmailProfilesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListEmailProfileConnectionRequest struct {
	Filter              *pagination.QueryOptions `json:"filter"`
	Cursor              pagination.CursorInfo    `json:"-"`
	EmailProfileColumns []string                 `json:"-"`
}

type EmailProfileSelectOptionsRequest struct {
	*pagination.SelectQueryRequest
}

type ListEmailMessagesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListEmailSuppressionsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetEmailEntityRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetEmailMessageByProviderIDRequest struct {
	Provider          email.Provider        `json:"provider"`
	ProviderMessageID string                `json:"providerMessageId"`
	TenantInfo        pagination.TenantInfo `json:"tenantInfo"`
}

type ListEmailAttachmentsRequest struct {
	MessageID  pulid.ID              `json:"messageId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetEmailWebhookConfigRequest struct {
	IntegrationType integration.Type `json:"integrationType"`
	Token           string           `json:"token"`
}

type EmailWebhookConfig struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	SigningSecret string                `json:"signingSecret"`
}

type EmailRepository interface {
	ListProfiles(context.Context, *ListEmailProfilesRequest) (*pagination.ListResult[*email.Profile], error)
	ListProfilesConnection(context.Context, *ListEmailProfileConnectionRequest) (*pagination.CursorListResult[*email.Profile], error)
	SelectProfileOptions(context.Context, *EmailProfileSelectOptionsRequest) (*pagination.ListResult[*email.Profile], error)
	GetProfile(context.Context, GetEmailEntityRequest) (*email.Profile, error)
	CreateProfile(context.Context, *email.Profile) (*email.Profile, error)
	UpdateProfile(context.Context, *email.Profile) (*email.Profile, error)
	DeleteProfile(context.Context, GetEmailEntityRequest) error
	ListAssignments(context.Context, pagination.TenantInfo) ([]*email.ProfileAssignment, error)
	UpsertAssignments(context.Context, pagination.TenantInfo, []*email.ProfileAssignment) ([]*email.ProfileAssignment, error)
	GetAssignedProfile(context.Context, pagination.TenantInfo, email.Purpose) (*email.Profile, error)
	CreateMessage(context.Context, *email.Message) (*email.Message, error)
	UpdateMessage(context.Context, *email.Message) (*email.Message, error)
	GetMessage(context.Context, GetEmailEntityRequest) (*email.Message, error)
	GetMessageByProviderID(context.Context, GetEmailMessageByProviderIDRequest) (*email.Message, error)
	CreateAttachments(context.Context, []*email.Attachment) ([]*email.Attachment, error)
	ListAttachments(context.Context, ListEmailAttachmentsRequest) ([]*email.Attachment, error)
	GetEmailWebhookConfig(context.Context, GetEmailWebhookConfigRequest) (*EmailWebhookConfig, error)
	ListMessages(context.Context, *ListEmailMessagesRequest) (*pagination.ListResult[*email.Message], error)
	CreateEvent(context.Context, *email.Event) (bool, error)
	ListSuppressions(context.Context, *ListEmailSuppressionsRequest) (*pagination.ListResult[*email.Suppression], error)
	CreateSuppression(context.Context, *email.Suppression) (*email.Suppression, error)
	DeleteSuppression(context.Context, GetEmailEntityRequest) error
	HasSuppression(context.Context, pagination.TenantInfo, string) (bool, error)
}
