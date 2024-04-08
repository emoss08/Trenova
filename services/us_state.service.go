package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
)

type UsStateOps struct {
	client *ent.Client
}

// NewUsStateOps creates a new US State service.
func NewUsStateOps() *UsStateOps {
	return &UsStateOps{
		client: database.GetClient(),
	}
}

// GetUsStates gets the accessorial charges for an organization.
func (r *UsStateOps) GetUsStates(ctx context.Context) ([]*ent.UsState, error) {
	usStates, err := r.client.UsState.Query().
		All(ctx)
	if err != nil {
		return nil, err
	}

	return usStates, nil
}
