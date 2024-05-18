package models

import (
	"context"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/billingcontrol"
	"github.com/emoss08/trenova/internal/ent/dispatchcontrol"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/shipmentcontrol"
	"github.com/google/uuid"
)

func GetShipmentControlByOrganization(
	ctx context.Context, client *ent.Client, organizationID, businessUnitID uuid.UUID,
) (*ent.ShipmentControl, error) {
	shipmentControl, err := client.ShipmentControl.Query().
		Where(
			shipmentcontrol.HasOrganizationWith(
				organization.IDEQ(organizationID),
				organization.BusinessUnitIDEQ(businessUnitID),
			),
		).Only(ctx)
	if err != nil {
		return nil, err
	}

	return shipmentControl, nil
}

func GetBillingControlByOrganization(
	ctx context.Context, client *ent.Client, organizationID, businessUnitID uuid.UUID,
) (*ent.BillingControl, error) {
	billingControl, err := client.BillingControl.Query().
		Where(
			billingcontrol.HasOrganizationWith(
				organization.IDEQ(organizationID),
				organization.BusinessUnitIDEQ(businessUnitID),
			),
		).Only(ctx)
	if err != nil {
		return nil, err
	}

	return billingControl, nil
}

func GetDispatchControlByOrganization(
	ctx context.Context, client *ent.Client, organizationID, businessUnitID uuid.UUID,
) (*ent.DispatchControl, error) {
	dispatchControl, err := client.DispatchControl.Query().
		Where(
			dispatchcontrol.HasOrganizationWith(
				organization.IDEQ(organizationID),
				organization.BusinessUnitIDEQ(businessUnitID),
			),
		).Only(ctx)
	if err != nil {
		return nil, err
	}

	return dispatchControl, nil
}
