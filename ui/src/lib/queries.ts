import { api } from "@/services/api";
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
        return await api.organization.getById({
          orgId,
          includeState,
          includeBu,
        });
      },
    }),
    getUserOrganizations: () => ({
      queryKey: ["organization/user"],
      queryFn: async () => {
        return await api.organization.list();
      },
    }),
    getShipmentControl: () => ({
      queryKey: ["shipmentControl"],
      queryFn: async () => {
        return await api.shipmentControl.get();
      },
    }),
    getBillingControl: (
      { enabled }: { enabled: boolean } = { enabled: true },
    ) => ({
      queryKey: ["billingControl"],
      queryFn: async () => {
        return await api.billingControl.get();
      },
      enabled,
    }),
    getDatabaseBackups: () => ({
      queryKey: ["databaseBackups"],
      queryFn: async () => {
        return await api.databaseBackups.get();
      },
    }),
  },
  usState: {
    list: () => ({
      queryKey: ["us-states"],
      queryFn: async () => api.usStates.getUsStates(),
    }),
    options: () => ({
      queryKey: ["us-states/options"],
      queryFn: async () => {
        return await api.usStates.getUsStateOptions();
      },
    }),
  },
  document: {
    getDocumentTypes: () => ({
      queryKey: ["document/types"],
      queryFn: async () => {
        return await api.documents.getDocumentTypes();
      },
    }),
    countByResource: () => ({
      queryKey: ["document/count-by-resource"],
      queryFn: async () => {
        return await api.documents.getDocumentCountByResource();
      },
    }),
    resourceSubFolders: (resourceType: Resource) => ({
      queryKey: ["document/resource-sub-folders", resourceType],
      queryFn: async () => {
        return await api.documents.getResourceSubFolders(resourceType);
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
        return await api.documents.getByResourceID(
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
        return await api.shipments.getShipmentByID(shipmentId, true);
      },
      enabled,
    }),
  },
  customer: {
    getDocumentRequirements: (customerId: string) => ({
      queryKey: ["customer/document-requirements", customerId],
      queryFn: async () => api.customers.getDocumentRequirements(customerId),
    }),
    getById: ({
      customerId,
      includeBillingProfile = false,
      enabled,
    }: GetCustomerByIDParams) => ({
      queryKey: ["customer/by-id", customerId, includeBillingProfile],
      queryFn: async () =>
        api.customers.getById(customerId, { includeBillingProfile }),
      enabled,
    }),
  },
  integration: {
    getIntegrations: () => ({
      queryKey: ["integrations"],
      queryFn: async () => api.integrations.get(),
    }),
    getIntegrationByType: (type: IntegrationType) => ({
      queryKey: ["integrations/type", type],
      queryFn: async () => api.integrations.getByType(type),
    }),
  },
  googleMaps: {
    checkAPIKey: () => ({
      queryKey: ["google-maps/check-api-key"],
      queryFn: async () => api.googleMaps.checkAPIKey(),
    }),
    locationAutocomplete: (input: string) => ({
      queryKey: ["google-maps/location-autocomplete", input],
      queryFn: async () => {
        if (!input || input.length < 3) {
          return { data: { details: [], count: 0 } };
        }
        return api.googleMaps.locationAutocomplete(input);
      },
      enabled: input.length >= 3,
    }),
  },
  analytics: {
    getAnalytics: (page: AnalyticsPage) => ({
      queryKey: ["analytics", page],
      queryFn: async () => api.analytics.getAnalytics({ page }),
    }),
  },
});
