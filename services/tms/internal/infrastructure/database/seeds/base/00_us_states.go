package base

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/uptrace/bun"
)

type USStatesSeed struct {
	seedhelpers.BaseSeed
}

func NewUSStatesSeed() *USStatesSeed {
	seed := &USStatesSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"USStates",
		"1.0.0",
		"Creates all US states required for the application",
		[]common.Environment{
			common.EnvProduction,
			common.EnvStaging,
			common.EnvDevelopment,
			common.EnvTest,
		},
	)
	return seed
}

func (s *USStatesSeed) Run(ctx context.Context, tx bun.Tx) error {
	var count int
	err := tx.NewSelect().
		Model((*usstate.UsState)(nil)).
		ColumnExpr("count(*)").
		Scan(ctx, &count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	loader := seedhelpers.NewDataLoader("./internal/infrastructure/database/seeds/base/data")

	var data struct {
		States []struct {
			Name         string `yaml:"name"`
			Abbreviation string `yaml:"abbreviation"`
			CountryName  string `yaml:"country_name"`
			CountryIso3  string `yaml:"country_iso3"`
		} `yaml:"states"`
	}

	if err := loader.LoadYAML("us_states.yaml", &data); err != nil {
		return err
	}

	states := make([]usstate.UsState, len(data.States))
	for i, stateData := range data.States {
		states[i] = usstate.UsState{
			Name:         stateData.Name,
			Abbreviation: stateData.Abbreviation,
			CountryName:  stateData.CountryName,
			CountryIso3:  stateData.CountryIso3,
		}
	}

	if _, err := tx.NewInsert().Model(&states).Exec(ctx); err != nil {
		return err
	}

	return nil
}
