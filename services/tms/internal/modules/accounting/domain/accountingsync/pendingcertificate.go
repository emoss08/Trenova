package accountingsync

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/pulid"
)

type AccountingPendingCertificate struct {
	keel.Table `keel:"accounting_pending_certificates"`

	ID                   pulid.ID         `json:"id" keel:"id,pk,autoid:acctpc_"`
	BusinessUnitID       pulid.ID         `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID         `json:"organizationId" keel:"organization_id,pk,tenant"`
	IntegrationType      integration.Type `json:"integrationType" keel:"integration_type"`
	PrivateKeyCiphertext string           `json:"-" keel:"private_key_ciphertext"`
	CertificatePEM       string           `json:"certificatePem" keel:"certificate_pem"`
	ExpiresAt            int64            `json:"expiresAt" keel:"expires_at"`
	PreparedByID         pulid.ID         `json:"preparedById" keel:"prepared_by_id"`
	PreparedAt           int64            `json:"preparedAt" keel:"prepared_at"`
	CreatedAt            int64            `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt            int64            `json:"updatedAt" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	PreparedBy   *tenant.User         `json:"preparedBy,omitempty" keel:"belongs:prepared_by_id"`
}

func (c *AccountingPendingCertificate) GetTableName() string {
	return "accounting_pending_certificates"
}

func (c *AccountingPendingCertificate) GetID() pulid.ID { return c.ID }

func (c *AccountingPendingCertificate) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *AccountingPendingCertificate) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *AccountingPendingCertificate) ClientCertificate(
	keyCiphertext, certificateID string,
) ClientCertificate {
	return ClientCertificate{
		KeyCiphertext: keyCiphertext,
		CertificateID: certificateID,
		ExpiresAt:     c.ExpiresAt,
	}
}
