import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const customerStatement = createQueryKeys("customerStatement", {
  sent: (customerId: string) => ({
    queryKey: ["sent", customerId],
    queryFn: async () => apiService.customerStatementService.listSent(customerId),
  }),
});
