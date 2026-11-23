import { getFragmentData } from "@trenova/graphql/fragment-data";
import type {
  DeskConversationFilterInput,
  DeskConversationOrigin,
  DeskConversationRowFieldsFragment,
  DeskTranscriptExportDownloadMutation,
  DeskTranscriptExportFieldsFragment,
  DeskTranscriptExportKind,
  DeskTranscriptExportPreviewQuery,
  DeskTranscriptExportStatus,
  DeskTranscriptTimeframePreset,
  RequestDeskTranscriptExportInput,
} from "@trenova/graphql/generated/graphql";
import {
  DeskConversationRowFieldsFragmentDoc,
  DeskConversationTableDocument,
  DeskTranscriptExportDownloadDocument,
  DeskTranscriptExportFieldsFragmentDoc,
  DeskTranscriptExportPreviewDocument,
  DeskTranscriptExportTableDocument,
  OpenDeskConversationDocument,
  RequestDeskTranscriptExportDocument,
} from "@trenova/graphql/generated/documents/agent/desk-transcripts";
import { AgentRunTranscriptFieldsFragmentDoc } from "@trenova/graphql/generated/documents/agent/runs";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";
import type { AgentRunTranscript } from "@/lib/graphql/agent-activity-tables";

export type {
  DeskConversationFilterInput,
  DeskConversationOrigin,
  DeskTranscriptExportKind,
  DeskTranscriptExportStatus,
  DeskTranscriptTimeframePreset,
  RequestDeskTranscriptExportInput,
};

type RequestOptions = { signal?: AbortSignal };

export const DESK_CONVERSATION_LIST_KEY = "desk-conversation-list";
export const DESK_TRANSCRIPT_EXPORT_LIST_KEY = "desk-transcript-export-list";

const deskConversationTableBase = defineDataTableGraphQLConfig({
  document: DeskConversationTableDocument,
  operationName: "DeskConversationTable",
  connectionKey: "deskConversations",
});

export type DeskConversationRow = DataTableConfigRow<typeof deskConversationTableBase>;

/** Everyone's conversations narrowed to the filter beside the table, newest last message first. */
export function deskConversationTableGraphQLConfig(filter: DeskConversationFilterInput) {
  return { ...deskConversationTableBase, extraVariables: { filter } };
}

export const deskTranscriptExportTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: DeskTranscriptExportTableDocument,
  operationName: "DeskTranscriptExportTable",
  connectionKey: "deskTranscriptExports",
});

export type DeskTranscriptExportRow = DataTableConfigRow<
  typeof deskTranscriptExportTableGraphQLConfig
>;

export type DeskConversation = Omit<DeskConversationRowFieldsFragment, " $fragmentName">;
export type DeskTranscriptExport = Omit<DeskTranscriptExportFieldsFragment, " $fragmentName">;
export type DeskTranscriptExportPreview =
  DeskTranscriptExportPreviewQuery["deskTranscriptExportPreview"];
export type DeskTranscriptDownload =
  DeskTranscriptExportDownloadMutation["deskTranscriptExportDownload"];

export type OpenedDeskConversation = {
  conversation: DeskConversation;
  transcript: AgentRunTranscript | null;
};

/**
 * Opens another person's conversation read-only. The server records every
 * opening in the audit log, so this is a mutation that is never retried.
 */
export async function openDeskConversation(id: string): Promise<OpenedDeskConversation> {
  const data = await requestGraphQL({
    document: OpenDeskConversationDocument,
    operationName: "OpenDeskConversation",
    variables: { id },
  });
  const opened = data.openDeskConversation;

  return {
    conversation: getFragmentData(DeskConversationRowFieldsFragmentDoc, opened.conversation),
    transcript: getFragmentData(AgentRunTranscriptFieldsFragmentDoc, opened.transcript) ?? null,
  };
}

/** How many conversations an export of the filter would hold, counted up to one past the limit. */
export async function fetchDeskTranscriptExportPreview(
  filter: DeskConversationFilterInput,
  options?: RequestOptions,
): Promise<DeskTranscriptExportPreview> {
  const data = await requestGraphQL({
    document: DeskTranscriptExportPreviewDocument,
    operationName: "DeskTranscriptExportPreview",
    variables: { filter },
    signal: options?.signal,
  });

  return data.deskTranscriptExportPreview;
}

export async function requestDeskTranscriptExport(
  input: RequestDeskTranscriptExportInput,
): Promise<DeskTranscriptExport> {
  const data = await requestGraphQL({
    document: RequestDeskTranscriptExportDocument,
    operationName: "RequestDeskTranscriptExport",
    variables: { input },
  });

  return getFragmentData(DeskTranscriptExportFieldsFragmentDoc, data.requestDeskTranscriptExport);
}

export async function fetchDeskTranscriptExportDownload(
  id: string,
): Promise<DeskTranscriptDownload> {
  const data = await requestGraphQL({
    document: DeskTranscriptExportDownloadDocument,
    operationName: "DeskTranscriptExportDownload",
    variables: { id },
  });

  return data.deskTranscriptExportDownload;
}
