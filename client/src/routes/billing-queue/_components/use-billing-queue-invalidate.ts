import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

export function useInvalidateBillingQueue() {
  const queryClient = useQueryClient();

  return useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
    void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
  }, [queryClient]);
}
