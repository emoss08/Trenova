package variablerepository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/variable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	variableservice "github.com/emoss08/trenova/internal/core/services/variable"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
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

func NewRepository(p Params) repositories.VariableRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.variable-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListVariableRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"var",
		req.Filter,
		(*variable.Variable)(nil),
	)

	if req.IncludeFormat {
		q = q.Relation("Format")
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListVariableRequest,
) (*pagination.ListResult[*variable.Variable], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*variable.Variable, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan variables", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*variable.Variable]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetVariableByIDRequest,
) (*variable.Variable, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(variable.Variable)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("var.id = ?", req.ID).
				Where("var.organization_id = ?", req.OrgID).
				Where("var.business_unit_id = ?", req.BuID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Variable")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	v *variable.Variable,
) (*variable.Variable, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("orgID", v.OrganizationID.String()),
		zap.String("buID", v.BusinessUnitID.String()),
		zap.String("key", v.Key),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(v).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert variable", zap.Error(err))
		return nil, err
	}

	return v, nil
}

func (r *repository) Update(
	ctx context.Context,
	v *variable.Variable,
) (*variable.Variable, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", v.GetID()),
		zap.Int64("version", v.Version),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := v.Version
	v.Version++

	results, err := db.NewUpdate().
		Model(v).
		Where("var.version = ?", ov).
		WherePK().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update variable", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Variable", v.GetID()); err != nil {
		return nil, err
	}

	return v, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.GetVariableByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	results, err := db.NewDelete().
		Model((*variable.Variable)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.
				Where("id = ?", req.ID).
				Where("organization_id = ?", req.OrgID).
				Where("business_unit_id = ?", req.BuID).
				Where("is_system = false")
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete variable", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "Variable", req.ID.String())
}

func (r *repository) GetVariablesByContext(
	ctx context.Context,
	req repositories.GetVariablesByContextRequest,
) ([]*variable.Variable, error) {
	log := r.l.With(
		zap.String("operation", "GetVariablesByContext"),
		zap.String("context", req.Context.String()),
		zap.String("orgID", req.OrgID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	var entities []*variable.Variable
	query := db.NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("var.organization_id = ?", req.OrgID).
				Where("var.applies_to = ?", req.Context)
		})

	if req.Active {
		query = query.Where("var.is_active = true")
	}

	if err = query.Scan(ctx); err != nil {
		log.Error("failed to get variables by context", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetActiveVariablesByKeys(
	ctx context.Context,
	req repositories.GetVariablesByKeysRequest,
) ([]*variable.Variable, error) {
	log := r.l.With(
		zap.String("operation", "GetActiveVariablesByKeys"),
		zap.String("orgID", req.OrgID.String()),
		zap.Int("keyCount", len(req.Keys)),
	)

	if len(req.Keys) == 0 {
		return []*variable.Variable{}, nil
	}

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	var entities []*variable.Variable
	err = db.NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("var.organization_id = ?", req.OrgID).
				Where("var.key IN (?)", bun.In(req.Keys)).
				Where("var.is_active = true")
		}).
		Relation("Format").
		Scan(ctx)
	if err != nil {
		log.Error("failed to get variables by keys", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByKey(
	ctx context.Context,
	orgID pulid.ID,
	key string,
) (*variable.Variable, error) {
	log := r.l.With(
		zap.String("operation", "GetByKey"),
		zap.String("orgID", orgID.String()),
		zap.String("key", key),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(variable.Variable)
	err = db.NewSelect().
		Model(entity).
		Where("var.organization_id = ?", orgID).
		Where("var.key = ?", key).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Variable")
	}

	return entity, nil
}

func (r *repository) ResolveVariable(
	ctx context.Context,
	req repositories.ResolveVariableRequest,
) (string, error) {
	log := r.l.With(
		zap.String("operation", "ResolveVariable"),
		zap.String("variableKey", req.Variable.Key),
		zap.String("variableID", req.Variable.GetID()),
	)

	if !req.Variable.IsValidated {
		return req.Variable.DefaultValue, fmt.Errorf(
			"variable %s has not been validated",
			req.Variable.Key,
		)
	}

	if !req.Variable.IsActive {
		return req.Variable.DefaultValue, fmt.Errorf("variable %s is not active", req.Variable.Key)
	}

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return req.Variable.DefaultValue, err
	}

	var result string

	err = db.RunInTx(
		ctx,
		&sql.TxOptions{ReadOnly: true},
		func(ctx context.Context, tx bun.Tx) error {
			queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			rawQuery := req.Variable.Query
			args := make([]any, 0)

			for k, v := range req.Params {
				placeholder := ":" + k
				count := strings.Count(rawQuery, placeholder)
				if count > 0 {
					rawQuery = strings.ReplaceAll(rawQuery, placeholder, "?")
					for range count {
						args = append(args, v)
					}
				}
			}

			if err = tx.NewRaw(rawQuery, args...).Scan(queryCtx, &result); err != nil {
				log.Error("failed to resolve variable",
					zap.Error(err),
					zap.String("query", req.Variable.Query),
					zap.Any("params", req.Params),
				)
				result = req.Variable.DefaultValue
				return nil
			}

			return nil
		},
	)
	if err != nil {
		log.Error("failed to execute variable resolution transaction", zap.Error(err))
		return req.Variable.DefaultValue, err
	}

	return result, nil
}

func (r *repository) ListFormats(
	ctx context.Context,
	req *repositories.ListVariableFormatRequest,
) (*pagination.ListResult[*variable.VariableFormat], error) {
	log := r.l.With(
		zap.String("operation", "ListFormats"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*variable.VariableFormat, 0, req.Filter.Limit)

	total, err := db.NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("vf.organization_id = ?", req.Filter.TenantOpts.OrgID).
				Where("vf.business_unit_id = ?", req.Filter.TenantOpts.BuID)
		}).
		Limit(req.Filter.Limit).
		Offset(req.Filter.Offset).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan variable formats", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*variable.VariableFormat]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetFormatByID(
	ctx context.Context,
	req repositories.GetVariableFormatByIDRequest,
) (*variable.VariableFormat, error) {
	log := r.l.With(
		zap.String("operation", "GetFormatByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(variable.VariableFormat)
	err = db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("vf.id = ?", req.ID).
				Where("vf.organization_id = ?", req.OrgID).
				Where("vf.business_unit_id = ?", req.BuID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "VariableFormat")
	}

	return entity, nil
}

func (r *repository) CreateFormat(
	ctx context.Context,
	f *variable.VariableFormat,
) (*variable.VariableFormat, error) {
	log := r.l.With(
		zap.String("operation", "CreateFormat"),
		zap.String("orgID", f.OrganizationID.String()),
		zap.String("buID", f.BusinessUnitID.String()),
		zap.String("name", f.Name),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(f).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert variable format", zap.Error(err))
		return nil, err
	}

	return f, nil
}

func (r *repository) UpdateFormat(
	ctx context.Context,
	f *variable.VariableFormat,
) (*variable.VariableFormat, error) {
	log := r.l.With(
		zap.String("operation", "UpdateFormat"),
		zap.String("id", f.GetID()),
		zap.Int64("version", f.Version),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := f.Version
	f.Version++

	results, err := db.NewUpdate().
		Model(f).
		Where("vf.version = ?", ov).
		WherePK().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update variable format", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "VariableFormat", f.GetID()); err != nil {
		return nil, err
	}

	return f, nil
}

func (r *repository) DeleteFormat(
	ctx context.Context,
	req repositories.GetVariableFormatByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "DeleteFormat"),
		zap.String("id", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	results, err := db.NewDelete().
		Model((*variable.VariableFormat)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.
				Where("id = ?", req.ID).
				Where("organization_id = ?", req.OrgID).
				Where("business_unit_id = ?", req.BuID).
				Where("is_system = false")
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete variable format", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "VariableFormat", req.ID.String())
}

func (r *repository) ExecuteFormatSQL(
	ctx context.Context,
	req repositories.ExecuteFormatSQLRequest,
) (string, error) {
	log := r.l.With(
		zap.String("operation", "ExecuteFormatSQL"),
	)

	validator := variableservice.NewFormatSQLValidator()
	if err := validator.Validate(req.FormatSQL); err != nil {
		log.Error("format SQL validation failed",
			zap.String("sql", req.FormatSQL),
			zap.Error(err),
		)
		return req.Value, err
	}

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return req.Value, err
	}

	var result string

	err = db.RunInTx(
		ctx,
		&sql.TxOptions{ReadOnly: true},
		func(ctx context.Context, tx bun.Tx) error {
			queryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			valueCount := strings.Count(req.FormatSQL, ":value")

			rawQuery := strings.ReplaceAll(req.FormatSQL, ":value", "?")
			selectQuery := fmt.Sprintf("SELECT %s", rawQuery)

			args := make([]any, valueCount)
			for i := range valueCount {
				args[i] = req.Value
			}

			if err = tx.NewRaw(selectQuery, args...).Scan(queryCtx, &result); err != nil {
				log.Error("failed to execute format SQL",
					zap.Error(err),
					zap.String("sql", req.FormatSQL),
					zap.String("value", req.Value),
				)
				result = req.Value
				return nil
			}

			return nil
		},
	)
	if err != nil {
		log.Error("failed to execute format SQL transaction", zap.Error(err))
		return req.Value, err
	}

	return result, nil
}
