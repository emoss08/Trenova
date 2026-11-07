import { createQueryKeys } from "@lukemorales/query-key-factory";
import { fetchAuditAlertPolicies, fetchAuditChainStatus } from "@/lib/graphql/audit-log-admin";
import {
  fetchAuditEntryDetail,
  fetchAuditFieldHistory,
  fetchAuditRequestEntries,
} from "@/lib/graphql/audit-log-table";

export const audit = createQueryKeys("audit", {
  history: (resourceId: string) => ({
    queryKey: ["audit-history", resourceId],
  }),
  entry: (id: string) => ({
    queryKey: ["entry", id],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAuditEntryDetail(id, { signal }),
  }),
  requestEntries: (correlationId: string) => ({
    queryKey: ["request", correlationId],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchAuditRequestEntries(correlationId, { signal }),
  }),
  fieldHistory: (resourceId: string, field: string) => ({
    queryKey: ["field-history", resourceId, field],
    queryFn: ({ signal }: { signal?: AbortSignal }) =>
      fetchAuditFieldHistory(resourceId, field, { signal }),
  }),
  chainStatus: () => ({
    queryKey: ["chain-status"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAuditChainStatus({ signal }),
  }),
  alertPolicies: () => ({
    queryKey: ["alert-policies"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchAuditAlertPolicies({ signal }),
  }),
});
