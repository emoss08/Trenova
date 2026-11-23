import type { WorkerActivityEntry } from "@/lib/graphql/worker-activity";
import { describe, expect, it } from "vitest";
import {
  activityDays,
  activityFieldFilters,
  activityPeople,
  activitySource,
  activityTarget,
  drawableTarget,
} from "../activity-model";

function entry(overrides: Partial<WorkerActivityEntry>): WorkerActivityEntry {
  return {
    id: "ae",
    timestamp: 1_795_000_000,
    resource: "worker",
    resourceId: "wrk_1",
    operation: "update",
    entityRef: null,
    comment: null,
    channel: "web",
    principalType: "user",
    principalId: "usr_1",
    actorName: null,
    changes: null,
    subjectResource: "worker",
    subjectId: "wrk_1",
    user: { id: "usr_1", name: "Dana Whitfield" },
    ...overrides,
  } as WorkerActivityEntry;
}

describe("activityFieldFilters", () => {
  it("narrows by the reader, by Dash, by agents or by one person, and by the kinds of record", () => {
    expect(activityFieldFilters("all", "all", "usr_me")).toEqual([]);
    expect(activityFieldFilters("me", "all", "usr_me")).toEqual([
      { field: "principalId", operator: "eq", value: "usr_me" },
    ]);
    expect(activityFieldFilters("dash", "all", null)).toEqual([
      { field: "channel", operator: "eq", value: "portal" },
    ]);
    expect(activityFieldFilters("agents", "all", null)).toEqual([
      { field: "channel", operator: "in", value: ["agent_conversation", "agent_run"] },
    ]);
    expect(activityFieldFilters("user:usr_9", "safety", null)).toEqual([
      { field: "principalId", operator: "eq", value: "usr_9" },
      {
        field: "resource",
        operator: "in",
        value: [
          "worker_safety_event",
          "worker_disciplinary_action",
          "worker_recognition",
          "worker_injury",
        ],
      },
    ]);
  });

  it("asks nothing of the reader filter when nobody is signed in", () => {
    expect(activityFieldFilters("me", "all", null)).toEqual([]);
  });
});

describe("activitySource", () => {
  it("names where an entry came from by its channel", () => {
    expect(activitySource("web")).toBe("web");
    expect(activitySource("portal")).toBe("dash");
    expect(activitySource("agent_run")).toBe("agent");
    expect(activitySource("agent_conversation")).toBe("agent");
    expect(activitySource("api")).toBe("integration");
    expect(activitySource("job")).toBe("system");
    expect(activitySource("support")).toBe("support");
  });
});

describe("activityTarget", () => {
  it("opens a sub-record on this page by its record param", () => {
    expect(
      activityTarget(entry({ resource: "worker_safety_event", resourceId: "wsev_1" })),
    ).toEqual({ tab: "safety", safetyEvent: "wsev_1" });
    expect(activityTarget(entry({ resource: "worker_credential", resourceId: "wcr_1" }))).toEqual({
      tab: "qualification",
      view: "credentials",
      credential: "wcr_1",
    });
    expect(
      activityTarget(entry({ resource: "profile_change_request", resourceId: "pcr_1" })),
    ).toEqual({ tab: "files", view: "driver-portal", request: "pcr_1" });
  });

  it("opens only the tab for a record that was deleted", () => {
    expect(
      activityTarget(
        entry({ resource: "worker_safety_event", resourceId: "wsev_1", operation: "delete" }),
      ),
    ).toEqual({ tab: "safety", safetyEvent: undefined });
  });

  it("has nowhere to go for a kind the page does not draw", () => {
    expect(activityTarget(entry({ resource: "shipment" }))).toBeNull();
  });

  it("links only to areas the reader may see", () => {
    const safety = entry({ resource: "worker_injury", resourceId: "inj_1" });
    expect(drawableTarget(safety, () => false)).toBeNull();
    expect(drawableTarget(safety, (area) => area.tab === "safety")).toEqual({
      tab: "safety",
      injury: "inj_1",
    });
  });
});

describe("activityDays", () => {
  it("cuts the trail into the local days its entries fall on, in order", () => {
    const days = activityDays(
      [
        entry({ id: "a", timestamp: Date.UTC(2026, 10, 20, 20, 41) / 1000 }),
        entry({ id: "b", timestamp: Date.UTC(2026, 10, 20, 15, 0) / 1000 }),
        entry({ id: "c", timestamp: Date.UTC(2026, 10, 14, 1, 5) / 1000 }),
      ],
      "America/Chicago",
    );
    expect(days.map((day) => [day.key, day.entries.map((item) => item.id)])).toEqual([
      ["2026-11-20", ["a", "b"]],
      ["2026-11-13", ["c"]],
    ]);
  });
});

describe("activityPeople", () => {
  it("lists the people seen once each, by name, leaving out agents", () => {
    expect(
      activityPeople([
        entry({ principalId: "usr_2", user: { id: "usr_2", name: "Marcus Bell" } }),
        entry({ principalId: "usr_1" }),
        entry({ principalId: "usr_2", user: { id: "usr_2", name: "Marcus Bell" } }),
        entry({ principalId: "agt_1", principalType: "agent", actorName: "Load planner agent" }),
        entry({ principalId: "usr_3", channel: "agent_run", actorName: "Driver check-in agent" }),
      ]),
    ).toEqual([
      { id: "usr_1", name: "Dana Whitfield" },
      { id: "usr_2", name: "Marcus Bell" },
    ]);
  });
});
