import type { InboundMessageDetail, InboundMessageFilter } from "@/lib/graphql/inbox";
import { inbox, type InboxFolderPages } from "@/lib/queries/inbox";
import { createInboundMessagePatcher } from "@/lib/triage/realtime";
import { QueryClient } from "@tanstack/react-query";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { messageRow } from "./message-fixture";

const NOW = 1_003_600;
const waiting: InboundMessageFilter = { lane: "Waiting", sort: "Due", collapseThreads: true };
const handled: InboundMessageFilter = { lane: "Handled", sort: "Newest", collapseThreads: true };

function detail(overrides: Partial<InboundMessageDetail> = {}): InboundMessageDetail {
  return { ...messageRow(), textBody: "", ...overrides } as InboundMessageDetail;
}

function folder(...rows: ReturnType<typeof messageRow>[]): InboxFolderPages {
  return { pages: [{ messages: rows, endCursor: null, hasNextPage: false }], pageParams: [null] };
}

function ids(client: QueryClient, filter: InboundMessageFilter): string[] {
  return (
    client
      .getQueryData<InboxFolderPages>(inbox.messages(filter).queryKey)
      ?.pages.flatMap((page) => page.messages.map((message) => message.id)) ?? []
  );
}

const event = (recordId: string, action = "updated") => ({
  organizationId: "org",
  businessUnitId: "bu",
  resource: "inbound_message",
  action,
  recordId,
});

describe("the inbound_message patcher", () => {
  let client: QueryClient;

  beforeEach(() => {
    vi.useFakeTimers();
    client = new QueryClient();
    client.setQueryData(inbox.messages(waiting).queryKey, folder(messageRow({ id: "a" })));
    client.setQueryData(inbox.messages(handled).queryKey, folder());
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("reads the one message and moves it between folders instead of refetching them", async () => {
    const fetchMessage = vi.fn(async (id: string) =>
      detail({ id, status: "Actioned", needsReview: false, dueAt: null }),
    );
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const patcher = createInboundMessagePatcher({
      queryClient: client,
      meId: "usr_me",
      fetchMessage,
      now: () => NOW,
    });

    patcher.handle(event("a"));
    patcher.handle(event("a"));
    await vi.advanceTimersByTimeAsync(300);

    expect(fetchMessage).toHaveBeenCalledTimes(1);
    expect(ids(client, waiting)).toEqual([]);
    expect(ids(client, handled)).toEqual(["a"]);
    const refetched = invalidate.mock.calls.map(([filters]) => filters?.queryKey);
    expect(refetched).not.toContainEqual(inbox.messages._def);
    expect(refetched).toContainEqual(inbox.counts().queryKey);
  });

  it("brings a new waiting message into the folder and announces it as an arrival", async () => {
    const onArrival = vi.fn();
    const patcher = createInboundMessagePatcher({
      queryClient: client,
      meId: "usr_me",
      onArrival,
      fetchMessage: async (id) =>
        detail({ id, threadKey: id, receivedAt: NOW - 60, dueAt: NOW + 60 }),
      now: () => NOW,
    });

    patcher.handle(event("new", "created"));
    await vi.advanceTimersByTimeAsync(300);

    expect(ids(client, waiting)).toEqual(["new", "a"]);
    expect(onArrival).toHaveBeenCalledWith(expect.objectContaining({ id: "new" }));
  });

  it("does not call a reopened old message an arrival", async () => {
    const onArrival = vi.fn();
    const patcher = createInboundMessagePatcher({
      queryClient: client,
      meId: "usr_me",
      onArrival,
      fetchMessage: async (id) => detail({ id, threadKey: id, receivedAt: NOW - 86_400 }),
      now: () => NOW,
    });

    patcher.handle(event("old"));
    await vi.advanceTimersByTimeAsync(300);

    expect(ids(client, waiting)).toContain("old");
    expect(onArrival).not.toHaveBeenCalled();
  });

  it("takes a message nobody can read any more out of every folder", async () => {
    const patcher = createInboundMessagePatcher({
      queryClient: client,
      meId: "usr_me",
      fetchMessage: async () => {
        throw new GraphQLRequestError({
          kind: "graphql",
          message: "not found",
          graphQLErrors: [
            {
              message: "not found",
              code: "NOT_FOUND",
              type: "https://trenova.app/problems/resource-not-found",
              extensions: {},
            },
          ],
        });
      },
      now: () => NOW,
    });

    patcher.handle(event("a", "deleted"));
    await vi.advanceTimersByTimeAsync(300);

    expect(ids(client, waiting)).toEqual([]);
  });

  it("refreshes the folders on screen for a bulk notice, which names no message", async () => {
    const invalidate = vi.spyOn(client, "invalidateQueries");
    const fetchMessage = vi.fn();
    const patcher = createInboundMessagePatcher({ queryClient: client, meId: null, fetchMessage });

    patcher.handle({
      organizationId: "org",
      businessUnitId: "bu",
      resource: "inbound_message",
      action: "bulk_updated",
    });

    expect(fetchMessage).not.toHaveBeenCalled();
    expect(invalidate).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: inbox.messages._def, refetchType: "active" }),
    );
  });
});
