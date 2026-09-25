import {
  fetchAccountingMappingSummary,
  fetchAccountingSyncStatus,
  searchAccountingReferenceObjects,
  type AccountingReferenceSearch,
} from "@/lib/graphql/accounting-sync";
import {
  fetchAccountingBackfills,
  fetchAccountingSyncAttempts,
  fetchAccountingSyncObjectStates,
  fetchAccountingSyncRecord,
  fetchAccountingSyncSummary,
} from "@/lib/graphql/accounting-sync-ledger";
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
  syncSummary: (integrationType: AccountingSystem) => ({
    queryKey: [integrationType],
    queryFn: ({ signal }) => fetchAccountingSyncSummary(integrationType, { signal }),
  }),
  syncRecord: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }) => fetchAccountingSyncRecord(id, { signal }),
  }),
  syncAttempts: (recordId: string) => ({
    queryKey: [recordId],
    queryFn: ({ signal }) => fetchAccountingSyncAttempts(recordId, { signal }),
  }),
  syncObjectStates: (objectIds: readonly string[]) => ({
    queryKey: [[...objectIds].sort()],
    queryFn: ({ signal }) => fetchAccountingSyncObjectStates(objectIds, { signal }),
  }),
  backfills: (integrationType: AccountingSystem) => ({
    queryKey: [integrationType],
    queryFn: ({ signal }) => fetchAccountingBackfills(integrationType, { signal }),
  }),
  referenceSearch: (search: AccountingReferenceSearch) => ({
    queryKey: [search],
    queryFn: ({ signal }) => searchAccountingReferenceObjects(search, { signal }),
  }),
});
