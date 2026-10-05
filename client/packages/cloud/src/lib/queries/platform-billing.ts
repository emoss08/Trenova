import { createQueryKeys } from "@lukemorales/query-key-factory";
import { platformBillingService } from "../../services/platform-billing";

export const platformBilling = createQueryKeys("platformBilling", {
  summary: () => ({
    queryKey: ["summary"],
    queryFn: async () => platformBillingService.getSummary(),
  }),
});
