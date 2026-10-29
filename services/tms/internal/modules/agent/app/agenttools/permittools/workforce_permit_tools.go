package permittools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcekit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/permit"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/permitports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramPermitID = "permitId"

const paramStateID = "stateId"

const paramPermitNumber = "permitNumber"

const paramPermitIssued = "issuedAt"

const paramPermitExpiry = workforcekit.FieldExpiresAt

const paramPermitCost = "cost"

const kindPermit = "permit"

const maxPermitNumber = 100

var permitStatuses = agenttoolschema.Source("permit.status", permit.StatusValues())

var permitFields = []string{
	paramStateID,
	paramPermitNumber,
	changepreview.FieldStatus,
	paramPermitIssued,
	paramPermitExpiry,
	paramPermitCost,
	workforcekit.FieldNotes,
}

func PermitToolProviders() []any {
	return []any{provideRecordShipmentPermitTool, provideUpdateShipmentPermitTool}
}

func permitRecord(entity *permit.Permit, id pulid.ID) toolpreview.Record {
	return workforcekit.Record(
		permission.ResourcePermit,
		id,
		"Permit "+entity.PermitNumber,
		entity.Version,
	)
}

func permitOptions() []toolpreview.Option {
	return workforcekit.Options(permitFields...)
}

func permitFactProperties() map[string]any {
	return map[string]any{
		paramStateID: agenttoolschema.KindID("The state that issued it, as the stateId of the "+
			"requirement it satisfies in shipment_list_permits. Never guess one.",
			recordkind.USState),
		paramPermitNumber: toolparams.StringProperty(
			"The permit number exactly as the state printed it.",
			maxPermitNumber,
		),
		changepreview.FieldStatus: agenttoolschema.Enum(
			"Where the permit stands. An active permit "+
				"records when it expires.",
			permitStatuses,
		),
		paramPermitIssued: agenttoolschema.DateTime("When the permit takes effect."),
		paramPermitExpiry: agenttoolschema.DateTime("When the permit stops covering the move."),
		paramPermitCost: toolparams.AmountProperty(
			"What the state charged, as a decimal string such " +
				"as 85.00.",
		),
		toolparams.ParamNote: workforcekit.NoteProperty(
			"Route conditions, escort or travel-time restrictions " +
				"the permit carries.",
		),
	}
}

func applyPermitFacts(entity *permit.Permit, params map[string]any) error {
	stateID, err := toolparams.OptionalID(params, paramStateID)
	if err != nil {
		return err
	}
	if !stateID.IsNil() {
		entity.StateID = stateID
	}
	if number, numErr := toolparams.OptionalBoundedText(params, paramPermitNumber,
		maxPermitNumber); numErr != nil {
		return numErr
	} else if number != nil {
		entity.PermitNumber = *number
	}
	if status, given, enumErr := toolparams.OptionalEnum(params, changepreview.FieldStatus,
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
		value, timeErr := toolparams.OptionalDateTime(params, field.key)
		if timeErr != nil {
			return timeErr
		}
		if value != nil {
			*field.dest = value
		}
	}
	if cost, costErr := workforcekit.OptionalNullDecimal(params, paramPermitCost); costErr != nil {
		return costErr
	} else if cost.Valid {
		entity.Cost = cost
	}
	if notes, notesErr := toolparams.OptionalBoundedText(
		params,
		toolparams.ParamNote,
		workforcekit.NoteChars,
	); notesErr != nil {
		return notesErr
	} else if notes != nil {
		entity.Notes = *notes
	}
	return nil
}

func permitResult(action string, entity *permit.Permit) *agentrecord.ToolExecutionResult {
	return &agentrecord.ToolExecutionResult{
		Action: action,
		Kind:   kindPermit,
		Name:   entity.PermitNumber,
		IDs: map[string]string{
			paramPermitID:              entity.ID.String(),
			toolparams.ParamShipmentID: entity.ShipmentID.String(),
		},
		Record: changepreview.RecordOf(changepreview.ShipmentRecordEntity, entity.ShipmentID),
	}
}

func permitSpec(
	name, description, rationale string,
	operation permission.Operation,
) *toolspec.ReceivableSpec {
	spec := workforcekit.Spec(name, description, rationale, permission.ResourcePermit, operation)
	spec.Artifact = changepreview.ShipmentRecordEntity
	spec.SearchTerms = []string{"oversize", "overweight", "OS/OW", "state permit"}
	return spec
}

func newRecordShipmentPermitTool(permits permitports.PermitService) agenttoolports.AgentTool {
	properties := permitFactProperties()
	properties[toolparams.ParamShipmentID] = shipmentpreview.ShipmentIDProperty(
		"The shipment the permit covers",
	)
	spec := workforcekit.Targeting(workforcekit.WithSchema(permitSpec(
		"shipment_record_permit",
		"Record an oversize or overweight permit a state issued for a shipment, from the "+
			"permit in hand: the state, its number, status, dates, cost and conditions. The "+
			"shipment's permit requirements are worked out again, so an active permit that "+
			"covers the trip satisfies its state and can lift the permit hold on dispatch.",
		"Records a permit on the shipment inside Trenova, which can release a dispatch "+
			"hold; nothing is sent to the state, and shipment_update_permit voids it.",
		permission.OpCreate,
	), properties, toolparams.ParamShipmentID, paramStateID, paramPermitNumber), toolparams.ParamShipmentID,
		permission.ResourceShipment)
	spec.Recipe = []string{"shipment_list_permits", "shipment_record_permit"}

	return toolspec.NewReportingReceivableTool(
		spec,
		toolspec.ReceivablePlan[*permit.Permit, *permit.Permit]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*permit.Permit, error) {
				shipmentID, err := toolparams.RequirePulid(
					params.Params,
					toolparams.ParamShipmentID,
				)
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
			Plan: func(
				ctx context.Context,
				entity *permit.Permit,
				_ *agenttoolports.ToolExecuteParams,
			) (*permit.Permit, error) {
				planned := *entity
				return permits.PlanCreatePermit(ctx, &planned)
			},
			Refused: func(*permit.Permit) string { return "Would record a permit on the shipment." },
			Render: func(_ *permit.Permit, planned *permit.Permit) (*agentpreview.ToolPreview, error) {
				created, err := toolpreview.Create(permitRecord(planned, pulid.Nil), planned,
					permitOptions()...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf("Would record permit %s as %s.",
					planned.PermitNumber, planned.Status), created), nil
			},
			Run: func(
				ctx context.Context,
				entity *permit.Permit,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				created, err := permits.CreatePermit(ctx, entity, params.Actor)
				if err != nil {
					return nil, err
				}
				return permitResult("recorded", created), nil
			},
		},
	)
}

type permitEdit struct {
	shipmentID pulid.ID
	permitID   pulid.ID
	params     *agenttoolports.ToolExecuteParams
}

func (e *permitEdit) entity(
	ctx context.Context,
	permits permitports.PermitService,
) (*permit.Permit, error) {
	recorded, err := permits.ListPermits(ctx, e.shipmentID, toolparams.TenantFrom(e.params))
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

func newUpdateShipmentPermitTool(permits permitports.PermitService) agenttoolports.AgentTool {
	properties := permitFactProperties()
	properties[toolparams.ParamShipmentID] = shipmentpreview.ShipmentIDProperty(
		"The shipment the permit covers",
	)
	properties[paramPermitID] = agenttoolschema.RecordIDText(permission.ResourcePermit,
		"The permit, from shipment_list_permits. Never guess one.")
	spec := workforcekit.Targeting(workforcekit.WithSchema(permitSpec(
		"shipment_update_permit",
		"Correct or move on a permit recorded on a shipment: activate it when the state "+
			"issues it, fix its number or dates, or void it. Give only what changes. The "+
			"shipment's permit requirements are worked out again.",
		"Changes a permit on the shipment inside Trenova, which can place or lift a "+
			"dispatch hold; nothing is sent to the state, and it is changed again the same way.",
		permission.OpUpdate,
	), properties, toolparams.ParamShipmentID, paramPermitID), paramPermitID, permission.ResourcePermit)

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*permitEdit, *kernelports.RecordChange[permit.Permit],
	]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*permitEdit, error) {
			shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
			if err != nil {
				return nil, err
			}
			permitID, err := toolparams.RequirePulid(params.Params, paramPermitID)
			if err != nil {
				return nil, err
			}
			return &permitEdit{shipmentID: shipmentID, permitID: permitID, params: params}, nil
		},
		Plan: func(
			ctx context.Context,
			edit *permitEdit,
			_ *agenttoolports.ToolExecuteParams,
		) (*kernelports.RecordChange[permit.Permit], error) {
			entity, err := edit.entity(ctx, permits)
			if err != nil {
				return nil, err
			}
			return permits.PlanUpdatePermit(ctx, entity)
		},
		Refused: func(*permitEdit) string { return "Would change a permit on the shipment." },
		Render: func(
			_ *permitEdit,
			change *kernelports.RecordChange[permit.Permit],
		) (*agentpreview.ToolPreview, error) {
			recorded, err := toolpreview.Changed(permitRecord(change.Before, change.Before.ID),
				change.Before, change.After, permitOptions()...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would change permit %s.",
				change.Before.PermitNumber), recorded), nil
		},
		Run: func(
			ctx context.Context,
			edit *permitEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
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

func provideRecordShipmentPermitTool(permits permitports.PermitService) agenttoolports.AgentTool {
	return newRecordShipmentPermitTool(permits)
}

func provideUpdateShipmentPermitTool(permits permitports.PermitService) agenttoolports.AgentTool {
	return newUpdateShipmentPermitTool(permits)
}
