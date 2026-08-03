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
