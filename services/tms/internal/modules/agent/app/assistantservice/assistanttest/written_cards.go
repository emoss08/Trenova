package assistanttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
)

type ShipmentReader struct {
	Asked map[string]any
}

func (r *ShipmentReader) Name() string { return "shipment_get" }

func (r *ShipmentReader) Description() string { return "Retrieve one shipment." }

func (r *ShipmentReader) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType:     toolschema.TypeObject,
		toolschema.KeyRequired: []string{"shipmentId"},
	}
}

func (r *ShipmentReader) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name: "shipment_get", Kind: agenttooltrust.ToolKindQuery,
		Resource: permission.ResourceShipment, Operation: permission.OpRead,
	}
}

func (r *ShipmentReader) Query(
	_ context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	r.Asked = params.Params

	return map[string]any{
		"id":        params.Params["shipmentId"],
		"proNumber": "S26100125117880",
		"status":    "New",
		"customer":  map[string]any{"name": "Harbor Line Foods"},
	}, nil
}
