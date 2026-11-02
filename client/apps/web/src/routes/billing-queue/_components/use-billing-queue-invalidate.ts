import { shipment } from "@/lib/queries/shipment";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

/**
 * Refreshes what a billing queue write can change: the queue, its items, and the
 * margin of the shipment behind them, which the server prices from the charges.
 */
export function useInvalidateBillingQueue() {
  const queryClient = useQueryClient();

  return useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
    void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
    void queryClient.invalidateQueries({ queryKey: shipment.profitability._def });
  }, [queryClient]);
}
