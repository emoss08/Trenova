import {
  getDocumentCountByResource,
  getDocumentsByResourceID,
  getResourceSubFolders,
} from "@/services/document";
import {
  getBillingControl,
  getDatabaseBackups,
  getOrgById,
  getShipmentControl,
  listOrganizations,
} from "@/services/organization";
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
    documentsByResourceID: (resourceType: Resource, resourceId: string) => ({
      queryKey: ["document/documents-by-resource-id", resourceType, resourceId],
      queryFn: async () => {
        return await getDocumentsByResourceID(resourceType, resourceId);
      },
    }),
  },
});
