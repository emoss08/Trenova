package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type AttachmentMeta struct {
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	URL         string `json:"url"`
	ContentID   string `json:"contentId,omitempty"`
}

type SystemTemplateKey string

const (
	TemplateUserWelcome               SystemTemplateKey = "user-welcome"
	TemplateShipmentOwnershipTransfer SystemTemplateKey = "shipment-ownership-transfer"
)

type TestConnectionRequest struct {
	ProviderType email.ProviderType `json:"providerType" form:"providerType"`
	Host         string             `json:"host"         form:"host"`
	Port         int                `json:"port"         form:"port"`
	Username     string             `json:"username"     form:"username"`
	Password     string             `json:"password"     form:"password"`
	APIKey       string             `json:"apiKey"       form:"apiKey"`
}

type SendSystemEmailRequest struct {
	TemplateKey SystemTemplateKey
	To          []string
	Variables   map[string]any
	OrgID       pulid.ID
	BuID        pulid.ID
	UserID      pulid.ID
}

type EmailService interface {
	SendEmail(ctx context.Context, req *SendEmailRequest) error
	SendSystemEmail(
		ctx context.Context,
		req *SendSystemEmailRequest,
	) error
	TestConnection(
		ctx context.Context,
		req *TestConnectionRequest,
	) (success bool, err error)
}

type EmailProfileService interface {
	Create(
		ctx context.Context,
		profile *email.EmailProfile,
		userID pulid.ID,
	) (*email.EmailProfile, error)
	Update(
		ctx context.Context,
		profile *email.EmailProfile,
		userID pulid.ID,
	) (*email.EmailProfile, error)
	Get(
		ctx context.Context,
		req repositories.GetEmailProfileByIDRequest,
	) (*email.EmailProfile, error)
	List(
		ctx context.Context,
		req *repositories.ListEmailProfileRequest,
	) (*pagination.ListResult[*email.EmailProfile], error)
	SetDefault(ctx context.Context, req repositories.GetEmailProfileByIDRequest) error
	GetDefault(
		ctx context.Context,
		req repositories.GetEmailProfileByIDRequest,
	) (*email.EmailProfile, error)
}

type SendEmailRequest struct {
	OrganizationID pulid.ID         `json:"organizationId"`
	BusinessUnitID pulid.ID         `json:"businessUnitId"`
	UserID         pulid.ID         `json:"userId"`
	ProfileID      *pulid.ID        `json:"profileId,omitempty"`
	To             []string         `json:"to"`
	CC             []string         `json:"cc,omitempty"`
	BCC            []string         `json:"bcc,omitempty"`
	Subject        string           `json:"subject"`
	HTMLBody       string           `json:"htmlBody"`
	TextBody       string           `json:"textBody,omitempty"`
	Attachments    []AttachmentMeta `json:"attachments,omitempty"`
	Priority       email.Priority   `json:"priority,omitempty"`
	Metadata       map[string]any   `json:"metadata,omitempty"`
}

func (r *SendEmailRequest) Validate() error {
	return validation.ValidateStruct(
		r,
		validation.Field(
			&r.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&r.BusinessUnitID,
			validation.Required.Error("Business Unit ID is required"),
		),
		validation.Field(&r.To,
			validation.Required.Error("At least one recipient is required"),
			validation.Each(is.Email.Error("Invalid email address in To field")),
		),
		validation.Field(
			&r.CC,
			validation.Each(is.Email.Error("Invalid email address in CC field")),
		),
		validation.Field(
			&r.BCC,
			validation.Each(is.Email.Error("Invalid email address in BCC field")),
		),
		validation.Field(&r.Subject,
			validation.Required.Error("Subject is required"),
			validation.Length(1, 500).Error("Subject must be between 1 and 500 characters"),
		),
		validation.Field(&r.TextBody,
			validation.Length(0, 5242880).Error("Text body must not exceed 5MB"),
		),
	)
}

type SendTemplatedEmailRequest struct {
	OrganizationID pulid.ID         `json:"organizationId"`
	BusinessUnitID pulid.ID         `json:"businessUnitId"`
	ProfileID      *pulid.ID        `json:"profileId,omitempty"`
	TemplateID     pulid.ID         `json:"templateId"`
	To             []string         `json:"to"`
	CC             []string         `json:"cc,omitempty"`
	BCC            []string         `json:"bcc,omitempty"`
	Variables      map[string]any   `json:"variables"`
	Attachments    []AttachmentMeta `json:"attachments,omitempty"`
	Priority       email.Priority   `json:"priority,omitempty"`
	Metadata       map[string]any   `json:"metadata,omitempty"`
}

func (r *SendTemplatedEmailRequest) Validate() error {
	return validation.ValidateStruct(
		r,
		validation.Field(
			&r.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&r.BusinessUnitID,
			validation.Required.Error("Business Unit ID is required"),
		),
		validation.Field(&r.TemplateID, validation.Required.Error("Template ID is required")),
		validation.Field(&r.To,
			validation.Required.Error("At least one recipient is required"),
			validation.Each(is.Email.Error("Invalid email address in To field")),
		),
		validation.Field(
			&r.CC,
			validation.Each(is.Email.Error("Invalid email address in CC field")),
		),
		validation.Field(
			&r.BCC,
			validation.Each(is.Email.Error("Invalid email address in BCC field")),
		),
		validation.Field(
			&r.Variables,
			validation.Required.Error("Template variables are required"),
		),
	)
}

type SendEmailResponse struct {
	QueueID   pulid.ID `json:"queueId"`
	MessageID string   `json:"messageId,omitempty"`
	Status    string   `json:"status"`
}
