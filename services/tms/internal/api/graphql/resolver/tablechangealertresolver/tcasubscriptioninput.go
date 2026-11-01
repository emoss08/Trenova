package tablechangealertresolver

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func tcaSubscriptionFromInput(
	input *gqlmodel.SaveTCASubscriptionInput,
) (*tablechangealert.TCASubscription, error) {
	conditions, err := conditionsFromInput(input.Conditions)
	if err != nil {
		return nil, err
	}

	var recordID *string
	if input.RecordID != nil && *input.RecordID != "" {
		recordID = input.RecordID
	}

	return &tablechangealert.TCASubscription{
		Name:           input.Name,
		TableName:      input.TableName,
		RecordID:       recordID,
		EventTypes:     input.EventTypes,
		Conditions:     conditions,
		ConditionMatch: input.ConditionMatch,
		WatchedColumns: input.WatchedColumns,
		CustomTitle:    input.CustomTitle,
		CustomMessage:  input.CustomMessage,
		Topic:          input.Topic,
		Priority:       input.Priority,
		Status:         tablechangealert.SubscriptionStatus(input.Status),
	}, nil
}

func conditionsFromInput(raw []map[string]any) ([]tablechangealert.Condition, error) {
	conditions := make([]tablechangealert.Condition, 0, len(raw))
	if len(raw) == 0 {
		return conditions, nil
	}

	encoded, err := sonic.Marshal(raw)
	if err != nil {
		return nil, err
	}
	if err = sonic.Unmarshal(encoded, &conditions); err != nil {
		return nil, errortypes.NewValidationError(
			"conditions",
			errortypes.ErrInvalid,
			"Each condition needs a field, an operator and a value",
		)
	}

	return conditions, nil
}
