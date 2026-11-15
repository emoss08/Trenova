package emailrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email/emaildb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/mailtext"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type DirectoryParams struct {
	fx.In

	DB *postgres.Connection
}

type directory struct {
	db *postgres.Connection
}

func NewRecipientDirectory(p DirectoryParams) emailports.RecipientDirectory {
	return &directory{db: p.DB}
}

func NewSystemMessageLocator(p DirectoryParams) emailports.SystemMessageLocator {
	return &directory{db: p.DB}
}

func (d *directory) MemberAddresses(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	addresses []string,
) ([]string, error) {
	normalized := mailtext.NormalizeEmailList(addresses)
	if len(normalized) == 0 {
		return []string{}, nil
	}
	ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())
	return dbtx.Read(ctx, d.db, func(ctx context.Context) ([]string, error) {
		u := tenantdb.Users
		m := tenantdb.OrganizationMemberships
		address := u.EmailAddress.Lower()
		members, err := keel.Into[string](
			ctx,
			d.db.Keel(ctx),
			u.Join(m, m.UserID.EqCol(u.ID)).
				Where(
					address.In(normalized...),
					keel.Or(m.ExpiresAt.IsNull(), m.ExpiresAt.Gt(timeutils.NowUnix())),
				).
				Group(address).
				Pick(address).
				Limit(len(normalized)),
		)
		if err != nil {
			return nil, err
		}
		if members == nil {
			members = []string{}
		}
		return members, nil
	})
}

type systemMessageRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

const systemMessageTenantReason = "find which organization a message the Trenova sender carried belongs to, before its tenant is known"

func (d *directory) TenantOfSystemMessage(
	ctx context.Context,
	provider email.Provider,
	providerMessageID string,
) (pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, systemMessageTenantReason)
	return dbtx.Read(ctx, d.db, func(ctx context.Context) (pagination.TenantInfo, error) {
		m := emaildb.Messages
		row, err := keel.IntoOne[systemMessageRow](
			ctx,
			d.db.Keel(ctx),
			m.AllTenants().
				Pick(m.OrganizationID, m.BusinessUnitID).
				Where(
					m.Sender.Eq(email.SenderTrenova),
					m.Provider.Eq(provider),
					m.ProviderMessageID.Eq(providerMessageID),
				).
				Limit(1),
		)
		if err != nil {
			return pagination.TenantInfo{}, messageErrors.Map(err, providerMessageID)
		}
		return pagination.TenantInfo{OrgID: row.OrganizationID, BuID: row.BusinessUnitID}, nil
	})
}
