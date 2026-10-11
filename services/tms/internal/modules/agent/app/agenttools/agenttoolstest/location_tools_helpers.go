package agenttoolstest

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
)

type FakeStates struct {
	State *usstate.UsState `json:"-"`
}

func (f *FakeStates) GetByAbbreviation(
	_ context.Context,
	abbreviation string,
) (*usstate.UsState, error) {
	if f.State == nil || f.State.Abbreviation != abbreviation {
		return nil, errors.New("state not found")
	}

	return f.State, nil
}
