import { fetchAccountingSyncStatus } from "@/lib/graphql/accounting-sync";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const accountingSync = createQueryKeys("accountingSync", {
  status: (integrationType: AccountingSystem) => ({
    queryKey: [integrationType],
    queryFn: ({ signal }) => fetchAccountingSyncStatus(integrationType, { signal }),
  }),
});
