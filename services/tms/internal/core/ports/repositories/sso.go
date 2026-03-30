package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type SSOConfigRepository interface {
	GetByOrganizationID(ctx context.Context, organizationID pulid.ID) (*tenant.SSOConfig, error)
	GetEnabledByOrganizationID(ctx context.Context, organizationID pulid.ID) (*tenant.SSOConfig, error)
	Save(ctx context.Context, config *tenant.SSOConfig) (*tenant.SSOConfig, error)
}

type SSOLoginState struct {
	State            string   `json:"state"`
	OrganizationID   pulid.ID `json:"organizationId"`
	OrganizationSlug string   `json:"organizationSlug"`
	CodeVerifier     string   `json:"codeVerifier"`
	Nonce            string   `json:"nonce"`
	ReturnTo         string   `json:"returnTo"`
}

type SSOLoginStateRepository interface {
	Save(ctx context.Context, state *SSOLoginState, ttl time.Duration) error
	Get(ctx context.Context, state string) (*SSOLoginState, error)
	Delete(ctx context.Context, state string) error
}
