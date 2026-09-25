import {
  fetchAccountingMappingSummary,
  fetchAccountingSyncStatus,
  searchAccountingReferenceObjects,
  type AccountingReferenceSearch,
} from "@/lib/graphql/accounting-sync";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const accountingSync = createQueryKeys("accountingSync", {
  status: (integrationType: AccountingSystem) => ({
    queryKey: [integrationType],
    queryFn: ({ signal }) => fetchAccountingSyncStatus(integrationType, { signal }),
  }),
  mappingSummary: (integrationType: AccountingSystem) => ({
    queryKey: [integrationType],
    queryFn: ({ signal }) => fetchAccountingMappingSummary(integrationType, { signal }),
  }),
  mappings: (integrationType: AccountingSystem, filterKey: string, pageSize: number) => ({
    queryKey: [integrationType, filterKey, pageSize],
  }),
  referenceSearch: (search: AccountingReferenceSearch) => ({
    queryKey: [search],
    queryFn: ({ signal }) => searchAccountingReferenceObjects(search, { signal }),
  }),
});
