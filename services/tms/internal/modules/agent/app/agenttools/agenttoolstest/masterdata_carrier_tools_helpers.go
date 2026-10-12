package agenttoolstest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/shared/pulid"
)

func (f *FakeStates) GetByIDs(_ context.Context, ids []pulid.ID) ([]*usstate.UsState, error) {
	found := make([]*usstate.UsState, 0, len(ids))
	for _, id := range ids {
		if f.State != nil && f.State.ID == id {
			found = append(found, f.State)
		}
	}

	return found, nil
}
