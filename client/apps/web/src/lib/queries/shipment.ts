import { getShipmentProfitabilityGraphQL } from "@/lib/graphql/shipment";
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
