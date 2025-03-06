import { http } from "@/lib/http-client";
import { OrganizationSchema } from "@/lib/schemas/organization-schema";
import { ShipmentControlSchema } from "@/lib/schemas/shipmentcontrol-schema";
import { type Organization } from "@/types/organization";
import { type LimitOffsetResponse } from "@/types/server";

type GetOrgByIdOptions = {
  orgId: string;
  includeState?: boolean;
  includeBu?: boolean;
};

export async function getOrgById({
  orgId,
  includeState = false,
  includeBu = false,
}: GetOrgByIdOptions) {
  return http.get<Organization>(`/organizations/${orgId}/`, {
    params: {
      includeState: includeState.toString(),
      includeBu: includeBu.toString(),
    },
  });
}

export async function listOrganizations() {
  return http.get<LimitOffsetResponse<Organization>>("/organizations/me/");
}

export async function updateOrganization(
  orgId: string,
  data: Organization | OrganizationSchema,
) {
  return http.put<Organization>(`/organizations/${orgId}/`, data);
}

export async function updateShipmentControl(data: ShipmentControlSchema) {
  return http.put<ShipmentControlSchema>(`/shipment-controls/`, data);
}

export async function getShipmentControl() {
  return http.get<ShipmentControlSchema>(`/shipment-controls/`);
}
