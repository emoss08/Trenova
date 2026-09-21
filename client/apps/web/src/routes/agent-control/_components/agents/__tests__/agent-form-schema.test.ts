import { describe, expect, it } from "vitest";
import { agentFormDefaults, agentFormSchema, toSaveRequest } from "../agent-form-schema";

function values(overrides: Record<string, unknown>) {
  return { ...agentFormDefaults, name: "Night desk", ...overrides };
}

function issuesOf(input: Record<string, unknown>): Record<string, string> {
  const result = agentFormSchema.safeParse(input);
  if (result.success) return {};
  return Object.fromEntries(
    result.error.issues.map((issue) => [issue.path.join("."), issue.message]),
  );
}

/**
 * The server's Validate (domain/agentdefinition/definition.go) rejects a
 * schedule without a cron, an event agent without events, a continuous agent
 * below a minute, and tool tiers for tools the agent does not have. The form
 * says so before the request leaves, at the field that is wrong.
 */
describe("agentFormSchema", () => {
  it("accepts a chat agent with only a name", () => {
    expect(issuesOf(values({}))).toEqual({});
  });

  // The server refuses a daily limit on a tool the agent does not hold; the
  // form says so, and drops a zero or a leftover limit before sending.
  it("keeps daily tool limits to the agent's own tools and drops empty ones", () => {
    expect(
      issuesOf(values({ toolNames: ["assign_move"], toolDailyLimits: { cancel_shipment: 3 } })),
    ).toHaveProperty("toolDailyLimits");

    const sent = toSaveRequest({
      ...agentFormDefaults,
      name: "Night desk",
      toolNames: ["assign_move"],
      toolDailyLimits: { assign_move: 4, cancel_shipment: 2, remember: 0 },
      monthlyBudgetUsd: 25,
    });
    expect(sent.toolDailyLimits).toEqual({ assign_move: 4 });
    expect(sent.monthlyBudgetUsd).toBe(25);
  });

  it("requires a cron expression for a scheduled agent", () => {
    expect(issuesOf(values({ triggerMode: "Scheduled", cronExpression: "" }))).toHaveProperty(
      "cronExpression",
    );
    expect(issuesOf(values({ triggerMode: "Scheduled", cronExpression: "*/30 * * * *" }))).toEqual(
      {},
    );
  });

  it("requires at least one event for an event-driven agent", () => {
    expect(issuesOf(values({ triggerMode: "Event", eventKinds: [] }))).toHaveProperty("eventKinds");
  });

  it("requires an interval of at least a minute for a continuous agent", () => {
    expect(issuesOf(values({ triggerMode: "Continuous", intervalSeconds: 30 }))).toHaveProperty(
      "intervalSeconds",
    );
    expect(issuesOf(values({ triggerMode: "Continuous", intervalSeconds: 60 }))).toEqual({});
  });

  it("refuses a tool tier above the ceiling or for a tool the agent lacks", () => {
    expect(
      issuesOf(
        values({
          autonomyCeiling: "Propose",
          toolNames: ["assign_move"],
          toolTiers: { assign_move: "AutoExecute" },
        }),
      ),
    ).toHaveProperty("toolTiers");
    expect(
      issuesOf(values({ toolNames: ["get_shipment"], toolTiers: { assign_move: "Propose" } })),
    ).toHaveProperty("toolTiers");
  });
});

describe("toSaveRequest", () => {
  it("clears the trigger fields that do not belong to the chosen mode", () => {
    const request = toSaveRequest(
      values({
        triggerMode: "Event",
        eventKinds: ["shipment.created"],
        cronExpression: "0 6 * * 1-5",
        cronTimezone: "America/Chicago",
        intervalSeconds: 300,
        endsAt: 1_800_000_000,
      }),
    );

    expect(request.eventKinds).toEqual(["shipment.created"]);
    expect(request.cronExpression).toBe("");
    expect(request.intervalSeconds).toBe(0);
    expect(request.endsAt).toBeNull();
  });

  it("drops tool tiers for tools that are no longer selected", () => {
    const request = toSaveRequest(
      values({
        toolNames: ["get_shipment"],
        toolTiers: { get_shipment: "Propose", assign_move: "ActWithApproval" },
      }),
    );

    expect(request.toolTiers).toEqual({ get_shipment: "Propose" });
  });
});
