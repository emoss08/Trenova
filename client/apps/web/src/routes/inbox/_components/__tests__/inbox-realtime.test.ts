import { inbox } from "@/lib/queries/inbox";
import { RESOURCE_QUERY_KEY_MAP, queryKeyPrefix } from "@trenova/shared/hooks/realtime-patching";
import { describe, expect, it } from "vitest";

/**
 * The server announces every Triage write as the inbound_message resource. An
 * open inbox only moves if that name reaches the keys the lanes, the counts and
 * the reading pane are cached under; without it a message a desk settled sits
 * in the waiting lane until somebody reloads.
 */
describe("inbound_message invalidation", () => {
  const roots = (RESOURCE_QUERY_KEY_MAP.inbound_message ?? []).map(queryKeyPrefix);

  function reaches(key: readonly unknown[]): boolean {
    return roots.some((root) => root.every((part, index) => key[index] === part));
  }

  it.each([
    [
      "the folders",
      inbox.messages({ lane: "Waiting", sort: "Due", collapseThreads: true }).queryKey,
    ],
    ["the counts", inbox.counts().queryKey],
    ["the reader", inbox.message("imsg_1").queryKey],
    ["the thread", inbox.thread("thread-1").queryKey],
    ["the views' waiting numbers", inbox.views().queryKey],
    ["the briefing", inbox.briefing().queryKey],
  ])("reaches %s", (_, key) => {
    expect(reaches(key)).toBe(true);
  });

  it("leaves mailbox administration and the rules alone", () => {
    expect(roots.length).toBeGreaterThan(0);
    expect(reaches(inbox.mailboxes().queryKey)).toBe(false);
    expect(reaches(inbox.rules().queryKey)).toBe(false);
  });
});
