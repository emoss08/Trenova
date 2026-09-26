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
import {
  fetchAccountingInboundApplyPreview,
  fetchAccountingInboundChange,
  fetchAccountingInboundOverview,
} from "@/lib/graphql/accounting-inbound";
import {
  fetchAccountingDriftFinding,
  fetchAccountingDriftFixPreview,
  fetchAccountingDriftOverview,
} from "@/lib/graphql/accounting-drift";
import type {
  AccountingDriftDirection,
  AccountingSystem,
} from "@trenova/graphql/generated/graphql";
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
  inboundOverview: (integrationType: AccountingSystem) => ({
    queryKey: [integrationType],
    queryFn: ({ signal }) => fetchAccountingInboundOverview(integrationType, { signal }),
  }),
  inboundChange: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }) => fetchAccountingInboundChange(id, { signal }),
  }),
  inboundPreview: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }) => fetchAccountingInboundApplyPreview(id, { signal }),
  }),
  driftOverview: (integrationType: AccountingSystem) => ({
    queryKey: [integrationType],
    queryFn: ({ signal }) => fetchAccountingDriftOverview(integrationType, { signal }),
  }),
  driftFinding: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }) => fetchAccountingDriftFinding(id, { signal }),
  }),
  driftPreview: (id: string, direction: AccountingDriftDirection) => ({
    queryKey: [id, direction],
    queryFn: ({ signal }) => fetchAccountingDriftFixPreview({ id, direction }, { signal }),
  }),
  referenceSearch: (search: AccountingReferenceSearch) => ({
    queryKey: [search],
    queryFn: ({ signal }) => searchAccountingReferenceObjects(search, { signal }),
  }),
});
