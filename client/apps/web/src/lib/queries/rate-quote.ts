import { rateQuoteService } from "@/services/rate";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const rateQuote = createQueryKeys("rateQuote", {
  /** The quote a shipment is currently billed from, with its full trace. */
  appliedForShipment: (shipmentId: string) => ({
    queryKey: [shipmentId],
    queryFn: async () => rateQuoteService.getAppliedForShipment(shipmentId),
  }),
  /** A shipment's rating history, newest first. */
  historyForShipment: (shipmentId: string) => ({
    queryKey: [shipmentId],
    queryFn: async () => rateQuoteService.listForShipment(shipmentId),
  }),
});
