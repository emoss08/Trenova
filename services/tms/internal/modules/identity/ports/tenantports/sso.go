package tenantports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/sso"
	"github.com/emoss08/trenova/shared/pulid"
)

type SSOConfigRepository interface {
	ListEnabledByOrganizationID(
		ctx context.Context,
		organizationID pulid.ID,
	) ([]*sso.SSOConfig, error)
	GetEnabledByID(ctx context.Context, providerID pulid.ID) (*sso.SSOConfig, error)
	GetByOrganizationID(
		ctx context.Context,
		organizationID pulid.ID,
		provider sso.SSOProvider,
	) (*sso.SSOConfig, error)
	GetEnabledByOrganizationID(
		ctx context.Context,
		organizationID pulid.ID,
		provider sso.SSOProvider,
	) (*sso.SSOConfig, error)
	Save(ctx context.Context, config *sso.SSOConfig) (*sso.SSOConfig, error)
}

type SSOLoginState struct {
	State            string          `json:"state"`
	Provider         sso.SSOProvider `json:"provider"`
	ProviderID       pulid.ID        `json:"providerId"`
	OrganizationID   pulid.ID        `json:"organizationId"`
	BusinessUnitID   pulid.ID        `json:"businessUnitId"`
	OrganizationSlug string          `json:"organizationSlug"`
	CodeVerifier     string          `json:"codeVerifier"`
	Nonce            string          `json:"nonce"`
	ReturnTo         string          `json:"returnTo"`
}

type SSOLoginStateRepository interface {
	Save(ctx context.Context, state *SSOLoginState, ttl time.Duration) error
	Get(ctx context.Context, state string) (*SSOLoginState, error)
	Delete(ctx context.Context, state string) error
}
