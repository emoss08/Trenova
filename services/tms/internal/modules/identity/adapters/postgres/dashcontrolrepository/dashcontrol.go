package dashcontrolrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var dashControlErrors = dberror.Mapping{Entity: "DashControl"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.DashControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dash-control-repository"),
	}
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.DashControl, error) {
	newDefault := func() *tenant.DashControl {
		return &tenant.DashControl{
			ID:                             pulid.MustNew("dashc_"),
			BusinessUnitID:                 tenantInfo.BuID,
			OrganizationID:                 tenantInfo.OrgID,
			RequireLoadAcknowledgment:      true,
			AllowLoadRefusals:              true,
			AllowStopActions:               true,
			AllowLoadDocumentUpload:        true,
			AllowLoadComments:              true,
			ShowLoadPay:                    true,
			ShowPayEstimates:               true,
			AllowExpenseSubmission:         true,
			AllowSettlementDisputes:        true,
			AllowProfileDocumentUpload:     true,
			AllowContactInfoEdit:           true,
			AllowPtoRequests:               true,
			SendCredentialReminders:        true,
			EnableDetentionAlerts:          true,
			DetentionAlertThresholdMinutes: 120,
		}
	}

	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*tenant.DashControl]{
		Find: r.selectControl,
		Create: func(ctx context.Context) error {
			dc := tenantdb.DashControls
			if _, err := dc.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*tenant.DashControl{newDefault()},
				dc.OrganizationID,
				dc.BusinessUnitID,
			); err != nil {
				return fmt.Errorf("create default dash control: %w", err)
			}

			return nil
		},
		Default: newDefault,
		MapError: func(err error) error {
			return dashControlErrors.Map(err, tenantInfo.OrgID.String())
		},
	})
}

func (r *repository) selectControl(ctx context.Context) (*tenant.DashControl, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.DashControl, error) {
		return tenantdb.DashControls.Cache(resultcache.Settings()...).One(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.DashControl,
) (*tenant.DashControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.DashControl, error) {
		dc := tenantdb.DashControls
		if err := dc.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			dc.RequireLoadAcknowledgment,
			dc.AllowLoadRefusals,
			dc.AllowStopActions,
			dc.AllowLoadDocumentUpload,
			dc.AllowLoadComments,
			dc.ShowLoadPay,
			dc.ShowPayEstimates,
			dc.AllowExpenseSubmission,
			dc.RequireExpenseReceipt,
			dc.AllowSettlementDisputes,
			dc.AllowProfileDocumentUpload,
			dc.AllowContactInfoEdit,
			dc.AllowPtoRequests,
			dc.SendCredentialReminders,
			dc.RequireContactChangeApproval,
			dc.EnableDetentionAlerts,
			dc.DetentionAlertThresholdMinutes,
		); err != nil {
			return nil, dashControlErrors.Map(err, entity.ID.String())
		}

		return r.GetOrCreate(ctx, pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		})
	})
}
