import {
  fetchInboundMailboxes,
  fetchInboundMessage,
  fetchInboundMessageCounts,
  fetchInboundMessages,
  type InboundMessageFilter,
} from "@/lib/graphql/inbox";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const inbox = createQueryKeys("inbox", {
  counts: () => ({
    queryKey: ["counts"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchInboundMessageCounts({ signal }),
  }),
  // The lane is part of the key: asking for what is waiting and asking for
  // what was handled are different questions, not the same one twice.
  messages: (filter: InboundMessageFilter) => ({
    queryKey: [filter],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchInboundMessages(filter, { signal }),
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
