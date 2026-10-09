package aiproviderresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/optional"
	"github.com/shopspring/decimal"
)

func aiProviderColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AIProviderSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	columns := selection.Columns
	if len(columns) == 0 {
		return columns
	}
	cols := buncolgen.ProviderColumns
	if graphql.FieldRequested(ctx, nodePathPrefix+".apiKey") {
		return append(columns,
			cols.APIKey.Name,
			cols.APIKeyPrefix.Name,
			cols.APIKeyLastFour.Name,
			cols.APIKeyAddedAt.Name,
			cols.APIKeyAddedByID.Name,
			cols.APIKeyLastUsedAt.Name,
			cols.RotationExpiresAt.Name,
			cols.CreatedAt.Name,
		)
	}
	if graphql.FieldRequested(ctx, nodePathPrefix+".hasApiKey") {
		columns = append(columns, cols.APIKey.Name)
	}

	return columns
}

func aiProviderConnectionToModel(
	result *pagination.CursorListResult[*aiprovider.Provider],
) (*gqlmodel.AIProviderConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *aiprovider.Provider, cursor string) *gqlmodel.AIProviderEdge {
			return &gqlmodel.AIProviderEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AIProviderEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AIProviderConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func endpointFromInput(input *gqlmodel.AIProviderEndpointInput) (services.AIProviderEndpoint, error) {
	if input == nil {
		return services.AIProviderEndpoint{}, errortypes.NewValidationError(
			"endpoint", errortypes.ErrRequired, "Endpoint is required",
		)
	}

	providerID, err := base.OptionalID(input.ProviderID)
	if err != nil {
		return services.AIProviderEndpoint{}, err
	}

	return services.AIProviderEndpoint{
		ProviderID:          providerID,
		Kind:                input.Kind,
		BaseURL:             input.BaseURL,
		APIKey:              input.APIKey,
		AllowPrivateNetwork: input.AllowPrivateNetwork,
	}, nil
}

func patchFromInput(input *gqlmodel.AIProviderPatchInput) (*services.PatchAIProviderRequest, error) {
	inputCost, err := omittableDecimal("inputCostPerMillion", input.InputCostPerMillion)
	if err != nil {
		return nil, err
	}
	outputCost, err := omittableDecimal("outputCostPerMillion", input.OutputCostPerMillion)
	if err != nil {
		return nil, err
	}

	req := &services.PatchAIProviderRequest{
		Enabled:              base.NullableOmittable(input.Enabled),
		Trusted:              base.NullableOmittable(input.Trusted),
		AllowPrivateNetwork:  base.NullableOmittable(input.AllowPrivateNetwork),
		APIKey:               base.NullableOmittable(input.APIKey),
		InputCostPerMillion:  inputCost,
		OutputCostPerMillion: outputCost,
	}
	if input.Tasks.IsSet() {
		req.Tasks = optional.Some(input.Tasks.Value())
	}

	return req, nil
}

func omittableDecimal(
	field string,
	value graphql.Omittable[*string],
) (optional.Value[*decimal.Decimal], error) {
	if !value.IsSet() {
		return optional.Value[*decimal.Decimal]{}, nil
	}

	parsed, err := base.NullDecimalFromStringPtr(value.Value(), field)
	if err != nil {
		return optional.Value[*decimal.Decimal]{}, err
	}
	if !parsed.Valid {
		return optional.Some[*decimal.Decimal](nil), nil
	}

	return optional.Some(&parsed.Decimal), nil
}

func modelOptionsToModel(models []services.AIProviderModelOption) []*gqlmodel.AIProviderModelOption {
	out := make([]*gqlmodel.AIProviderModelOption, 0, len(models))
	for idx := range models {
		model := &models[idx]
		option := &gqlmodel.AIProviderModelOption{
			ID:                   model.ID,
			DisplayName:          model.DisplayName,
			ContextWindow:        model.ContextWindow,
			Loaded:               model.Loaded,
			Embedding:            model.Embedding,
			InputCostPerMillion:  base.DecimalPtrToStringPtr(model.InputCostPerMillion),
			OutputCostPerMillion: base.DecimalPtrToStringPtr(model.OutputCostPerMillion),
			PriceSource:          priceSourceToModel(model.PriceSource),
			CreatedAt:            base.Int64PtrToIntPtr(model.CreatedAt),
		}
		if model.SizeBytes != nil {
			size := float64(*model.SizeBytes)
			option.SizeBytes = &size
		}
		out = append(out, option)
	}

	return out
}

func priceSourceToModel(source services.ModelPriceSource) *gqlmodel.AIModelPriceSource {
	if source == "" {
		return nil
	}

	value := gqlmodel.AIModelPriceSource(source)
	return &value
}
