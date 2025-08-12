package repositories

import (
	"context"
	"database/sql"

	domainedi "github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/db"
	portrepo "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type EDIPartnerConfigRepoParams struct {
	fx.In
	DB     db.Connection
	Logger *logger.Logger
}

type ediPartnerConfigRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewEDIPartnerConfigRepository(
	p EDIPartnerConfigRepoParams,
) portrepo.EDIPartnerConfigRepository {
	log := p.Logger.With().Str("repository", "edi_partner_config").Logger()
	return &ediPartnerConfigRepository{db: p.DB, l: &log}
}

func (r *ediPartnerConfigRepository) GetByID(
	ctx context.Context,
	buID, orgID pulid.ID,
	id pulid.ID,
) (*domainedi.PartnerConfig, error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}
	pc := new(domainedi.PartnerConfig)
	q := dba.NewSelect().Model(pc).
		Where("ep.id = ?", id).
		Where("ep.business_unit_id = ?", buID).
		Where("ep.organization_id = ?", orgID)
	if err := q.Limit(1).Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewValidationError(
				"id",
				errors.ErrNotFound,
				"EDI partner config not found",
			)
		}
		r.l.Error().Err(err).Msg("GetByID failed")
		return nil, err
	}
	return pc, nil
}

func (r *ediPartnerConfigRepository) GetByKey(
	ctx context.Context,
	buID, orgID pulid.ID,
	name string,
) (*domainedi.PartnerConfig, error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}
	pc := new(domainedi.PartnerConfig)
	q := dba.NewSelect().Model(pc).
		Where("ep.name = ?", name).
		Where("ep.business_unit_id = ?", buID).
		Where("ep.organization_id = ?", orgID)
	if err := q.Limit(1).Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewValidationError(
				"name",
				errors.ErrNotFound,
				"EDI partner config not found",
			)
		}
		r.l.Error().Err(err).Msg("GetByKey failed")
		return nil, err
	}
	return pc, nil
}

func (r *ediPartnerConfigRepository) List(
	ctx context.Context,
	buID, orgID pulid.ID,
	limit int,
	afterName string,
	afterID pulid.ID,
) ([]*domainedi.PartnerConfig, string, error) {
	dba, err := r.db.ReadDB(ctx)
	if err != nil {
		return nil, "", err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows := make([]*domainedi.PartnerConfig, 0, limit+1)
	q := dba.NewSelect().Model(&rows).
		Where("ep.business_unit_id = ?", buID).
		Where("ep.organization_id = ?", orgID)
	if afterName != "" && !afterID.IsNil() {
		q = q.Where("(ep.name > ?) OR (ep.name = ? AND ep.id > ?)", afterName, afterName, afterID)
	}
	if err := q.Order("ep.name ASC, ep.id ASC").Limit(limit + 1).Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return []*domainedi.PartnerConfig{}, "", nil
		}
		r.l.Error().Err(err).Msg("List failed")
		return nil, "", err
	}
	next := ""
	if len(rows) > limit {
		rows = rows[:limit]
		// next token returned empty here; the gRPC service handles encoding
		next = "more"
	}
	return rows, next, nil
}
