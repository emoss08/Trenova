import type { ToolCatalogEntry } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { agentFormDefaults, type AgentFormValues } from "../../agent-form-schema";
import {
  changeTiers,
  dateIn,
  endOfDay,
  checklist,
  nextRun,
  readiness,
  startValues,
  triggerProblem,
  weekStrip,
} from "../builder-model";

function tool(name: string, kind: "query" | "action", core = false): ToolCatalogEntry {
  return {
    name,
    description: "",
    kind,
    resource: "shipment",
    operation: kind === "query" ? "read" : "update",
    defaultAutonomyTier: "",
    reversible: false,
    core,
    prerequisites: [],
    extension: "",
    grantedToEveryAgent: false,
  } as ToolCatalogEntry;
}

const catalog = [
  tool("recall_memory", "query", true),
  tool("search_shipments", "query"),
  tool("schedule_report_email", "action"),
  tool("send_customer_reply", "action"),
  tool("assign_driver", "action"),
];

function values(overrides: Partial<AgentFormValues>): AgentFormValues {
  return { ...agentFormDefaults, ...overrides };
}

describe("startValues", () => {
  it("starts in shadow, asking first, with the trigger and tool its start implies", () => {
    const scheduled = startValues("scheduled", { timezone: "America/Chicago", catalog });
    expect(scheduled.shadowMode).toBe(true);
    expect(scheduled.autonomyCeiling).toBe("ActWithApproval");
    expect(scheduled.triggerMode).toBe("Scheduled");
    expect(scheduled.cronExpression).toBe("0 6 * * 1-5");
    expect(scheduled.cronTimezone).toBe("America/Chicago");
    expect(scheduled.toolNames).toEqual(["schedule_report_email"]);
    expect(scheduled.toolTiers).toEqual({ schedule_report_email: "Propose" });
  });

  it("chooses no tool the catalog does not offer, and no schedule for a chat agent", () => {
    const event = startValues("event", { timezone: "UTC", catalog });
    expect(event.triggerMode).toBe("Event");
    expect(event.toolNames).toEqual([]);
    const chat = startValues("blank", { timezone: "UTC", catalog });
    expect(chat.triggerMode).toBe("Chat");
    expect(chat.cronExpression).toBe("");
    expect(chat.cronTimezone).toBe("");
  });
});

describe("triggerProblem", () => {
  it("names what each trigger is missing", () => {
    expect(triggerProblem(values({ triggerMode: "Event", eventKinds: [] }))).toBe("events");
    expect(triggerProblem(values({ triggerMode: "Scheduled", cronExpression: "  " }))).toBe("schedule");
    expect(triggerProblem(values({ triggerMode: "Scheduled", cronExpression: "0 17 L * *" }))).toBe(
      "badSchedule",
    );
    expect(triggerProblem(values({ triggerMode: "Continuous", intervalSeconds: 30 }))).toBe("interval");
    expect(triggerProblem(values({ triggerMode: "Continuous", intervalSeconds: 60 }))).toBeNull();
    expect(triggerProblem(values({ triggerMode: "Chat" }))).toBeNull();
  });
});

describe("checklist and readiness", () => {
  const base = values({
    name: "Detention desk",
    description: "Starts the detention clock",
    instructions: "You are the detention desk for {{organization}}. Check arrival times first.",
  });

  it("is ready only when every required part is done and the draft was tried as it stands", () => {
    const status = checklist({ values: base, findings: 0, openRisk: 0, chosen: 3, tried: "d1", draft: "d1" });
    expect(Object.values(status).every((value) => value === "ok")).toBe(true);
    expect(readiness(status)).toEqual({ done: 5, of: 5 });
  });

  it("warns of lint findings, outside reach and a draft changed since it was tried", () => {
    const status = checklist({ values: base, findings: 2, openRisk: 1, chosen: 3, tried: "d1", draft: "d2" });
    expect(status.instr).toBe("warn");
    expect(status.tools).toBe("warn");
    expect(status.test).toBe("warn");
    expect(readiness(status)).toEqual({ done: 2, of: 5 });
  });

  it("leaves unwritten parts to do", () => {
    const status = checklist({
      values: values({ name: "X", description: "", instructions: "short" }),
      findings: 0,
      openRisk: 0,
      chosen: 0,
      tried: null,
      draft: "d",
    });
    expect([status.who, status.instr, status.tools, status.test]).toEqual(["todo", "todo", "todo", "todo"]);
  });
});

describe("changeTiers", () => {
  it("counts reads and caps each change at the ceiling, leaving out what every agent holds", () => {
    expect(
      changeTiers(
        values({
          toolNames: ["recall_memory", "search_shipments", "assign_driver", "send_customer_reply"],
          toolTiers: { assign_driver: "AutoExecute", send_customer_reply: "Propose" },
          autonomyCeiling: "ActWithApproval",
        }),
        catalog,
      ),
    ).toEqual({ reads: 1, byTier: { Propose: 1, ActWithApproval: 1, AutoExecute: 0 } });
  });
});

describe("weekStrip and nextRun", () => {
  const wednesdayNoon = Date.UTC(2026, 9, 7, 12, 0) / 1000;

  it("lays weekday mornings across the next seven days, today first", () => {
    const strip = weekStrip("0 6 * * 1-5", "UTC", wednesdayNoon);
    expect(strip.kind).toBe("week");
    if (strip.kind !== "week") return;
    expect(strip.days.map((day) => day.weekday)).toEqual([3, 4, 5, 6, 0, 1, 2]);
    expect(strip.days[0]?.runs).toEqual([{ hour: 6, minute: 0, past: true }]);
    expect(strip.days[3]?.runs).toEqual([]);
    expect(strip.upcoming).toBe(4);
    expect(strip.nowFraction).toBeCloseTo(0.5);
  });

  it("says nothing runs in the week, or that it cannot read the schedule", () => {
    expect(weekStrip("0 8 1 1 *", "UTC", wednesdayNoon).kind).toBe("quiet");
    expect(weekStrip("whenever", "UTC", wednesdayNoon).kind).toBe("unreadable");
  });

  it("finds the next run in the agent's time zone", () => {
    expect(nextRun("0 6 * * 1-5", "America/New_York", wednesdayNoon)).toBe(
      Date.UTC(2026, 9, 8, 10, 0) / 1000,
    );
    expect(nextRun("bad", "UTC", wednesdayNoon)).toBeNull();
  });
});

describe("endOfDay and dateIn", () => {
  it("ends the day in the agent's time zone, and reads it back as the same date", () => {
    const chicago = endOfDay("2026-10-31", "America/Chicago");
    expect(chicago).toBe(Date.UTC(2026, 10, 1, 4, 59, 59) / 1000);
    expect(dateIn(chicago!, "America/Chicago")).toBe("2026-10-31");
    expect(endOfDay("2026-10-31", "UTC")).toBe(Date.UTC(2026, 9, 31, 23, 59, 59) / 1000);
  });

  it("is nothing for a date it cannot read", () => {
    expect(endOfDay("31/10/2026", "UTC")).toBeNull();
  });
});
