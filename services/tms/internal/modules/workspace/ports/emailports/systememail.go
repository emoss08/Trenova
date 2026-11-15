package emailports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/systememail"
	"github.com/emoss08/trenova/pkg/pagination"
)

type EmailRouting struct {
	Assignments            []*email.ProfileAssignment `json:"assignments"`
	SystemEmailInUse       bool                       `json:"systemEmailInUse"`
	SystemEmailFromAddress string                     `json:"systemEmailFromAddress"`
}

type SystemSenderRepository interface {
	Get(ctx context.Context) (*systememail.Sender, error)
	Save(ctx context.Context, sender *systememail.Sender) error
	Delete(ctx context.Context) (bool, error)
}

type SystemSenders interface {
	Current(ctx context.Context) (*systememail.Credential, error)
}

type RecipientDirectory interface {
	MemberAddresses(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		addresses []string,
	) ([]string, error)
}

type SystemMessageLocator interface {
	TenantOfSystemMessage(
		ctx context.Context,
		provider email.Provider,
		providerMessageID string,
	) (pagination.TenantInfo, error)
}
