package dashcontrolrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) repositories.DashControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dash-control-repository"),
	}
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.DashControl, error) {
	entity, err := r.selectControl(ctx, tenantInfo)
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, dberror.HandleNotFoundError(err, "DashControl")
	}

	control := &tenant.DashControl{
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
	if _, err = r.db.DBForContext(ctx).
		NewInsert().
		Model(control).
		On("CONFLICT (organization_id, business_unit_id) DO NOTHING").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("create default dash control: %w", err)
	}

	entity, err = r.selectControl(ctx, tenantInfo)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DashControl")
	}
	return entity, nil
}

func (r *repository) selectControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.DashControl, error) {
	entity := new(tenant.DashControl)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dashc.organization_id = ?", tenantInfo.OrgID).
		Where("dashc.business_unit_id = ?", tenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.DashControl,
) (*tenant.DashControl, error) {
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("require_load_acknowledgment = ?", entity.RequireLoadAcknowledgment).
		Set("allow_load_refusals = ?", entity.AllowLoadRefusals).
		Set("allow_stop_actions = ?", entity.AllowStopActions).
		Set("allow_load_document_upload = ?", entity.AllowLoadDocumentUpload).
		Set("allow_load_comments = ?", entity.AllowLoadComments).
		Set("show_load_pay = ?", entity.ShowLoadPay).
		Set("show_pay_estimates = ?", entity.ShowPayEstimates).
		Set("allow_expense_submission = ?", entity.AllowExpenseSubmission).
		Set("require_expense_receipt = ?", entity.RequireExpenseReceipt).
		Set("allow_settlement_disputes = ?", entity.AllowSettlementDisputes).
		Set("allow_profile_document_upload = ?", entity.AllowProfileDocumentUpload).
		Set("allow_contact_info_edit = ?", entity.AllowContactInfoEdit).
		Set("allow_pto_requests = ?", entity.AllowPtoRequests).
		Set("send_credential_reminders = ?", entity.SendCredentialReminders).
		Set("enable_detention_alerts = ?", entity.EnableDetentionAlerts).
		Set("detention_alert_threshold_minutes = ?", entity.DetentionAlertThresholdMinutes).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update dash control: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "DashControl", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetOrCreate(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
}
