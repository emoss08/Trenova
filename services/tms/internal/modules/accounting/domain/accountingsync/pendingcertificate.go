package accountingsync

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*AccountingPendingCertificate)(nil)

type AccountingPendingCertificate struct {
	bun.BaseModel `bun:"table:accounting_pending_certificates,alias:acctpc" json:"-"`
	keel.Table    `keel:"accounting_pending_certificates"`

	ID                   pulid.ID         `json:"id"              bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:acctpc_"`
	BusinessUnitID       pulid.ID         `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID         `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	IntegrationType      integration.Type `json:"integrationType" bun:"integration_type,type:VARCHAR(50),notnull" keel:"integration_type"`
	PrivateKeyCiphertext string           `json:"-"               bun:"private_key_ciphertext,type:TEXT,notnull" keel:"private_key_ciphertext"`
	CertificatePEM       string           `json:"certificatePem"  bun:"certificate_pem,type:TEXT,notnull" keel:"certificate_pem"`
	ExpiresAt            int64            `json:"expiresAt"       bun:"expires_at,type:BIGINT,notnull" keel:"expires_at"`
	PreparedByID         pulid.ID         `json:"preparedById"    bun:"prepared_by_id,type:VARCHAR(100),notnull" keel:"prepared_by_id"`
	PreparedAt           int64            `json:"preparedAt"      bun:"prepared_at,type:BIGINT,notnull" keel:"prepared_at"`
	CreatedAt            int64            `json:"createdAt"       bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt            int64            `json:"updatedAt"       bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	PreparedBy   *tenant.User         `json:"preparedBy,omitempty"   bun:"rel:belongs-to,join:prepared_by_id=id" keel:"belongs:prepared_by_id"`
}

func (c *AccountingPendingCertificate) GetTableName() string {
	return "accounting_pending_certificates"
}

func (c *AccountingPendingCertificate) GetID() pulid.ID { return c.ID }

func (c *AccountingPendingCertificate) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *AccountingPendingCertificate) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *AccountingPendingCertificate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("acctpc_")
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

func (c *AccountingPendingCertificate) ClientCertificate(
	keyCiphertext, certificateID string,
) ClientCertificate {
	return ClientCertificate{
		KeyCiphertext: keyCiphertext,
		CertificateID: certificateID,
		ExpiresAt:     c.ExpiresAt,
	}
}
