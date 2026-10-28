import { assistant } from "@/lib/queries/assistant";
import {
  invalidationFor,
  queryKeyPrefix,
  type QueryKeyRoot,
} from "@trenova/shared/hooks/realtime-patching";
import { describe, expect, it } from "vitest";

function reaches(roots: readonly QueryKeyRoot[]) {
  const prefixes = roots.map(queryKeyPrefix);

  return (key: readonly unknown[]) =>
    prefixes.some((prefix) => prefix.every((part, index) => key[index] === part));
}

function scheduleEvent(action: string, threadId?: string) {
  return {
    organizationId: "org_1",
    businessUnitId: "bu_1",
    resource: "conversation_schedules",
    action,
    recordId: "csch_1",
    entity: threadId ? { id: "csch_1", threadId, userId: "usr_1" } : undefined,
  };
}

/*
 * A schedule an agent proposed is kept when the person approves it, and its
 * card is a message in the conversation no message box put on screen. The
 * event that announces it refetches that conversation's messages where it is
 * open, and nothing else's.
 */
describe("conversation_schedules invalidation", () => {
  it("a schedule made refetches its conversation's messages on screen", () => {
    const { roots, activeOnly, activeRoots } = invalidationFor(scheduleEvent("created", "athr_1"));

    expect(activeOnly).toBe(true);
    const lists = reaches(roots);
    expect(lists(assistant.schedules("athr_1").queryKey)).toBe(true);
    expect(lists(assistant.allSchedules().queryKey)).toBe(true);

    const messages = reaches(activeRoots);
    expect(messages(assistant.messages("athr_1").queryKey)).toBe(true);
    expect(messages(assistant.messages("athr_2").queryKey)).toBe(false);
  });

  it("a pause, edit or run moves only the schedule lists", () => {
    for (const action of ["updated", "ran", "deleted"]) {
      const { roots, activeRoots } = invalidationFor(scheduleEvent(action, "athr_1"));

      expect(reaches(roots)(assistant.schedules("athr_1").queryKey)).toBe(true);
      expect(activeRoots).toEqual([]);
    }
  });

  it("a schedule made with no conversation named refetches no messages", () => {
    expect(invalidationFor(scheduleEvent("created")).activeRoots).toEqual([]);
  });
});
