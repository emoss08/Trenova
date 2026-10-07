import { describe, expect, it } from "vitest";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { WorkingRun } from "@/lib/graphql/ai-control";
import { workingAgents } from "../agents-at-work";

const agent = (id: string) => ({ id, name: id }) as AgentDefinitionRow;
const run = (id: string, agentDefinitionId: string) =>
  ({ id, agentDefinitionId, status: "Diagnosing", summary: "", startedAt: 1 }) as WorkingRun;

describe("workingAgents", () => {
  it("names each running agent once, with its newest run, and skips runs of unknown agents", () => {
    const result = workingAgents(
      [agent("a"), agent("b")],
      [run("r3", "a"), run("r2", "a"), run("r1", "b"), run("r0", "gone")],
    );

    expect(result.map(({ agent: a, run: r }) => [a.id, r.id])).toEqual([
      ["a", "r3"],
      ["b", "r1"],
    ]);
  });
});
