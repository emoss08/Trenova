import { getAnalytics } from "@/services/analytics";
import {
  getCustomerById,
  getCustomerDocumentRequirements,
} from "@/services/customer";
import {
  getDocumentCountByResource,
  getDocumentsByResourceID,
  getDocumentTypes,
  getResourceSubFolders,
} from "@/services/document";
import { checkAPIKey, locationAutocomplete } from "@/services/google-maps";
import { getIntegrationByType, getIntegrations } from "@/services/integration";
import {
  getBillingControl,
  getDatabaseBackups,
  getOrgById,
  getShipmentControl,
  listOrganizations,
} from "@/services/organization";
import { getShipmentByID } from "@/services/shipment";
import { getUsStateOptions, getUsStates } from "@/services/us-state";
import type { AnalyticsPage } from "@/types/analytics";
import { Resource } from "@/types/audit-entry";
import type { GetCustomerByIDParams } from "@/types/customer";
import type { IntegrationType } from "@/types/integration";
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
    getBillingControl: (
      { enabled }: { enabled: boolean } = { enabled: true },
    ) => ({
      queryKey: ["billingControl"],
      queryFn: async () => {
        const response = await getBillingControl();
        return response.data;
      },
      enabled,
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
      enabled: boolean = true,
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
      enabled,
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
    getById: ({
      customerId,
      includeBillingProfile = false,
      enabled,
    }: GetCustomerByIDParams) => ({
      queryKey: ["customer/by-id", customerId, includeBillingProfile],
      queryFn: async () =>
        getCustomerById(customerId, { includeBillingProfile }),
      enabled,
    }),
  },
  integration: {
    getIntegrations: () => ({
      queryKey: ["integrations"],
      queryFn: async () => getIntegrations(),
    }),
    getIntegrationByType: (type: IntegrationType) => ({
      queryKey: ["integrations/type", type],
      queryFn: async () => getIntegrationByType(type),
    }),
  },
  googleMaps: {
    checkAPIKey: () => ({
      queryKey: ["google-maps/check-api-key"],
      queryFn: async () => checkAPIKey(),
    }),
    locationAutocomplete: (input: string) => ({
      queryKey: ["google-maps/location-autocomplete", input],
      queryFn: async () => {
        if (!input || input.length < 3) {
          return { data: { details: [], count: 0 } };
        }
        return locationAutocomplete(input);
      },
      enabled: input.length >= 3,
    }),
  },
  analytics: {
    getAnalytics: (page: AnalyticsPage) => ({
      queryKey: ["analytics", page],
      queryFn: async () => getAnalytics({ page }),
    }),
  },
});
