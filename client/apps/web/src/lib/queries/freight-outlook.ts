import { fetchFreightArea, fetchFreightOutlook } from "@/lib/graphql/freight-outlook";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/**
 * The outlook is computed from the nightly forecast and what is booked and who is free now, so
 * it polls as a floor; a positioning sent or called off refreshes it at once (realtime).
 */
export const FREIGHT_OUTLOOK_REFETCH_MS = 300_000;

export const freightOutlook = createQueryKeys("freightOutlook", {
  outlook: () => ({
    queryKey: ["outlook"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => fetchFreightOutlook({ signal }),
  }),
  area: (cell: number, slotStart: number | null) => ({
    queryKey: ["area", cell, slotStart ?? "worst"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchFreightArea({ cell, slotStart }, { signal }),
  }),
});
