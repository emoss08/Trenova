import { invoiceAdjustmentControlService } from "@/services/invoice-adjustment-control";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const invoiceAdjustmentControl = createQueryKeys("invoiceAdjustmentControl", {
  get: () => ({
    queryKey: ["get"],
    queryFn: async () => invoiceAdjustmentControlService.get(),
  }),
});
