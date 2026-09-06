import { getShipmentGraphQL } from "@/lib/graphql/shipment";

/** Prefix shared by every shipment list, map, and panel-detail cache entry. */
export const SHIPMENT_LIST_KEY = "shipment-list";

/** The table-configuration resource the command center saves views under. */
export const SHIPMENT_TABLE_RESOURCE_NAME = "Shipment";

/**
 * The row the edit panel opens on. Kept under the list prefix so the list's
 * invalidations after a save reach it too.
 */
export function shipmentPanelDetailQuery(shipmentId: string) {
  return {
    queryKey: [SHIPMENT_LIST_KEY, "detail", shipmentId] as const,
    queryFn: ({ signal }: { signal?: AbortSignal }) => getShipmentGraphQL(shipmentId, { signal }),
  };
}
