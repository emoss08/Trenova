import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { MEMORY_TOKEN_BUDGET } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  accessOf,
  agentFormDefaults,
  agentFormSchema,
  toAgentPanelRow,
  toSaveRequest,
} from "../agent-form-schema";

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

  // agentdefinition.validateMemoryBudget: 1000 to 16000 tokens, or nothing
  // for the default of 6000.
  it("bounds the memory budget the way the server does", () => {
    expect(issuesOf(values({ memoryTokenBudget: null }))).toEqual({});
    expect(issuesOf(values({ memoryTokenBudget: MEMORY_TOKEN_BUDGET.min }))).toEqual({});
    expect(issuesOf(values({ memoryTokenBudget: MEMORY_TOKEN_BUDGET.max }))).toEqual({});
    expect(issuesOf(values({ memoryTokenBudget: 999 }))).toEqual({
      memoryTokenBudget: "Memory in the prompt must be at least 1000 tokens",
    });
    expect(issuesOf(values({ memoryTokenBudget: 16001 }))).toEqual({
      memoryTokenBudget: "Memory in the prompt can be at most 16000 tokens",
    });
    expect(issuesOf(values({ memoryTokenBudget: 1500.5 }))).toHaveProperty("memoryTokenBudget");
    expect(MEMORY_TOKEN_BUDGET.default).toBe(6000);
  });

  it("sends the memory budget as set and none when it was left empty", () => {
    expect(
      toSaveRequest({ ...agentFormDefaults, name: "Rates", memoryTokenBudget: 9000 })
        .memoryTokenBudget,
    ).toBe(9000);
    expect(toSaveRequest({ ...agentFormDefaults, name: "Rates" }).memoryTokenBudget).toBeNull();
    expect(toAgentPanelRow({ ...agentRow(), memoryTokenBudget: 12000 }).memoryTokenBudget).toBe(
      12000,
    );
    expect(toAgentPanelRow(agentRow()).memoryTokenBudget).toBeNull();
  });

  // The server refuses a daily limit on a tool the agent does not hold. A
  // tool taken off the agent leaves its limit behind with no box to clear it
  // from, so the form does not refuse it; the save drops it, with any zero.
  it("keeps daily tool limits to the agent's own tools and drops empty ones", () => {
    expect(
      issuesOf(values({ toolNames: ["assign_move"], toolDailyLimits: { cancel_shipment: 3 } })),
    ).toEqual({});

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

  // A limit box the person never filled holds nothing; saving must not
  // fail on it, and it is not a limit.
  it("accepts an empty daily limit box and does not send it", () => {
    expect(
      issuesOf(
        values({
          toolNames: ["assign_move"],
          toolDailyLimits: { assign_move: undefined, get_insight: null },
        }),
      ),
    ).toEqual({});

    const sent = toSaveRequest({
      ...agentFormDefaults,
      name: "Home builder",
      toolNames: ["assign_move", "get_insight"],
      toolDailyLimits: { assign_move: undefined, get_insight: null },
    });
    expect(sent.toolDailyLimits).toEqual({});
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

  it("refuses a tool tier above the ceiling, but not one left by a removed tool", () => {
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
      issuesOf(
        values({
          autonomyCeiling: "Propose",
          toolNames: ["get_shipment"],
          toolTiers: { assign_move: "AutoExecute" },
        }),
      ),
    ).toEqual({});
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

/**
 * The allowlist (domain/agentdefinition Definition.Validate and the save
 * handler): at most eight, no agent twice, only on an agent people talk to.
 * The body's `delegateIds` replaces the list, so the form always sends the
 * whole of it, and an absent list would keep a stale one.
 */
describe("delegateIds", () => {
  const eight = Array.from({ length: 8 }, (_, index) => `agdef_${index}`);

  it("accepts up to eight agents and refuses a ninth at the list", () => {
    expect(issuesOf(values({ delegateIds: eight }))).toEqual({});
    expect(issuesOf(values({ delegateIds: [...eight, "agdef_8"] }))).toHaveProperty("delegateIds");
  });

  it("points at the agent listed twice rather than at the list", () => {
    expect(issuesOf(values({ delegateIds: ["agdef_a", "agdef_b", "agdef_a"] }))).toEqual({
      "delegateIds.2": "This agent is already on the list",
    });
  });

  it("reads an absent list as empty", () => {
    const { delegateIds: _absent, ...rest } = values({});
    const parsed = agentFormSchema.safeParse(rest);

    expect(parsed.success && parsed.data.delegateIds).toEqual([]);
  });

  it("sends the whole list for a chat agent, once each, in order", () => {
    const request = toSaveRequest(values({ delegateIds: ["agdef_b", "agdef_a", "agdef_b"] }));

    expect(request.delegateIds).toEqual(["agdef_b", "agdef_a"]);
  });

  it("sends an empty list for an agent that does not run from chat", () => {
    const request = toSaveRequest(
      values({
        triggerMode: "Scheduled",
        cronExpression: "0 6 * * *",
        delegateIds: ["agdef_a"],
      }),
    );

    expect(request.delegateIds).toEqual([]);
  });

  // The roster's enable switch saves a row built from the agent itself; the
  // names drawn beside the list must not ride along in the body.
  it("saves a row's allowlist as ids only", () => {
    const row = toAgentPanelRow({
      ...agentRow(),
      delegateIds: ["agdef_b", "agdef_gone", "agdef_a"],
      delegates: [
        {
          id: "agdef_a",
          name: "Dispatch desk",
          icon: "",
          accent: "",
          enabled: true,
          triggerMode: "Chat",
        },
        {
          id: "agdef_b",
          name: "Report Builder",
          icon: "",
          accent: "",
          enabled: false,
          triggerMode: "Chat",
        },
      ],
    });

    expect(row.delegateIds).toEqual(["agdef_b", "agdef_a"]);
    expect(row.delegates.map((delegate) => [delegate.name, delegate.enabled])).toEqual([
      ["Report Builder", false],
      ["Dispatch desk", true],
    ]);

    const request = toSaveRequest(row);
    expect(request.delegateIds).toEqual(["agdef_b", "agdef_a"]);
    expect(request).not.toHaveProperty("delegates");
  });

  // Who may use an agent rides with the save only when the caller hands it
  // over. A save that is not given it (a toggle, a prompt preview, an edit
  // that left access alone) never carries it, so it cannot change it.
  it("loads who may use the agent into the form and saves it only when handed it", () => {
    const row = toAgentPanelRow({
      ...agentRow(),
      accessMode: "Roles",
      accessRoles: [
        { id: "role_dispatch", name: "Dispatch" },
        { id: "role_billing", name: "Billing" },
      ],
    });

    expect(row.accessMode).toBe("Roles");
    expect(row.accessRoleIds).toEqual(["role_dispatch", "role_billing"]);
    expect(row.accessRoles.map((role) => role.name)).toEqual(["Dispatch", "Billing"]);
    expect(accessOf(row)).toEqual({ mode: "Roles", roleIds: ["role_dispatch", "role_billing"] });

    const request = toSaveRequest(row);
    expect(request).not.toHaveProperty("accessMode");
    expect(request).not.toHaveProperty("accessRoleIds");
    expect(request).not.toHaveProperty("accessRoles");

    const withAccess = toSaveRequest(row, {
      mode: "Everyone",
      roleIds: ["role_dispatch", "role_dispatch"],
    });
    expect(withAccess.accessMode).toBe("Everyone");
    expect(withAccess.accessRoleIds).toEqual(["role_dispatch"]);
    expect(withAccess).not.toHaveProperty("accessRoles");
  });

  it("reads a repeated role once", () => {
    expect(
      accessOf({ accessMode: "Roles", accessRoleIds: ["role_a", "role_b", "role_a"] }),
    ).toEqual({ mode: "Roles", roleIds: ["role_a", "role_b"] });
  });

  it("starts a new agent open to everyone, with no roles", () => {
    expect(accessOf(agentFormDefaults)).toEqual({ mode: "Everyone", roleIds: [] });
  });
});

describe("data access", () => {
  it("starts a new agent at Internal", () => {
    expect(agentFormDefaults.dataAccessCeiling).toBe("Internal");
  });

  it("loads the saved setting and sends it back", () => {
    const row = toAgentPanelRow(agentRow());

    expect(row.dataAccessCeiling).toBe("Restricted");
    expect(toSaveRequest(row).dataAccessCeiling).toBe("Restricted");
  });

  it("refuses a setting the server does not know", () => {
    const parsed = agentFormSchema.safeParse({
      ...agentFormDefaults,
      name: "Payroll desk",
      dataAccessCeiling: "Confidential",
    });

    expect(parsed.success).toBe(false);
  });
});

function agentRow(): AgentDefinitionRow {
  return {
    id: "agdef_widgets",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    name: "Homepage Widget Builder",
    description: "",
    template: null,
    icon: "",
    accent: "",
    instructions: "",
    guardrails: [],
    toolNames: [],
    toolTiers: {},
    autonomyCeiling: "Propose",
    dataAccessCeiling: "Restricted",
    enabled: true,
    shadowMode: false,
    decisionTimeoutSeconds: 86400,
    triggerMode: "Chat",
    cronExpression: "",
    cronTimezone: "",
    eventKinds: [],
    intervalSeconds: 0,
    endsAt: null,
    maxConcurrentRuns: 1,
    runTimeoutSeconds: 600,
    maxToolCalls: 12,
    monthlyBudgetUsd: null,
    dailyRunLimit: 0,
    toolDailyLimits: {},
    simulationMode: false,
    memoryTokenBudget: null,
    contextProviders: [],
    outputMode: "Conversational",
    preferredProviderId: "",
    systemKey: "",
    delegateIds: [],
    delegates: [],
    accessMode: "Everyone",
    accessRoles: [],
    starters: [],
    lastRunAt: null,
    nextRunAt: null,
    pendingProposals: 0,
    openRuns: 0,
    version: 3,
    createdAt: 1_758_000_000,
    updatedAt: 1_758_000_100,
  };
}
