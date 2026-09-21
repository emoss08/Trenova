import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { describe, expect, it } from "vitest";
import { composerBlock } from "../thread-guard";

const agent = { id: "agdef_1", name: "Report Builder" } as AgentDefinitionRow;

describe("composerBlock", () => {
  it("opens the composer when the agent is known and the thread has room", () => {
    expect(composerBlock({ agent, agentsUnavailable: false, threadFull: false })).toBeNull();
  });

  it("closes a full thread whatever else is true", () => {
    expect(composerBlock({ agent, agentsUnavailable: true, threadFull: true })).toBe("full");
  });

  // An agent missing from a list that loaded was disabled or is no longer a
  // chat agent. An agent missing because the list never loaded is unknown,
  // and saying it was disabled would be a false report.
  it("tells a failed agent list apart from a disabled agent", () => {
    expect(composerBlock({ agent: null, agentsUnavailable: false, threadFull: false })).toBe(
      "agent-missing",
    );
    expect(composerBlock({ agent: null, agentsUnavailable: true, threadFull: false })).toBe(
      "agents-unavailable",
    );
  });
});
