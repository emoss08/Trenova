import { invoiceService } from "@/services/invoice";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function usePostInvoice() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (invoiceId: string) => invoiceService.post(invoiceId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["invoice"] });
      void queryClient.invalidateQueries({ queryKey: ["invoice-list"] });
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
    },
    onError: (error) => {
      handleMutationError({ error, resourceName: "invoice posting" });
    },
  });
}
