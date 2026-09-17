import { agentDefinitionSchema, agentTemplateSchema } from "@/types/assistant";
import { describe, expect, it } from "vitest";

/**
 * Fixtures come from the Go contract, not from what the UI sends. The
 * definition's list columns (`guardrails`, `tool_names`, `event_kinds`,
 * `context_providers`) are `nullzero` arrays and `tool_tiers` is a nullable
 * JSONB map, so an agent saved with none of them is serialized with `null` in
 * each, not `[]`, `{}` or an absent key.
 *
 * The two unset scalars go the opposite ways: `template` is a Go string type,
 * so an agent built without a starter serializes as `""`, while an unset PULID
 * such as `preferredProviderId` serializes as `null`.
 */
const customAgent = {
  id: "agdef_01JAGENT0000000000000000",
  businessUnitId: "bu_01JBUSINESSUNIT000000000",
  organizationId: "org_01JORGANIZATION00000000",
  name: "Help",
  description: "Explains how to do things in Trenova.",
  template: "",
  instructions: "",
  guardrails: null,
  toolNames: null,
  toolTiers: null,
  autonomyCeiling: "Propose",
  enabled: true,
  shadowMode: false,
  decisionTimeoutSeconds: 86400,
  triggerMode: "Chat",
  cronExpression: "",
  cronTimezone: "UTC",
  eventKinds: null,
  intervalSeconds: 0,
  endsAt: null,
  maxConcurrentRuns: 1,
  runTimeoutSeconds: 600,
  maxToolCalls: 12,
  contextProviders: null,
  outputMode: "Conversational",
  preferredProviderId: null,
  systemKey: "",
  lastRunAt: null,
  nextRunAt: null,
  version: 1,
  createdAt: 1_758_000_000,
  updatedAt: 1_758_000_100,
};

describe("agentDefinitionSchema", () => {
  it("reads an agent whose list columns the server wrote as null", () => {
    const parsed = agentDefinitionSchema.parse(customAgent);

    expect(parsed.template).toBeNull();
    expect(parsed.preferredProviderId).toBe("");
    expect(parsed.toolNames).toEqual([]);
    expect(parsed.guardrails).toEqual([]);
    expect(parsed.eventKinds).toEqual([]);
    expect(parsed.contextProviders).toEqual([]);
    expect(parsed.toolTiers).toEqual({});
  });

  it("keeps the tools, tiers and template of an agent that has them", () => {
    const parsed = agentDefinitionSchema.parse({
      ...customAgent,
      template: "DispatchAssistant",
      toolNames: ["get_shipment", "assign_move"],
      toolTiers: { assign_move: "ActWithApproval" },
      triggerMode: "Scheduled",
      cronExpression: "*/30 * * * *",
      nextRunAt: 1_758_001_000,
    });

    expect(parsed.template).toBe("DispatchAssistant");
    expect(parsed.toolNames).toEqual(["get_shipment", "assign_move"]);
    expect(parsed.toolTiers).toEqual({ assign_move: "ActWithApproval" });
    expect(parsed.nextRunAt).toBe(1_758_001_000);
  });

  it("still fills in list fields when a key is absent", () => {
    const { toolNames: _omitted, ...withoutTools } = customAgent;

    expect(agentDefinitionSchema.parse(withoutTools).toolNames).toEqual([]);
  });
});

describe("agentTemplateSchema", () => {
  it("reads a starter whose optional lists the server wrote as null", () => {
    const parsed = agentTemplateSchema.parse({
      template: "GeneralAssistant",
      label: "General assistant",
      description: "Answers questions across the system.",
      starterInstructions: "You explain how Trenova works.",
      starterTools: null,
      starterTrigger: "Chat",
      starterEvents: null,
      starterCron: "",
      starterCeiling: "Propose",
      starterOutput: "Conversational",
      systemKey: "",
      contextProviders: null,
    });

    expect(parsed.starterTools).toEqual([]);
    expect(parsed.starterEvents).toEqual([]);
  });
});
