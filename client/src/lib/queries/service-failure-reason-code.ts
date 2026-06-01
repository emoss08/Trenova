import { apiService } from "@/services/api";
import type { ServiceFailureReasonCodeAppliesTo } from "@/types/service-failure-reason-code";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const serviceFailureReasonCode = createQueryKeys(
  "serviceFailureReasonCode",
  {
    get: (id: string) => ({
      queryKey: ["get", id],
      queryFn: async () => apiService.serviceFailureReasonCodeService.getById(id),
    }),
    selectOptions: (appliesTo?: ServiceFailureReasonCodeAppliesTo) => ({
      queryKey: ["select-options", appliesTo],
      queryFn: async () =>
        apiService.serviceFailureReasonCodeService.selectOptions({
          appliesTo,
          limit: 100,
        }),
    }),
  },
);
