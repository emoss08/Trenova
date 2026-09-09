package iftarepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	jurisdictionKeySeparator = "_"
	defaultJurisdictionCount = 64
)

func (r *repository) ListJurisdictions(
	ctx context.Context,
	req *repositories.ListJurisdictionsRequest,
) ([]*ifta.Jurisdiction, error) {
	cols := buncolgen.JurisdictionColumns
	entities := make([]*ifta.Jurisdiction, 0, defaultJurisdictionCount)

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Order(cols.SortOrder.OrderAsc()).
		Order(cols.CountryCode.OrderAsc()).
		Order(cols.Code.OrderAsc())

	if req.MembersOnly {
		q = q.Where(cols.IsIftaMember.IsTrue())
	}
	if req.CountryCode != "" {
		q = q.Where(cols.CountryCode.Eq(), strings.ToUpper(strings.TrimSpace(req.CountryCode)))
	}
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.In(req.Statuses))
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list ifta jurisdictions", zap.Error(err))
		return nil, fmt.Errorf("list ifta jurisdictions: %w", err)
	}

	return entities, nil
}

func (r *repository) GetJurisdictionByID(
	ctx context.Context,
	id pulid.ID,
) (*ifta.Jurisdiction, error) {
	entity := new(ifta.Jurisdiction)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(buncolgen.JurisdictionColumns.ID.Eq(), id).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "IFTA jurisdiction")
	}

	return entity, nil
}

func (r *repository) GetJurisdictionsByIDs(
	ctx context.Context,
	ids []pulid.ID,
) ([]*ifta.Jurisdiction, error) {
	if len(ids) == 0 {
		return []*ifta.Jurisdiction{}, nil
	}

	entities := make([]*ifta.Jurisdiction, 0, len(ids))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(buncolgen.JurisdictionColumns.ID.In(), bun.List(ids)).
		Order(buncolgen.JurisdictionColumns.SortOrder.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to load ifta jurisdictions by id", zap.Error(err))
		return nil, fmt.Errorf("get ifta jurisdictions by ids: %w", err)
	}

	return entities, nil
}

func (r *repository) GetJurisdictionByCode(
	ctx context.Context,
	countryCode, code string,
) (*ifta.Jurisdiction, error) {
	cols := buncolgen.JurisdictionColumns
	entity := new(ifta.Jurisdiction)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.CountryCode.Eq(), strings.ToUpper(strings.TrimSpace(countryCode))).
		Where(cols.Code.Eq(), strings.ToUpper(strings.TrimSpace(code))).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "IFTA jurisdiction")
	}

	return entity, nil
}

func splitJurisdictionKey(key string) (countryCode, code string, ok bool) {
	trimmed := strings.ToUpper(strings.TrimSpace(key))
	if trimmed == "" {
		return "", "", false
	}

	if idx := strings.IndexAny(trimmed, jurisdictionKeySeparator+"-"); idx >= 0 {
		countryCode = trimmed[:idx]
		code = trimmed[idx+1:]
	} else {
		countryCode = ifta.CountryCodeUS
		code = trimmed
	}

	if countryCode == "" || code == "" {
		return "", "", false
	}

	return countryCode, code, true
}

func (r *repository) FindJurisdictionsByCodes(
	ctx context.Context,
	keys []string,
) (map[string]*ifta.Jurisdiction, error) {
	out := make(map[string]*ifta.Jurisdiction, len(keys))
	if len(keys) == 0 {
		return out, nil
	}

	normalised := make(map[string]string, len(keys))
	countries := make([]string, 0, 2)
	codes := make([]string, 0, len(keys))
	seenCountry := make(map[string]struct{}, 2)
	seenCode := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		countryCode, code, ok := splitJurisdictionKey(key)
		if !ok {
			continue
		}
		normalised[key] = ifta.JurisdictionKey(countryCode, code)
		if _, dup := seenCountry[countryCode]; !dup {
			seenCountry[countryCode] = struct{}{}
			countries = append(countries, countryCode)
		}
		if _, dup := seenCode[code]; !dup {
			seenCode[code] = struct{}{}
			codes = append(codes, code)
		}
	}
	if len(codes) == 0 {
		return out, nil
	}

	cols := buncolgen.JurisdictionColumns
	entities := make([]*ifta.Jurisdiction, 0, len(codes))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.CountryCode.In(), bun.In(countries)).
		Where(cols.Code.In(), bun.In(codes)).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to find ifta jurisdictions by code", zap.Error(err))
		return nil, fmt.Errorf("find ifta jurisdictions by codes: %w", err)
	}

	byKey := make(map[string]*ifta.Jurisdiction, len(entities))
	for _, entity := range entities {
		byKey[entity.Key()] = entity
	}
	for key, canonical := range normalised {
		if entity, ok := byKey[canonical]; ok {
			out[key] = entity
		}
	}

	return out, nil
}
