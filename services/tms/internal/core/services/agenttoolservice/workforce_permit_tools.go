package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramPermitID     = "permitId"
	paramStateID      = "stateId"
	paramPermitNumber = "permitNumber"
	paramPermitIssued = "issuedAt"
	paramPermitExpiry = wfFieldExpiresAt
	paramPermitCost   = "cost"
	kindPermit        = "permit"
	maxPermitNumber   = 100
)

var (
	permitStatuses = agenttoolschema.Source("permit.status", permit.StatusValues())
	permitFields   = []string{
		paramStateID, paramPermitNumber, fieldStatus, paramPermitIssued, paramPermitExpiry,
		paramPermitCost, wfFieldNotes,
	}
)

func permitToolProviders() []any {
	return []any{provideRecordShipmentPermitTool, provideUpdateShipmentPermitTool}
}

func permitRecord(entity *permit.Permit, id pulid.ID) toolpreview.Record {
	return wfRecord(permission.ResourcePermit, id, "Permit "+entity.PermitNumber, entity.Version)
}

func permitOptions() []toolpreview.Option {
	return wfOptions(permitFields...)
}

func permitFactProperties() map[string]any {
	return map[string]any{
		paramStateID: agenttoolschema.KindID("The state that issued it, as the stateId of the "+
			"requirement it satisfies in list_shipment_permits. Never guess one.",
			permission.KindUSState),
		paramPermitNumber: stringProperty("The permit number exactly as the state printed it.",
			maxPermitNumber),
		fieldStatus: agenttoolschema.Enum("Where the permit stands. An active permit "+
			"records when it expires.", permitStatuses),
		paramPermitIssued: agenttoolschema.DateTime("When the permit takes effect."),
		paramPermitExpiry: agenttoolschema.DateTime("When the permit stops covering the move."),
		paramPermitCost: amountProperty("What the state charged, as a decimal string such " +
			"as 85.00."),
		fieldNotes: wfNoteProperty("Route conditions, escort or travel-time restrictions " +
			"the permit carries."),
	}
}

func applyPermitFacts(entity *permit.Permit, params map[string]any) error {
	stateID, err := optionalID(params, paramStateID)
	if err != nil {
		return err
	}
	if !stateID.IsNil() {
		entity.StateID = stateID
	}
	if number, numErr := optionalBoundedText(params, paramPermitNumber,
		maxPermitNumber); numErr != nil {
		return numErr
	} else if number != nil {
		entity.PermitNumber = *number
	}
	if status, given, enumErr := optionalEnum(params, fieldStatus,
		permitStatuses.Values); enumErr != nil {
		return enumErr
	} else if given {
		entity.Status = status
	}
	for _, field := range []struct {
		key  string
		dest **int64
	}{
		{paramPermitIssued, &entity.IssuedAt},
		{paramPermitExpiry, &entity.ExpiresAt},
	} {
		value, timeErr := optionalDateTime(params, field.key)
		if timeErr != nil {
			return timeErr
		}
		if value != nil {
			*field.dest = value
		}
	}
	if cost, costErr := optionalNullDecimal(params, paramPermitCost); costErr != nil {
		return costErr
	} else if cost.Valid {
		entity.Cost = cost
	}
	if notes, notesErr := optionalBoundedText(params, fieldNotes, wfNoteChars); notesErr != nil {
		return notesErr
	} else if notes != nil {
		entity.Notes = *notes
	}
	return nil
}

func permitResult(action string, entity *permit.Permit) *agent.ToolExecutionResult {
	return &agent.ToolExecutionResult{
		Action: action,
		Kind:   kindPermit,
		Name:   entity.PermitNumber,
		IDs: map[string]string{
			paramPermitID:   entity.ID.String(),
			paramShipmentID: entity.ShipmentID.String(),
		},
		Record: recordOf(shipmentRecordEntity, entity.ShipmentID),
	}
}

func permitSpec(
	name, description, rationale string,
	operation permission.Operation,
) *receivableSpec {
	spec := wfSpec(name, description, rationale, permission.ResourcePermit, operation)
	spec.artifact = shipmentRecordEntity
	spec.searchTerms = []string{"oversize", "overweight", "OS/OW", "state permit"}
	return spec
}

func newRecordShipmentPermitTool(permits serviceports.PermitService) serviceports.AgentTool {
	properties := permitFactProperties()
	properties[paramShipmentID] = shipmentIDProperty("The shipment the permit covers")
	spec := targeting(withSchema(permitSpec(
		"record_shipment_permit",
		"Record an oversize or overweight permit a state issued for a shipment, from the "+
			"permit in hand: the state, its number, status, dates, cost and conditions. The "+
			"shipment's permit requirements are worked out again, so an active permit that "+
			"covers the trip satisfies its state and can lift the permit hold on dispatch.",
		"Records a permit on the shipment inside Trenova, which can release a dispatch "+
			"hold; nothing is sent to the state, and update_shipment_permit voids it.",
		permission.OpCreate,
	), properties, paramShipmentID, paramStateID, paramPermitNumber), paramShipmentID,
		permission.ResourceShipment)
	spec.recipe = []string{"list_shipment_permits", "record_shipment_permit"}

	return newReportingReceivableTool(spec, receivablePlan[*permit.Permit, *permit.Permit]{
		request: func(params *serviceports.ToolExecuteParams) (*permit.Permit, error) {
			shipmentID, err := requirePulid(params.Params, paramShipmentID)
			if err != nil {
				return nil, err
			}
			entity := &permit.Permit{
				OrganizationID: params.OrganizationID,
				BusinessUnitID: params.BusinessUnitID,
				ShipmentID:     shipmentID,
				Status:         permit.StatusPending,
			}
			return entity, applyPermitFacts(entity, params.Params)
		},
		plan: func(
			ctx context.Context,
			entity *permit.Permit,
			_ *serviceports.ToolExecuteParams,
		) (*permit.Permit, error) {
			planned := *entity
			return permits.PlanCreatePermit(ctx, &planned)
		},
		refused: func(*permit.Permit) string { return "Would record a permit on the shipment." },
		render: func(_ *permit.Permit, planned *permit.Permit) (*agent.ToolPreview, error) {
			created, err := toolpreview.Create(permitRecord(planned, pulid.Nil), planned,
				permitOptions()...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would record permit %s as %s.",
				planned.PermitNumber, planned.Status), created), nil
		},
		run: func(
			ctx context.Context,
			entity *permit.Permit,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := permits.CreatePermit(ctx, entity, params.Actor)
			if err != nil {
				return nil, err
			}
			return permitResult("recorded", created), nil
		},
	})
}

type permitEdit struct {
	shipmentID pulid.ID
	permitID   pulid.ID
	params     *serviceports.ToolExecuteParams
}

func (e *permitEdit) entity(
	ctx context.Context,
	permits serviceports.PermitService,
) (*permit.Permit, error) {
	recorded, err := permits.ListPermits(ctx, e.shipmentID, tenantFrom(*e.params))
	if err != nil {
		return nil, err
	}
	for _, current := range recorded {
		if current == nil || current.ID != e.permitID {
			continue
		}
		entity := *current
		entity.State = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
		return &entity, applyPermitFacts(&entity, e.params.Params)
	}
	return nil, errortypes.NewNotFoundError("This shipment has no permit {0}",
		e.permitID.String())
}

func newUpdateShipmentPermitTool(permits serviceports.PermitService) serviceports.AgentTool {
	properties := permitFactProperties()
	properties[paramShipmentID] = shipmentIDProperty("The shipment the permit covers")
	properties[paramPermitID] = agenttoolschema.RecordIDText(permission.ResourcePermit,
		"The permit, from list_shipment_permits. Never guess one.")
	spec := targeting(withSchema(permitSpec(
		"update_shipment_permit",
		"Correct or move on a permit recorded on a shipment: activate it when the state "+
			"issues it, fix its number or dates, or void it. Give only what changes. The "+
			"shipment's permit requirements are worked out again.",
		"Changes a permit on the shipment inside Trenova, which can place or lift a "+
			"dispatch hold; nothing is sent to the state, and it is changed again the same way.",
		permission.OpUpdate,
	), properties, paramShipmentID, paramPermitID), paramPermitID, permission.ResourcePermit)

	return newReportingReceivableTool(spec, receivablePlan[
		*permitEdit, *serviceports.RecordChange[permit.Permit],
	]{
		request: func(params *serviceports.ToolExecuteParams) (*permitEdit, error) {
			shipmentID, err := requirePulid(params.Params, paramShipmentID)
			if err != nil {
				return nil, err
			}
			permitID, err := requirePulid(params.Params, paramPermitID)
			if err != nil {
				return nil, err
			}
			return &permitEdit{shipmentID: shipmentID, permitID: permitID, params: params}, nil
		},
		plan: func(
			ctx context.Context,
			edit *permitEdit,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.RecordChange[permit.Permit], error) {
			entity, err := edit.entity(ctx, permits)
			if err != nil {
				return nil, err
			}
			return permits.PlanUpdatePermit(ctx, entity)
		},
		refused: func(*permitEdit) string { return "Would change a permit on the shipment." },
		render: func(
			_ *permitEdit,
			change *serviceports.RecordChange[permit.Permit],
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(permitRecord(change.Before, change.Before.ID),
				change.Before, change.After, permitOptions()...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would change permit %s.",
				change.Before.PermitNumber), recorded), nil
		},
		run: func(
			ctx context.Context,
			edit *permitEdit,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := edit.entity(ctx, permits)
			if err != nil {
				return nil, err
			}
			updated, err := permits.UpdatePermit(ctx, entity, params.Actor)
			if err != nil {
				return nil, err
			}
			return permitResult("updated", updated), nil
		},
	})
}

func provideRecordShipmentPermitTool(permits serviceports.PermitService) serviceports.AgentTool {
	return newRecordShipmentPermitTool(permits)
}

func provideUpdateShipmentPermitTool(permits serviceports.PermitService) serviceports.AgentTool {
	return newUpdateShipmentPermitTool(permits)
}
