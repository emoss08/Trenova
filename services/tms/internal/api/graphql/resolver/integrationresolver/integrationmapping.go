package integrationresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func configurationFromInput(
	values []*gqlmodel.IntegrationConfigValueInput,
) (map[string]string, error) {
	configuration := make(map[string]string, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		if value.Key == "" {
			return nil, errortypes.NewValidationError(
				"configuration",
				errortypes.ErrRequired,
				"Every setting needs a key",
			)
		}
		if _, seen := configuration[value.Key]; seen {
			return nil, errortypes.NewValidationError(
				"configuration."+value.Key,
				errortypes.ErrDuplicate,
				"This setting is given more than once",
			)
		}
		configuration[value.Key] = value.Value
	}

	return configuration, nil
}

func runtimeConfigValues(config map[string]string) map[string]any {
	values := make(map[string]any, len(config))
	for key, value := range config {
		values[key] = value
	}

	return values
}
