package bulkeditservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/services/carrierservice"
	"github.com/emoss08/trenova/internal/core/services/customerservice"
	"github.com/emoss08/trenova/internal/core/services/locationservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/bulkeditports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const (
	statusField = "status"
	ownerField  = "ownerId"
	applyChunk  = 100
)

var (
	ErrRecordNotFound = errors.New("the record no longer exists")
	ErrUnknownField   = errors.New("this field cannot be changed in bulk")
)

type EditorResult struct {
	fx.Out

	Editor bulkeditports.BulkEditor `group:"bulk_editors"`
}

type statusEditor[S ~string] struct {
	resource permission.Resource
	options  []bulkeditports.BulkEditOption
	valid    func(S) bool
	load     func(ctx context.Context, tenant pagination.TenantInfo, ids []pulid.ID) (map[pulid.ID]S, error)
	apply    func(ctx context.Context, tenant pagination.TenantInfo, ids []pulid.ID, status S) error
}

func (e *statusEditor[S]) Resource() permission.Resource {
	return e.resource
}

func (e *statusEditor[S]) Fields() []bulkeditports.BulkEditField {
	return []bulkeditports.BulkEditField{{
		Name:    statusField,
		Label:   "Status",
		Kind:    bulkeditports.BulkEditFieldSelect,
		Options: e.options,
	}}
}

func (e *statusEditor[S]) Apply(
	ctx context.Context,
	req *bulkeditports.BulkEditApplyRequest,
) ([]bulkeditports.BulkEditOutcome, error) {
	if req.Field != statusField {
		return nil, ErrUnknownField
	}
	status := S(req.Value)
	if !e.valid(status) {
		return nil, errortypes.NewValidationError(
			"value",
			errortypes.ErrInvalid,
			fmt.Sprintf("%q is not a status this record can have", req.Value),
		)
	}

	current, err := e.load(ctx, req.TenantInfo, req.IDs)
	if err != nil {
		return nil, err
	}

	outcomes := make([]bulkeditports.BulkEditOutcome, 0, len(req.IDs))
	toChange := make([]pulid.ID, 0, len(req.IDs))
	for _, id := range req.IDs {
		previous, found := current[id]
		switch {
		case !found:
			outcomes = append(outcomes, bulkeditports.BulkEditOutcome{ID: id, Err: ErrRecordNotFound})
		case previous == status:
			outcomes = append(
				outcomes,
				bulkeditports.BulkEditOutcome{ID: id, Previous: string(previous)},
			)
		default:
			toChange = append(toChange, id)
		}
	}

	for start := 0; start < len(toChange); start += applyChunk {
		chunk := toChange[start:min(start+applyChunk, len(toChange))]
		applyErr := e.apply(ctx, req.TenantInfo, chunk, status)
		for _, id := range chunk {
			outcome := bulkeditports.BulkEditOutcome{ID: id, Previous: string(current[id])}
			if applyErr != nil {
				outcome.Err = applyErr
			} else {
				outcome.Changed = true
			}
			outcomes = append(outcomes, outcome)
		}
	}

	return outcomes, nil
}

func activeInactiveOptions() []bulkeditports.BulkEditOption {
	return []bulkeditports.BulkEditOption{
		{Value: string(domaintypes.StatusActive), Label: "Active"},
		{Value: string(domaintypes.StatusInactive), Label: "Inactive"},
	}
}

type CustomerEditorParams struct {
	fx.In

	Repo    customerports.CustomerRepository
	Service *customerservice.Service
}

func NewCustomerEditor(p CustomerEditorParams) EditorResult {
	return EditorResult{Editor: &statusEditor[domaintypes.Status]{
		resource: permission.ResourceCustomer,
		options:  activeInactiveOptions(),
		valid:    domaintypes.Status.IsValid,
		load: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
		) (map[pulid.ID]domaintypes.Status, error) {
			found, err := p.Repo.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
				TenantInfo:  tenant,
				CustomerIDs: ids,
			})
			if err != nil {
				return nil, err
			}
			return statusesByID(
				found,
				func(entity *customer.Customer) (pulid.ID, domaintypes.Status) {
					return entity.ID, entity.Status
				},
			), nil
		},
		apply: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status domaintypes.Status,
		) error {
			_, err := p.Service.BulkUpdateStatus(ctx, &customerports.BulkUpdateCustomerStatusRequest{
				TenantInfo:  tenant,
				CustomerIDs: ids,
				Status:      status,
			})
			return err
		},
	}}
}

type LocationEditorParams struct {
	fx.In

	Repo    locationports.LocationRepository
	Service *locationservice.Service
}

func NewLocationEditor(p LocationEditorParams) EditorResult {
	return EditorResult{Editor: &statusEditor[domaintypes.Status]{
		resource: permission.ResourceLocation,
		options:  activeInactiveOptions(),
		valid:    domaintypes.Status.IsValid,
		load: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
		) (map[pulid.ID]domaintypes.Status, error) {
			found, err := p.Repo.GetByIDs(ctx, locationports.GetLocationsByIDsRequest{
				TenantInfo:  tenant,
				LocationIDs: ids,
			})
			if err != nil {
				return nil, err
			}
			return statusesByID(
				found,
				func(entity *location.Location) (pulid.ID, domaintypes.Status) {
					return entity.ID, entity.Status
				},
			), nil
		},
		apply: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status domaintypes.Status,
		) error {
			_, err := p.Service.BulkUpdateStatus(ctx, &locationports.BulkUpdateLocationStatusRequest{
				TenantInfo:  tenant,
				LocationIDs: ids,
				Status:      status,
			})
			return err
		},
	}}
}

type CarrierEditorParams struct {
	fx.In

	Repo    carrierports.CarrierRepository
	Service *carrierservice.Service
}

func NewCarrierEditor(p CarrierEditorParams) EditorResult {
	return EditorResult{Editor: &statusEditor[carrier.Status]{
		resource: permission.ResourceCarrier,
		options: []bulkeditports.BulkEditOption{
			{Value: string(carrier.StatusActive), Label: "Active"},
			{Value: string(carrier.StatusInactive), Label: "Inactive"},
			{Value: string(carrier.StatusDoNotUse), Label: "Do not use"},
		},
		valid: carrier.Status.IsValid,
		load: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
		) (map[pulid.ID]carrier.Status, error) {
			found, err := p.Repo.GetByIDs(ctx, carrierports.GetCarriersByIDsRequest{
				TenantInfo: tenant,
				CarrierIDs: ids,
			})
			if err != nil {
				return nil, err
			}
			return statusesByID(found, func(entity *carrier.Carrier) (pulid.ID, carrier.Status) {
				return entity.ID, entity.Status
			}), nil
		},
		apply: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status carrier.Status,
		) error {
			_, err := p.Service.BulkUpdateStatus(ctx, &carrierports.BulkUpdateCarrierStatusRequest{
				TenantInfo: tenant,
				CarrierIDs: ids,
				Status:     status,
			})
			return err
		},
	}}
}

func statusesByID[T any, S ~string](entities []T, read func(T) (pulid.ID, S)) map[pulid.ID]S {
	statuses := make(map[pulid.ID]S, len(entities))
	for _, entity := range entities {
		id, status := read(entity)
		statuses[id] = status
	}
	return statuses
}

type ShipmentEditorParams struct {
	fx.In

	Repo    shipmentports.ShipmentRepository
	Service shipmentports.ShipmentService
}

type shipmentEditor struct {
	repo    shipmentports.ShipmentRepository
	service shipmentports.ShipmentService
}

func NewShipmentEditor(p ShipmentEditorParams) EditorResult {
	return EditorResult{Editor: &shipmentEditor{repo: p.Repo, service: p.Service}}
}

func (e *shipmentEditor) Resource() permission.Resource {
	return permission.ResourceShipment
}

func (e *shipmentEditor) Fields() []bulkeditports.BulkEditField {
	return []bulkeditports.BulkEditField{{
		Name:   ownerField,
		Label:  "Owner",
		Kind:   bulkeditports.BulkEditFieldRecord,
		Record: "USER",
	}}
}

func (e *shipmentEditor) Apply(
	ctx context.Context,
	req *bulkeditports.BulkEditApplyRequest,
) ([]bulkeditports.BulkEditOutcome, error) {
	if req.Field != ownerField {
		return nil, ErrUnknownField
	}
	ownerID, err := pulid.MustParse(req.Value)
	if err != nil || ownerID.IsNil() {
		return nil, errortypes.NewValidationError(
			"value",
			errortypes.ErrInvalid,
			"Choose the new owner",
		)
	}

	found, err := e.repo.GetByIDs(ctx, &shipmentports.GetShipmentsByIDsRequest{
		TenantInfo:  req.TenantInfo,
		ShipmentIDs: req.IDs,
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[pulid.ID]*shipment.Shipment, len(found))
	for _, entity := range found {
		byID[entity.ID] = entity
	}

	outcomes := make([]bulkeditports.BulkEditOutcome, 0, len(req.IDs))
	for _, id := range req.IDs {
		entity, ok := byID[id]
		if !ok {
			outcomes = append(outcomes, bulkeditports.BulkEditOutcome{ID: id, Err: ErrRecordNotFound})
			continue
		}
		previous := entity.OwnerID.String()
		if entity.OwnerID == ownerID {
			outcomes = append(outcomes, bulkeditports.BulkEditOutcome{ID: id, Previous: previous})
			continue
		}

		_, transferErr := e.service.TransferOwnership(ctx, &shipmentports.TransferOwnershipRequest{
			TenantInfo: req.TenantInfo,
			ShipmentID: id,
			OwnerID:    ownerID,
		}, req.Actor)
		outcomes = append(outcomes, bulkeditports.BulkEditOutcome{
			ID:       id,
			Previous: previous,
			Changed:  transferErr == nil,
			Err:      transferErr,
		})
	}

	return outcomes, nil
}
