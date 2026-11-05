import { customerStatementService } from "@/services/customer-statement";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const customerStatement = createQueryKeys("customerStatement", {
  sent: (customerId: string) => ({
    queryKey: ["sent", customerId],
    queryFn: async () => customerStatementService.listSent(customerId),
  }),
});
