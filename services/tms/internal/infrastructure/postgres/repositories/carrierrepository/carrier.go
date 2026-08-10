package carrierrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
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

func New(p Params) repositories.CarrierRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-repository"),
	}
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	opts repositories.CarrierFilterOptions,
) *bun.SelectQuery {
	if opts.IncludeState {
		q = q.Relation("State")
		q = q.Relation("RemitState")
	}

	if opts.IncludeContacts {
		q = q.Relation("Contacts")
	}

	if opts.IncludeInsurancePolicies {
		q = q.Relation("InsurancePolicies")
	}

	if opts.IncludeEDIChannels {
		q = q.Relation("EDIChannels")
		q = q.Relation("EDIChannels.EDIPartner")
	}

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListCarrierRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.CarrierTable.Alias,
		req.Filter,
		(*carrier.Carrier)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.CarrierFilterOptions)
	})

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListCarrierRequest,
) (*pagination.ListResult[*carrier.Carrier], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*carrier.Carrier, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count carriers", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*carrier.Carrier]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListCarrierConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.CarrierTable.Alias,
		req.Filter,
		req.Cursor,
		(*carrier.Carrier)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListCarrierConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.CarrierTable.Alias,
		req.Filter,
		(*carrier.Carrier)(nil),
	)
}

func applyCarrierColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.CarrierTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListCarrierConnectionRequest,
) (*pagination.CursorListResult[*carrier.Carrier], error) {
	log := r.l.With(
		zap.String("operation", "ListConnection"),
		zap.Any("request", req),
	)

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*carrier.Carrier)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count carriers", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*carrier.Carrier]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*carrier.Carrier) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyCarrierColumns(sq, req.CarrierColumns)
					}).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.addOptions(sq, req.CarrierFilterOptions)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		})
	if err != nil {
		log.Error("failed to scan carriers", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetCarrierByIDRequest,
) (*carrier.Carrier, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	cols := buncolgen.CarrierColumns
	entity := new(carrier.Carrier)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.CarrierFilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get carrier", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Carrier")
	}

	return entity, nil
}

func (r *repository) FindByIdentity(
	ctx context.Context,
	req *repositories.FindCarrierByIdentityRequest,
) (*carrier.Carrier, error) {
	if req.SCAC == "" && req.DOTNumber == "" {
		return nil, nil //nolint:nilnil // nothing to look up without an identity
	}

	cols := buncolgen.CarrierColumns
	entity := new(carrier.Carrier)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierScopeTenant(sq, req.TenantInfo).
				WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
					if req.SCAC != "" {
						inner = inner.Where(
							cols.SCAC.Expr("UPPER({}) = ?"),
							strings.ToUpper(req.SCAC),
						)
					}
					if req.DOTNumber != "" {
						inner = inner.WhereOr(cols.DOTNumber.Eq(), req.DOTNumber)
					}
					return inner
				})
		}).
		Order(cols.CreatedAt.OrderAsc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil carrier represents an optional absence
		}
		return nil, fmt.Errorf("find carrier by identity: %w", err)
	}

	return entity, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetCarriersByIDsRequest,
) ([]*carrier.Carrier, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	cols := buncolgen.CarrierColumns
	entities := make([]*carrier.Carrier, 0, len(req.CarrierIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.CarrierIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get carriers", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Carrier")
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.CarrierSelectOptionsRequest,
) (*pagination.ListResult[*carrier.Carrier], error) {
	return dbhelper.SelectOptions[*carrier.Carrier](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"code",
				"name",
			},
			OrgColumn: "carr.organization_id",
			BuColumn:  "carr.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("carr.status = ?", carrier.StatusActive)
			},
			EntityName: "Carrier",
			SearchColumns: []string{
				"carr.code",
				"carr.name",
			},
		},
	)
}

// stampChildren pins every child row to its parent and tenant. IDs are left
// alone: rows the caller kept retain their identity (and created_at history);
// BeforeAppendModel mints IDs for the new, ID-less rows only.
func (r *repository) stampChildren(entity *carrier.Carrier) {
	for _, contact := range entity.Contacts {
		contact.CarrierID = entity.ID
		contact.OrganizationID = entity.OrganizationID
		contact.BusinessUnitID = entity.BusinessUnitID
	}

	for _, policy := range entity.InsurancePolicies {
		policy.CarrierID = entity.ID
		policy.OrganizationID = entity.OrganizationID
		policy.BusinessUnitID = entity.BusinessUnitID
	}

	for _, channel := range entity.EDIChannels {
		channel.CarrierID = entity.ID
		channel.OrganizationID = entity.OrganizationID
		channel.BusinessUnitID = entity.BusinessUnitID
	}
}

func (r *repository) insertChildren(ctx context.Context, entity *carrier.Carrier) error {
	if len(entity.Contacts) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.Contacts).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}
	}

	if len(entity.InsurancePolicies) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.InsurancePolicies).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}
	}

	if len(entity.EDIChannels) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.EDIChannels).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

func keepIDs[T interface{ GetID() pulid.ID }](children []T) []pulid.ID {
	ids := make([]pulid.ID, 0, len(children))
	for _, child := range children {
		if id := child.GetID(); !id.IsNil() {
			ids = append(ids, id)
		}
	}
	return ids
}

// deleteMissingChildren removes only the child rows the payload no longer
// carries; rows whose IDs are still present keep their identity and history.
func (r *repository) deleteMissingChildren(ctx context.Context, entity *carrier.Carrier) error {
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	contactQuery := r.db.DBForContext(ctx).
		NewDelete().
		Model((*carrier.CarrierContact)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			dq = buncolgen.CarrierContactScopeTenantDelete(dq, tenantInfo).
				Where(buncolgen.CarrierContactColumns.CarrierID.Eq(), entity.ID)
			if kept := keepIDs(entity.Contacts); len(kept) > 0 {
				dq = dq.Where(buncolgen.CarrierContactColumns.ID.NotIn(), bun.List(kept))
			}
			return dq
		})
	if _, err := contactQuery.Exec(ctx); err != nil {
		return err
	}

	policyQuery := r.db.DBForContext(ctx).
		NewDelete().
		Model((*carrier.CarrierInsurancePolicy)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			dq = buncolgen.CarrierInsurancePolicyScopeTenantDelete(dq, tenantInfo).
				Where(buncolgen.CarrierInsurancePolicyColumns.CarrierID.Eq(), entity.ID)
			if kept := keepIDs(entity.InsurancePolicies); len(kept) > 0 {
				dq = dq.Where(buncolgen.CarrierInsurancePolicyColumns.ID.NotIn(), bun.List(kept))
			}
			return dq
		})
	if _, err := policyQuery.Exec(ctx); err != nil {
		return err
	}

	channelQuery := r.db.DBForContext(ctx).
		NewDelete().
		Model((*carrier.CarrierEDIChannel)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			dq = buncolgen.CarrierEDIChannelScopeTenantDelete(dq, tenantInfo).
				Where(buncolgen.CarrierEDIChannelColumns.CarrierID.Eq(), entity.ID)
			if kept := keepIDs(entity.EDIChannels); len(kept) > 0 {
				dq = dq.Where(buncolgen.CarrierEDIChannelColumns.ID.NotIn(), bun.List(kept))
			}
			return dq
		})
	if _, err := channelQuery.Exec(ctx); err != nil {
		return err
	}

	return nil
}

// clearEDIChannelDefaults drops every default flag before the channel upsert:
// uq_carrier_edi_channels_default is a non-deferrable partial unique index, so
// moving is_default between two existing rows in one statement would collide
// mid-upsert.
func (r *repository) clearEDIChannelDefaults(ctx context.Context, entity *carrier.Carrier) error {
	cols := buncolgen.CarrierEDIChannelColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*carrier.CarrierEDIChannel)(nil)).
		Set("is_default = FALSE").
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CarrierEDIChannelScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).Where(cols.CarrierID.Eq(), entity.ID).
				Where(cols.IsDefault.IsTrue())
		}).
		Exec(ctx)
	return err
}

// upsertChildren inserts new (ID-less) child rows and updates the kept ones in
// place, preserving their IDs and created_at for insurance history and audit
// diffs.
func (r *repository) upsertChildren(ctx context.Context, entity *carrier.Carrier) error {
	now := timeutils.NowUnix()

	if len(entity.Contacts) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.Contacts).
			On("CONFLICT (id, business_unit_id, organization_id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("title = EXCLUDED.title").
			Set("email = EXCLUDED.email").
			Set("phone = EXCLUDED.phone").
			Set("is_primary = EXCLUDED.is_primary").
			Set("receives_rate_confirmations = EXCLUDED.receives_rate_confirmations").
			Set("version = carc.version + 1").
			Set("updated_at = ?", now).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}
	}

	if len(entity.InsurancePolicies) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.InsurancePolicies).
			On("CONFLICT (id, business_unit_id, organization_id) DO UPDATE").
			Set("policy_type = EXCLUDED.policy_type").
			Set("policy_number = EXCLUDED.policy_number").
			Set("provider_name = EXCLUDED.provider_name").
			Set("coverage_amount = EXCLUDED.coverage_amount").
			Set("effective_date = EXCLUDED.effective_date").
			Set("expiration_date = EXCLUDED.expiration_date").
			Set("is_verified = EXCLUDED.is_verified").
			Set("version = carins.version + 1").
			Set("updated_at = ?", now).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}
	}

	if len(entity.EDIChannels) > 0 {
		if err := r.clearEDIChannelDefaults(ctx, entity); err != nil {
			return err
		}

		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.EDIChannels).
			On("CONFLICT (id, business_unit_id, organization_id) DO UPDATE").
			Set("edi_partner_id = EXCLUDED.edi_partner_id").
			Set("partner_document_profile_id = EXCLUDED.partner_document_profile_id").
			Set("communication_profile_id = EXCLUDED.communication_profile_id").
			Set("scac_override = EXCLUDED.scac_override").
			Set("is_default = EXCLUDED.is_default").
			Set("version = cec.version + 1").
			Set("updated_at = ?", now).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *carrier.Carrier,
) (*carrier.Carrier, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("code", entity.Code),
	)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}

		r.stampChildren(entity)

		return r.insertChildren(c, entity)
	})
	if err != nil {
		log.Error("failed to create carrier", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *carrier.Carrier,
) (*carrier.Carrier, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		ov := entity.Version
		entity.Version++

		results, uErr := r.db.DBForContext(c).NewUpdate().
			Model(entity).
			WherePK().
			Where("version = ?", ov).
			Returning("*").
			Exec(c)
		if uErr != nil {
			return uErr
		}

		if uErr = dberror.CheckRowsAffected(results, "Carrier", entity.ID.String()); uErr != nil {
			return uErr
		}

		if dErr := r.deleteMissingChildren(c, entity); dErr != nil {
			return dErr
		}

		r.stampChildren(entity)

		return r.upsertChildren(c, entity)
	})
	if err != nil {
		log.Error("failed to update carrier", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Carrier is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateCarrierStatusRequest,
) ([]*carrier.Carrier, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	cols := buncolgen.CarrierColumns
	entities := make([]*carrier.Carrier, 0, len(req.CarrierIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CarrierScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.CarrierIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update carrier status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "Carrier", req.CarrierIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) ListEDIChannels(
	ctx context.Context,
	req repositories.GetCarrierEDIChannelRequest,
) ([]*carrier.CarrierEDIChannel, error) {
	cols := buncolgen.CarrierEDIChannelColumns
	entities := make([]*carrier.CarrierEDIChannel, 0, 4)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("EDIPartner").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierEDIChannelScopeTenant(sq, req.TenantInfo).
				Where(cols.CarrierID.Eq(), req.CarrierID)
		}).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list carrier EDI channels", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetDefaultEDIChannel(
	ctx context.Context,
	req repositories.GetCarrierEDIChannelRequest,
) (*carrier.CarrierEDIChannel, error) {
	cols := buncolgen.CarrierEDIChannelColumns
	entity := new(carrier.CarrierEDIChannel)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("EDIPartner").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierEDIChannelScopeTenant(sq, req.TenantInfo).
				Where(cols.CarrierID.Eq(), req.CarrierID).
				Where(cols.IsDefault.IsTrue())
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil channel represents an optional absence
		}
		r.l.Error("failed to get default carrier EDI channel", zap.Error(err))
		return nil, err
	}

	return entity, nil
}
