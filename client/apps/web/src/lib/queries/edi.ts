import {
  EdiPartnerScorecardsDocument,
  EdiSummaryDocument,
  EdiVolumeSeriesDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { fetchDeskEdiPartner, fetchEdiPartnerMessages } from "@/lib/graphql/edi-partner-card";
import { fetchGraphQLSelectOptions } from "@/lib/graphql/select-options";
import { EDI_TRANSACTION_SET_OPTIONS_LIMIT } from "@/lib/edi-transaction-set-options";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import { infiniteQueryOptions } from "@tanstack/react-query";

export const edi = createQueryKeys("edi", {
  connections: (query = "") => ({
    queryKey: ["connections", query],
    queryFn: async () => apiService.ediService.listConnections(query),
  }),
  mappingProfile: (partnerId: string) => ({
    queryKey: ["mapping-profile", partnerId],
    queryFn: async () => apiService.ediService.getMappingProfile(partnerId),
  }),
  mappingPreview: (transferId: string) => ({
    queryKey: ["mapping-preview", transferId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ediService.getMappingPreview(transferId, { signal }),
  }),
  transfer: (transferId: string) => ({
    queryKey: ["transfer", transferId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ediService.getTransfer(transferId, { signal }),
  }),
  inboundFileTransfers: (fileId: string) => ({
    queryKey: ["inbound-file-transfers", fileId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ediService.listInboundFileTransfers(fileId, { signal }),
  }),
  transactionSetOptions: () => ({
    queryKey: ["transaction-set-options"],
    queryFn: async ({ signal }) =>
      fetchGraphQLSelectOptions(
        {
          resource: "EDI_TRANSACTION_SET",
          initialLimit: EDI_TRANSACTION_SET_OPTIONS_LIMIT,
          filters: { status: "Active" },
        },
        { signal },
      ),
  }),
  inboundFile: (fileId: string) => ({
    queryKey: ["inbound-file", fileId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ediService.getInboundFile(fileId, { signal }),
  }),
  inboundFileInspection: (fileId: string) => ({
    queryKey: ["inbound-file-inspection", fileId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ediService.inspectInboundFile(fileId, { signal }),
  }),
  messages: (query = "") => ({
    queryKey: ["messages", query],
    queryFn: async () => apiService.ediService.listMessages(query),
  }),
  partner: (partnerId: string) => ({
    queryKey: ["partner", partnerId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ediService.getPartner(partnerId, { signal }),
  }),
  partnerCard: (partnerId: string) => ({
    queryKey: ["partner-card", partnerId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchDeskEdiPartner(partnerId, { signal }),
  }),
  partnerMessages: (partnerId: string) => ({
    queryKey: ["partner-messages", partnerId],
  }),
  message: (messageId: string) => ({
    queryKey: ["message", messageId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ediService.getMessage(messageId, { signal }),
  }),
  messageInspection: (messageId: string) => ({
    queryKey: ["message-inspection", messageId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ediService.inspectMessage(messageId, { signal }),
  }),
  summary: (sinceHours?: number) => ({
    queryKey: ["summary", sinceHours],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: EdiSummaryDocument,
        operationName: "EdiSummary",
        variables: { sinceHours },
        signal,
      }),
  }),
  partnerScorecards: (sinceHours?: number) => ({
    queryKey: ["partner-scorecards", sinceHours],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: EdiPartnerScorecardsDocument,
        operationName: "EdiPartnerScorecards",
        variables: { sinceHours },
        signal,
      }),
  }),
  volumeSeries: (sinceHours?: number) => ({
    queryKey: ["volume-series", sinceHours],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: EdiVolumeSeriesDocument,
        operationName: "EdiVolumeSeries",
        variables: { sinceHours },
        signal,
      }),
  }),
});

/** A partner's messages both ways, newest first, a page at a time. */
export function ediPartnerMessagesQuery(partnerId: string, pageSize: number) {
  return infiniteQueryOptions({
    queryKey: [...edi.partnerMessages(partnerId).queryKey, pageSize],
    queryFn: ({ pageParam, signal }) =>
      fetchEdiPartnerMessages(partnerId, { first: pageSize, after: pageParam }, { signal }),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => (last.hasNextPage && last.endCursor ? last.endCursor : undefined),
  });
}
