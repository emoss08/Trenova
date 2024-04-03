package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
)

type UsStateOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewUsStateOps creates a new US State service.
func NewUsStateOps(ctx context.Context) *UsStateOps {
	return &UsStateOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetUsStates gets the accessorial charges for an organization.
func (r *UsStateOps) GetUsStates() ([]*ent.UsState, error) {
	usStates, err := r.client.UsState.Query().
		All(r.ctx)
	if err != nil {
		return nil, err
	}

	return usStates, nil
}
