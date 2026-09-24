import { describe, expect, it } from "vitest";
import {
  EGRESS_ACCENT,
  EGRESS_ORDER,
  KIND_ORDER,
  answerChoices,
  egressChoices,
  externalReadChoices,
  heldByChoices,
  heldByLabel,
  heldByOf,
  kindChoices,
  kindLabel,
  readsOutsideLabel,
  reachLabel,
  resourceChoices,
  sortResources,
  tierChoices,
} from "../safety-model";
import { policy, safety, tool } from "./fixtures";

const t = (text: string | null | undefined, ...args: unknown[]) =>
  (text ?? "").replace(/\{(\d+)\}/g, (_m, i) => String(args[Number(i)]));

describe("sortResources", () => {
  // The server names resources by key; a person reads them by label, and a
  // tool that needs no grant is filed under general.
  it("orders resources by the label a person reads", () => {
    expect(sortResources(["shipment_move", "general", "customer"])).toEqual([
      "customer",
      "general",
      "shipment_move",
    ]);
    expect(sortResources([])).toEqual([]);
  });

  it("does not reorder the list it was given", () => {
    const resources = ["shipment", "customer"];
    sortResources(resources);
    expect(resources).toEqual(["shipment", "customer"]);
  });
});

describe("labels", () => {
  it("names every kind of tool", () => {
    expect(KIND_ORDER.map((kind) => kindLabel(t, kind))).toEqual(["Reads", "Changes", "Runtime"]);
  });

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

describe("heldByOf", () => {
  // A reason that holds the call both before and after outside text is one
  // chip, not two; the order is the order the server gave.
  it("names each reason once, clean first", () => {
    expect(
      heldByOf(
        tool(
          "email_customer",
          { heldBy: ["egress_class"] },
          { heldBy: ["egress_class", "tainted"] },
        ),
      ),
    ).toEqual(["egress_class", "tainted"]);
    expect(heldByOf(tool("assign_move", { heldBy: [] }))).toEqual([]);
  });
});

describe("filter choices", () => {
  // The values are the enum names the server's filter reads; the labels are
  // the words the cells show, so a filter chip reads like the column.
  it("offers every class, tier, kind, answer and reason by the server's value", () => {
    expect(egressChoices(t).map((choice) => choice.value)).toEqual([...EGRESS_ORDER]);
    expect(egressChoices(t)[3]).toEqual({ value: "CustomerVisible", label: "Customer" });
    expect(tierChoices(t).map((choice) => choice.value)).toEqual([
      "Propose",
      "ActWithApproval",
      "AutoExecute",
    ]);
    expect(kindChoices(t).map((choice) => choice.label)).toEqual(["Reads", "Changes", "Runtime"]);
    expect(answerChoices(t).map((choice) => choice.value)).toEqual([
      "RUNS_ON_ITS_OWN",
      "CONDITIONAL",
      "NEEDS_APPROVAL",
      "PROPOSE_ONLY",
      "SIMULATED",
    ]);
    expect(heldByChoices(t).every((choice) => choice.label !== choice.value)).toBe(true);
    expect(externalReadChoices(t)).toEqual([
      { value: "Never", label: "Never" },
      { value: "Marked", label: "When marked" },
      { value: "Always", label: "Always" },
    ]);
  });

  it("lists resources by the label a person reads", () => {
    expect(resourceChoices(["shipment_move", "general", "customer"]).map((c) => c.value)).toEqual([
      "customer",
      "general",
      "shipment_move",
    ]);
  });
});
