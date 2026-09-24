import { describe, expect, it } from "vitest";
import {
  EGRESS_ACCENT,
  EGRESS_ORDER,
  filterPolicies,
  heldByLabel,
  readsOutsideLabel,
  reachLabel,
  resourceOptions,
  safetyFigures,
  selectedAgents,
  sortToolsByExposure,
} from "../safety-model";
import { policy, safety, tool } from "./fixtures";

const t = (text: string | null | undefined, ...args: unknown[]) =>
  (text ?? "").replace(/\{(\d+)\}/g, (_m, i) => String(args[Number(i)]));

describe("safetyFigures", () => {
  const policies = [
    policy({ name: "assign_move" }),
    policy({
      name: "email_customer",
      egress: ["ExternalRecipient"],
      leavesOrganization: true,
      promotableTier: "ActWithApproval",
    }),
    policy({ name: "get_shipment", kind: "Query", effect: "Lookup", egress: ["None"] }),
    policy({
      name: "create_report",
      egress: ["Personal", "Internal"],
      hasClassify: true,
      personalExemption: true,
    }),
  ];

  // A read is not something anyone worries an agent did on its own, so only
  // tools that change something count, and each tool counts once however many
  // agents run it.
  it("counts changing tools that run without a person on any agent, once each", () => {
    const figures = safetyFigures(policies, [
      safety("a1", "Dispatch", {
        tools: [
          tool("assign_move", { answer: "RUNS_ON_ITS_OWN" }),
          tool("get_shipment", { answer: "RUNS_ON_ITS_OWN" }),
          tool("email_customer", { answer: "NEEDS_APPROVAL", tier: "ActWithApproval" }),
        ],
      }),
      safety("a2", "Reports", {
        tools: [
          tool("assign_move", { answer: "RUNS_ON_ITS_OWN" }),
          tool("create_report", { answer: "CONDITIONAL" }),
        ],
      }),
    ]);

    expect(figures.runWithoutPerson).toBe(2);
    expect(figures.leaveOrganization).toBe(1);
    expect(figures.openWithSensitive).toBe(0);
  });

  // The clean answer is the one that says what an agent may do; a run that
  // read outside text only ever does less.
  it("reads the clean answer, not the tainted one", () => {
    const figures = safetyFigures(policies, [
      safety("a1", "Desk", {
        tools: [
          tool(
            "assign_move",
            { answer: "PROPOSE_ONLY", tier: "Propose" },
            { answer: "RUNS_ON_ITS_OWN" },
          ),
        ],
      }),
    ]);

    expect(figures.runWithoutPerson).toBe(0);
  });

  it("does not count a simulated write as one made without a person", () => {
    const figures = safetyFigures(policies, [
      safety("a1", "Rehearsal", {
        tools: [tool("assign_move", { answer: "SIMULATED", heldBy: ["simulation_mode"] })],
      }),
    ]);

    expect(figures.runWithoutPerson).toBe(0);
  });

  it("counts open agents holding sensitive tools", () => {
    const figures = safetyFigures(policies, [
      safety("a1", "Open", {
        reach: {
          accessMode: "Everyone",
          roles: [],
          warnings: [{ kind: "OpenWithSensitiveTools", tools: ["email_customer"] }],
        },
      }),
      safety("a2", "Nobody", {
        reach: {
          accessMode: "Roles",
          roles: [],
          warnings: [{ kind: "NoAudience", tools: [] }],
        },
      }),
      safety("a3", "Quiet"),
    ]);

    expect(figures.openWithSensitive).toBe(1);
  });

  it("is all zeros with no agents and no tools", () => {
    expect(safetyFigures([], [])).toEqual({
      runWithoutPerson: 0,
      leaveOrganization: 0,
      openWithSensitive: 0,
    });
  });
});

describe("filterPolicies", () => {
  const policies = [
    policy({ name: "assign_move" }),
    policy({
      name: "add_shipment_comment",
      egress: ["Internal", "CustomerVisible", "DriverVisible"],
      needs: { resource: "shipment", operation: "update" },
    }),
    policy({ name: "add_home_widget", egress: ["Personal"], needs: null }),
  ];

  // A tool whose calls reach several audiences is found under each of them.
  it("finds a tool under every class it can reach", () => {
    expect(
      filterPolicies(policies, { egress: "CustomerVisible", resource: "all" }).map((p) => p.name),
    ).toEqual(["add_shipment_comment"]);
    expect(
      filterPolicies(policies, { egress: "Internal", resource: "all" }).map((p) => p.name),
    ).toEqual(["assign_move", "add_shipment_comment"]);
  });

  it("files a tool that needs no grant under general", () => {
    expect(
      filterPolicies(policies, { egress: "all", resource: "general" }).map((p) => p.name),
    ).toEqual(["add_home_widget"]);
    expect(resourceOptions(policies)).toEqual(["general", "shipment", "shipment_move"]);
  });

  it("combines both filters", () => {
    expect(filterPolicies(policies, { egress: "Personal", resource: "shipment" })).toEqual([]);
    expect(filterPolicies(policies, { egress: "all", resource: "all" })).toHaveLength(3);
  });
});

describe("labels", () => {
  it("gives every class its own accent", () => {
    const accents = EGRESS_ORDER.map((egress) => EGRESS_ACCENT[egress]);
    expect(new Set(accents).size).toBe(EGRESS_ORDER.length);
  });

  it("names every reason the server holds a call for", () => {
    for (const key of [
      "agent_ceiling",
      "tool_max",
      "egress_class",
      "condition",
      "tainted",
      "tool_tier",
      "personal_exemption",
      "shadow_mode",
      "simulation_mode",
    ]) {
      expect(heldByLabel(t, key)).not.toBe(key);
    }
    expect(heldByLabel(t, "a_future_reason")).toBe("a_future_reason");
  });

  it("says where outside text comes from, and nothing for a tool that reads none", () => {
    expect(
      readsOutsideLabel(t, policy({ readsExternal: "Always", source: "InboundMessage" })),
    ).toBe("Always, from inbound messages");
    expect(readsOutsideLabel(t, policy({ readsExternal: "Marked", source: "Memory" }))).toBe(
      "When marked, from memory",
    );
    expect(readsOutsideLabel(t, policy({ readsExternal: "Never" }))).toBeNull();
    expect(readsOutsideLabel(t, policy({ readsExternal: "Never", carriesTaint: true }))).toBe(
      "Carries outside text forward",
    );
  });

  it("says who can reach an agent", () => {
    expect(reachLabel(t, safety("a1", "Open"))).toBe("Everyone who can use the assistant");
    expect(
      reachLabel(
        t,
        safety("a2", "Billing", {
          reach: {
            accessMode: "Roles",
            roles: [
              { id: "rol_1", name: "Billing" },
              { id: "rol_2", name: "Finance" },
            ],
            warnings: [],
          },
        }),
      ),
    ).toBe("Roles: Billing, Finance");
    expect(
      reachLabel(
        t,
        safety("a3", "Nobody", { reach: { accessMode: "Roles", roles: [], warnings: [] } }),
      ),
    ).toBe("Restricted to roles");
  });
});

describe("selectedAgents", () => {
  const agents = [safety("a1", "Alpha"), safety("a2", "Beta"), safety("a3", "Gamma")];

  it("shows the first agent until someone picks", () => {
    expect(selectedAgents(agents, []).map((a) => a.agentId)).toEqual(["a1"]);
    expect(selectedAgents([], [])).toEqual([]);
  });

  it("keeps the agents' own order and drops ids no longer listed", () => {
    expect(selectedAgents(agents, ["a3", "a1", "gone"]).map((a) => a.agentId)).toEqual([
      "a1",
      "a3",
    ]);
  });
});

describe("sortToolsByExposure", () => {
  it("leads with what runs without a person", () => {
    const sorted = sortToolsByExposure([
      tool("b_propose", { answer: "PROPOSE_ONLY" }),
      tool("a_simulated", { answer: "SIMULATED" }),
      tool("z_auto", { answer: "RUNS_ON_ITS_OWN" }),
      tool("c_conditional", { answer: "CONDITIONAL" }),
      tool("a_auto", { answer: "RUNS_ON_ITS_OWN" }),
      tool("d_approval", { answer: "NEEDS_APPROVAL" }),
    ]);

    expect(sorted.map((entry) => entry.policyName)).toEqual([
      "a_auto",
      "z_auto",
      "c_conditional",
      "d_approval",
      "b_propose",
      "a_simulated",
    ]);
  });
});
