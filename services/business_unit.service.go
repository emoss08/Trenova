package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
)

// BusinessUnitOps is the service for billing control settings
type BusinessUnitOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewBusinessUnitOps creates a new billing control service
func NewBusinessUnitOps(ctx context.Context) *BusinessUnitOps {
	return &BusinessUnitOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetBusinessUnits creates a new billing control settings for an organization
func (r *BusinessUnitOps) GetBusinessUnits(limit, offset int) ([]*ent.BusinessUnit, int, error) {
	businessUnitCount, err := r.client.BusinessUnit.Query().Count(r.ctx)

	businessUnits, err := r.client.Debug().BusinessUnit.Query().
		Limit(limit).
		Offset(offset).
		All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return businessUnits, businessUnitCount, nil
}
