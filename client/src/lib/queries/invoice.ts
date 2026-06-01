import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const invoice = createQueryKeys("invoice", {
  get: (invoiceId: string) => ({
    queryKey: ["get", invoiceId],
    queryFn: async () => apiService.invoiceService.getById(invoiceId),
  }),
  sendPlan: (invoiceId: string) => ({
    queryKey: ["send-plan", invoiceId],
    queryFn: async () => apiService.invoiceService.getSendPlan(invoiceId),
  }),
  emailAttempts: (invoiceId: string) => ({
    queryKey: ["email-attempts", invoiceId],
    queryFn: async () => apiService.invoiceService.listEmailAttempts(invoiceId),
  }),
});
