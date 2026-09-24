package agentredteam

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

const getShipmentTool = "get_shipment"

var liveQueries = []string{getShipmentTool}

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
