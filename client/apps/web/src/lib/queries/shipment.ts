import {
  listCustomerStatusNotifications,
  listCustomerTrackingLinks,
} from "@/lib/graphql/customer-tracking";
import { getCarrierTrackingLink, listShipmentCheckCalls } from "@/lib/graphql/check-calls";
import {
  getShipmentProfitabilityGraphQL,
  listShipmentTrackingEventsGraphQL,
} from "@/lib/graphql/shipment";
import { apiService } from "@/services/api";
import type { PaginationInfo } from "@trenova/shared/types/server";
import type { Shipment } from "@trenova/shared/types/shipment";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const shipment = createQueryKeys("shipment", {
  uiPolicy: () => ({
    queryKey: ["ui-policy"],
    queryFn: async () => apiService.shipmentService.getUIPolicy(),
  }),
  get: (shipmentId: Shipment["id"], params?: Record<string, string>) => ({
    queryKey: ["get", shipmentId, params],
    queryFn: async () => apiService.shipmentService.get(shipmentId, params),
  }),
  eta: (shipmentId: NonNullable<Shipment["id"]>) => ({
    queryKey: ["eta", shipmentId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.shipmentService.getEta(shipmentId, { signal }),
  }),
  billingReadiness: (shipmentId: Shipment["id"]) => ({
    queryKey: ["billing-readiness", shipmentId],
    queryFn: async () => apiService.shipmentService.getBillingReadiness(shipmentId),
  }),
  // The version participates in the key so a save refetches the assessment.
  // The shipment panel invalidates under "shipment-list", not this namespace,
  // so without it the panel would keep rendering the pre-save envelope while
  // looking authoritative.
  permitAssessment: (shipmentId: NonNullable<Shipment["id"]>, version?: number) => ({
    queryKey: ["permit-assessment", shipmentId, version],
    queryFn: async () => apiService.shipmentService.getPermitAssessment(shipmentId),
  }),
  permits: (shipmentId: NonNullable<Shipment["id"]>) => ({
    queryKey: ["permits", shipmentId],
    queryFn: async () => apiService.shipmentService.listPermits(shipmentId),
  }),
  permitRequirements: (shipmentId: NonNullable<Shipment["id"]>) => ({
    queryKey: ["permit-requirements", shipmentId],
    queryFn: async () => apiService.shipmentService.listPermitRequirements(shipmentId),
  }),
  profitability: (shipmentId: Shipment["id"]) => ({
    queryKey: ["profitability", shipmentId],
    queryFn: async () => getShipmentProfitabilityGraphQL(shipmentId),
  }),
  listUnassigned: (req: { limit: number; after?: string | null }) => ({
    queryKey: ["list-unassigned", req],
    queryFn: async () => apiService.shipmentService.listUnassigned(req),
  }),
  // The latest events on one shipment, newest first, for a card's activity.
  // Realtime refreshes them under ["shipment", "events"] with the shipment.
  events: (shipmentId: NonNullable<Shipment["id"]>, limit: number) => ({
    queryKey: ["events", shipmentId, limit],
    queryFn: async () => apiService.shipmentEventService.list({ shipmentId, limit }),
  }),
  // Every reported arrival and departure on the shipment's stops, newest first,
  // with what became of each when the stop times were settled.
  trackingEvents: (shipmentId: NonNullable<Shipment["id"]>) => ({
    queryKey: ["tracking-events", shipmentId],
    queryFn: async ({ signal }) => listShipmentTrackingEventsGraphQL(shipmentId, { signal }),
  }),
  // Every check call on the shipment's moves, newest first: dispatcher calls,
  // carrier posts through the tracking link, and agent reports.
  checkCalls: (shipmentId: NonNullable<Shipment["id"]>) => ({
    queryKey: ["check-calls", shipmentId],
    queryFn: async ({ signal }) => listShipmentCheckCalls(shipmentId, { signal }),
  }),
  // The live carrier tracking link on one move, or null when none is out.
  trackingLink: (moveId: string) => ({
    queryKey: ["tracking-link", moveId],
    queryFn: async ({ signal }) => getCarrierTrackingLink(moveId, { signal }),
  }),
  // Every customer tracking link shared for the shipment, live or not.
  customerTrackingLinks: (shipmentId: string) => ({
    queryKey: ["customer-tracking-links", shipmentId],
    queryFn: async ({ signal }) => listCustomerTrackingLinks(shipmentId, { signal }),
  }),
  // The status emails the shipment has owed its customer, and what became of each.
  customerStatusNotifications: (shipmentId: string) => ({
    queryKey: ["customer-status-notifications", shipmentId],
    queryFn: async ({ signal }) => listCustomerStatusNotifications(shipmentId, { signal }),
  }),
  listComments: (req: PaginationInfo & { shipmentId: Shipment["id"] }) => ({
    queryKey: ["comments", req.shipmentId, req],
    queryFn: async () => apiService.shipmentService.getComments(req),
  }),
});

/**
 * Every cache entry the permit engine feeds, for invalidating after a permit
 * write. The server re-syncs on each write — recording a permit can release the
 * dispatch hold — so the assessment and the shipment itself go stale too, not
 * just the permit list.
 *
 * The keys are derived from the query definitions rather than written out.
 * createQueryKeys prepends ["shipment", <methodName>] to each declared array, so
 * a hand-written key has to restate an internal naming convention and silently
 * matches nothing when it gets it wrong — which is exactly what happened here.
 */
export function permitViewQueryKeys(shipmentId: NonNullable<Shipment["id"]>) {
  return [
    shipment.permits(shipmentId).queryKey,
    shipment.permitRequirements(shipmentId).queryKey,
    // The version is the last element of the assessment key. Dropping it leaves
    // a prefix that matches every cached version for this shipment; keeping it
    // would match only the one whose version happens to be undefined.
    shipment.permitAssessment(shipmentId).queryKey.slice(0, -1),
    // The shipment row itself carries the hold badge.
    ["shipment-list"],
  ];
}
