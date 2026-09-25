package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

type locationView struct {
	Name         string `json:"name"`
	AddressLine1 string `json:"addressLine1"`
	AddressLine2 string `json:"addressLine2"`
	City         string `json:"city"`
	State        string `json:"state"`
	PostalCode   string `json:"postalCode"`
	Category     string `json:"category"`
	Status       string `json:"status"`
}

func locationViewOf(
	entity *location.Location,
	state *usstate.UsState,
	category *locationcategory.LocationCategory,
) *locationView {
	return &locationView{
		Name:         entity.Name,
		AddressLine1: entity.AddressLine1,
		AddressLine2: entity.AddressLine2,
		City:         entity.City,
		State:        state.Abbreviation,
		PostalCode:   entity.PostalCode,
		Category:     category.Name,
		Status:       string(entity.Status),
	}
}

func (t *createLocationTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	plan, err := t.plan(ctx, params)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build("Would create a location."), err), nil
		}

		return nil, err
	}

	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceLocation,
		Label:    plan.entity.Name,
	}, locationViewOf(plan.entity, plan.state, plan.category))
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would create the location %s at %s, %s, %s %s. Its code is assigned when it is saved.",
		plan.entity.Name,
		plan.entity.AddressLine1,
		plan.entity.City,
		plan.state.Abbreviation,
		plan.entity.PostalCode,
	), change), nil
}
