import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AuditEntryDetailDocument,
  AuditLogTableRowFieldsFragmentDoc,
  type AuditLogTableRowFieldsFragment,
  AuditFieldHistoryDocument,
  AuditLogTableDocument,
  AuditRequestEntriesDocument,
  type AuditEntryDetailQuery,
  type AuditFieldHistoryQuery,
  type AuditRequestEntriesQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

type RequestOptions = { signal?: AbortSignal };

export const auditLogTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AuditLogTableDocument,
  operationName: "AuditLogTable",
  connectionKey: "auditEntries",
});

export type AuditEntryRow = DataTableConfigRow<typeof auditLogTableGraphQLConfig>;

type AuditEntryDetailExtras = Omit<
  NonNullable<AuditEntryDetailQuery["auditEntry"]>,
  " $fragmentRefs"
>;

export type AuditEntryDetail = Omit<AuditLogTableRowFieldsFragment, " $fragmentName"> &
  AuditEntryDetailExtras;

export type AuditRequestEntry = AuditRequestEntriesQuery["auditEntries"]["edges"][number]["node"];

export type AuditFieldChange =
  AuditFieldHistoryQuery["auditEntriesByResourceId"]["edges"][number]["node"];

const REQUEST_ENTRY_LIMIT = 50;
const FIELD_HISTORY_LIMIT = 25;

/** One entry with its diff, states and metadata, or null when the reader cannot see it. */
export async function fetchAuditEntryDetail(
  id: string,
  options?: RequestOptions,
): Promise<AuditEntryDetail | null> {
  const data = await requestGraphQL({
    document: AuditEntryDetailDocument,
    operationName: "AuditEntryDetail",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.auditEntry) {
    return null;
  }
  const { " $fragmentRefs": _refs, ...extras } = data.auditEntry;
  return {
    ...getFragmentData(AuditLogTableRowFieldsFragmentDoc, data.auditEntry),
    ...extras,
  };
}

/** Every entry written while serving one request, oldest first. */
export async function fetchAuditRequestEntries(
  correlationId: string,
  options?: RequestOptions,
): Promise<AuditRequestEntry[]> {
  const data = await requestGraphQL({
    document: AuditRequestEntriesDocument,
    operationName: "AuditRequestEntries",
    variables: {
      input: {
        first: REQUEST_ENTRY_LIMIT,
        fieldFilters: [{ field: "correlationId", operator: "eq", value: correlationId }],
        sort: [{ field: "timestamp", direction: "asc" }],
      },
    },
    signal: options?.signal,
  });
  return data.auditEntries.edges.map((edge) => edge.node);
}

/** The entries of one record that changed one field, newest first. */
export async function fetchAuditFieldHistory(
  resourceId: string,
  field: string,
  options?: RequestOptions,
): Promise<AuditFieldChange[]> {
  const data = await requestGraphQL({
    document: AuditFieldHistoryDocument,
    operationName: "AuditFieldHistory",
    variables: {
      resourceId,
      field,
      input: {
        first: FIELD_HISTORY_LIMIT,
        sort: [{ field: "timestamp", direction: "desc" }],
      },
    },
    signal: options?.signal,
  });
  return data.auditEntriesByResourceId.edges.map((edge) => edge.node);
}
