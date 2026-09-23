import type { ResourceInvalidationEvent } from "@trenova/shared/hooks/realtime-patching";
import { RESOURCE_QUERY_KEY_MAP } from "@trenova/shared/hooks/realtime-patching";
import { describe, expect, it } from "vitest";
import { ASSISTANT_TURNS_RESOURCE, isOtherUsersTurnEvent } from "../assistant-turn-realtime";

/*
Fixtures follow the realtime contract: resource "assistant_turns", action
"started" or "finished", entity { turnId, threadId, status, userId }. The
channel is tenant-wide, and userId is always present.
*/
function turnEvent(
  entity: Record<string, unknown> | undefined,
  overrides: Partial<ResourceInvalidationEvent> = {},
): ResourceInvalidationEvent {
  return {
    organizationId: "org_1",
    businessUnitId: "bu_1",
    resource: ASSISTANT_TURNS_RESOURCE,
    action: "started",
    entity,
    ...overrides,
  };
}

const entity = { turnId: "atrn_1", threadId: "athr_1", status: "Running" };

describe("isOtherUsersTurnEvent", () => {
  it.each(["started", "finished"])("is true for another person's reply %s", (action) => {
    expect(
      isOtherUsersTurnEvent(turnEvent({ ...entity, userId: "usr_2" }, { action }), "usr_1"),
    ).toBe(true);
  });

  it.each(["started", "finished"])("is false for the person's own reply %s", (action) => {
    expect(
      isOtherUsersTurnEvent(turnEvent({ ...entity, userId: "usr_1" }, { action }), "usr_1"),
    ).toBe(false);
  });

  // Every socket in the organization receives every member's turns; one that
  // does not name the person is not taken as theirs.
  it("is true when the event does not name the person", () => {
    expect(isOtherUsersTurnEvent(turnEvent(entity), "usr_1")).toBe(true);
    expect(isOtherUsersTurnEvent(turnEvent({ ...entity, userId: "" }), "usr_1")).toBe(true);
    expect(isOtherUsersTurnEvent(turnEvent(undefined), "usr_1")).toBe(true);
  });

  it("is true before the current user is known", () => {
    expect(isOtherUsersTurnEvent(turnEvent({ ...entity, userId: "usr_1" }), undefined)).toBe(true);
  });

  it("leaves every other resource alone", () => {
    expect(
      isOtherUsersTurnEvent(
        turnEvent({ userId: "usr_2" }, { resource: "shipments", action: "updated" }),
        "usr_1",
      ),
    ).toBe(false);
  });
});

describe("assistant_turns invalidation", () => {
  it("refreshes the live replies and the conversation list together", () => {
    expect(RESOURCE_QUERY_KEY_MAP[ASSISTANT_TURNS_RESOURCE]).toEqual([
      ["assistant", "activeTurns"],
      ["assistant", "threads"],
    ]);
  });
});
