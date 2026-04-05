package documentparsingrulerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentparsingrule"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.DocumentParsingRuleRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-parsing-rule-repository"),
	}
}

func (r *repository) ListRuleSets(
	ctx context.Context,
	req repositories.ListDocumentParsingRuleSetsRequest,
) ([]*documentparsingrule.RuleSet, error) {
	items := make([]*documentparsingrule.RuleSet, 0)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dprs.organization_id = ?", req.TenantInfo.OrgID).
		Where("dprs.business_unit_id = ?", req.TenantInfo.BuID)
	if req.DocumentKind != "" {
		q = q.Where("dprs.document_kind = ?", req.DocumentKind)
	}
	err := q.OrderExpr("dprs.priority DESC, dprs.created_at ASC").Scan(ctx)
	return items, err
}

func (r *repository) GetRuleSet(
	ctx context.Context,
	req repositories.GetDocumentParsingRuleSetRequest,
) (*documentparsingrule.RuleSet, error) {
	entity := new(documentparsingrule.RuleSet)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dprs.id = ?", req.ID).
		Where("dprs.organization_id = ?", req.TenantInfo.OrgID).
		Where("dprs.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DocumentParsingRuleSet")
	}
	return entity, nil
}

func (r *repository) CreateRuleSet(
	ctx context.Context,
	entity *documentparsingrule.RuleSet,
) (*documentparsingrule.RuleSet, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdateRuleSet(
	ctx context.Context,
	entity *documentparsingrule.RuleSet,
) (*documentparsingrule.RuleSet, error) {
	ov := entity.Version
	entity.Version++
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(result, "DocumentParsingRuleSet", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) DeleteRuleSet(
	ctx context.Context,
	req repositories.GetDocumentParsingRuleSetRequest,
) error {
	result, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*documentparsingrule.RuleSet)(nil)).
		Where("id = ?", req.ID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}
	return dberror.CheckRowsAffected(result, "DocumentParsingRuleSet", req.ID.String())
}

func (r *repository) ListVersions(
	ctx context.Context,
	req repositories.ListDocumentParsingRuleVersionsRequest,
) ([]*documentparsingrule.RuleVersion, error) {
	items := make([]*documentparsingrule.RuleVersion, 0)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dprv.rule_set_id = ?", req.RuleSetID).
		Where("dprv.organization_id = ?", req.TenantInfo.OrgID).
		Where("dprv.business_unit_id = ?", req.TenantInfo.BuID)
	if !req.IncludeAll {
		q = q.Where("dprv.status != ?", documentparsingrule.VersionStatusArchived)
	}
	err := q.OrderExpr("dprv.version_number DESC, dprv.created_at DESC").Scan(ctx)
	return items, err
}

func (r *repository) GetVersion(
	ctx context.Context,
	req repositories.GetDocumentParsingRuleVersionRequest,
) (*documentparsingrule.RuleVersion, error) {
	entity := new(documentparsingrule.RuleVersion)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dprv.id = ?", req.ID).
		Where("dprv.organization_id = ?", req.TenantInfo.OrgID).
		Where("dprv.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DocumentParsingRuleVersion")
	}
	return entity, nil
}

func (r *repository) GetVersionWithRuleSet(
	ctx context.Context,
	req repositories.GetDocumentParsingRuleVersionRequest,
) (*documentparsingrule.RuleVersion, *documentparsingrule.RuleSet, error) {
	version, err := r.GetVersion(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	set, err := r.GetRuleSet(ctx, repositories.GetDocumentParsingRuleSetRequest{
		ID: version.RuleSetID,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	return version, set, nil
}

func (r *repository) CreateVersion(
	ctx context.Context,
	entity *documentparsingrule.RuleVersion,
) (*documentparsingrule.RuleVersion, error) {
	if entity.VersionNumber > 0 {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
			return nil, r.mapCreateVersionError(err)
		}
		return entity, nil
	}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		lockToken := 0
		if err := r.db.DBForContext(c).
			NewSelect().
			Model((*documentparsingrule.RuleSet)(nil)).
			ColumnExpr("1").
			Where("id = ?", entity.RuleSetID).
			Where("organization_id = ?", entity.OrganizationID).
			Where("business_unit_id = ?", entity.BusinessUnitID).
			For("UPDATE").
			Scan(c, &lockToken); err != nil {
			return dberror.HandleNotFoundError(err, "DocumentParsingRuleSet")
		}

		next, err := r.NextVersionNumber(
			c,
			entity.RuleSetID,
			entity.OrganizationID,
			entity.BusinessUnitID,
		)
		if err != nil {
			return err
		}
		entity.VersionNumber = next

		_, err = r.db.DBForContext(c).NewInsert().Model(entity).Returning("*").Exec(c)
		return r.mapCreateVersionError(err)
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(
			err,
			"The parsing rule version is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *repository) UpdateVersion(
	ctx context.Context,
	entity *documentparsingrule.RuleVersion,
) (*documentparsingrule.RuleVersion, error) {
	ov := entity.Version
	entity.Version++
	cols := buncolgen.RuleVersionColumns
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(result, "DocumentParsingRuleVersion", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) ArchivePublishedVersions(
	ctx context.Context,
	ruleSetID, orgID, buID pulid.ID,
) error {
	cols := buncolgen.RuleVersionColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*documentparsingrule.RuleVersion)(nil)).
		Set(cols.Status.Set(), documentparsingrule.VersionStatusArchived).
		Set(cols.UpdatedAt.SetExpr("extract(epoch from current_timestamp)::bigint")).
		Where(cols.RuleSetID.Eq(), ruleSetID).
		Where(cols.OrganizationID.Eq(), orgID).
		Where(cols.BusinessUnitID.Eq(), buID).
		Where(cols.Status.Eq(), documentparsingrule.VersionStatusPublished).
		Exec(ctx)
	return err
}

func (r *repository) SetPublishedVersion(
	ctx context.Context,
	ruleSetID, versionID, orgID, buID pulid.ID,
) error {
	cols := buncolgen.RuleSetColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*documentparsingrule.RuleSet)(nil)).
		Set(cols.PublishedVersionID.Set(), versionID).
		Set(cols.UpdatedAt.SetExpr("extract(epoch from current_timestamp)::bigint")).
		Where(cols.ID.Eq(), ruleSetID).
		Where(cols.OrganizationID.Eq(), orgID).
		Where(cols.BusinessUnitID.Eq(), buID).
		Exec(ctx)
	return err
}

func (r *repository) NextVersionNumber(
	ctx context.Context,
	ruleSetID, orgID, buID pulid.ID,
) (int, error) {
	var next int
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*documentparsingrule.RuleVersion)(nil)).
		ColumnExpr("COALESCE(MAX(version_number), 0) + 1").
		Where("rule_set_id = ?", ruleSetID).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Scan(ctx, &next); err != nil {
		return 0, err
	}
	if next <= 0 {
		next = 1
	}
	return next, nil
}

func (r *repository) ListPublishedVersionsByDocumentKind(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	documentKind string,
) ([]*repositories.PublishedDocumentParsingRuleVersion, error) {
	type joinedRow struct {
		RuleSetID             pulid.ID                          `bun:"rule_set_id"`
		RuleSetOrganizationID pulid.ID                          `bun:"rule_set_organization_id"`
		RuleSetBusinessUnitID pulid.ID                          `bun:"rule_set_business_unit_id"`
		RuleSetName           string                            `bun:"rule_set_name"`
		RuleSetDescription    string                            `bun:"rule_set_description"`
		RuleSetDocumentKind   documentparsingrule.DocumentKind  `bun:"rule_set_document_kind"`
		RuleSetPriority       int                               `bun:"rule_set_priority"`
		PublishedVersionID    *pulid.ID                         `bun:"rule_set_published_version_id"`
		RuleSetVersion        int64                             `bun:"rule_set_version"`
		RuleSetCreatedAt      int64                             `bun:"rule_set_created_at"`
		RuleSetUpdatedAt      int64                             `bun:"rule_set_updated_at"`
		VersionID             pulid.ID                          `bun:"version_id"`
		VersionRuleSetID      pulid.ID                          `bun:"version_rule_set_id"`
		VersionOrganizationID pulid.ID                          `bun:"version_organization_id"`
		VersionBusinessUnitID pulid.ID                          `bun:"version_business_unit_id"`
		VersionNumber         int                               `bun:"version_number"`
		VersionStatus         documentparsingrule.VersionStatus `bun:"version_status"`
		VersionLabel          string                            `bun:"version_label"`
		VersionParserMode     documentparsingrule.ParserMode    `bun:"version_parser_mode"`
		VersionMatchConfig    documentparsingrule.MatchConfig   `bun:"version_match_config"`
		VersionRuleDocument   documentparsingrule.RuleDocument  `bun:"version_rule_document"`
		VersionValidation     map[string]any                    `bun:"version_validation_summary"`
		VersionPublishedAt    *int64                            `bun:"version_published_at"`
		VersionPublishedByID  *pulid.ID                         `bun:"version_published_by_id"`
		VersionOptimisticLock int64                             `bun:"version_version"`
		VersionCreatedAt      int64                             `bun:"version_created_at"`
		VersionUpdatedAt      int64                             `bun:"version_updated_at"`
	}
	rows := make([]*joinedRow, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("document_parsing_rule_sets AS dprs").
		Join("JOIN document_parsing_rule_versions AS dprv ON dprv.id = dprs.published_version_id AND dprv.rule_set_id = dprs.id AND dprv.organization_id = dprs.organization_id AND dprv.business_unit_id = dprs.business_unit_id").
		ColumnExpr("dprs.id AS rule_set_id, dprs.organization_id AS rule_set_organization_id, dprs.business_unit_id AS rule_set_business_unit_id, dprs.name AS rule_set_name, dprs.description AS rule_set_description, dprs.document_kind AS rule_set_document_kind, dprs.priority AS rule_set_priority, dprs.published_version_id AS rule_set_published_version_id, dprs.version AS rule_set_version, dprs.created_at AS rule_set_created_at, dprs.updated_at AS rule_set_updated_at").
		ColumnExpr("dprv.id AS version_id, dprv.rule_set_id AS version_rule_set_id, dprv.organization_id AS version_organization_id, dprv.business_unit_id AS version_business_unit_id, dprv.version_number AS version_number, dprv.status AS version_status, dprv.label AS version_label, dprv.parser_mode AS version_parser_mode, dprv.match_config AS version_match_config, dprv.rule_document AS version_rule_document, dprv.validation_summary AS version_validation_summary, dprv.published_at AS version_published_at, dprv.published_by_id AS version_published_by_id, dprv.version AS version_version, dprv.created_at AS version_created_at, dprv.updated_at AS version_updated_at").
		Where("dprs.organization_id = ?", tenantInfo.OrgID).
		Where("dprs.business_unit_id = ?", tenantInfo.BuID).
		Where("dprs.document_kind = ?", documentKind).
		Where("dprs.published_version_id IS NOT NULL").
		Where("dprv.status = ?", documentparsingrule.VersionStatusPublished).
		OrderExpr("dprs.priority DESC, dprv.published_at DESC NULLS LAST").
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	items := make([]*repositories.PublishedDocumentParsingRuleVersion, 0, len(rows))
	for _, row := range rows {
		set := documentparsingrule.RuleSet{
			ID:                 row.RuleSetID,
			OrganizationID:     row.RuleSetOrganizationID,
			BusinessUnitID:     row.RuleSetBusinessUnitID,
			Name:               row.RuleSetName,
			Description:        row.RuleSetDescription,
			DocumentKind:       row.RuleSetDocumentKind,
			Priority:           row.RuleSetPriority,
			PublishedVersionID: row.PublishedVersionID,
			Version:            row.RuleSetVersion,
			CreatedAt:          row.RuleSetCreatedAt,
			UpdatedAt:          row.RuleSetUpdatedAt,
		}
		version := documentparsingrule.RuleVersion{
			ID:                row.VersionID,
			RuleSetID:         row.VersionRuleSetID,
			OrganizationID:    row.VersionOrganizationID,
			BusinessUnitID:    row.VersionBusinessUnitID,
			VersionNumber:     row.VersionNumber,
			Status:            row.VersionStatus,
			Label:             row.VersionLabel,
			ParserMode:        row.VersionParserMode,
			MatchConfig:       row.VersionMatchConfig,
			RuleDocument:      row.VersionRuleDocument,
			ValidationSummary: row.VersionValidation,
			PublishedAt:       row.VersionPublishedAt,
			PublishedByID:     row.VersionPublishedByID,
			Version:           row.VersionOptimisticLock,
			CreatedAt:         row.VersionCreatedAt,
			UpdatedAt:         row.VersionUpdatedAt,
		}
		items = append(items, &repositories.PublishedDocumentParsingRuleVersion{
			RuleSet: &set,
			Version: &version,
		})
	}
	return items, nil
}

func (r *repository) ListFixtures(
	ctx context.Context,
	req repositories.ListDocumentParsingRuleFixturesRequest,
) ([]*documentparsingrule.Fixture, error) {
	items := make([]*documentparsingrule.Fixture, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dprf.rule_set_id = ?", req.RuleSetID).
		Where("dprf.organization_id = ?", req.TenantInfo.OrgID).
		Where("dprf.business_unit_id = ?", req.TenantInfo.BuID).
		OrderExpr("dprf.created_at ASC").
		Scan(ctx)
	return items, err
}

func (r *repository) GetFixture(
	ctx context.Context,
	req repositories.GetDocumentParsingRuleFixtureRequest,
) (*documentparsingrule.Fixture, error) {
	entity := new(documentparsingrule.Fixture)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dprf.id = ?", req.ID).
		Where("dprf.organization_id = ?", req.TenantInfo.OrgID).
		Where("dprf.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DocumentParsingRuleFixture")
	}
	return entity, nil
}

func (r *repository) CreateFixture(
	ctx context.Context,
	entity *documentparsingrule.Fixture,
) (*documentparsingrule.Fixture, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdateFixture(
	ctx context.Context,
	entity *documentparsingrule.Fixture,
) (*documentparsingrule.Fixture, error) {
	ov := entity.Version
	entity.Version++
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(result, "DocumentParsingRuleFixture", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) DeleteFixture(
	ctx context.Context,
	req repositories.GetDocumentParsingRuleFixtureRequest,
) error {
	result, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*documentparsingrule.Fixture)(nil)).
		Where("id = ?", req.ID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}
	return dberror.CheckRowsAffected(result, "DocumentParsingRuleFixture", req.ID.String())
}

func (r *repository) mapCreateVersionError(err error) error {
	if err == nil {
		return nil
	}
	if dberror.IsUniqueConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == "uq_document_parsing_rule_versions_rule_set_version" {
		return errortypes.NewConflictError("A parsing rule version was created concurrently. Retry the request.").
			WithInternal(err)
	}
	return err
}
