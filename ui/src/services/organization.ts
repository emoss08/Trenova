import { http } from "@/lib/http-client";
import { type BillingControlSchema } from "@/lib/schemas/billing-schema";
import { type OrganizationSchema } from "@/lib/schemas/organization-schema";
import { type ShipmentControlSchema } from "@/lib/schemas/shipmentcontrol-schema";
import { type DatabaseBackupListResponse } from "@/types/database-backup";
import { type Organization } from "@/types/organization";
import { type LimitOffsetResponse } from "@/types/server";

/**
 * Get the organization for the current user
 */
type GetOrganizationByIdParams = {
  orgId: OrganizationSchema["id"];
  includeState?: boolean;
  includeBu?: boolean;
};

/**
 * Get the organization for the current user
 */
export async function getOrgById({
  orgId,
  includeState = false,
  includeBu = false,
}: GetOrganizationByIdParams) {
  return http.get<OrganizationSchema>(`/organizations/${orgId}/`, {
    params: {
      includeState: includeState.toString(),
      includeBu: includeBu.toString(),
    },
  });
}

/**
 * Get the list of organizations for the current user
 */
export async function listOrganizations() {
  return http.get<LimitOffsetResponse<OrganizationSchema>>(
    "/organizations/me/",
  );
}

/**
 * Update the organization for the current user
 */
export async function updateOrganization(
  orgId: Organization["id"],
  data: OrganizationSchema,
) {
  return http.put<OrganizationSchema>(`/organizations/${orgId}/`, data);
}

/**
 * Get the billing control for the current organization
 */
export async function getBillingControl() {
  return http.get<BillingControlSchema>("/billing-controls/");
}

/**
 * Update the billing control for the current organization
 */
export async function updateBillingControl(data: BillingControlSchema) {
  return http.put<BillingControlSchema>(`/billing-controls/`, data);
}

/**
 * Get the shipment control for the current organization
 */
export async function getShipmentControl() {
  return http.get<ShipmentControlSchema>("/shipment-controls/");
}

/**
 * Update the shipment control for the current organization
 */
export async function updateShipmentControl(data: ShipmentControlSchema) {
  return http.put<ShipmentControlSchema>(`/shipment-controls/`, data);
}

/**
 * Get the list of database backups
 */
export async function getDatabaseBackups() {
  return http.get<DatabaseBackupListResponse>("/database-backups/");
}

/**
 * Delete a database backup
 */
export async function deleteDatabaseBackup(fileName: string) {
  return http.delete(`/database-backups/${fileName}/`);
}

/**
 * Restore a database backup
 */
export async function restoreDatabaseBackup(fileName: string) {
  return http.post(`/database-backups/restore/`, { filename: fileName });
}

/**
 * Create a new database backup
 */
export async function createDatabaseBackup() {
  return http.post(`/database-backups/`);
}
