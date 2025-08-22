/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { api } from "@/services/api";
import type { GetDedicatedLaneByShipmentRequest } from "@/services/dedicated-lane";
import type { GetPreviousRatesRequest } from "@/services/shipment";
import type { AnalyticsPage } from "@/types/analytics";
import { Resource } from "@/types/audit-entry";
import type { GetCustomerByIDParams } from "@/types/customer";
import type { IntegrationType } from "@/types/integration";
import type { NotificationQueryParams } from "@/types/notification";
import type { ShipmentQueryParams } from "@/types/shipment";
import { createQueryKeyStore } from "@lukemorales/query-key-factory";
import type { AccessorialChargeSchema } from "./schemas/accessorial-charge-schema";
import { HoldReasonSchema } from "./schemas/hold-reason-schema";
import type { PatternConfigSchema } from "./schemas/pattern-config-schema";
import { ShipmentSchema } from "./schemas/shipment-schema";
import type { TableConfigurationSchema } from "./schemas/table-configuration-schema";

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
    getConsolidationSettings: () => ({
      queryKey: ["consolidationSettings"],
      queryFn: async () => {
        return await api.consolidationSettings.get();
      },
    }),
    getDatabaseBackups: () => ({
      queryKey: ["databaseBackups"],
      queryFn: async () => {
        return await api.databaseBackups.get();
      },
    }),
  },
  accessorialCharge: {
    getById: (accId: AccessorialChargeSchema["id"]) => ({
      queryKey: ["accessorial-charge", accId],
      queryFn: async () => api.accessorialCharge.getById(accId),
    }),
  },
  user: {
    getUserById: (userId: string) => ({
      queryKey: ["user", userId],
      queryFn: async () => api.user.getUserById(userId),
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
  holdReason: {
    getById: (id: HoldReasonSchema["id"]) => ({
      queryKey: ["hold-reason", id],
      queryFn: async () => api.holdReasons.getById(id),
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
    list: (params: ShipmentQueryParams) => ({
      queryKey: ["shipment/list", params],
      queryFn: async () => api.shipments.getShipments(params),
    }),
    getShipment: (
      shipmentId: ShipmentSchema["id"],
      enabled: boolean = true,
    ) => ({
      queryKey: ["shipment", shipmentId],
      queryFn: async () => {
        return await api.shipments.getShipmentByID(shipmentId, true);
      },
      enabled,
    }),
    getPreviousRates: (values: GetPreviousRatesRequest) => ({
      queryKey: ["shipment/previous-rates", values],
      queryFn: async () => api.shipments.getPreviousRates(values),
    }),
    listComments: (
      shipmentId: ShipmentSchema["id"],
      enabled: boolean = true,
    ) => ({
      queryKey: ["shipment/comments", shipmentId],
      queryFn: async () => api.shipments.listComments(shipmentId),
      enabled,
    }),
    getCommentCount: (shipmentId: ShipmentSchema["id"]) => ({
      queryKey: ["shipment/comments/count", shipmentId],
      queryFn: async () => api.shipments.getCommentCount(shipmentId),
    }),
    getHolds: (shipmentId: ShipmentSchema["id"]) => ({
      queryKey: ["shipment/holds", shipmentId],
      queryFn: async () => api.shipments.getHolds(shipmentId),
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
  tableConfiguration: {
    listUserConfigurations: (resource: Resource) => ({
      queryKey: ["table-configurations", resource],
      queryFn: async () =>
        api.tableConfigurations.listUserConfigurations(resource),
    }),
    listPublicConfigurations: (resource: Resource) => ({
      queryKey: ["table-configurations", resource],
      queryFn: async () =>
        api.tableConfigurations.listPublicConfigurations(resource),
    }),
    getDefaultOrLatestConfiguration: (resource: Resource) => ({
      queryKey: ["table-configurations", resource],
      queryFn: async () =>
        api.tableConfigurations.getDefaultOrLatestConfiguration(resource),
    }),
    create: (payload: TableConfigurationSchema) => ({
      queryKey: ["table-configurations", payload],
      queryFn: async () => api.tableConfigurations.create(payload),
    }),
  },
  favorite: {
    list: () => ({
      queryKey: ["favorites"],
      queryFn: async () => api.favorites.list(),
    }),
    check: (pageUrl: string) => ({
      queryKey: ["favorite", pageUrl],
      queryFn: async () => api.favorites.checkFavorite(pageUrl),
    }),
  },
  permission: {
    list: (limit: number, offset: number) => ({
      queryKey: ["permissions", limit, offset],
      queryFn: async () => api.permissions.list(limit, offset),
    }),
    getById: (id: string) => ({
      queryKey: ["permissions", id],
      queryFn: async () => api.permissions.getById(id),
    }),
  },
  role: {
    list: (limit: number, offset: number) => ({
      queryKey: ["roles", limit, offset],
      queryFn: async () => api.roles.list(limit, offset),
    }),
    getById: (id: string) => ({
      queryKey: ["roles", id],
      queryFn: async () => api.roles.getById(id),
    }),
  },
  dedicatedLane: {
    getByShipment: (req: GetDedicatedLaneByShipmentRequest) => ({
      queryKey: ["dedicated-lane/by-shipment", req],
      queryFn: async () => api.dedicatedLane.getByShipment(req),
    }),
  },
  dedicatedLaneSuggestion: {
    getSuggestions: (limit: number = 10, offset: number = 0) => ({
      queryKey: ["dedicated-lane-suggestions", limit, offset],
      queryFn: async () =>
        api.dedicatedLaneSuggestions.getSuggestions(limit, offset),
    }),
    getSuggestionByID: (id: string) => ({
      queryKey: ["dedicated-lane-suggestions", id],
      queryFn: async () => api.dedicatedLaneSuggestions.getSuggestionByID(id),
    }),
    analyzePatterns: () => ({
      queryKey: ["dedicated-lane-suggestions", "analyze-patterns"],
      queryFn: async () => api.dedicatedLaneSuggestions.analyzePatterns(),
    }),
    expireOldSuggestions: () => ({
      queryKey: ["dedicated-lane-suggestions", "expire-old"],
      queryFn: async () => api.dedicatedLaneSuggestions.expireOldSuggestions(),
    }),
  },
  patternConfig: {
    get: () => ({
      queryKey: ["pattern-config"],
      queryFn: async () => api.patternConfig.get(),
    }),
    update: (payload: PatternConfigSchema) => ({
      queryKey: ["pattern-config", payload],
      queryFn: async () => api.patternConfig.update(payload),
    }),
  },
  notification: {
    list: (params?: NotificationQueryParams) => ({
      queryKey: ["notifications", params],
      queryFn: async () => api.notifications.list(params),
    }),
  },
});
