import { createQueryKeys } from "@lukemorales/query-key-factory";

export const audit = createQueryKeys("audit", {
  history: (resourceId: string) => ({
    queryKey: ["audit-history", resourceId],
  }),
});
