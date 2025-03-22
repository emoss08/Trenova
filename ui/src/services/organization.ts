import { http } from "@/lib/http-client";
import { OrganizationSchema } from "@/lib/schemas/organization-schema";
import { ShipmentControlSchema } from "@/lib/schemas/shipmentcontrol-schema";
import { DatabaseBackupListResponse } from "@/types/database-backup";
import { type Organization } from "@/types/organization";
import { type LimitOffsetResponse } from "@/types/server";

type GetOrgByIdOptions = {
  orgId: Organization["id"];
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
  orgId: Organization["id"],
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

export async function getDatabaseBackups() {
  return http.get<DatabaseBackupListResponse>(`/database-backups/`);
}

export async function deleteDatabaseBackup(fileName: string) {
  return http.delete(`/database-backups/${fileName}/`);
}

export async function restoreDatabaseBackup(fileName: string) {
  return http.post(`/database-backups/restore/`, { filename: fileName });
}

export async function createDatabaseBackup() {
  return http.post(`/database-backups/`);
}
