import { queries } from "@/lib/queries";
import { modificationsKey } from "@/lib/queries/agent-preview";
import { RESOURCE_QUERY_KEY_MAP, queryKeyPrefix } from "@trenova/shared/hooks/realtime-patching";
import { describe, expect, it } from "vitest";

/** Whether a realtime root reaches a key: TanStack matches from the start. */
function reaches(resource: string, queryKey: readonly unknown[]): boolean {
  return (RESOURCE_QUERY_KEY_MAP[resource] ?? []).some((root) => {
    const prefix = queryKeyPrefix(root);
    return prefix.every((part, index) => queryKey[index] === part);
  });
}

/*
A proposal decided in another tab, by another approver, or by the agent's own
plan moving on changes what the proposals still waiting would do. The preview a
person is looking at has to be read again when that happens, or the approval it
carries is refused with a digest the server no longer agrees with.
*/
describe("preview realtime invalidation", () => {
  it("reaches every proposal preview from a proposal event", () => {
    expect(reaches("agent_proposal", queries.agentPreview.proposal("mine", "p").queryKey)).toBe(
      true,
    );
    expect(
      reaches(
        "agent_proposal",
        queries.agentPreview.proposal("approver", "p", { status: "Late" }).queryKey,
      ),
    ).toBe(true);
  });

  it("reaches every plan preview from a plan event", () => {
    expect(reaches("agent_plan", queries.agentPreview.plan("approver", "pl").queryKey)).toBe(true);
  });
});

describe("modificationsKey", () => {
  // The same draft typed in a different order is the same preview.
  it("keys a draft the same whatever order its values were set in", () => {
    expect(modificationsKey({ a: 1, b: { d: 2, c: 3 } })).toBe(
      modificationsKey({ b: { c: 3, d: 2 }, a: 1 }),
    );
  });

  // Every surface showing a proposal as proposed shares one cache entry and
  // one digest.
  it("keys an unchanged draft as the proposal as proposed", () => {
    expect(modificationsKey(null)).toBe("");
    expect(modificationsKey({})).toBe("");
    expect(queries.agentPreview.proposal("mine", "p", {}).queryKey).toEqual(
      queries.agentPreview.proposal("mine", "p").queryKey,
    );
  });
});
