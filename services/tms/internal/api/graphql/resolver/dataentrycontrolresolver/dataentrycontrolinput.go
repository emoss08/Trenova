package dataentrycontrolresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlinput"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
)

func dataEntryControlFromInput(
	existing *dataentrycontrol.DataEntryControl,
	input *gqlmodel.UpdateDataEntryControlInput,
) (*dataentrycontrol.DataEntryControl, error) {
	parse := gqlinput.NewParser()
	entity := &dataentrycontrol.DataEntryControl{
		ID:             existing.ID,
		BusinessUnitID: existing.BusinessUnitID,
		OrganizationID: existing.OrganizationID,
		Version:        int64(input.Version),
		CreatedAt:      existing.CreatedAt,
		UpdatedAt:      existing.UpdatedAt,
		CodeCase:       input.CodeCase,
		NameCase:       input.NameCase,
		EmailCase:      input.EmailCase,
		CityCase:       input.CityCase,
	}
	if err := parse.Err(); err != nil {
		return nil, err
	}

	return entity, nil
}
