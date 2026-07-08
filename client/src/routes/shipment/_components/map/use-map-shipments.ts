import { listMapShipmentsGraphQL } from "@/lib/graphql/shipment";
import { useQuery } from "@tanstack/react-query";

const MAX_MAP_SHIPMENTS = 100;

/**
 * Active shipments to plot on the map. We exclude terminal states (Canceled,
 * Invoiced) so the map only shows routes a dispatcher can act on. Stops are
 * expanded so origin/destination lat/lon are available via shipment-utils.
 */
export function useMapShipments(enabled = true) {
  return useQuery({
    queryKey: ["shipment-list", "map", { limit: MAX_MAP_SHIPMENTS }],
    queryFn: () =>
      listMapShipmentsGraphQL({
        limit: MAX_MAP_SHIPMENTS,
        fieldFilters: [
          {
            field: "status",
            operator: "notin",
            value: ["Canceled", "Invoiced"],
          },
        ],
      }),
    staleTime: 60_000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled,
  });
}
