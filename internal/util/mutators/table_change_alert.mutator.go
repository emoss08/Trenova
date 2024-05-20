package mutators

import (
	"context"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/hook"
	"github.com/emoss08/trenova/internal/ent/tablechangealert"
	"github.com/emoss08/trenova/internal/util/types"
)

func MutateTableChangeAlerts(next ent.Mutator) ent.Mutator {
	return hook.TableChangeAlertFunc(func(ctx context.Context, m *ent.TableChangeAlertMutation) (ent.Value, error) {
		if _, err := mutateSourceAndDatabase(ctx, next, m); err != nil {
			return nil, err
		}

		return next.Mutate(ctx, m)
	})
}

func mutateSourceAndDatabase(ctx context.Context, next ent.Mutator, m *ent.TableChangeAlertMutation) (ent.Value, error) {
	source, exists := m.Source()
	if !exists {
		return nil, &types.ValidationErrorResponse{
			Type: "validationError",
			Errors: []types.ValidationErrorDetail{
				{
					Attr:   "source",
					Code:   "required",
					Detail: "source is required",
				},
			},
		}
	}

	// If the database is the source, set the topic name to an empty string.
	if source == tablechangealert.SourceDatabase {
		m.SetTopicName("")
	} else {
		m.SetTableName("")
	}

	return next.Mutate(ctx, m)
}
