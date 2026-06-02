//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edipartnerdocumentprofilerepository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.EDIPartnerDocumentProfileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-partner-document-profile-repository"),
	}
}

func (r *repository) ListPartnerDocumentProfiles(
	ctx context.Context,
	req *repositories.ListEDIPartnerDocumentProfilesRequest,
) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
	entities := make([]*edi.EDIPartnerDocumentProfile, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.EDIPartnerDocumentProfileColumns
	rel := buncolgen.EDIPartnerDocumentProfileRelations

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(rel.Partner).
		Relation(rel.DocumentType).
		Relation(rel.Template).
		Relation(rel.PartnerSettingsSchema).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EDIPartnerDocumentProfileScopeTenant(sq, req.Filter.TenantInfo)
		})

	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}

	if req.Direction != "" {
		query = query.Where(cols.Direction.Eq(), req.Direction)
	}
	if req.Standard != "" {
		query = query.Where(cols.Standard.Eq(), req.Standard)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	if req.PartnerID.IsNotNil() {
		query = query.Where(cols.EDIPartnerID.Eq(), req.PartnerID)
	}
	query = applyPartnerDocumentProfileSearch(query, req.Filter.Query)

	total, err := query.
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDIPartnerDocumentProfile]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) SelectPartnerDocumentProfileOptions(
	ctx context.Context,
	req *repositories.EDIPartnerDocumentProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
	entities := make(
		[]*edi.EDIPartnerDocumentProfile,
		0,
		req.SelectQueryRequest.Pagination.SafeLimit(),
	)
	cols := buncolgen.EDIPartnerDocumentProfileColumns
	rel := buncolgen.EDIPartnerDocumentProfileRelations

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(rel.Partner).
		Relation(rel.Template).
		Relation(rel.DocumentType).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EDIPartnerDocumentProfileScopeTenant(
				sq,
				req.SelectQueryRequest.TenantInfo,
			)
		})
	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where(cols.Direction.Eq(), req.Direction)
	}
	if req.Standard != "" {
		query = query.Where(cols.Standard.Eq(), req.Standard)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	if req.PartnerID.IsNotNil() {
		query = query.Where(cols.EDIPartnerID.Eq(), req.PartnerID)
	}
	query = applyPartnerDocumentProfileSearch(query, req.SelectQueryRequest.Query)

	total, err := query.
		Order(cols.Name.OrderAsc()).
		Limit(req.SelectQueryRequest.Pagination.SafeLimit()).
		Offset(req.SelectQueryRequest.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDIPartnerDocumentProfile]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetPartnerDocumentProfileByID(
	ctx context.Context,
	req repositories.GetEDIPartnerDocumentProfileByIDRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	entity := new(edi.EDIPartnerDocumentProfile)
	cols := buncolgen.EDIPartnerDocumentProfileColumns
	rel := buncolgen.EDIPartnerDocumentProfileRelations

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(rel.Partner).
		Relation(rel.DocumentType).
		Relation(rel.Template).
		Relation(rel.TemplateVersion).
		Relation(rel.PartnerSettingsSchema).
		Where(cols.ID.Eq(), req.ID).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EDIPartnerDocumentProfileScopeTenant(sq, req.TenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIPartnerDocumentProfile")
	}
	return entity, nil
}

func (r *repository) GetActivePartnerDocumentProfile(
	ctx context.Context,
	req repositories.GetActiveEDIPartnerDocumentProfileRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	entities := make([]*edi.EDIPartnerDocumentProfile, 0, 2)
	cols := buncolgen.EDIPartnerDocumentProfileColumns
	rel := buncolgen.EDIPartnerDocumentProfileRelations

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(rel.DocumentType).
		Relation(rel.Template).
		Relation(rel.PartnerSettingsSchema).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(cols.EDIPartnerID.Eq(), req.PartnerID).
				Where(cols.Status.Eq(), edi.DocumentStatusActive)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EDIPartnerDocumentProfileApplyTenant(req.TenantInfo)(sq)
		})

	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where(cols.Direction.Eq(), req.Direction)
	}

	err := query.
		Order(cols.CreatedAt.OrderDesc()).
		Limit(2).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIPartnerDocumentProfile")
	}

	if len(entities) == 0 {
		return nil, dberror.HandleNotFoundError(sql.ErrNoRows, "EDIPartnerDocumentProfile")
	}
	if (req.TransactionSet == "" || req.Direction == "") && len(entities) > 1 {
		return nil, errors.New(
			"multiple active EDI document profiles match partner; transaction set and direction are required",
		)
	}

	return entities[0], nil
}

func (r *repository) CreatePartnerDocumentProfile(
	ctx context.Context,
	entity *edi.EDIPartnerDocumentProfile,
) (*edi.EDIPartnerDocumentProfile, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdatePartnerDocumentProfile(
	ctx context.Context,
	entity *edi.EDIPartnerDocumentProfile,
) (*edi.EDIPartnerDocumentProfile, error) {
	ov := entity.Version
	entity.Version++
	cols := buncolgen.EDIPartnerDocumentProfileColumns
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Column(
			cols.TemplateID.Bare(),
			cols.TemplateVersionID.Bare(),
			cols.DocumentTypeID.Bare(),
			cols.Name.Bare(),
			cols.Status.Bare(),
			cols.Direction.Bare(),
			cols.Standard.Bare(),
			cols.TransactionSet.Bare(),
			cols.X12VersionOverride.Bare(),
			cols.FunctionalGroupID.Bare(),
			cols.Envelope.Bare(),
			cols.Acknowledgment.Bare(),
			cols.ValidationMode.Bare(),
			cols.PartnerSettings.Bare(),
			cols.PartnerSettingsSchemaID.Bare(),
			cols.PartnerSettingsSchemaVersion.Bare(),
			cols.Version.Bare(),
			cols.UpdatedAt.Bare(),
		).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(
		results,
		"EDIPartnerDocumentProfile",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}
	return entity, nil
}

func applyPartnerDocumentProfileSearch(query *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}

	term := "%" + strings.ToLower(search) + "%"
	cols := buncolgen.EDIPartnerDocumentProfileColumns
	partnerCols := buncolgen.EDIPartnerColumns
	documentTypeCols := buncolgen.EDIDocumentTypeColumns

	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(cols.Name.LowerLike(), term).
			WhereOr(partnerCols.Code.WithAlias("partner").LowerLike(), term).
			WhereOr(partnerCols.Name.WithAlias("partner").LowerLike(), term).
			WhereOr(documentTypeCols.Name.WithAlias("document_type").LowerLike(), term)
	})
}
