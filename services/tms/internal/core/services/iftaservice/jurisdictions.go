package iftaservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) ListJurisdictions(
	ctx context.Context,
	req *repositories.ListJurisdictionsRequest,
) ([]*ifta.Jurisdiction, error) {
	if req == nil {
		req = &repositories.ListJurisdictionsRequest{}
	}
	return s.repo.ListJurisdictions(ctx, req)
}

func (s *Service) GetJurisdiction(ctx context.Context, id pulid.ID) (*ifta.Jurisdiction, error) {
	return s.repo.GetJurisdictionByID(ctx, id)
}

func (s *Service) GetJurisdictionsByIDs(
	ctx context.Context,
	ids []pulid.ID,
) ([]*ifta.Jurisdiction, error) {
	return s.repo.GetJurisdictionsByIDs(ctx, ids)
}

func (s *Service) jurisdictionsByID(
	ctx context.Context,
	ids []pulid.ID,
) (map[pulid.ID]*ifta.Jurisdiction, error) {
	out := make(map[pulid.ID]*ifta.Jurisdiction, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	entities, err := s.repo.GetJurisdictionsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, entity := range entities {
		out[entity.ID] = entity
	}
	return out, nil
}
