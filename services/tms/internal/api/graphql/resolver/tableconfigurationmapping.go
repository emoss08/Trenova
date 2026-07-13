package resolver

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func tableConfigToMap(cfg *tableconfiguration.TableConfig) (map[string]any, error) {
	result := make(map[string]any)
	if cfg == nil {
		return result, nil
	}

	data, err := sonic.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err = sonic.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func tableConfigFromMap(raw map[string]any) (*tableconfiguration.TableConfig, error) {
	cfg := new(tableconfiguration.TableConfig)
	if len(raw) == 0 {
		return cfg, nil
	}

	data, err := sonic.Marshal(raw)
	if err != nil {
		return nil, err
	}
	if err = sonic.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func tableConfigurationFromInput(
	input gqlmodel.TableConfigurationInput,
	id pulid.ID,
	authCtx *authctx.AuthContext,
) (*tableconfiguration.TableConfiguration, error) {
	cfg, err := tableConfigFromMap(input.TableConfig)
	if err != nil {
		return nil, err
	}

	visibility := tableconfiguration.VisibilityPrivate
	if input.Visibility != nil {
		visibility = *input.Visibility
	}

	return &tableconfiguration.TableConfiguration{
		ID:             id,
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
		Name:           input.Name,
		Description:    stringValue(input.Description),
		Resource:       input.Resource,
		TableConfig:    cfg,
		Visibility:     visibility,
		IsDefault:      boolValue(input.IsDefault),
	}, nil
}

func applyTableConfigurationPatch(
	existing *tableconfiguration.TableConfiguration,
	input gqlmodel.TableConfigurationPatchInput,
) error {
	if input.Name != nil {
		existing.Name = *input.Name
	}
	if description, ok := input.Description.ValueOK(); ok {
		existing.Description = stringValue(description)
	}
	if input.Resource != nil {
		existing.Resource = *input.Resource
	}
	if input.TableConfig != nil {
		cfg, err := tableConfigFromMap(input.TableConfig)
		if err != nil {
			return err
		}
		existing.TableConfig = cfg
	}
	if input.Visibility != nil {
		existing.Visibility = *input.Visibility
	}
	if isDefault, ok := input.IsDefault.ValueOK(); ok {
		existing.IsDefault = boolValue(isDefault)
	}

	return nil
}

func tableConfigurationConnectionToModel(
	result *pagination.CursorListResult[*tableconfiguration.TableConfiguration],
) (*gqlmodel.TableConfigurationConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(
			node *tableconfiguration.TableConfiguration,
			cursor string,
		) *gqlmodel.TableConfigurationEdge {
			return &gqlmodel.TableConfigurationEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.TableConfigurationEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.TableConfigurationConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
