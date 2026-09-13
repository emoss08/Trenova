package workercredentialrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultExpiringPageSize = 500
	secondsPerDay           = int64(86400)
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

func New(p Params) repositories.WorkerCredentialRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-credential-repository"),
	}
}

func orderTypes(sq *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.WorkerCredentialTypeColumns
	return sq.Order(cols.SortOrder.OrderAsc()).Order(cols.Name.OrderAsc())
}

func (r *repository) applyTypeFilters(
	q *bun.SelectQuery,
	req *repositories.ListCredentialTypesRequest,
) *bun.SelectQuery {
	cols := buncolgen.WorkerCredentialTypeColumns
	if req.Status != "" {
		q = q.Where(cols.Status.Eq(), req.Status)
	}
	if req.Category != "" {
		q = q.Where(cols.Category.Eq(), req.Category)
	}
	return q
}

func (r *repository) ListTypes(
	ctx context.Context,
	req *repositories.ListCredentialTypesRequest,
) (*pagination.CursorListResult[*worker.WorkerCredentialType], error) {
	log := r.l.With(zap.String("operation", "ListTypes"))

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*worker.WorkerCredentialType)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.WorkerCredentialTypeTable.Alias,
					req.Filter,
					(*worker.WorkerCredentialType)(nil),
				)
				return r.applyTypeFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count credential types", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*worker.WorkerCredentialType]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*worker.WorkerCredentialType) *bun.SelectQuery {
				return dba.NewSelect().
					Model(items).
					ColumnExpr(buncolgen.WorkerCredentialTypeTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.WorkerCredentialTypeTable.Alias,
					req.Filter,
					req.Cursor,
					(*worker.WorkerCredentialType)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				return r.applyTypeFilters(sq, req), nil
			},
		},
	)
	if err != nil {
		log.Error("failed to list credential types", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) ListActiveTypes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.WorkerCredentialType, error) {
	cols := buncolgen.WorkerCredentialTypeColumns
	entities := make([]*worker.WorkerCredentialType, 0, 16)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerCredentialTypeScopeTenant(sq, tenantInfo).
				Where(cols.Status.Eq(), domaintypes.StatusActive)
		}).
		Apply(orderTypes).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list active credential types", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) TypeSelectOptions(
	ctx context.Context,
	req *repositories.WorkerCredentialTypeSelectOptionsRequest,
) (*pagination.ListResult[*worker.WorkerCredentialType], error) {
	cols := buncolgen.WorkerCredentialTypeColumns
	return dbhelper.SelectOptions[*worker.WorkerCredentialType](
		ctx,
		r.db.DBForContext(ctx),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				cols.ID,
				cols.Code,
				cols.Name,
				cols.Description,
				cols.Category,
				cols.IsRequired,
				cols.ValidityMonths,
				cols.RequiresNumber,
				cols.RequiresDocument,
				cols.ProfileField,
				cols.SortOrder,
				cols.CreatedAt,
			},
			OrgColumnRef: &cols.OrganizationID,
			BuColumnRef:  &cols.BusinessUnitID,
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return orderTypes(q.Where(cols.Status.Eq(), domaintypes.StatusActive))
			},
			EntityName: "WorkerCredentialType",
			SearchColumnRefs: []buncolgen.Column{
				cols.Code,
				cols.Name,
				cols.Description,
			},
		},
	)
}

func (r *repository) GetTypeByID(
	ctx context.Context,
	req *repositories.GetCredentialTypeByIDRequest,
) (*worker.WorkerCredentialType, error) {
	entity := new(worker.WorkerCredentialType)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerCredentialTypeScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerCredentialTypeColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerCredentialType")
	}

	return entity, nil
}

func (r *repository) TypeCodeExists(
	ctx context.Context,
	req *repositories.CredentialTypeCodeExistsRequest,
) (bool, error) {
	cols := buncolgen.WorkerCredentialTypeColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerCredentialType)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerCredentialTypeScopeTenant(sq, req.TenantInfo).
				Where("LOWER("+cols.Code.String()+") = ?", strings.ToLower(req.Code))
		})
	if !req.ExcludeID.IsNil() {
		q = q.Where(cols.ID.Ne(), req.ExcludeID)
	}

	exists, err := q.Exists(ctx)
	if err != nil {
		r.l.Error("failed to check credential type code", zap.Error(err))
		return false, err
	}

	return exists, nil
}

func (r *repository) CreateType(
	ctx context.Context,
	entity *worker.WorkerCredentialType,
) (*worker.WorkerCredentialType, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"code",
				errortypes.ErrDuplicate,
				"A credential type with this code already exists",
			)
		}
		r.l.Error("failed to create credential type", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateType(
	ctx context.Context,
	entity *worker.WorkerCredentialType,
) (*worker.WorkerCredentialType, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerCredentialTypeColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"code",
				errortypes.ErrDuplicate,
				"A credential type with this code already exists",
			)
		}
		r.l.Error("failed to update credential type", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "WorkerCredentialType", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) EnsureSystemTypes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	types []*worker.WorkerCredentialType,
) (int, error) {
	if len(types) == 0 {
		return 0, nil
	}

	rows := make([]*worker.WorkerCredentialType, 0, len(types))
	for _, typ := range types {
		if typ == nil {
			continue
		}
		row := *typ
		row.OrganizationID = tenantInfo.OrgID
		row.BusinessUnitID = tenantInfo.BuID
		row.Status = domaintypes.StatusActive
		row.IsSystem = true
		if row.RequiredForDriverTypes == nil {
			row.RequiredForDriverTypes = []worker.DriverType{}
		}
		rows = append(rows, &row)
	}

	result, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&rows).
		On("CONFLICT DO NOTHING").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to ensure system credential types", zap.Error(err))
		return 0, fmt.Errorf("ensure system credential types: %w", err)
	}
	inserted, _ := result.RowsAffected()

	return int(inserted), nil
}

func (r *repository) CountCredentialsByType(
	ctx context.Context,
	req *repositories.CountCredentialsByTypeRequest,
) (int, error) {
	cols := buncolgen.WorkerCredentialColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerCredential)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerCredentialScopeTenant(sq, req.TenantInfo).
				Where(cols.CredentialTypeID.Eq(), req.TypeID)
		})
	if req.ActiveOnly {
		q = q.Where(cols.Status.Eq(), worker.CredentialStatusActive)
	}

	count, err := q.Count(ctx)
	if err != nil {
		r.l.Error("failed to count credentials by type", zap.Error(err))
		return 0, err
	}

	return count, nil
}

func (r *repository) CountCredentialsByTypeIDs(
	ctx context.Context,
	req *repositories.CountCredentialsByTypeIDsRequest,
) (map[pulid.ID]int, error) {
	if len(req.TypeIDs) == 0 {
		return map[pulid.ID]int{}, nil
	}

	cols := buncolgen.WorkerCredentialColumns
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*worker.WorkerCredential)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerCredentialScopeTenant(sq, req.TenantInfo).
				Where(cols.CredentialTypeID.In(), bun.In(req.TypeIDs))
		})
	if req.ActiveOnly {
		q = q.Where(cols.Status.Eq(), worker.CredentialStatusActive)
	}

	counts, err := dbhelper.CountByID(ctx, q, cols.CredentialTypeID, len(req.TypeIDs))
	if err != nil {
		r.l.Error("failed to count credentials by type ids", zap.Error(err))
		return nil, err
	}

	return counts, nil
}

func (r *repository) ListForWorker(
	ctx context.Context,
	req *repositories.ListWorkerCredentialsRequest,
) ([]*worker.WorkerCredential, error) {
	cols := buncolgen.WorkerCredentialColumns
	entities := make([]*worker.WorkerCredential, 0, 16)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.WorkerCredentialScopeTenant(sq, req.TenantInfo).
				Where(cols.WorkerID.Eq(), req.WorkerID)
			if !req.IncludeArchived {
				sq = sq.Where(cols.Status.Eq(), worker.CredentialStatusActive)
			}
			return sq
		}).
		Order(cols.Status.OrderAsc()).
		Order(cols.CreatedAt.OrderDesc())
	if req.IncludeType {
		q = q.Relation(buncolgen.WorkerCredentialRelations.CredentialType)
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerCredentialRelations.Document)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list worker credentials", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetWorkerCredentialByIDRequest,
) (*worker.WorkerCredential, error) {
	entity := new(worker.WorkerCredential)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.WorkerCredentialScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.WorkerCredentialColumns.ID.Eq(), req.ID)
		})
	if req.IncludeType {
		q = q.Relation(buncolgen.WorkerCredentialRelations.CredentialType)
	}
	if req.IncludeWorker {
		q = q.Relation(
			buncolgen.WorkerCredentialRelations.Worker,
			func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation(buncolgen.WorkerRelations.Profile)
			},
		)
	}
	if req.IncludeDocument {
		q = q.Relation(buncolgen.WorkerCredentialRelations.Document)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "WorkerCredential")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	req *repositories.CreateWorkerCredentialRequest,
) (*worker.WorkerCredential, error) {
	entity := req.Entity
	cols := buncolgen.WorkerCredentialColumns
	tenant := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if req.SupersedeReason != "" {
			now := timeutils.NowUnix()
			if _, uErr := r.db.DBForContext(c).
				NewUpdate().
				Model((*worker.WorkerCredential)(nil)).
				Set(cols.Status.Set(), worker.CredentialStatusArchived).
				Set(cols.ArchivedAt.Set(), now).
				Set(cols.ArchivedByID.Set(), req.SupersededByID).
				Set(cols.ArchiveReason.Set(), req.SupersedeReason).
				Set(cols.UpdatedAt.Set(), now).
				Set(cols.Version.Set()+" + 1", bun.Safe(cols.Version.String())).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return buncolgen.WorkerCredentialScopeTenantUpdate(uq, tenant).
						Where(cols.WorkerID.Eq(), entity.WorkerID).
						Where(cols.CredentialTypeID.Eq(), entity.CredentialTypeID).
						Where(cols.Status.Eq(), worker.CredentialStatusActive)
				}).
				Exec(c); uErr != nil {
				return uErr
			}
		}

		_, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c)
		return iErr
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"credentialTypeId",
				errortypes.ErrDuplicate,
				"This worker already holds an active credential of this type. Renew it instead.",
			)
		}
		r.l.Error("failed to create worker credential", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Credential is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *worker.WorkerCredential,
) (*worker.WorkerCredential, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.WorkerCredentialColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"credentialTypeId",
				errortypes.ErrDuplicate,
				"This worker already holds an active credential of this type",
			)
		}
		r.l.Error("failed to update worker credential", zap.Error(err))
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "WorkerCredential", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListExpiring(
	ctx context.Context,
	req *repositories.ListExpiringWorkerCredentialsRequest,
) ([]*worker.WorkerCredential, error) {
	now := timeutils.NowUnix()
	horizon := now + int64(req.HorizonDays)*secondsPerDay
	grace := now - int64(req.GraceDays)*secondsPerDay
	limit := req.Limit
	if limit <= 0 {
		limit = defaultExpiringPageSize
	}

	cols := buncolgen.WorkerCredentialColumns
	typeAlias := "credential_type"
	workerAlias := "worker"
	entities := make([]*worker.WorkerCredential, 0, limit)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.WorkerCredentialRelations.CredentialType).
		Relation(buncolgen.WorkerCredentialRelations.Worker, func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Relation(buncolgen.WorkerRelations.Profile)
		}).
		Where(cols.Status.Eq(), worker.CredentialStatusActive).
		Where(cols.ExpiresAt.Between(), grace, horizon).
		Where(buncolgen.WorkerColumns.Status.WithAlias(workerAlias).Eq(), domaintypes.StatusActive).
		Where(buncolgen.WorkerCredentialTypeColumns.Status.WithAlias(typeAlias).Eq(), domaintypes.StatusActive).
		Order(cols.ID.OrderAsc()).
		Limit(limit)
	if !req.TenantInfo.OrgID.IsNil() {
		q = q.Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID)
	}
	if !req.TenantInfo.BuID.IsNil() {
		q = q.Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID)
	}
	if !req.AfterID.IsNil() {
		q = q.Where(cols.ID.Gt(), req.AfterID)
	}
	if req.RequiredOnly {
		q = q.Where(
			buncolgen.WorkerCredentialTypeColumns.IsRequired.WithAlias(typeAlias).Eq(),
			true,
		)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list expiring worker credentials", zap.Error(err))
		return nil, fmt.Errorf("list expiring worker credentials: %w", err)
	}

	return entities, nil
}
