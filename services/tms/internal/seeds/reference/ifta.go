package reference

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type iftaJurisdictionFile struct {
	Jurisdictions []iftaJurisdictionRow `yaml:"jurisdictions"`
}

type iftaJurisdictionRow struct {
	ID                  string `yaml:"id"`
	CountryCode         string `yaml:"country_code"`
	Code                string `yaml:"code"`
	Name                string `yaml:"name"`
	IsIftaMember        bool   `yaml:"is_ifta_member"`
	HasSurcharge        bool   `yaml:"has_surcharge"`
	SortOrder           int    `yaml:"sort_order"`
	UsStateAbbreviation string `yaml:"us_state_abbreviation"`
}

func buildIFTAJurisdictions(
	rows []iftaJurisdictionRow,
	stateIDs map[string]pulid.ID,
) ([]*ifta.Jurisdiction, error) {
	jurisdictions := make([]*ifta.Jurisdiction, 0, len(rows))
	seenIDs := make(map[string]struct{}, len(rows))
	seenKeys := make(map[string]struct{}, len(rows))

	for i := range rows {
		row := &rows[i]
		if _, dup := seenIDs[row.ID]; dup {
			return nil, fmt.Errorf("ifta jurisdiction %s: duplicate id %s", row.Code, row.ID)
		}
		seenIDs[row.ID] = struct{}{}

		jurisdiction := &ifta.Jurisdiction{
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
