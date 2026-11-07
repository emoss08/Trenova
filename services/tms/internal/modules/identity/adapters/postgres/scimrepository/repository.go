package scimrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	credentialScopeReason = "resolve a SCIM bearer token by its prefix before its directory, and so its tenant, is known"
	touchInterval         = 60
)

var (
	credentialErrors   = dberror.Mapping{Entity: "SCIM token"}
	userErrors         = dberror.Mapping{Entity: "SCIM user"}
	groupErrors        = dberror.Mapping{Entity: "SCIM group"}
	memberErrors       = dberror.Mapping{Entity: "SCIM group member"}
	grantErrors        = dberror.Mapping{Entity: "SCIM role grant"}
	provisioningErrors = dberror.Mapping{Entity: "Provisioning record"}
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) scimports.Repository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.scim-repository"),
	}
}

type credentialRow struct {
	TokenID          pulid.ID
	DirectoryID      pulid.ID
	OrganizationID   pulid.ID
	BusinessUnitID   pulid.ID
	TokenHash        string
	Status           iam.SCIMTokenStatus
	ExpiresAt        int64
	LastUsedAt       int64
	DirectoryEnabled bool
	TenantSlug       string
}

func (r *repository) FindCredential(
	ctx context.Context,
	prefix string,
) (*scimports.Credential, error) {
	ctx = dbscope.WithSystem(ctx, credentialScopeReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*scimports.Credential, error) {
		st := iamdb.SCIMTokens
		sd := iamdb.SCIMDirectories

		row, err := keel.IntoOne[credentialRow](
			ctx,
			r.db.Keel(ctx),
			st.AllTenants().
				Join(sd, sd.ID.EqCol(st.DirectoryID), sd.OrganizationID.EqCol(st.OrganizationID)).
				Pick(
					st.ID,
					st.DirectoryID,
					st.OrganizationID,
					sd.BusinessUnitID,
					st.TokenHash,
					st.Status,
					st.ExpiresAt.Coalesce(0),
					st.LastUsedAt.Coalesce(0),
					sd.Enabled,
					sd.TenantSlug,
				).
				Where(st.Prefix.Eq(prefix)).
				Limit(1),
		)
		if err != nil {
			return nil, credentialErrors.Map(err, prefix)
		}

		credential := scimports.Credential(row)
		return &credential, nil
	})
}

func (r *repository) TouchCredential(ctx context.Context, tokenID pulid.ID, now int64) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		st := iamdb.SCIMTokens
		_, err := st.Where(
			st.ID.Eq(tokenID),
			keel.Or(st.LastUsedAt.IsNull(), st.LastUsedAt.Lt(now-touchInterval)),
		).Update(ctx, r.db.Keel(ctx), st.LastUsedAt.To(now))
		if err != nil {
			return credentialErrors.Map(err, tokenID.String())
		}

		return nil
	})
}

func (r *repository) RecordProvisioning(
	ctx context.Context,
	record *iam.ProvisioningAuditRecord,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := iamdb.ProvisioningAuditRecords.Create(ctx, r.db.Keel(ctx), record); err != nil {
			return provisioningErrors.Map(err, record.ID.String())
		}

		return nil
	})
}
