package agenttoolservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	capacityRecordEntity = "carrierCapacityPosting"
	paramPostingID       = "carrierCapacityPostingId"
	capacityKind         = "carrier capacity posting"
)

var (
	capacityRateMethods = agenttoolschema.Source(
		"carrierCapacity.rateMethod",
		carriercapacity.RateMethods(),
	)
	capacitySources = agenttoolschema.Source("carrierCapacity.source", carriercapacity.Sources())
)

type capacityKeeper interface {
	Get(
		ctx context.Context,
		req *repositories.GetCarrierCapacityPostingRequest,
	) (*carriercapacity.Posting, error)
	PlanCreate(
		ctx context.Context,
		entity *carriercapacity.Posting,
	) (*carriercapacity.Posting, error)
	Create(
		ctx context.Context,
		entity *carriercapacity.Posting,
		actor *serviceports.RequestActor,
	) (*carriercapacity.Posting, error)
	PlanUpdate(
		ctx context.Context,
		entity *carriercapacity.Posting,
	) (*serviceports.RecordChange[carriercapacity.Posting], error)
	Update(
		ctx context.Context,
		entity *carriercapacity.Posting,
		actor *serviceports.RequestActor,
	) (*carriercapacity.Posting, error)
	PlanDelete(
		ctx context.Context,
		req *repositories.DeleteCarrierCapacityPostingRequest,
	) (*carriercapacity.Posting, error)
	Delete(
		ctx context.Context,
		req *repositories.DeleteCarrierCapacityPostingRequest,
		actor *serviceports.RequestActor,
	) error
}

type capacityView struct {
	CarrierID         string `json:"carrierId"`
	OriginLocationID  string `json:"originLocationId,omitempty"`
	OriginState       string `json:"originState,omitempty"`
	OriginRadiusMiles *int   `json:"originRadiusMiles,omitempty"`
	DestinationState  string `json:"destinationState,omitempty"`
	EquipmentTypeID   string `json:"equipmentTypeId,omitempty"`
	AvailableFrom     string `json:"availableFrom"`
	AvailableTo       string `json:"availableTo"`
	TruckCount        int    `json:"truckCount"`
	RateMethod        string `json:"rateMethod"`
	Rate              string `json:"rate,omitempty"`
	Source            string `json:"source"`
	Notes             string `json:"notes,omitempty"`
}

func idPointerText(id *pulid.ID) string {
	if id == nil {
		return ""
	}
	return idText(*id)
}

func instantText(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func capacityViewOf(entity *carriercapacity.Posting, states map[pulid.ID]string) any {
	view := &capacityView{
		CarrierID:         entity.CarrierID.String(),
		OriginLocationID:  idPointerText(entity.OriginLocationID),
		OriginState:       stateNamePointer(states, entity.OriginStateID),
		OriginRadiusMiles: entity.OriginRadiusMiles,
		DestinationState:  stateNamePointer(states, entity.DestinationStateID),
		EquipmentTypeID:   idPointerText(entity.EquipmentTypeID),
		AvailableFrom:     instantText(entity.AvailableFrom),
		AvailableTo:       instantText(entity.AvailableTo),
		TruckCount:        entity.TruckCount,
		RateMethod:        string(entity.RateMethod),
		Source:            string(entity.Source),
		Notes:             entity.Notes,
	}
	if entity.Rate.Valid {
		view.Rate = entity.Rate.Decimal.String()
	}

	return view
}

func capacityLabel(entity *carriercapacity.Posting) string {
	label := fmt.Sprintf("%d %s from %s", entity.TruckCount,
		pluralTrucks(entity.TruckCount), instantText(entity.AvailableFrom))
	if entity.Carrier != nil && entity.Carrier.Name != "" {
		return entity.Carrier.Name + ": " + label
	}
	return label
}

func pluralTrucks(count int) string {
	if count == 1 {
		return "truck"
	}
	return "trucks"
}

var capacityRecord = &masterRecord[carriercapacity.Posting]{
	kind:     capacityKind,
	resource: permission.ResourceCarrierCapacityPosting,
	entity:   capacityRecordEntity,
	idParam:  paramPostingID,
	supplier: "from list_carrier_capacity_postings",
	label:    capacityLabel,
	id:       func(entity *carriercapacity.Posting) pulid.ID { return entity.ID },
	version:  func(entity *carriercapacity.Posting) int64 { return entity.Version },
	detach: func(entity *carriercapacity.Posting) {
		entity.Carrier = nil
		entity.OriginLocation = nil
		entity.OriginState = nil
		entity.DestinationState = nil
		entity.EquipmentType = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	stateIDs: func(entity *carriercapacity.Posting) []pulid.ID {
		ids := make([]pulid.ID, 0, 2)
		if entity.OriginStateID != nil {
			ids = append(ids, *entity.OriginStateID)
		}
		if entity.DestinationStateID != nil {
			ids = append(ids, *entity.DestinationStateID)
		}
		return ids
	},
	view:       capacityViewOf,
	linkEntity: carrierRecordEntity,
	linkID:     func(entity *carriercapacity.Posting) pulid.ID { return entity.CarrierID },
	options: []toolpreview.Option{toolpreview.WithRefs(map[string]permission.Resource{
		paramCarrierID:     permission.ResourceCarrier,
		"originLocationId": permission.ResourceLocation,
		"equipmentTypeId":  permission.ResourceEquipmentType,
	})},
}

func capacityFields() []masterField[carriercapacity.Posting] {
	return []masterField[carriercapacity.Posting]{
		masterID(paramCarrierID, permission.ResourceCarrier,
			"The carrier offering the trucks, from list_carriers.",
			func(p *carriercapacity.Posting) *pulid.ID { return &p.CarrierID }),
		masterOptionalIDPointer("originLocationId", permission.ResourceLocation,
			"Where the trucks are, from list_locations. Give this, originState, or both.",
			func(p *carriercapacity.Posting) **pulid.ID { return &p.OriginLocationID }),
		masterStatePointer("originState", "The state the trucks are in, when no location "+
			"is named.",
			func(p *carriercapacity.Posting) **pulid.ID { return &p.OriginStateID }),
		masterOptionalInt("originRadiusMiles", "How many miles from the origin location the "+
			"carrier will pick up; only with an origin location.", 1,
			carriercapacity.MaxRadiusMiles,
			func(p *carriercapacity.Posting) **int { return &p.OriginRadiusMiles }),
		masterStatePointer("destinationState", "Where the carrier wants to go, when it said.",
			func(p *carriercapacity.Posting) **pulid.ID { return &p.DestinationStateID }),
		masterOptionalIDPointer("equipmentTypeId", permission.ResourceEquipmentType,
			"The trailer type offered, from list_equipment_types.",
			func(p *carriercapacity.Posting) **pulid.ID { return &p.EquipmentTypeID }),
		masterDateTime("availableFrom", "When the trucks are free.",
			func(p *carriercapacity.Posting) *int64 { return &p.AvailableFrom }),
		masterDateTime("availableTo", "When the offer ends, at most 60 days after it starts.",
			func(p *carriercapacity.Posting) *int64 { return &p.AvailableTo }),
		masterInt("truckCount", "How many trucks are offered.", 1,
			carriercapacity.MaxTruckCount,
			func(p *carriercapacity.Posting) *int { return &p.TruckCount }),
		masterEnum("rateMethod", "Whether the rate is per loaded mile or for the whole move.",
			capacityRateMethods,
			func(p *carriercapacity.Posting) *carriercapacity.RateMethod { return &p.RateMethod }),
		masterNullDecimal("rate", "The rate the carrier asked.",
			func(p *carriercapacity.Posting) *decimal.NullDecimal { return &p.Rate }),
		masterEnum("source", "How the offer reached you.", capacitySources,
			func(p *carriercapacity.Posting) *carriercapacity.Source { return &p.Source }),
		masterText("notes", "Anything else the carrier said.", carriercapacity.MaxNotesLength,
			false, func(p *carriercapacity.Posting) *string { return &p.Notes }),
	}
}

func getCapacity(keeper capacityKeeper) func(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*carriercapacity.Posting, error) {
	return func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		id pulid.ID,
	) (*carriercapacity.Posting, error) {
		return keeper.Get(ctx, &repositories.GetCarrierCapacityPostingRequest{
			ID:         id,
			TenantInfo: tenant,
		})
	}
}

func newCreateCarrierCapacityPostingTool(
	keeper capacityKeeper,
	states stateLookup,
) serviceports.AgentTool {
	return newMasterCreateTool(&masterCreateSpec[carriercapacity.Posting]{
		record: capacityRecord,
		name:   "create_carrier_capacity_posting",
		description: "Record trucks a carrier says it has available: where they are, " +
			"when, how many, the equipment and the rate it asked. Coverage suggestions " +
			"and the capacity strip offer these trucks against uncovered loads.",
		rationale: "Adds a capacity posting inside Trenova; nothing is sent to the carrier, " +
			"and delete_carrier_capacity_posting removes it.",
		fields: capacityFields(),
		required: []string{
			paramCarrierID,
			"availableFrom",
			"availableTo",
		},
		searchTerms: []string{
			"carrier capacity",
			"truck posting",
			"available trucks",
			"lane offer",
		},
		policy: mdInternalAsk,
		states: states,
		fresh: func(tenant pagination.TenantInfo) *carriercapacity.Posting {
			return &carriercapacity.Posting{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				TruckCount:     1,
				RateMethod:     carriercapacity.RateMethodPerMile,
				Source:         carriercapacity.SourceManual,
			}
		},
		plan:   keeper.PlanCreate,
		create: keeper.Create,
	})
}

func newUpdateCarrierCapacityPostingTool(
	keeper capacityKeeper,
	states stateLookup,
) serviceports.AgentTool {
	return newMasterUpdateTool(&masterUpdateSpec[carriercapacity.Posting]{
		record: capacityRecord,
		name:   "update_carrier_capacity_posting",
		description: "Change a carrier capacity posting: its origin, destination, window, " +
			"truck count, equipment or rate. Fields left out keep their value.",
		rationale: "Changes a capacity posting inside Trenova; nothing is sent, and a later " +
			"change puts it back.",
		fields:      capacityFields(),
		searchTerms: []string{"edit carrier capacity", "change truck posting", "capacity rate"},
		policy:      mdInternalAsk,
		states:      states,
		get:         getCapacity(keeper),
		plan:        keeper.PlanUpdate,
		update:      keeper.Update,
	})
}

func newDeleteCarrierCapacityPostingTool(keeper capacityKeeper) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name:        "delete_carrier_capacity_posting",
		description: "Remove a carrier capacity posting the carrier withdrew or that was entered by mistake.",
		resource:    permission.ResourceCarrierCapacityPosting,
		operation:   permission.OpDelete,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		rationale: "Removes trucks a carrier offered from Trenova; nothing is sent, and " +
			"create_carrier_capacity_posting records them again.",
		properties: map[string]any{
			paramPostingID: agenttoolschema.RecordIDText(
				permission.ResourceCarrierCapacityPosting,
				"The posting to remove, "+capacityRecord.supplier+". Never guess one.",
			),
		},
		required:    []string{paramPostingID},
		searchTerms: []string{"remove carrier capacity", "withdraw truck posting"},
		target:      capacityRecord.target,
	}, receivablePlan[pulid.ID, *carriercapacity.Posting]{
		request: func(params *serviceports.ToolExecuteParams) (pulid.ID, error) {
			return requirePulid(params.Params, paramPostingID)
		},
		plan: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*carriercapacity.Posting, error) {
			return keeper.PlanDelete(ctx, &repositories.DeleteCarrierCapacityPostingRequest{
				ID:         id,
				TenantInfo: tenantFrom(*params),
			})
		},
		refused: func(pulid.ID) string { return "Would remove a carrier capacity posting." },
		render: func(_ pulid.ID, posting *carriercapacity.Posting) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(capacityRecord.record(posting),
				capacityRecord.viewOf(posting, nil), capacityRecord.options...)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would remove the posting %s.",
				capacityLabel(posting)), change), nil
		},
		run: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			tenant := tenantFrom(*params)
			current, err := keeper.Get(ctx, &repositories.GetCarrierCapacityPostingRequest{
				ID:         id,
				TenantInfo: tenant,
			})
			if err != nil {
				return nil, err
			}
			if err = keeper.Delete(ctx, &repositories.DeleteCarrierCapacityPostingRequest{
				ID:         id,
				Version:    current.Version,
				TenantInfo: tenant,
			}, params.Actor); err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: actionDeleted,
				Kind:   capacityKind,
				IDs:    map[string]string{paramPostingID: id.String()},
			}, nil
		},
	})
}
