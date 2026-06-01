package servicefailurereasoncoderepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	coreports "github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.ServiceFailureReasonCodeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.service-failure-reason-code-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListServiceFailureReasonCodesRequest,
) (*pagination.ListResult[*servicefailure.ReasonCode], error) {
	if req.Filter.Pagination.Limit <= 0 {
		req.Filter.Pagination.Limit = 50
	}

	entities := make([]*servicefailure.ReasonCode, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(q *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFilters(q, "sfrc", req.Filter, (*servicefailure.ReasonCode)(nil)).
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset())
		}).
		Order("sfrc.sort_order ASC", "sfrc.code ASC").
		ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list service failure reason codes: %w", err)
	}

	return &pagination.ListResult[*servicefailure.ReasonCode]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetServiceFailureReasonCodeByIDRequest,
) (*servicefailure.ReasonCode, error) {
	entity := new(servicefailure.ReasonCode)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("sfrc.id = ?", req.ID).
		Where("sfrc.organization_id = ?", req.TenantInfo.OrgID).
		Where("sfrc.business_unit_id = ?", req.TenantInfo.BuID).
		Relation("ArchivedBy").
		Relation("ActivatedBy").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Service failure reason code")
	}

	return entity, nil
}

func (r *repository) FindDefault(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	appliesTo servicefailure.ReasonCodeAppliesTo,
) (*servicefailure.ReasonCode, error) {
	entity := new(servicefailure.ReasonCode)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("sfrc.organization_id = ?", tenantInfo.OrgID).
		Where("sfrc.business_unit_id = ?", tenantInfo.BuID).
		Where("sfrc.active = TRUE").
		Where("sfrc.applies_to IN (?)", bun.In([]servicefailure.ReasonCodeAppliesTo{
			appliesTo,
			servicefailure.ReasonCodeAppliesToBoth,
		})).
		OrderExpr("CASE WHEN sfrc.applies_to = ? THEN 0 ELSE 1 END", appliesTo).
		Order("sfrc.sort_order ASC").
		Order("sfrc.code ASC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Service failure reason code")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *servicefailure.ReasonCode,
) (*servicefailure.ReasonCode, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, mapReasonCodeConstraint(err)
	}

	return r.GetByID(ctx, repositories.GetServiceFailureReasonCodeByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo(entity),
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *servicefailure.ReasonCode,
) (*servicefailure.ReasonCode, error) {
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*servicefailure.ReasonCode)(nil)).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("code = ?", entity.Code).
		Set("label = ?", entity.Label).
		Set("description = ?", entity.Description).
		Set("category = ?", entity.Category).
		Set("applies_to = ?", entity.AppliesTo).
		Set("default_status_code = ?", entity.DefaultStatusCode).
		Set("default_reason_code = ?", entity.DefaultReasonCode).
		Set("default_exception_code = ?", entity.DefaultExceptionCode).
		Set("default_note = ?", entity.DefaultNote).
		Set("active = ?", entity.Active).
		Set("sort_order = ?", entity.SortOrder).
		Set("external_map = ?", entity.ExternalMap).
		Set("version = version + 1").
		Set("updated_at = ?", timeutils.NowUnix()).
		Exec(ctx)
	if err != nil {
		return nil, mapReasonCodeConstraint(err)
	}
	if err = dberror.CheckRowsAffected(result, "Service failure reason code", entity.ID.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, repositories.GetServiceFailureReasonCodeByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantInfo(entity),
	})
}

func (r *repository) Archive(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	actorID pulid.ID,
) (*servicefailure.ReasonCode, error) {
	now := timeutils.NowUnix()
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*servicefailure.ReasonCode)(nil)).
		Where("id = ?", id).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Set("active = FALSE").
		Set("archived_at = ?", now).
		Set("archived_by_id = ?", nullableID(actorID)).
		Set("activated_at = NULL").
		Set("activated_by_id = NULL").
		Set("version = version + 1").
		Set("updated_at = ?", now).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("archive service failure reason code: %w", err)
	}
	if err = dberror.CheckRowsAffected(result, "Service failure reason code", id.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, repositories.GetServiceFailureReasonCodeByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

func (r *repository) Activate(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	actorID pulid.ID,
) (*servicefailure.ReasonCode, error) {
	now := timeutils.NowUnix()
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*servicefailure.ReasonCode)(nil)).
		Where("id = ?", id).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Set("active = TRUE").
		Set("activated_at = ?", now).
		Set("activated_by_id = ?", nullableID(actorID)).
		Set("version = version + 1").
		Set("updated_at = ?", now).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("activate service failure reason code: %w", err)
	}
	if err = dberror.CheckRowsAffected(result, "Service failure reason code", id.String()); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, repositories.GetServiceFailureReasonCodeByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

func (r *repository) Reorder(
	ctx context.Context,
	req *repositories.ReorderServiceFailureReasonCodesRequest,
) ([]*servicefailure.ReasonCode, error) {
	if err := r.db.WithTx(ctx, coreports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		for idx, id := range req.ReasonIDs {
			_, err := r.db.DBForContext(txCtx).
				NewUpdate().
				Model((*servicefailure.ReasonCode)(nil)).
				Where("id = ?", id).
				Where("organization_id = ?", req.TenantInfo.OrgID).
				Where("business_unit_id = ?", req.TenantInfo.BuID).
				Set("sort_order = ?", int32((idx+1)*10)).
				Set("version = version + 1").
				Set("updated_at = ?", timeutils.NowUnix()).
				Exec(txCtx)
			if err != nil {
				return fmt.Errorf("reorder service failure reason code: %w", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	entities := make([]*servicefailure.ReasonCode, 0, len(req.ReasonIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("sfrc.organization_id = ?", req.TenantInfo.OrgID).
		Where("sfrc.business_unit_id = ?", req.TenantInfo.BuID).
		Where("sfrc.id IN (?)", bun.In(req.ReasonIDs)).
		Order("sfrc.sort_order ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list reordered service failure reason codes: %w", err)
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.ServiceFailureReasonCodeSelectOptionsRequest,
) (*pagination.ListResult[*servicefailure.ReasonCode], error) {
	return dbhelper.SelectOptions[*servicefailure.ReasonCode](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"code",
				"label",
				"category",
				"applies_to",
				"default_status_code",
				"default_reason_code",
			},
			OrgColumn: "sfrc.organization_id",
			BuColumn:  "sfrc.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				q = q.Where("sfrc.active = TRUE")
				if req.AppliesTo.IsValid() {
					q = q.Where("sfrc.applies_to IN (?)", bun.In([]servicefailure.ReasonCodeAppliesTo{
						req.AppliesTo,
						servicefailure.ReasonCodeAppliesToBoth,
					}))
				}
				return q.Order("sfrc.sort_order ASC", "sfrc.code ASC")
			},
			EntityName: "ServiceFailureReasonCode",
			SearchColumns: []string{
				"sfrc.code",
				"sfrc.label",
				"sfrc.description",
			},
		},
	)
}

func mapReasonCodeConstraint(err error) error {
	if dberror.IsUniqueConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == "ux_sfrc_tenant_code" {
		return errortypes.NewValidationError(
			"code",
			errortypes.ErrDuplicate,
			"Service failure reason code already exists in your organization",
		)
	}
	return err
}

func tenantInfo(entity *servicefailure.ReasonCode) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
}

func nullableID(id pulid.ID) any {
	if id.IsNil() {
		return nil
	}
	return id
}
