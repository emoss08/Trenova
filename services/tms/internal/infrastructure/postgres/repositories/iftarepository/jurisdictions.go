package iftarepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	jurisdictionKeySeparator = "_"
	defaultJurisdictionCount = 64
)

func (r *repository) allJurisdictions(ctx context.Context) ([]*ifta.Jurisdiction, error) {
	cached, err := r.jurisdictionCache.GetAll(ctx)
	if err == nil && len(cached) > 0 {
		return cached, nil
	}

	cols := buncolgen.JurisdictionColumns
	entities := make([]*ifta.Jurisdiction, 0, defaultJurisdictionCount)
	if err = r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Order(cols.SortOrder.OrderAsc()).
		Order(cols.CountryCode.OrderAsc()).
		Order(cols.Code.OrderAsc()).
		Scan(ctx); err != nil {
		r.l.Error("failed to load ifta jurisdictions", zap.Error(err))
		return nil, fmt.Errorf("load ifta jurisdictions: %w", err)
	}

	if cacheErr := r.jurisdictionCache.Set(ctx, entities); cacheErr != nil {
		r.l.Warn("failed to populate ifta jurisdiction cache", zap.Error(cacheErr))
	}

	return entities, nil
}

func (r *repository) ListJurisdictions(
	ctx context.Context,
	req *repositories.ListJurisdictionsRequest,
) ([]*ifta.Jurisdiction, error) {
	all, err := r.allJurisdictions(ctx)
	if err != nil {
		return nil, err
	}

	countryCode := strings.ToUpper(strings.TrimSpace(req.CountryCode))
	statuses := make(map[ifta.JurisdictionStatus]struct{}, len(req.Statuses))
	for _, status := range req.Statuses {
		statuses[status] = struct{}{}
	}

	out := make([]*ifta.Jurisdiction, 0, len(all))
	for _, entity := range all {
		if req.MembersOnly && !entity.IsIftaMember {
			continue
		}
		if countryCode != "" && entity.CountryCode != countryCode {
			continue
		}
		if len(statuses) > 0 {
			if _, ok := statuses[entity.Status]; !ok {
				continue
			}
		}
		out = append(out, entity)
	}

	return out, nil
}

func (r *repository) GetJurisdictionByID(
	ctx context.Context,
	id pulid.ID,
) (*ifta.Jurisdiction, error) {
	all, err := r.allJurisdictions(ctx)
	if err != nil {
		return nil, err
	}

	for _, entity := range all {
		if entity.ID == id {
			return entity, nil
		}
	}

	return nil, errortypes.NewNotFoundError("IFTA jurisdiction not found")
}

func (r *repository) GetJurisdictionsByIDs(
	ctx context.Context,
	ids []pulid.ID,
) ([]*ifta.Jurisdiction, error) {
	if len(ids) == 0 {
		return []*ifta.Jurisdiction{}, nil
	}

	all, err := r.allJurisdictions(ctx)
	if err != nil {
		return nil, err
	}

	wanted := make(map[pulid.ID]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}

	out := make([]*ifta.Jurisdiction, 0, len(ids))
	for _, entity := range all {
		if _, ok := wanted[entity.ID]; ok {
			out = append(out, entity)
		}
	}

	return out, nil
}

func (r *repository) GetJurisdictionByCode(
	ctx context.Context,
	countryCode, code string,
) (*ifta.Jurisdiction, error) {
	all, err := r.allJurisdictions(ctx)
	if err != nil {
		return nil, err
	}

	wantedCountry := strings.ToUpper(strings.TrimSpace(countryCode))
	wantedCode := strings.ToUpper(strings.TrimSpace(code))
	for _, entity := range all {
		if entity.CountryCode == wantedCountry && entity.Code == wantedCode {
			return entity, nil
		}
	}

	return nil, errortypes.NewNotFoundError("IFTA jurisdiction not found")
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
	for _, key := range keys {
		countryCode, code, ok := splitJurisdictionKey(key)
		if !ok {
			continue
		}
		normalised[key] = ifta.JurisdictionKey(countryCode, code)
	}
	if len(normalised) == 0 {
		return out, nil
	}

	all, err := r.allJurisdictions(ctx)
	if err != nil {
		return nil, err
	}

	byKey := make(map[string]*ifta.Jurisdiction, len(all))
	for _, entity := range all {
		byKey[entity.Key()] = entity
	}
	for key, canonical := range normalised {
		if entity, ok := byKey[canonical]; ok {
			out[key] = entity
		}
	}

	return out, nil
}
