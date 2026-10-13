//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package accountingsyncrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/pkg/buncolgen/accountingsynccol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const pendingCertificateEntity = "Pending accounting certificate"

type PendingCertificateParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type pendingCertificateRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPendingCertificateRepository(
	p PendingCertificateParams,
) accountingsyncports.AccountingPendingCertificateRepository {
	return &pendingCertificateRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-pending-certificate-repository"),
	}
}

func (r *pendingCertificateRepository) GetByType(
	ctx context.Context,
	req accountingsyncports.GetAccountingConnectionRequest,
) (*accountingsync.AccountingPendingCertificate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingPendingCertificate, error) {
			entity := new(accountingsync.AccountingPendingCertificate)
			cols := accountingsynccol.AccountingPendingCertificateColumns

			if err := r.db.DBForContext(ctx).
				NewSelect().
				Model(entity).
				Where(cols.IntegrationType.Eq(), req.IntegrationType).
				Apply(accountingsynccol.AccountingPendingCertificateApplyTenant(req.TenantInfo)).
				Scan(ctx); err != nil {
				return nil, dberror.HandleNotFoundError(err, pendingCertificateEntity)
			}

			return entity, nil
		},
	)
}

func (r *pendingCertificateRepository) Replace(
	ctx context.Context,
	entity *accountingsync.AccountingPendingCertificate,
) (*accountingsync.AccountingPendingCertificate, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingPendingCertificate, error) {
			if err := r.deleteByType(ctx, accountingsyncports.GetAccountingConnectionRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
				IntegrationType: entity.IntegrationType,
			}); err != nil {
				return nil, err
			}

			if _, err := r.db.DBForContext(ctx).
				NewInsert().
				Model(entity).
				Returning("*").
				Exec(ctx); err != nil {
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *pendingCertificateRepository) Delete(
	ctx context.Context,
	req accountingsyncports.GetAccountingConnectionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return r.deleteByType(ctx, req)
	})
}

func (r *pendingCertificateRepository) deleteByType(
	ctx context.Context,
	req accountingsyncports.GetAccountingConnectionRequest,
) error {
	cols := accountingsynccol.AccountingPendingCertificateColumns

	_, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*accountingsync.AccountingPendingCertificate)(nil)).
		WhereGroup(" AND ", func(q *bun.DeleteQuery) *bun.DeleteQuery {
			return accountingsynccol.AccountingPendingCertificateScopeTenantDelete(q, req.TenantInfo).
				Where(cols.IntegrationType.Eq(), req.IntegrationType)
		}).
		Exec(ctx)
	return err
}
