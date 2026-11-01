package casechecklistresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlinput"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/agent/domain/deskcase"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskcaseports"
)

func saveTemplateRequestFromInput(
	input *gqlmodel.SaveCaseChecklistTemplateInput,
) (*deskcaseports.SaveCaseChecklistTemplateRequest, error) {
	parse := gqlinput.NewParser()
	items := make(deskcase.TemplateItems, 0, len(input.Items))
	for _, item := range input.Items {
		if item == nil {
			continue
		}
		items = append(items, deskcase.TemplateItem{
			Key:    deskcase.ItemKey(item.Key),
			Mode:   item.Mode,
			Custom: customItemFromInput(parse, item.Custom),
		})
	}
	req := &deskcaseports.SaveCaseChecklistTemplateRequest{
		ID:         parse.OptionalID("id", input.ID),
		Version:    int64(input.Version),
		Kind:       input.Kind,
		CustomerID: parse.OptionalID("customerId", input.CustomerID),
		Items:      items,
	}
	if err := parse.Err(); err != nil {
		return nil, err
	}

	return req, nil
}

func customItemFromInput(
	parse *gqlinput.Parser,
	input *gqlmodel.CaseChecklistCustomItemInput,
) *deskcase.CustomItem {
	if input == nil {
		return nil
	}

	return &deskcase.CustomItem{
		Label:          input.Label,
		Check:          input.Check,
		DocumentTypeID: parse.OptionalID("items.custom.documentTypeId", input.DocumentTypeID),
		StepLabel:      gqlscalar.StringValue(input.StepLabel),
		Prompt:         gqlscalar.StringValue(input.Prompt),
	}
}
