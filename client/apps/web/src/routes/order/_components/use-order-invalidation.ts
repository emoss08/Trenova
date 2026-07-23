import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

// Order membership/charge mutations change the order's derived status and total, the
// table row the edit panel re-seeds from, and the shipments whose orderId moved — so
// all three query families must refetch together.
export function useOrderInvalidation() {
  const queryClient = useQueryClient();

  return useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["order-detail"] });
    void queryClient.invalidateQueries({ queryKey: ["order-list"] });
    void queryClient.invalidateQueries({ queryKey: ["shipment-list"] });
  }, [queryClient]);
}

export function useOrderInvoiceInvalidation() {
  const queryClient = useQueryClient();
  const invalidateOrders = useOrderInvalidation();

  return useCallback(() => {
    invalidateOrders();
    void queryClient.invalidateQueries({ queryKey: ["invoice-list"] });
    void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
  }, [invalidateOrders, queryClient]);
}
