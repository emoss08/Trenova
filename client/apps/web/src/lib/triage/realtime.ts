import { fetchInboundMessage, type InboundMessageDetail } from "@/lib/graphql/inbox";
import { inbox } from "@/lib/queries/inbox";
import { applyRows, forgetRows, knownRow, type MembershipContext } from "@/lib/triage/cache";
import { isWaiting } from "@/lib/triage/message-state";
import type { QueryClient } from "@tanstack/react-query";
import {
  isBulkAction,
  resolveEntityID,
  type ResourceInvalidationEvent,
} from "@trenova/shared/hooks/realtime-patching";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";

export const INBOUND_MESSAGE_RESOURCE = "inbound_message";

/** Notices for one message inside this window are read once. */
const COALESCE_MS = 250;
/** A message this recent that appears in a waiting folder is a new arrival, not a reopened one. */
const ARRIVAL_WINDOW_SECONDS = 10 * 60;

export function isInboundMessageEvent(event: ResourceInvalidationEvent): boolean {
  return event.resource === INBOUND_MESSAGE_RESOURCE;
}

export type InboundMessagePatcherOptions = {
  queryClient: QueryClient;
  /** The signed-in person, for "Assigned to me". */
  meId: string | null;
  /** Told about a message that just arrived and now waits in a folder on screen. */
  onArrival?: (message: InboundMessageDetail) => void;
  fetchMessage?: (id: string) => Promise<InboundMessageDetail>;
  now?: () => number;
  coalesceMs?: number;
};

export type InboundMessagePatcher = {
  handle: (event: ResourceInvalidationEvent) => void;
  dispose: () => void;
};

function gone(error: unknown): boolean {
  return (
    error instanceof GraphQLRequestError &&
    (error.isNotFoundError() || error.isAuthorizationError())
  );
}

/**
 * Applies the server's notice that a message changed by reading that one
 * message and placing it in every cached folder, instead of refetching every
 * folder: it stays where it still belongs, leaves the folders it no longer
 * matches and joins the ones it now does. A message that can no longer be
 * read leaves every folder. The counts, the saved views' waiting numbers, the
 * briefing and the thread are small reads and refresh where they are shown.
 * A bulk notice names no message, so it refreshes the folders on screen.
 */
export function createInboundMessagePatcher({
  queryClient,
  meId,
  onArrival,
  fetchMessage = (id) => fetchInboundMessage(id),
  now = () => Math.floor(Date.now() / 1000),
  coalesceMs = COALESCE_MS,
}: InboundMessagePatcherOptions): InboundMessagePatcher {
  const pending = new Map<string, ReturnType<typeof setTimeout>>();
  let disposed = false;

  const refreshSummaries = (threadKey: string | null) => {
    void queryClient.invalidateQueries({ queryKey: inbox.counts().queryKey, refetchType: "active" });
    void queryClient.invalidateQueries({ queryKey: inbox.views().queryKey, refetchType: "active" });
    void queryClient.invalidateQueries({
      queryKey: inbox.briefing().queryKey,
      refetchType: "active",
    });
    if (threadKey !== null) {
      void queryClient.invalidateQueries({
        queryKey: inbox.thread(threadKey).queryKey,
        refetchType: "active",
      });
    }
  };

  const read = async (id: string) => {
    pending.delete(id);
    const context: MembershipContext = { meId, now: now() };
    const before = knownRow(queryClient, id);
    let message: InboundMessageDetail;
    try {
      message = await fetchMessage(id);
    } catch (error) {
      if (disposed) {
        return;
      }
      if (gone(error)) {
        if (before !== undefined) {
          forgetRows(queryClient, [before], context);
        } else {
          queryClient.removeQueries({ queryKey: inbox.message(id).queryKey, exact: true });
        }
        refreshSummaries(before?.threadKey ?? null);
        return;
      }
      void queryClient.invalidateQueries({
        queryKey: inbox.messages._def,
        refetchType: "active",
      });
      refreshSummaries(before?.threadKey ?? null);
      return;
    }
    if (disposed) {
      return;
    }

    const applied = applyRows(queryClient, [message], new Map([[id, before]]), context);
    if (queryClient.getQueryData(inbox.message(id).queryKey) !== undefined) {
      queryClient.setQueryData(inbox.message(id).queryKey, message);
    }
    refreshSummaries(message.threadKey);

    if (
      before === undefined &&
      applied.inserted.has(id) &&
      isWaiting(message) &&
      context.now - message.receivedAt <= ARRIVAL_WINDOW_SECONDS
    ) {
      onArrival?.(message);
    }
  };

  return {
    handle: (event) => {
      if (disposed) {
        return;
      }
      const id = resolveEntityID(event);
      if (id === "" || isBulkAction(event.action ?? "")) {
        void queryClient.invalidateQueries({
          queryKey: inbox.messages._def,
          refetchType: "active",
        });
        refreshSummaries(null);
        return;
      }
      const waiting = pending.get(id);
      if (waiting !== undefined) {
        clearTimeout(waiting);
      }
      pending.set(
        id,
        setTimeout(() => void read(id), coalesceMs),
      );
    },
    dispose: () => {
      disposed = true;
      for (const timer of pending.values()) {
        clearTimeout(timer);
      }
      pending.clear();
    },
  };
}
