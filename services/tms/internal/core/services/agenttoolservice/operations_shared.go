package agenttoolservice

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxOperationNoteChars      = 2000
	shipmentRecordEntity       = "shipment"
	orderRecordEntity          = "order"
	serviceFailureRecordEntity = "service_failure"
	workerRecordEntity         = "worker"
	carrierRecordEntity        = "carrier"
	customerRecordEntity       = "customer"
	detentionRecordEntity      = "detention_occurrence"
	fieldReviewedByID          = "reviewedById"
	fieldReviewedAt            = "reviewedAt"
	fieldVoidedByID            = "voidedById"
	fieldFreightChargeAmount   = "freightChargeAmount"
	paramStopID                = "stopId"
	fieldCanceledAt            = "canceledAt"
)

func dateTimeProperty(description string) map[string]any {
	return stringProperty(description+" An ISO 8601 date and time with its UTC offset, "+
		"such as 2026-10-01T08:00:00-05:00.", 0)
}

func booleanProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeBoolean,
		toolschema.KeyDescription: description,
	}
}

func integerProperty(description string, minimum, maximum int) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeInteger,
		toolschema.KeyMinimum:     minimum,
		toolschema.KeyMaximum:     maximum,
		toolschema.KeyDescription: description,
	}
}

func requireDateTime(params map[string]any, key string) (int64, error) {
	raw, err := requireString(params, key)
	if err != nil {
		return 0, err
	}

	return parseDateTime(key, raw)
}

func optionalDateTime(params map[string]any, key string) (*int64, error) {
	raw := strings.TrimSpace(optionalString(params, key))
	if raw == "" {
		if value, given := params[key]; given && value != nil {
			if _, isText := value.(string); !isText {
				return nil, fmt.Errorf("parameter %q must be an ISO 8601 date and time", key)
			}
		}

		return nil, nil //nolint:nilnil // an absent time is no change and no error
	}

	seconds, err := parseDateTime(key, raw)
	if err != nil {
		return nil, err
	}

	return &seconds, nil
}

func parseDateTime(key, raw string) (int64, error) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf(
			"parameter %q must be an ISO 8601 date and time with its UTC offset, such as "+
				"2026-10-01T08:00:00-05:00, got %q", key, raw)
	}

	return parsed.Unix(), nil
}

func optionalBoolPointer(params map[string]any, key string) (*bool, error) {
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

func optionalBoundedText(params map[string]any, key string, limit int) (*string, error) {
	if _, ok := params[key]; !ok {
		return nil, nil //nolint:nilnil // an absent text is no change and no error
	}

	text, err := boundedText(params, key, limit)
	if err != nil {
		return nil, err
	}

	return &text, nil
}

func requireIntInRange(params map[string]any, key string, minimum, maximum int) (int, error) {
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
	fieldScheduledWindowStart: assistantartifact.DisplayDateTime,
	fieldScheduledWindowEnd:   assistantartifact.DisplayDateTime,
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

func recordOf(entity string, id pulid.ID) *agent.RecordRef {
	if id.IsNil() {
		return nil
	}

	return &agent.RecordRef{EntityType: entity, ID: id.String()}
}
