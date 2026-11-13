import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useMarkSeen } from "../use-mark-seen";

const server = vi.hoisted(() => ({
  markWatchtowerItemSeen: vi.fn<(id: string) => Promise<unknown>>(),
}));
const cache = vi.hoisted(() => ({
  patchQueueItem: vi.fn(),
  patchAttentionItem: vi.fn(),
}));

vi.mock("@/lib/graphql/watchtower", () => ({
  markWatchtowerItemSeen: server.markWatchtowerItemSeen,
}));
vi.mock("@/lib/watchtower-cache", () => ({ patchQueueItem: cache.patchQueueItem }));
vi.mock("@/lib/desk-attention-cache", () => ({ patchAttentionItem: cache.patchAttentionItem }));

type Props = { current: { id: string; seen: boolean } | null; defer: boolean };

function renderSeen(initial: Props) {
  const queryClient = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return renderHook(({ current, defer }: Props) => useMarkSeen(current, defer), {
    initialProps: initial,
    wrapper,
  });
}

/** The id a patch was aimed at and what it sets `seen` to, whatever the item said before. */
function patched(call: unknown[]): [string, boolean] {
  const [, id, patch] = call as [unknown, string, (item: { seen: boolean }) => { seen: boolean }];
  return [id, patch({ seen: false }).seen && patch({ seen: true }).seen];
}

beforeEach(() => {
  server.markWatchtowerItemSeen.mockReset().mockResolvedValue({});
  cache.patchQueueItem.mockReset();
  cache.patchAttentionItem.mockReset();
});

describe("useMarkSeen, what opening an item does to its dot", () => {
  it("marks an unseen item seen as it opens, in both cached lists and on the server", () => {
    renderSeen({ current: { id: "wt_1", seen: false }, defer: false });

    expect(server.markWatchtowerItemSeen.mock.calls).toEqual([["wt_1"]]);
    expect(cache.patchQueueItem.mock.calls.map(patched)).toEqual([["wt_1", true]]);
    expect(cache.patchAttentionItem.mock.calls.map(patched)).toEqual([["wt_1", true]]);
  });

  it("leaves an item that is already seen, and nothing in front, alone", () => {
    const view = renderSeen({ current: { id: "wt_1", seen: true }, defer: false });
    view.rerender({ current: null, defer: false });

    expect(server.markWatchtowerItemSeen).not.toHaveBeenCalled();
    expect(cache.patchAttentionItem).not.toHaveBeenCalled();
  });

  it("puts the dot back when the mark does not reach the server", async () => {
    server.markWatchtowerItemSeen.mockRejectedValue(new Error("offline"));
    renderSeen({ current: { id: "wt_1", seen: false }, defer: false });

    await waitFor(() => expect(cache.patchAttentionItem).toHaveBeenCalledTimes(2));
    expect(cache.patchAttentionItem.mock.calls.map(patched)).toEqual([
      ["wt_1", true],
      ["wt_1", false],
    ]);
    expect(cache.patchQueueItem.mock.calls.map(patched)).toEqual([
      ["wt_1", true],
      ["wt_1", false],
    ]);
  });

  it("holds the mark while a deferred item is in front, however often the list is read again", () => {
    const view = renderSeen({ current: { id: "wt_1", seen: false }, defer: true });
    view.rerender({ current: { id: "wt_1", seen: false }, defer: true });

    expect(server.markWatchtowerItemSeen).not.toHaveBeenCalled();
    expect(cache.patchAttentionItem).not.toHaveBeenCalled();
  });

  it("marks a deferred item once the reader moves to another, and holds the new one", () => {
    const view = renderSeen({ current: { id: "wt_1", seen: false }, defer: true });
    view.rerender({ current: { id: "wt_2", seen: false }, defer: true });

    expect(server.markWatchtowerItemSeen.mock.calls).toEqual([["wt_1"]]);

    view.rerender({ current: null, defer: true });
    expect(server.markWatchtowerItemSeen.mock.calls).toEqual([["wt_1"], ["wt_2"]]);
  });

  it("marks the item in front when the list stops deferring", () => {
    const view = renderSeen({ current: { id: "wt_1", seen: false }, defer: true });
    view.rerender({ current: { id: "wt_1", seen: false }, defer: false });

    expect(server.markWatchtowerItemSeen.mock.calls).toEqual([["wt_1"]]);
  });
});
