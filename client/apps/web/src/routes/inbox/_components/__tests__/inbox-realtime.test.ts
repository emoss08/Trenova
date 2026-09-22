import { inbox } from "@/lib/queries/inbox";
import { RESOURCE_QUERY_KEY_MAP, queryKeyPrefix } from "@trenova/shared/hooks/realtime-patching";
import { describe, expect, it } from "vitest";

/**
 * The server announces every inbox write as the inbound_message resource. An
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
    ["the lanes", inbox.messages({ statuses: ["InReview", "Quarantined"] }).queryKey],
    ["the counts", inbox.counts().queryKey],
    ["the reading pane", inbox.message("imsg_1").queryKey],
  ])("reaches %s", (_, key) => {
    expect(reaches(key)).toBe(true);
  });

  it("leaves mailbox administration alone", () => {
    expect(roots.length).toBeGreaterThan(0);
    expect(reaches(inbox.mailboxes().queryKey)).toBe(false);
  });
});
