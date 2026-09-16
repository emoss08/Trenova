import { agentDefinitionSchema, agentTemplateSchema } from "@/types/assistant";
import { describe, expect, it } from "vitest";

/**
 * Fixtures come from the Go contract, not from what the UI sends. An agent's
 * `ToolNames` is a `[]string` with `nullzero`, so an agent that only answers
 * (the seeded "Help" general assistant, or any agent saved with no tools) is
 * serialized as `"toolNames": null`, not `[]` and not absent. A tool's
 * `Parameters` is a `map[string]any` and is `null` for a tool without one.
 */
const answersOnlyAgent = {
  id: "agdef_01JAGENT0000000000000000",
  businessUnitId: "bu_01JBUSINESSUNIT000000000",
  organizationId: "org_01JORGANIZATION00000000",
  name: "Help",
  description: "Explains how to do things in Trenova. Cannot read or change records.",
  kind: "GeneralAssistant",
  focus: "",
  toolNames: null,
  autonomyCeiling: "Propose",
  enabled: true,
  version: 1,
  createdAt: 1_758_000_000,
  updatedAt: 1_758_000_100,
};

describe("agentDefinitionSchema", () => {
  it("reads an agent whose toolNames the server wrote as null", () => {
    const parsed = agentDefinitionSchema.parse(answersOnlyAgent);

    expect(parsed.toolNames).toEqual([]);
  });

  it("keeps the tools of an agent that has some", () => {
    const parsed = agentDefinitionSchema.parse({
      ...answersOnlyAgent,
      kind: "DispatchAssistant",
      toolNames: ["get_shipment", "search_shipments"],
    });

    expect(parsed.toolNames).toEqual(["get_shipment", "search_shipments"]);
  });

  it("still fills in toolNames when the key is absent", () => {
    const { toolNames: _omitted, ...withoutTools } = answersOnlyAgent;

    expect(agentDefinitionSchema.parse(withoutTools).toolNames).toEqual([]);
  });
});

describe("agentTemplateSchema", () => {
  it("reads a tool whose parameters the server wrote as null", () => {
    const parsed = agentTemplateSchema.parse({
      kind: "DispatchAssistant",
      label: "Dispatch assistant",
      description: "Looks up shipments and drivers.",
      mutatingAllowed: true,
      availableTools: [
        {
          name: "list_open_holds",
          description: "Lists holds that are still open.",
          parameters: null,
          autonomyTier: "Propose",
        },
      ],
    });

    expect(parsed.availableTools[0].parameters).toBeFalsy();
  });
});
