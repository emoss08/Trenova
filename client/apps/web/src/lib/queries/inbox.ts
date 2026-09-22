import {
  fetchInboundMailboxes,
  fetchInboundMessage,
  fetchInboundMessageCounts,
  fetchInboundMessages,
  type InboundMessageFilter,
  type InboundMessagePage,
} from "@/lib/graphql/inbox";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const inbox = createQueryKeys("inbox", {
  counts: () => ({
    queryKey: ["counts"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchInboundMessageCounts({ signal }),
  }),
  // The folder and the search are part of the key: what is waiting and what
  // was handled are different questions, not the same one twice.
  messages: (filter: Omit<InboundMessageFilter, "after">) => ({
    queryKey: [filter],
  }),
  message: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchInboundMessage(id, { signal }),
  }),
  mailboxes: () => ({
    queryKey: ["mailboxes"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchInboundMailboxes({ signal }),
  }),
});

/**
 * A folder of mail, paged by arrival. Each page's end cursor carries the sort,
 * so asking for the next page continues where the last one stopped rather
 * than starting over.
 */
export function inboxMessagesQuery(filter: Omit<InboundMessageFilter, "after">) {
  return {
    queryKey: inbox.messages(filter).queryKey,
    queryFn: ({ pageParam, signal }: { pageParam?: unknown; signal?: AbortSignal }) =>
      fetchInboundMessages(
        { ...filter, after: typeof pageParam === "string" ? pageParam : null },
        { signal },
      ),
    initialPageParam: null as string | null,
    getNextPageParam: (last: InboundMessagePage): string | null =>
      last.hasNextPage ? last.endCursor : null,
  };
}
