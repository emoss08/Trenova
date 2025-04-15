import { getCustomerDocumentRequirements } from "@/services/customer";
import {
  getDocumentCountByResource,
  getDocumentsByResourceID,
  getDocumentTypes,
  getResourceSubFolders,
} from "@/services/document";
import { checkAPIKey } from "@/services/google-maps";
import {
  getBillingControl,
  getDatabaseBackups,
  getOrgById,
  getShipmentControl,
  listOrganizations,
} from "@/services/organization";
import { getShipmentByID } from "@/services/shipment";
import { getUsStateOptions, getUsStates } from "@/services/us-state";
import { Resource } from "@/types/audit-entry";
import { createQueryKeyStore } from "@lukemorales/query-key-factory";

export const queries = createQueryKeyStore({
  organization: {
    getOrgById: (
      orgId: string,
      includeState: boolean = false,
      includeBu: boolean = false,
    ) => ({
      queryKey: ["organization", orgId, includeState, includeBu],
      queryFn: async () => {
        const response = await getOrgById({
          orgId,
          includeState,
          includeBu,
        });
        return response.data;
      },
    }),
    getUserOrganizations: () => ({
      queryKey: ["organization/user"],
      queryFn: async () => {
        const response = await listOrganizations();
        return response.data;
      },
    }),
    getShipmentControl: () => ({
      queryKey: ["shipmentControl"],
      queryFn: async () => {
        const response = await getShipmentControl();
        return response.data;
      },
    }),
    getBillingControl: () => ({
      queryKey: ["billingControl"],
      queryFn: async () => {
        const response = await getBillingControl();
        return response.data;
      },
    }),
    getDatabaseBackups: () => ({
      queryKey: ["databaseBackups"],
      queryFn: async () => {
        const response = await getDatabaseBackups();
        return response.data;
      },
    }),
  },
  usState: {
    list: () => ({
      queryKey: ["us-states"],
      queryFn: async () => getUsStates(),
    }),
    options: () => ({
      queryKey: ["us-states/options"],
      queryFn: async () => {
        return await getUsStateOptions();
      },
    }),
  },
  document: {
    getDocumentTypes: () => ({
      queryKey: ["document/types"],
      queryFn: async () => {
        return await getDocumentTypes();
      },
    }),
    countByResource: () => ({
      queryKey: ["document/count-by-resource"],
      queryFn: async () => {
        return await getDocumentCountByResource();
      },
    }),
    resourceSubFolders: (resourceType: Resource) => ({
      queryKey: ["document/resource-sub-folders", resourceType],
      queryFn: async () => {
        return await getResourceSubFolders(resourceType);
      },
    }),
    documentsByResourceID: (
      resourceType: Resource,
      resourceId: string,
      limit?: number,
      offset?: number,
    ) => ({
      queryKey: [
        "document/documents-by-resource-id",
        resourceType,
        resourceId,
        limit,
        offset,
      ],
      queryFn: async () => {
        return await getDocumentsByResourceID(
          resourceType,
          resourceId,
          limit,
          offset,
        );
      },
    }),
  },
  shipment: {
    getShipment: (shipmentId: string, enabled: boolean = true) => ({
      queryKey: ["shipment", shipmentId],
      queryFn: async () => {
        return await getShipmentByID(shipmentId, true);
      },
      enabled,
    }),
  },
  customer: {
    getDocumentRequirements: (customerId: string) => ({
      queryKey: ["customer/document-requirements", customerId],
      queryFn: async () => getCustomerDocumentRequirements(customerId),
    }),
  },
  googleMaps: {
    checkAPIKey: () => ({
      queryKey: ["google-maps/check-api-key"],
      queryFn: async () => checkAPIKey(),
    }),
  },
});
