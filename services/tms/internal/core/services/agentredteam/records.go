package agentredteam

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	getShipmentTool    = "get_shipment"
	listWatchtowerTool = "list_watchtower_items"
)

var liveQueries = []string{getShipmentTool, listWatchtowerTool}

type shipmentDesk struct {
	repositories.ShipmentRepository

	rec      *recorder
	response map[string]any
}

func (d *shipmentDesk) GetByID(
	_ context.Context,
	req *repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	d.rec.read(ReadShipmentRepo, "GetByID", req.TenantInfo)

	entity, err := plantedShipment(d.response)
	if err != nil {
		return nil, err
	}
	entity.ID = req.ID
	entity.OrganizationID = req.TenantInfo.OrgID
	entity.BusinessUnitID = req.TenantInfo.BuID
	entity.Comments = nil

	return entity, nil
}

type commentDesk struct {
	repositories.ShipmentCommentRepository

	rec      *recorder
	response map[string]any
}

func (d *commentDesk) ListByShipmentID(
	_ context.Context,
	req *repositories.ListShipmentCommentsRequest,
) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
	tenant := pagination.TenantInfo{}
	if req.Filter != nil {
		tenant = req.Filter.TenantInfo
	}
	d.rec.read(ReadCommentRepo, "ListByShipmentID", tenant)

	entity, err := plantedShipment(d.response)
	if err != nil {
		return nil, err
	}

	items := make([]*shipment.ShipmentComment, 0, len(entity.Comments))
	for _, comment := range entity.Comments {
		if comment == nil {
			continue
		}
		comment.ShipmentID = req.ShipmentID
		comment.OrganizationID = tenant.OrgID
		comment.BusinessUnitID = tenant.BuID
		items = append(items, comment)
	}

	return &pagination.CursorListResult[*shipment.ShipmentComment]{Items: items}, nil
}

func plantedShipment(response map[string]any) (*shipment.Shipment, error) {
	entity := &shipment.Shipment{}
	if len(response) == 0 {
		return entity, nil
	}

	raw, err := sonic.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("encode the planted %s response: %w", getShipmentTool, err)
	}
	if err = sonic.Unmarshal(raw, entity); err != nil {
		return nil, fmt.Errorf("decode the planted %s response: %w", getShipmentTool, err)
	}

	return entity, nil
}

type watchtowerDesk struct {
	serviceports.WatchtowerService

	rec      *recorder
	response map[string]any
}

func (d *watchtowerDesk) List(
	_ context.Context,
	req serviceports.ListWatchtowerItemsRequest, //nolint:gocritic // the port takes it by value
	_ *serviceports.RequestActor,
) (*serviceports.WatchtowerPage, error) {
	d.rec.read(ReadWatchtower, "List", req.TenantInfo)

	items, err := plantedWatchtowerItems(d.response)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		item.OrganizationID = req.TenantInfo.OrgID
		item.BusinessUnitID = req.TenantInfo.BuID
	}

	return &serviceports.WatchtowerPage{Items: items}, nil
}

func plantedWatchtowerItems(response map[string]any) ([]*watchtower.Item, error) {
	planted := struct {
		Items []*watchtower.Item `json:"items"`
	}{}
	if len(response) == 0 {
		return []*watchtower.Item{}, nil
	}

	raw, err := sonic.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("encode the planted %s response: %w", listWatchtowerTool, err)
	}
	if err = sonic.Unmarshal(raw, &planted); err != nil {
		return nil, fmt.Errorf("decode the planted %s response: %w", listWatchtowerTool, err)
	}

	return planted.Items, nil
}

type permissionDesk struct {
	serviceports.PermissionEngine

	rec *recorder
}

func (d *permissionDesk) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	d.rec.read(ReadPermissions, req.Resource, pagination.TenantInfo{
		OrgID: req.OrganizationID,
		BuID:  req.BusinessUnitID,
	})

	return &serviceports.PermissionCheckResult{Allowed: true}, nil
}
