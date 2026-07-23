import { apiService } from "@/services/api";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

export function usePostInvoice() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (invoiceId: string) => apiService.invoiceService.post(invoiceId),
    onSuccess: (updated) => {
      void queryClient.invalidateQueries({ queryKey: ["invoice"] });
      void queryClient.invalidateQueries({ queryKey: ["invoice-list"] });
      void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
      void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
      toast.success(`${updated.number} posted`);
    },
    onError: (error) => {
      handleMutationError({ error, resourceName: "invoice posting" });
    },
  });
}
