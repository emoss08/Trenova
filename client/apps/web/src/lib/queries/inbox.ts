import {
  fetchInboundMailboxes,
  fetchInboundMessage,
  fetchInboundMessageCounts,
  fetchInboundMessages,
  fetchInboundSenderProfile,
  fetchInboundThread,
  fetchMyTriageViews,
  fetchTriageAssignees,
  fetchTriageBriefing,
  fetchTriageCustomerPeek,
  fetchTriageReplyTemplates,
  fetchTriageResponseTargets,
  fetchTriageRuleMatches,
  fetchTriageRulePreview,
  fetchTriageRules,
  fetchTriageSenderDomains,
  fetchTriageSettings,
  fetchTriageShipmentPeek,
  fetchTriageStopTimes,
  renderTriageReplyTemplate,
  type InboundMessageFilter,
  type InboundMessagePage,
  type TriageRuleInput,
} from "@/lib/graphql/inbox";
import { fetchGraphQLSelectOptions } from "@/lib/graphql/select-options";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import type { InfiniteData } from "@tanstack/react-query";

type QueryContext = { signal?: AbortSignal };

/** How long a reference list (assignees, templates, document types) is fresh. */
const REFERENCE_STALE_MS = 5 * 60_000;

/**
 * Every Triage read, under one scope so a write can reach all of it.
 *
 * Keys are normalized: a folder page is keyed by its whole filter (lane, kind,
 * mailbox, search, sort, thread collapse) and nothing else, so two components
 * asking for the same folder share one cache entry, and a message is keyed by
 * its id alone, so the list row, the reader and a prefetch on hover agree.
 */
export const inbox = createQueryKeys("inbox", {
  counts: () => ({
    queryKey: ["counts"],
    queryFn: ({ signal }: QueryContext) => fetchInboundMessageCounts({ signal }),
  }),
  messages: (filter: InboundMessageFilter) => ({
    queryKey: [filter],
  }),
  message: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }: QueryContext) => fetchInboundMessage(id, { signal }),
  }),
  thread: (threadKey: string) => ({
    queryKey: [threadKey],
    queryFn: ({ signal }: QueryContext) => fetchInboundThread(threadKey, { signal }),
  }),
  mailboxes: () => ({
    queryKey: ["mailboxes"],
    queryFn: ({ signal }: QueryContext) => fetchInboundMailboxes({ signal }),
  }),
  views: () => ({
    queryKey: ["views"],
    queryFn: ({ signal }: QueryContext) => fetchMyTriageViews({ signal }),
  }),
  briefing: () => ({
    queryKey: ["briefing"],
    queryFn: ({ signal }: QueryContext) => fetchTriageBriefing({ signal }),
  }),
  rules: () => ({
    queryKey: ["rules"],
    queryFn: ({ signal }: QueryContext) => fetchTriageRules({ signal }),
  }),
  rulePreview: (input: TriageRuleInput) => ({
    queryKey: [input],
    queryFn: ({ signal }: QueryContext) => fetchTriageRulePreview(input, { signal }),
  }),
  ruleMatches: (ruleId: string) => ({
    queryKey: [ruleId],
    queryFn: ({ signal }: QueryContext) => fetchTriageRuleMatches(ruleId, { signal }),
  }),
  senderDomains: (query: string) => ({
    queryKey: [query],
    queryFn: ({ signal }: QueryContext) => fetchTriageSenderDomains(query, { signal }),
  }),
  templates: () => ({
    queryKey: ["templates"],
    queryFn: ({ signal }: QueryContext) => fetchTriageReplyTemplates({ signal }),
  }),
  renderedTemplate: (templateId: string, messageId: string) => ({
    queryKey: [templateId, messageId],
    queryFn: ({ signal }: QueryContext) =>
      renderTriageReplyTemplate(templateId, messageId, { signal }),
  }),
  assignees: () => ({
    queryKey: ["assignees"],
    queryFn: ({ signal }: QueryContext) => fetchTriageAssignees({ signal }),
  }),
  settings: () => ({
    queryKey: ["settings"],
    queryFn: ({ signal }: QueryContext) => fetchTriageSettings({ signal }),
  }),
  targets: () => ({
    queryKey: ["targets"],
    queryFn: ({ signal }: QueryContext) => fetchTriageResponseTargets({ signal }),
  }),
  senderProfile: (messageId: string) => ({
    queryKey: [messageId],
    queryFn: ({ signal }: QueryContext) => fetchInboundSenderProfile(messageId, { signal }),
  }),
  shipmentPeek: (shipmentId: string) => ({
    queryKey: [shipmentId],
    queryFn: ({ signal }: QueryContext) => fetchTriageShipmentPeek(shipmentId, { signal }),
  }),
  customerPeek: (customerId: string) => ({
    queryKey: [customerId],
    queryFn: ({ signal }: QueryContext) => fetchTriageCustomerPeek(customerId, { signal }),
  }),
  stopTimes: (messageId: string) => ({
    queryKey: [messageId],
    queryFn: ({ signal }: QueryContext) => fetchTriageStopTimes(messageId, { signal }),
  }),
  documentTypes: () => ({
    queryKey: ["documentTypes"],
    queryFn: ({ signal }: QueryContext) =>
      fetchGraphQLSelectOptions({ resource: "DOCUMENT_TYPE", initialLimit: 100 }, { signal }),
  }),
});

/** A folder's cached pages, as `useInfiniteQuery` holds them. */
export type InboxFolderPages = InfiniteData<InboundMessagePage, string | null>;

/**
 * A folder of the queue, paged by the server's cursor. Each page's end cursor
 * carries the sort, so the next page continues where the last one stopped.
 */
export function inboxMessagesQuery(filter: InboundMessageFilter) {
  return {
    queryKey: inbox.messages(filter).queryKey,
    queryFn: ({ pageParam, signal }: { pageParam?: unknown; signal?: AbortSignal }) =>
      fetchInboundMessages(filter, typeof pageParam === "string" ? pageParam : null, { signal }),
    initialPageParam: null as string | null,
    getNextPageParam: (last: InboundMessagePage): string | null =>
      last.hasNextPage ? last.endCursor : null,
  };
}

/** Reference lists change rarely; reading them once per few minutes is enough. */
export function inboxReferenceQuery<T extends object>(query: T): T & { staleTime: number } {
  return { ...query, staleTime: REFERENCE_STALE_MS };
}
