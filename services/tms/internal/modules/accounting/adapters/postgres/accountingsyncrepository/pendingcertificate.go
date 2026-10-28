package accountingsyncrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const pendingCertificateEntity = "Pending accounting certificate"

var pendingCertificateErrors = dberror.Mapping{Entity: pendingCertificateEntity}

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
) accountingconnectionports.AccountingPendingCertificateRepository {
	return &pendingCertificateRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-pending-certificate-repository"),
	}
}

func (r *pendingCertificateRepository) GetByType(
	ctx context.Context,
	req accountingconnectionports.GetAccountingConnectionRequest,
) (*accountingsync.AccountingPendingCertificate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingPendingCertificate, error) {
			pc := accountingsyncdb.AccountingPendingCertificates
			entity, err := pc.Where(pc.IntegrationType.Eq(req.IntegrationType)).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, pendingCertificateErrors.Map(err, "")
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
			if err := r.deleteByType(ctx, accountingconnectionports.GetAccountingConnectionRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
				IntegrationType: entity.IntegrationType,
			}); err != nil {
				return nil, err
			}

			err := accountingsyncdb.AccountingPendingCertificates.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			)
			if err != nil {
				return nil, pendingCertificateErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *pendingCertificateRepository) Delete(
	ctx context.Context,
	req accountingconnectionports.GetAccountingConnectionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return r.deleteByType(ctx, req)
	})
}

func (r *pendingCertificateRepository) deleteByType(
	ctx context.Context,
	req accountingconnectionports.GetAccountingConnectionRequest,
) error {
	pc := accountingsyncdb.AccountingPendingCertificates
	_, err := pc.Where(pc.IntegrationType.Eq(req.IntegrationType)).Delete(ctx, r.db.Keel(ctx))
	return err
}
