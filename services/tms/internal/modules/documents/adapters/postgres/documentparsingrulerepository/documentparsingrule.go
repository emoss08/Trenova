package documentparsingrulerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentparsingrule"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentparsingrule/documentparsingruledb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentparsingruleports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const versionNumberConstraint = "uq_document_parsing_rule_versions_rule_set_version"

var ruleSetErrors = dberror.Mapping{
	Entity: "DocumentParsingRuleSet",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_document_parsing_rule_sets_name_kind_scope",
			Field:   "name",
			Message: "A parsing rule set with this name already exists for this document kind",
		},
	},
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_document_parsing_rule_sets_published_version",
			Field:   "publishedVersionId",
			Message: "Published version does not exist",
		},
	},
}

var ruleVersionErrors = dberror.Mapping{
	Entity: "DocumentParsingRuleVersion",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_document_parsing_rule_versions_rule_set",
			Field:   "ruleSetId",
			Message: "Parsing rule set does not exist",
		},
	},
}

var fixtureErrors = dberror.Mapping{
	Entity: "DocumentParsingRuleFixture",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_document_parsing_rule_fixtures_rule_set",
			Field:   "ruleSetId",
			Message: "Parsing rule set does not exist",
		},
	},
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

func New(p Params) documentparsingruleports.DocumentParsingRuleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-parsing-rule-repository"),
	}
}

func (r *repository) ListRuleSets(
	ctx context.Context,
	req documentparsingruleports.ListDocumentParsingRuleSetsRequest,
) ([]*documentparsingrule.RuleSet, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*documentparsingrule.RuleSet, error) {
		s := documentparsingruledb.RuleSets
		items, err := s.Where(
			keel.When(
				req.DocumentKind != "",
				s.DocumentKind.Eq(documentparsingrule.DocumentKind(req.DocumentKind)),
			),
		).
			Order(s.Priority.Desc(), s.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))

		return orEmpty(items), err
	})
}

func (r *repository) GetRuleSet(
	ctx context.Context,
	req documentparsingruleports.GetDocumentParsingRuleSetRequest,
) (*documentparsingrule.RuleSet, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*documentparsingrule.RuleSet, error) {
		entity, err := documentparsingruledb.RuleSets.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, ruleSetErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateRuleSet(
	ctx context.Context,
	entity *documentparsingrule.RuleSet,
) (*documentparsingrule.RuleSet, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*documentparsingrule.RuleSet, error) {
		if err := documentparsingruledb.RuleSets.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, ruleSetErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateRuleSet(
	ctx context.Context,
	entity *documentparsingrule.RuleSet,
) (*documentparsingrule.RuleSet, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*documentparsingrule.RuleSet, error) {
		if err := documentparsingruledb.RuleSets.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, ruleSetErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeleteRuleSet(
	ctx context.Context,
	req documentparsingruleports.GetDocumentParsingRuleSetRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := documentparsingruledb.RuleSets.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&documentparsingrule.RuleSet{ID: req.ID},
		)

		return ruleSetErrors.Map(err, req.ID.String())
	})
}

func (r *repository) ListVersions(
	ctx context.Context,
	req documentparsingruleports.ListDocumentParsingRuleVersionsRequest,
) ([]*documentparsingrule.RuleVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*documentparsingrule.RuleVersion, error) {
			v := documentparsingruledb.RuleVersions
			items, err := v.Where(
				v.RuleSetID.Eq(req.RuleSetID),
				keel.When(!req.IncludeAll, v.Status.Ne(documentparsingrule.VersionStatusArchived)),
			).
				Order(v.VersionNumber.Desc(), v.CreatedAt.Desc()).
				All(ctx, r.db.Keel(ctx))

			return orEmpty(items), err
		},
	)
}

func (r *repository) GetVersion(
	ctx context.Context,
	req documentparsingruleports.GetDocumentParsingRuleVersionRequest,
) (*documentparsingrule.RuleVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*documentparsingrule.RuleVersion, error) {
			entity, err := documentparsingruledb.RuleVersions.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, ruleVersionErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetVersionWithRuleSet(
	ctx context.Context,
	req documentparsingruleports.GetDocumentParsingRuleVersionRequest,
) (*documentparsingrule.RuleVersion, *documentparsingrule.RuleSet, error) {
	return dbtx.Read2(
		ctx,
		r.db,
		func(ctx context.Context) (*documentparsingrule.RuleVersion, *documentparsingrule.RuleSet, error) {
			v := documentparsingruledb.RuleVersions
			version, err := v.Where(v.ID.Eq(req.ID)).Load(v.RuleSet()).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, nil, ruleVersionErrors.Map(err, req.ID.String())
			}
			if version.RuleSet == nil {
				return nil, nil, ruleSetErrors.Map(keel.ErrNotFound, version.RuleSetID.String())
			}

			set := version.RuleSet
			version.RuleSet = nil

			return version, set, nil
		},
	)
}

func (r *repository) CreateVersion(
	ctx context.Context,
	entity *documentparsingrule.RuleVersion,
) (*documentparsingrule.RuleVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documentparsingrule.RuleVersion, error) {
			db := r.db.Keel(ctx)
			v := documentparsingruledb.RuleVersions

			if entity.VersionNumber > 0 {
				if err := v.Create(ctx, db, entity); err != nil {
					return nil, mapCreateVersionError(err, entity)
				}

				return entity, nil
			}

			s := documentparsingruledb.RuleSets
			if _, err := s.Pick(s.ID).
				Where(s.ID.Eq(entity.RuleSetID)).
				Lock(keel.ForUpdate).
				One(ctx, db); err != nil {
				return nil, dberror.MapRetryableTransactionError(
					ruleSetErrors.Map(err, entity.RuleSetID.String()),
					"The parsing rule version is busy. Retry the request.",
				)
			}

			next, err := nextVersionNumber(ctx, db, entity.RuleSetID)
			if err != nil {
				return nil, err
			}
			entity.VersionNumber = next

			if err = v.Create(ctx, db, entity); err != nil {
				return nil, dberror.MapRetryableTransactionError(
					mapCreateVersionError(err, entity),
					"The parsing rule version is busy. Retry the request.",
				)
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateVersion(
	ctx context.Context,
	entity *documentparsingrule.RuleVersion,
) (*documentparsingrule.RuleVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documentparsingrule.RuleVersion, error) {
			err := documentparsingruledb.RuleVersions.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				return nil, ruleVersionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ArchivePublishedVersions(
	ctx context.Context,
	ruleSetID, _, _ pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		v := documentparsingruledb.RuleVersions
		_, err := v.Where(
			v.RuleSetID.Eq(ruleSetID),
			v.Status.Eq(documentparsingrule.VersionStatusPublished),
		).Update(ctx, r.db.Keel(ctx), v.Status.To(documentparsingrule.VersionStatusArchived))

		return err
	})
}

func (r *repository) SetPublishedVersion(
	ctx context.Context,
	ruleSetID, versionID, _, _ pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		s := documentparsingruledb.RuleSets
		_, err := s.Where(s.ID.Eq(ruleSetID)).
			Update(ctx, r.db.Keel(ctx), s.PublishedVersionID.To(versionID))

		return ruleSetErrors.Map(err, ruleSetID.String())
	})
}

func (r *repository) NextVersionNumber(
	ctx context.Context,
	ruleSetID, _, _ pulid.ID,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		return nextVersionNumber(ctx, r.db.Keel(ctx), ruleSetID)
	})
}

func nextVersionNumber(ctx context.Context, db *keel.DB, ruleSetID pulid.ID) (int, error) {
	v := documentparsingruledb.RuleVersions
	current, err := keel.IntoOne[*int64](
		ctx,
		db,
		v.Pick(v.VersionNumber.Max()).Where(v.RuleSetID.Eq(ruleSetID)),
	)
	if err != nil {
		return 0, err
	}
	if current == nil || *current < 0 {
		return 1, nil
	}

	return int(*current) + 1, nil
}

func (r *repository) ListPublishedVersionsByDocumentKind(
	ctx context.Context,
	_ pagination.TenantInfo,
	documentKind string,
) ([]*documentparsingruleports.PublishedDocumentParsingRuleVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*documentparsingruleports.PublishedDocumentParsingRuleVersion, error) {
			v := documentparsingruledb.RuleVersions
			set := v.RuleSet()
			versions, err := v.Where(
				set.DocumentKind.Eq(documentparsingrule.DocumentKind(documentKind)),
				set.PublishedVersionID.EqCol(v.ID),
				v.Status.Eq(documentparsingrule.VersionStatusPublished),
			).
				Load(set).
				Order(set.Priority.Desc(), v.PublishedAt.Desc().NullsLast()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}

			items := make(
				[]*documentparsingruleports.PublishedDocumentParsingRuleVersion,
				0,
				len(versions),
			)
			for _, version := range versions {
				ruleSet := version.RuleSet
				version.RuleSet = nil
				items = append(items, &documentparsingruleports.PublishedDocumentParsingRuleVersion{
					RuleSet: ruleSet,
					Version: version,
				})
			}

			return items, nil
		},
	)
}

func (r *repository) ListFixtures(
	ctx context.Context,
	req documentparsingruleports.ListDocumentParsingRuleFixturesRequest,
) ([]*documentparsingrule.Fixture, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*documentparsingrule.Fixture, error) {
		f := documentparsingruledb.Fixtures
		items, err := f.Where(f.RuleSetID.Eq(req.RuleSetID)).
			Order(f.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))

		return orEmpty(items), err
	})
}

func (r *repository) GetFixture(
	ctx context.Context,
	req documentparsingruleports.GetDocumentParsingRuleFixtureRequest,
) (*documentparsingrule.Fixture, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*documentparsingrule.Fixture, error) {
		entity, err := documentparsingruledb.Fixtures.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, fixtureErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateFixture(
	ctx context.Context,
	entity *documentparsingrule.Fixture,
) (*documentparsingrule.Fixture, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*documentparsingrule.Fixture, error) {
		if err := documentparsingruledb.Fixtures.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, fixtureErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateFixture(
	ctx context.Context,
	entity *documentparsingrule.Fixture,
) (*documentparsingrule.Fixture, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*documentparsingrule.Fixture, error) {
		if err := documentparsingruledb.Fixtures.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, fixtureErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeleteFixture(
	ctx context.Context,
	req documentparsingruleports.GetDocumentParsingRuleFixtureRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := documentparsingruledb.Fixtures.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&documentparsingrule.Fixture{ID: req.ID},
		)

		return fixtureErrors.Map(err, req.ID.String())
	})
}

func mapCreateVersionError(err error, entity *documentparsingrule.RuleVersion) error {
	if dberror.IsUniqueConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == versionNumberConstraint {
		return errortypes.NewConflictError(
			"A parsing rule version was created concurrently. Retry the request.",
		).WithInternal(err)
	}

	return ruleVersionErrors.Map(err, entity.ID.String())
}

func orEmpty[T any](items []*T) []*T {
	if items == nil {
		return []*T{}
	}

	return items
}
