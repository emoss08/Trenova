import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const rateQuote = createQueryKeys("rateQuote", {
  /** The quote a shipment is currently billed from, with its full trace. */
  appliedForShipment: (shipmentId: string) => ({
    queryKey: [shipmentId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.rateQuoteService.getAppliedForShipment(shipmentId, { signal }),
  }),
});
