import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AiAuditChainStatusDocument,
  AiAuditChainStatusFieldsFragmentDoc,
  AiAuditEventDetailDocument,
  AiAuditEventDetailFieldsFragmentDoc,
  AiAuditEventRowFieldsFragmentDoc,
  AiAuditEventTableDocument,
  AiAuditExportDownloadDocument,
  AiAuditExportFieldsFragmentDoc,
  AiAuditExportTableDocument,
  AuditLogTableRowFieldsFragmentDoc,
  RequestAiAuditExportDocument,
  VerifyAiAuditChainDocument,
  type AiAuditChainStatusFieldsFragment,
  type AiAuditEventDetailFieldsFragment,
  type AiAuditEventKind,
  type AiAuditEventOutcome,
  type AiAuditEventRowFieldsFragment,
  type AiAuditExportDownloadMutation,
  type AiAuditExportFieldsFragment,
  type AiAuditExportFormat,
  type AiAuditExportStatus,
  type AiAuditVerificationStatus,
  type AuditLogTableRowFieldsFragment,
  type RequestAiAuditExportInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export type {
  AiAuditEventKind,
  AiAuditEventOutcome,
  AiAuditExportFormat,
  AiAuditExportStatus,
  AiAuditVerificationStatus,
  RequestAiAuditExportInput,
};

type RequestOptions = { signal?: AbortSignal };

export const AI_AUDIT_EVENT_LIST_KEY = "ai-audit-event-list";
export const AI_AUDIT_EXPORT_LIST_KEY = "ai-audit-export-list";

/** The trail, newest first, paged, searched, filtered and sorted on the server. */
export const aiAuditEventTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AiAuditEventTableDocument,
  operationName: "AIAuditEventTable",
  connectionKey: "aiAuditEvents",
});

export type AIAuditEventRow = DataTableConfigRow<typeof aiAuditEventTableGraphQLConfig>;

/** Every export of the trail the tenant has asked for, newest first. */
export const aiAuditExportTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AiAuditExportTableDocument,
  operationName: "AIAuditExportTable",
  connectionKey: "aiAuditExports",
});

export type AIAuditExportRow = DataTableConfigRow<typeof aiAuditExportTableGraphQLConfig>;

export type AIAuditChainStatus = AiAuditChainStatusFieldsFragment;
export type AIAuditExport = AiAuditExportFieldsFragment;
export type AIAuditLinkedEntry = AuditLogTableRowFieldsFragment;

/** One event with everything the trail recorded about it, its linked audit entries unmasked. */
export type AIAuditEventDetail = Omit<AiAuditEventRowFieldsFragment, " $fragmentName"> &
  Omit<AiAuditEventDetailFieldsFragment, " $fragmentRefs" | " $fragmentName" | "auditEntries"> & {
    auditEntries: AIAuditLinkedEntry[];
  };

export type AIAuditDownload = AiAuditExportDownloadMutation["aiAuditExportDownload"];

/** One event of the trail, or null when it is not the reader's tenant's or does not exist. */
export async function fetchAIAuditEvent(
  id: string,
  options?: RequestOptions,
): Promise<AIAuditEventDetail | null> {
  const data = await requestGraphQL({
    document: AiAuditEventDetailDocument,
    operationName: "AIAuditEventDetail",
    variables: { id },
    signal: options?.signal,
  });
  const detail = getFragmentData(AiAuditEventDetailFieldsFragmentDoc, data.aiAuditEvent);
  if (!detail) {
    return null;
  }
  const row = getFragmentData(AiAuditEventRowFieldsFragmentDoc, detail);

  return {
    ...detail,
    ...row,
    auditEntries: [...getFragmentData(AuditLogTableRowFieldsFragmentDoc, detail.auditEntries)],
  };
}

/** Where the reader's tenant's chain runs, how far it is sealed and what its last check found. */
export async function fetchAIAuditChainStatus(
  options?: RequestOptions,
): Promise<AIAuditChainStatus> {
  const data = await requestGraphQL({
    document: AiAuditChainStatusDocument,
    operationName: "AIAuditChainStatus",
    signal: options?.signal,
  });

  return getFragmentData(AiAuditChainStatusFieldsFragmentDoc, data.aiAuditChainStatus);
}

/** Starts a check of the chain now; the status comes back with `verifying` set. */
export async function verifyAIAuditChain(): Promise<AIAuditChainStatus> {
  const data = await requestGraphQL({
    document: VerifyAiAuditChainDocument,
    operationName: "VerifyAIAuditChain",
  });

  return getFragmentData(AiAuditChainStatusFieldsFragmentDoc, data.verifyAIAuditChain);
}

/**
 * Records an export of the trail. A small one comes back already written
 * (`Succeeded`); a large one comes back `Running`, and its requester is told
 * when the file is ready.
 */
export async function requestAIAuditExport(
  input: RequestAiAuditExportInput,
): Promise<AIAuditExport> {
  const data = await requestGraphQL({
    document: RequestAiAuditExportDocument,
    operationName: "RequestAIAuditExport",
    variables: { input },
  });

  return getFragmentData(AiAuditExportFieldsFragmentDoc, data.requestAIAuditExport);
}

/** A link to an export's file, good for a minute; only its requester is given one. */
export async function fetchAIAuditExportDownload(id: string): Promise<AIAuditDownload> {
  const data = await requestGraphQL({
    document: AiAuditExportDownloadDocument,
    operationName: "AIAuditExportDownload",
    variables: { id },
  });

  return data.aiAuditExportDownload;
}
