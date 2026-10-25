package workercredentialrepository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultExpiringPageSize = 500
	secondsPerDay           = int64(86400)
)

var errExpiringKeysetBySoonest = errors.New(
	"expiring credentials ordered soonest first page by offset, not by afterId",
)

var typeErrors = dberror.Mapping{Entity: "WorkerCredentialType"}

var credentialErrors = dberror.Mapping{Entity: "WorkerCredential"}

type typeCount struct {
	TypeID pulid.ID
	Count  int64
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) workerports.WorkerCredentialRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-credential-repository"),
	}
}

func typeOrder() []keel.Order {
	t := workerdb.WorkerCredentialTypes
	return []keel.Order{t.SortOrder.Asc(), t.Name.Asc()}
}

func (r *repository) ListTypes(
	ctx context.Context,
	req *workerports.ListCredentialTypesRequest,
) (*pagination.CursorListResult[*worker.WorkerCredentialType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*worker.WorkerCredentialType], error) {
			t := workerdb.WorkerCredentialTypes
			q := t.Where(
				keel.When(req.Status != "", t.Status.Eq(domaintypes.Status(req.Status))),
				keel.When(
					req.Category != "",
					t.Category.Eq(worker.CredentialCategory(req.Category)),
				),
			)

			result, err := listquery.Connection(ctx, r.db.Keel(ctx), q,
				&listquery.Request[worker.WorkerCredentialType]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				})
			if err != nil {
				r.l.Error("failed to list credential types", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListActiveTypes(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*worker.WorkerCredentialType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.WorkerCredentialType, error) {
		t := workerdb.WorkerCredentialTypes
		entities, err := t.Where(t.Status.Eq(domaintypes.StatusActive)).
			Order(typeOrder()...).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list active credential types", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) TypeSelectOptions(
	ctx context.Context,
	req *workerports.WorkerCredentialTypeSelectOptionsRequest,
) (*pagination.ListResult[*worker.WorkerCredentialType], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*worker.WorkerCredentialType], error) {
			t := workerdb.WorkerCredentialTypes
			q := t.Pick(
				t.ID,
				t.Code,
				t.Name,
				t.Description,
				t.Category,
				t.IsRequired,
				t.ValidityMonths,
				t.RequiresNumber,
				t.RequiresDocument,
				t.ProfileField,
				t.SortOrder,
				t.CreatedAt,
			).
				Where(t.Status.Eq(domaintypes.StatusActive)).
				Order(typeOrder()...)

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						t.Code.ContainsFold(s),
						t.Name.ContainsFold(s),
						t.Description.ContainsFold(s),
					)
				},
			)
		},
	)
}

func (r *repository) GetTypeByID(
	ctx context.Context,
	req *workerports.GetCredentialTypeByIDRequest,
) (*worker.WorkerCredentialType, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.WorkerCredentialType, error) {
		entity, err := workerdb.WorkerCredentialTypes.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, typeErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) TypeCodeExists(
	ctx context.Context,
	req *workerports.CredentialTypeCodeExistsRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		t := workerdb.WorkerCredentialTypes
		exists, err := t.Where(
			t.Code.Lower().Eq(strings.ToLower(req.Code)),
			keel.When(req.ExcludeID.IsNotNil(), t.ID.Ne(req.ExcludeID)),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to check credential type code", zap.Error(err))
			return false, err
		}

		return exists, nil
	})
}

func (r *repository) CreateType(
	ctx context.Context,
	entity *worker.WorkerCredentialType,
) (*worker.WorkerCredentialType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerCredentialType, error) {
		if err := workerdb.WorkerCredentialTypes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create credential type", zap.Error(err))
			return nil, typeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateType(
	ctx context.Context,
	entity *worker.WorkerCredentialType,
) (*worker.WorkerCredentialType, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerCredentialType, error) {
		if err := workerdb.WorkerCredentialTypes.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update credential type", zap.Error(err))
			return nil, typeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) EnsureSystemTypes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	types []*worker.WorkerCredentialType,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
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
		if len(rows) == 0 {
			return 0, nil
		}

		inserted, err := workerdb.WorkerCredentialTypes.CreateOrSkip(ctx, r.db.Keel(ctx), rows)
		if err != nil {
			r.l.Error("failed to ensure system credential types", zap.Error(err))
			return 0, fmt.Errorf("ensure system credential types: %w", err)
		}

		return int(inserted), nil
	})
}

func (r *repository) CountCredentialsByType(
	ctx context.Context,
	req *workerports.CountCredentialsByTypeRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		c := workerdb.WorkerCredentials
		count, err := c.Where(
			c.CredentialTypeID.Eq(req.TypeID),
			keel.When(req.ActiveOnly, c.Status.Eq(worker.CredentialStatusActive)),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count credentials by type", zap.Error(err))
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) CountCredentialsByTypeIDs(
	ctx context.Context,
	req *workerports.CountCredentialsByTypeIDsRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		if len(req.TypeIDs) == 0 {
			return map[pulid.ID]int{}, nil
		}

		c := workerdb.WorkerCredentials
		rows, err := keel.Into[typeCount](ctx, r.db.Keel(ctx), c.
			Group(c.CredentialTypeID).
			Pick(c.CredentialTypeID, keel.Count()).
			Where(
				c.CredentialTypeID.In(req.TypeIDs...),
				keel.When(req.ActiveOnly, c.Status.Eq(worker.CredentialStatusActive)),
			))
		if err != nil {
			r.l.Error("failed to count credentials by type ids", zap.Error(err))
			return nil, err
		}

		counts := make(map[pulid.ID]int, len(rows))
		for _, row := range rows {
			counts[row.TypeID] = int(row.Count)
		}

		return counts, nil
	})
}

func (r *repository) ListForWorker(
	ctx context.Context,
	req *workerports.ListWorkerCredentialsRequest,
) ([]*worker.WorkerCredential, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.WorkerCredential, error) {
		c := workerdb.WorkerCredentials
		loads := append(
			listquery.When(req.IncludeType, c.CredentialType()),
			listquery.When(req.IncludeDocument, c.Document())...,
		)

		entities, err := c.Where(
			c.WorkerID.Eq(req.WorkerID),
			keel.When(!req.IncludeArchived, c.Status.Eq(worker.CredentialStatusActive)),
		).
			Load(loads...).
			Order(c.Status.Asc(), c.CreatedAt.Desc(), c.ID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list worker credentials", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req *workerports.GetWorkerCredentialByIDRequest,
) (*worker.WorkerCredential, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.WorkerCredential, error) {
		c := workerdb.WorkerCredentials
		loads := make([]keel.Loader, 0, 3)
		loads = append(loads, listquery.When(req.IncludeType, c.CredentialType())...)
		loads = append(loads, listquery.When(req.IncludeWorker, c.Worker().Profile())...)
		loads = append(loads, listquery.When(req.IncludeDocument, c.Document())...)

		entity, err := c.Where(c.ID.Eq(req.ID)).Load(loads...).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, credentialErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	req *workerports.CreateWorkerCredentialRequest,
) (*worker.WorkerCredential, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerCredential, error) {
		if err := r.create(ctx, req); err != nil {
			r.l.Error("failed to create worker credential", zap.Error(err))
			return nil, dberror.MapRetryableTransactionError(
				credentialErrors.Map(err, req.Entity.ID.String()),
				"Credential is busy. Retry the request.",
			)
		}

		return req.Entity, nil
	})
}

func (r *repository) create(
	ctx context.Context,
	req *workerports.CreateWorkerCredentialRequest,
) error {
	db := r.db.Keel(ctx)
	entity := req.Entity
	c := workerdb.WorkerCredentials

	if req.SupersedeReason != "" {
		_, err := c.Where(
			c.WorkerID.Eq(entity.WorkerID),
			c.CredentialTypeID.Eq(entity.CredentialTypeID),
			c.Status.Eq(worker.CredentialStatusActive),
		).Update(
			ctx,
			db,
			c.Status.To(worker.CredentialStatusArchived),
			c.ArchivedAt.To(timeutils.NowUnix()),
			c.ArchivedByID.To(req.SupersededByID),
			c.ArchiveReason.To(req.SupersedeReason),
		)
		if err != nil {
			return err
		}
	}

	return c.Create(ctx, db, entity)
}

func (r *repository) Update(
	ctx context.Context,
	entity *worker.WorkerCredential,
) (*worker.WorkerCredential, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.WorkerCredential, error) {
		if err := workerdb.WorkerCredentials.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to update worker credential", zap.Error(err))
			return nil, credentialErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func expiryWindow(
	req *workerports.ListExpiringWorkerCredentialsRequest,
) (grace, horizon int64) {
	now := timeutils.NowUnix()
	horizon = now + int64(req.HorizonDays)*secondsPerDay
	grace = now - int64(req.GraceDays)*secondsPerDay
	if req.AsOf > 0 {
		horizon = req.AsOf + int64(req.HorizonDays+1)*secondsPerDay - 1
		grace = req.AsOf - int64(req.GraceDays)*secondsPerDay
	}

	return grace, horizon
}

func expiringQuery(
	ctx context.Context,
	req *workerports.ListExpiringWorkerCredentialsRequest,
) *keel.Query[worker.WorkerCredential] {
	c := workerdb.WorkerCredentials
	ct := c.CredentialType()
	grace, horizon := expiryWindow(req)
	limit := req.Limit
	if limit <= 0 {
		limit = defaultExpiringPageSize
	}

	order := []keel.Order{c.ID.Asc()}
	if req.SoonestFirst {
		order = []keel.Order{c.ExpiresAt.Asc(), c.ID.Asc()}
	}

	q := c.Where(
		c.Status.Eq(worker.CredentialStatusActive),
		c.ExpiresAt.Between(grace, horizon),
		c.Worker().Status.Eq(domaintypes.StatusActive),
		ct.Status.Eq(domaintypes.StatusActive),
		keel.When(req.WorkerID.IsNotNil(), c.WorkerID.Eq(req.WorkerID)),
		keel.When(req.TenantInfo.OrgID.IsNotNil(), c.OrganizationID.Eq(req.TenantInfo.OrgID)),
		keel.When(req.TenantInfo.BuID.IsNotNil(), c.BusinessUnitID.Eq(req.TenantInfo.BuID)),
		keel.When(req.AfterID.IsNotNil(), c.ID.Gt(req.AfterID)),
		keel.When(req.RequiredOnly, ct.IsRequired.IsTrue()),
		keel.When(len(req.CredentialTypeCodes) > 0, ct.Code.In(req.CredentialTypeCodes...)),
	).
		Load(ct, c.Worker().Profile()).
		Order(order...).
		Limit(limit)
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	if dbscope.IsSystem(ctx) {
		q = q.AllTenants()
	}

	return q
}

func (r *repository) ListExpiring(
	ctx context.Context,
	req *workerports.ListExpiringWorkerCredentialsRequest,
) ([]*worker.WorkerCredential, error) {
	if req.TenantInfo.OrgID.IsNil() {
		ctx = dbscope.WithSystem(
			ctx,
			"list expiring credentials across every organization for the expiry sweep",
		)
	}
	if req.SoonestFirst && !req.AfterID.IsNil() {
		return nil, errExpiringKeysetBySoonest
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*worker.WorkerCredential, error) {
		entities, err := expiringQuery(ctx, req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list expiring worker credentials", zap.Error(err))
			return nil, fmt.Errorf("list expiring worker credentials: %w", err)
		}

		return entities, nil
	})
}
