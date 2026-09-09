package base

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const (
	iftaJurisdictionDataPath = "./internal/infrastructure/database/seeds/base/data"
	iftaJurisdictionDataFile = "ifta_jurisdictions.yaml"
)

type iftaJurisdictionRow struct {
	ID                  string `json:"id"`
	CountryCode         string `json:"country_code"`
	Code                string `json:"code"`
	Name                string `json:"name"`
	IsIftaMember        bool   `json:"is_ifta_member"`
	HasSurcharge        bool   `json:"has_surcharge"`
	SortOrder           int    `json:"sort_order"`
	UsStateAbbreviation string `json:"us_state_abbreviation"`
}

type IFTAJurisdictionsSeed struct {
	seedhelpers.BaseSeed
}

func NewIFTAJurisdictionsSeed() *IFTAJurisdictionsSeed {
	seed := &IFTAJurisdictionsSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"IFTAJurisdictions",
		"1.0.0",
		"Creates the global IFTA jurisdiction reference rows for US states, DC and Canadian provinces",
		[]common.Environment{
			common.EnvProduction,
			common.EnvStaging,
			common.EnvDevelopment,
			common.EnvTest,
		},
	)
	seed.SetDependencies(seedhelpers.SeedUSStates)
	return seed
}

func (s *IFTAJurisdictionsSeed) Run(ctx context.Context, tx bun.Tx) error {
	rows, err := loadIFTAJurisdictionRows(seedhelpers.NewDataLoader(iftaJurisdictionDataPath))
	if err != nil {
		return err
	}

	var states []usstate.UsState
	if err = tx.NewSelect().
		Model(&states).
		Column("id", "abbreviation").
		Scan(ctx); err != nil {
		return fmt.Errorf("load us states: %w", err)
	}

	stateIDs := make(map[string]pulid.ID, len(states))
	for i := range states {
		stateIDs[states[i].Abbreviation] = states[i].ID
	}

	jurisdictions, err := buildIFTAJurisdictions(rows, stateIDs)
	if err != nil {
		return err
	}

	if _, err = tx.NewInsert().
		Model(&jurisdictions).
		On("CONFLICT (country_code, code) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("sort_order = EXCLUDED.sort_order").
		Set("has_surcharge = EXCLUDED.has_surcharge").
		Set("us_state_id = EXCLUDED.us_state_id").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx); err != nil {
		return fmt.Errorf("upsert ifta jurisdictions: %w", err)
	}

	return nil
}

func loadIFTAJurisdictionRows(loader *seedhelpers.DataLoader) ([]iftaJurisdictionRow, error) {
	var data struct {
		Jurisdictions []iftaJurisdictionRow `json:"jurisdictions"`
	}

	if err := loader.LoadYAML(iftaJurisdictionDataFile, &data); err != nil {
		return nil, fmt.Errorf("load ifta jurisdictions: %w", err)
	}

	return data.Jurisdictions, nil
}

func buildIFTAJurisdictions(
	rows []iftaJurisdictionRow,
	stateIDs map[string]pulid.ID,
) ([]ifta.Jurisdiction, error) {
	jurisdictions := make([]ifta.Jurisdiction, 0, len(rows))
	seenIDs := make(map[string]struct{}, len(rows))
	seenKeys := make(map[string]struct{}, len(rows))

	for i := range rows {
		row := &rows[i]

		if _, dup := seenIDs[row.ID]; dup {
			return nil, fmt.Errorf("ifta jurisdiction %s: duplicate id %s", row.Code, row.ID)
		}
		seenIDs[row.ID] = struct{}{}

		jurisdiction := ifta.Jurisdiction{
			ID:           pulid.ID(row.ID),
			CountryCode:  row.CountryCode,
			Code:         row.Code,
			Name:         row.Name,
			IsIftaMember: row.IsIftaMember,
			HasSurcharge: row.HasSurcharge,
			SortOrder:    row.SortOrder,
			Status:       ifta.JurisdictionStatusActive,
		}
		jurisdiction.Normalize()

		if _, dup := seenKeys[jurisdiction.Key()]; dup {
			return nil, fmt.Errorf("ifta jurisdiction %s: listed twice", jurisdiction.Key())
		}
		seenKeys[jurisdiction.Key()] = struct{}{}

		if jurisdiction.ID.IsNil() {
			return nil, fmt.Errorf("ifta jurisdiction %s: id is required", jurisdiction.Key())
		}

		if row.UsStateAbbreviation != "" {
			if stateID, ok := stateIDs[row.UsStateAbbreviation]; ok {
				jurisdiction.UsStateID = &stateID
			}
		}

		multiErr := errortypes.NewMultiError()
		jurisdiction.Validate(multiErr)
		if multiErr.HasErrors() {
			return nil, fmt.Errorf(
				"ifta jurisdiction %s is invalid: %w",
				jurisdiction.Key(),
				multiErr,
			)
		}

		jurisdictions = append(jurisdictions, jurisdiction)
	}

	return jurisdictions, nil
}
