package common

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const MaxOperationNoteChars = 2000

const ShipmentRecordEntity = "shipment"

const OrderRecordEntity = "order"

const ServiceFailureRecordEntity = "service_failure"

const WorkerRecordEntity = "worker"

const CarrierRecordEntity = "carrier"

const CustomerRecordEntity = "customer"

const DetentionRecordEntity = "detention_occurrence"

const FieldReviewedByID = "reviewedById"

const FieldReviewedAt = "reviewedAt"

const FieldVoidedByID = "voidedById"

const FieldFreightChargeAmount = "freightChargeAmount"

const ParamStopID = "stopId"

const FieldCanceledAt = "canceledAt"

func BooleanProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeBoolean,
		toolschema.KeyDescription: description,
	}
}

func IntegerProperty(description string, minimum, maximum int) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeInteger,
		toolschema.KeyMinimum:     minimum,
		toolschema.KeyMaximum:     maximum,
		toolschema.KeyDescription: description,
	}
}

func RequireDateTime(params map[string]any, key string) (int64, error) {
	raw, err := RequireString(params, key)
	if err != nil {
		return 0, err
	}

	return ParseDateTime(key, raw)
}

func OptionalDateTime(params map[string]any, key string) (*int64, error) {
	raw := strings.TrimSpace(OptionalString(params, key))
	if raw == "" {
		if value, given := params[key]; given && value != nil {
			if _, isText := value.(string); !isText {
				return nil, DateRefusal(key, value, toolschema.FormatDateTime)
			}
		}

		return nil, nil //nolint:nilnil // an absent time is no change and no error
	}

	seconds, err := ParseDateTime(key, raw)
	if err != nil {
		return nil, err
	}

	return &seconds, nil
}

func ParseDateTime(key, raw string) (int64, error) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return 0, DateRefusal(key, raw, toolschema.FormatDateTime)
	}

	return parsed.Unix(), nil
}

func OptionalBoolPointer(params map[string]any, key string) (*bool, error) {
	raw, ok := params[key]
	if !ok || raw == nil {
		return nil, nil //nolint:nilnil // an absent flag is no change and no error
	}

	value, ok := raw.(bool)
	if !ok {
		return nil, fmt.Errorf("parameter %q must be true or false", key)
	}

	return &value, nil
}

func OptionalBoundedText(params map[string]any, key string, limit int) (*string, error) {
	if _, ok := params[key]; !ok {
		return nil, nil //nolint:nilnil // an absent text is no change and no error
	}

	text, err := BoundedText(params, key, limit)
	if err != nil {
		return nil, err
	}

	return &text, nil
}

func RequireIntInRange(params map[string]any, key string, minimum, maximum int) (int, error) {
	raw, ok := params[key]
	if !ok {
		return 0, fmt.Errorf("missing required parameter %q", key)
	}

	value, ok := raw.(float64)
	if !ok {
		if whole, isInt := raw.(int); isInt {
			value = float64(whole)
		} else {
			return 0, fmt.Errorf("parameter %q must be a whole number", key)
		}
	}
	if value != float64(int(value)) || int(value) < minimum || int(value) > maximum {
		return 0, fmt.Errorf("parameter %q must be a whole number from %d to %d", key, minimum,
			maximum)
	}

	return int(value), nil
}

var operationDateTimes = map[string]assistantartifact.DisplayType{
	FieldScheduledWindowStart: assistantartifact.DisplayDateTime,
	FieldScheduledWindowEnd:   assistantartifact.DisplayDateTime,
	"firstPickupAt":           assistantartifact.DisplayDateTime,
	"lastDeliveryAt":          assistantartifact.DisplayDateTime,
	"startedAt":               assistantartifact.DisplayDateTime,
	"effectiveFrom":           assistantartifact.DisplayDate,
	"effectiveTo":             assistantartifact.DisplayDate,
	"shiftDate":               assistantartifact.DisplayDate,
	"counterpartyShiftDate":   assistantartifact.DisplayDate,
	"startDate":               assistantartifact.DisplayDate,
	"endDate":                 assistantartifact.DisplayDate,
}

func RecordOf(entity string, id pulid.ID) *agent.RecordRef {
	if id.IsNil() {
		return nil
	}

	return &agent.RecordRef{EntityType: entity, ID: id.String()}
}
