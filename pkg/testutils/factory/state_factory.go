package factory

import (
	"context"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

type StateFactory struct {
	db *bun.DB
}

func NewStateFactory(db *bun.DB) *StateFactory {
	return &StateFactory{db: db}
}

func (f *StateFactory) CreateUSState(ctx context.Context) (*models.UsState, error) {
	state := &models.UsState{
		Name:         "North Carolina",
		Abbreviation: "NC",
		CountryName:  "United States",
		CountryIso3:  "USA",
	}

	_, err := f.db.NewInsert().Model(state).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return state, nil
}
