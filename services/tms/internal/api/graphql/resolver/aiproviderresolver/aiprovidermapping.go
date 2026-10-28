package aiproviderresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider/aiproviderdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
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
	p := aiproviderdb.Providers
	if graphql.FieldRequested(ctx, nodePathPrefix+".apiKey") {
		return append(columns, projection.ColumnNames(
			p.APIKey.Col,
			p.APIKeyPrefix.Col,
			p.APIKeyLastFour.Col,
			p.APIKeyAddedAt.Col,
			p.APIKeyAddedByID.Col,
			p.APIKeyLastUsedAt.Col,
			p.RotationExpiresAt.Col,
			p.CreatedAt.Col,
		)...)
	}
	if graphql.FieldRequested(ctx, nodePathPrefix+".hasApiKey") {
		columns = append(columns, projection.ColumnNames(p.APIKey.Col)...)
	}

	return columns
}

func aiProviderConnectionToModel(
	result *pagination.CursorListResult[*aiprovider.Provider],
) (*gqlmodel.AIProviderConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
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
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func endpointFromInput(
	input *gqlmodel.AIProviderEndpointInput,
) (aiproviderports.AIProviderEndpoint, error) {
	if input == nil {
		return aiproviderports.AIProviderEndpoint{}, errortypes.NewValidationError(
			"endpoint", errortypes.ErrRequired, "Endpoint is required",
		)
	}

	providerID, err := gqlid.OptionalID(input.ProviderID)
	if err != nil {
		return aiproviderports.AIProviderEndpoint{}, err
	}

	endpoint := aiproviderports.AIProviderEndpoint{
		ProviderID:          providerID,
		Kind:                input.Kind,
		Platform:            aiprovider.PlatformDirect,
		PlatformSettings:    platformSettingsFromInput(input.PlatformSettings),
		BaseURL:             input.BaseURL,
		APIKey:              input.APIKey,
		AllowPrivateNetwork: input.AllowPrivateNetwork,
	}
	if input.Platform != nil {
		endpoint.Platform = *input.Platform
	}

	return endpoint, nil
}

func platformSettingsFromInput(
	input *gqlmodel.AIProviderPlatformSettingsInput,
) *aiprovider.PlatformSettings {
	if input == nil {
		return nil
	}

	settings := &aiprovider.PlatformSettings{
		Region:      gqlscalar.StringValue(input.Region),
		ProjectID:   gqlscalar.StringValue(input.ProjectID),
		WorkspaceID: gqlscalar.StringValue(input.WorkspaceID),
		AccessKeyID: gqlscalar.StringValue(input.AccessKeyID),
		Resource:    gqlscalar.StringValue(input.Resource),
		TenantID:    gqlscalar.StringValue(input.TenantID),
		ClientID:    gqlscalar.StringValue(input.ClientID),
	}
	if input.Auth != nil {
		settings.Auth = *input.Auth
	}

	return settings
}

func patchFromInput(
	input *gqlmodel.AIProviderPatchInput,
) (*aiproviderports.PatchAIProviderRequest, error) {
	inputCost, err := omittableDecimal("inputCostPerMillion", input.InputCostPerMillion)
	if err != nil {
		return nil, err
	}
	outputCost, err := omittableDecimal("outputCostPerMillion", input.OutputCostPerMillion)
	if err != nil {
		return nil, err
	}
	cacheReadCost, err := omittableDecimal("cacheReadCostPerMillion", input.CacheReadCostPerMillion)
	if err != nil {
		return nil, err
	}
	cacheWriteCost, err := omittableDecimal(
		"cacheWriteCostPerMillion",
		input.CacheWriteCostPerMillion,
	)
	if err != nil {
		return nil, err
	}

	req := &aiproviderports.PatchAIProviderRequest{
		Enabled:                  gqlscalar.NullableOmittable(input.Enabled),
		Trusted:                  gqlscalar.NullableOmittable(input.Trusted),
		AllowPrivateNetwork:      gqlscalar.NullableOmittable(input.AllowPrivateNetwork),
		APIKey:                   gqlscalar.NullableOmittable(input.APIKey),
		InputCostPerMillion:      inputCost,
		OutputCostPerMillion:     outputCost,
		CacheReadCostPerMillion:  cacheReadCost,
		CacheWriteCostPerMillion: cacheWriteCost,
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

	parsed, err := gqldecimal.NullDecimalFromStringPtr(value.Value(), field)
	if err != nil {
		return optional.Value[*decimal.Decimal]{}, err
	}
	if !parsed.Valid {
		return optional.Some[*decimal.Decimal](nil), nil
	}

	return optional.Some(&parsed.Decimal), nil
}

func modelOptionsToModel(
	models []aiproviderports.AIProviderModelOption,
) []*gqlmodel.AIProviderModelOption {
	out := make([]*gqlmodel.AIProviderModelOption, 0, len(models))
	for idx := range models {
		model := &models[idx]
		option := &gqlmodel.AIProviderModelOption{
			ID:                   model.ID,
			DisplayName:          model.DisplayName,
			ContextWindow:        model.ContextWindow,
			Embedding:            model.Embedding,
			InputCostPerMillion:  gqldecimal.DecimalPtrToStringPtr(model.InputCostPerMillion),
			OutputCostPerMillion: gqldecimal.DecimalPtrToStringPtr(model.OutputCostPerMillion),
			PriceSource:          priceSourceToModel(model.PriceSource),
			CreatedAt:            gqlscalar.Int64PtrToIntPtr(model.CreatedAt),
		}
		out = append(out, option)
	}

	return out
}

func priceSourceToModel(source aiproviderports.ModelPriceSource) *gqlmodel.AIModelPriceSource {
	if source == "" {
		return nil
	}

	value := gqlmodel.AIModelPriceSource(source)
	return &value
}
