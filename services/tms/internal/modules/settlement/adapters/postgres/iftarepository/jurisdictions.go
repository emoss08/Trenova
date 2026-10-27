package iftarepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const jurisdictionKeySeparator = "_"

func (r *repository) allJurisdictions(ctx context.Context) ([]*ifta.Jurisdiction, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ifta.Jurisdiction, error) {
		j := iftadb.Jurisdictions
		entities, err := j.Query().
			Order(j.SortOrder.Asc(), j.CountryCode.Asc(), j.Code.Asc()).
			Cache(resultcache.Catalog()...).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to load ifta jurisdictions", zap.Error(err))
			return nil, fmt.Errorf("load ifta jurisdictions: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) ListJurisdictions(
	ctx context.Context,
	req *iftaports.ListJurisdictionsRequest,
) ([]*ifta.Jurisdiction, error) {
	all, err := r.allJurisdictions(ctx)
	if err != nil {
		return nil, err
	}

	return filterJurisdictions(all, req), nil
}

func filterJurisdictions(
	all []*ifta.Jurisdiction,
	req *iftaports.ListJurisdictionsRequest,
) []*ifta.Jurisdiction {
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

	return out
}

func (r *repository) GetJurisdictionByID(
	ctx context.Context,
	id pulid.ID,
) (*ifta.Jurisdiction, error) {
	all, err := r.allJurisdictions(ctx)
	if err != nil {
		return nil, err
	}

	return jurisdictionByID(all, id)
}

func jurisdictionByID(all []*ifta.Jurisdiction, id pulid.ID) (*ifta.Jurisdiction, error) {
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

	return jurisdictionsByIDs(all, ids), nil
}

func jurisdictionsByIDs(all []*ifta.Jurisdiction, ids []pulid.ID) []*ifta.Jurisdiction {
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

	return out
}

func (r *repository) GetJurisdictionByCode(
	ctx context.Context,
	countryCode, code string,
) (*ifta.Jurisdiction, error) {
	all, err := r.allJurisdictions(ctx)
	if err != nil {
		return nil, err
	}

	return jurisdictionByCode(all, countryCode, code)
}

func jurisdictionByCode(
	all []*ifta.Jurisdiction,
	countryCode, code string,
) (*ifta.Jurisdiction, error) {
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
	normalised := normaliseJurisdictionKeys(keys)
	if len(normalised) == 0 {
		return make(map[string]*ifta.Jurisdiction), nil
	}

	all, err := r.allJurisdictions(ctx)
	if err != nil {
		return nil, err
	}

	return jurisdictionsByKeys(all, normalised), nil
}

func normaliseJurisdictionKeys(keys []string) map[string]string {
	normalised := make(map[string]string, len(keys))
	for _, key := range keys {
		countryCode, code, ok := splitJurisdictionKey(key)
		if !ok {
			continue
		}
		normalised[key] = ifta.JurisdictionKey(countryCode, code)
	}

	return normalised
}

func jurisdictionsByKeys(
	all []*ifta.Jurisdiction,
	normalised map[string]string,
) map[string]*ifta.Jurisdiction {
	byKey := make(map[string]*ifta.Jurisdiction, len(all))
	for _, entity := range all {
		byKey[entity.Key()] = entity
	}

	out := make(map[string]*ifta.Jurisdiction, len(normalised))
	for key, canonical := range normalised {
		if entity, ok := byKey[canonical]; ok {
			out[key] = entity
		}
	}

	return out
}
