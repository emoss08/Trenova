import type { AgentDraft } from "@/lib/graphql/agent-builder";
import { describe, expect, it } from "vitest";
import { draftedValues } from "../use-agent-drafting";

function draft(patch: Partial<AgentDraft> = {}): AgentDraft {
  return {
    name: "Detention chaser",
    description: "Chases detention",
    icon: "bot",
    accent: "indigo",
    instructions: "Chase detention when a truck waits.",
    guardrails: [],
    triggerMode: "Event",
    cronExpression: "",
    cronTimezone: "",
    eventKinds: ["shipment.stop.arrived"],
    intervalSeconds: 0,
    toolNames: [],
    toolTiers: {},
    autonomyCeiling: "ActWithApproval",
    dataAccessCeiling: "Internal",
    outputMode: "Conversational",
    enabled: true,
    shadowMode: true,
    decisionTimeoutSeconds: 86_400,
    runTimeoutSeconds: 600,
    maxToolCalls: 10,
    maxConcurrentRuns: 1,
    notes: [],
    ...patch,
  } as AgentDraft;
}

describe("draftedValues", () => {
  // The drafter marks every draft shadow; the mode is the person's to pick in the
  // builder, so a draft must not decide it.
  it("leaves the mode to the builder whatever the draft says", () => {
    expect(draftedValues(draft({ shadowMode: true }))).not.toHaveProperty("shadowMode");
    expect(draftedValues(draft({ shadowMode: false }))).not.toHaveProperty("shadowMode");
    expect(draftedValues(draft())).not.toHaveProperty("simulationMode");
  });

  it("keeps what the draft chose for the agent itself", () => {
    const values = draftedValues(draft({ toolTiers: { flag_for_manual_review: "Propose" } }));
    expect(values.name).toBe("Detention chaser");
    expect(values.triggerMode).toBe("Event");
    expect(values.eventKinds).toEqual(["shipment.stop.arrived"]);
    expect(values.toolTiers).toEqual({ flag_for_manual_review: "Propose" });
  });
});
