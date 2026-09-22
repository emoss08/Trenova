import { applyDecision, decidedStatus } from "@/lib/proposal-cache";
import type { AssistantProposal } from "@/types/assistant";
import { describe, expect, it } from "vitest";

function pending(id: string): AssistantProposal {
  return {
    id,
    status: "Pending",
    toolName: "create_report",
    executionError: "",
  } as AssistantProposal;
}

/*
Deciding and executing are two steps. The resolve call records the decision and
returns; the tool runs after it. While the card waited for a refetch to tell it
that, it kept its buttons — and a second click in that window reached a
proposal the server had already decided, refusing it with a message that reads
like a bug: "this proposal has already been decided: it is executed".
*/
describe("decidedStatus", () => {
  it("records what the server will record", () => {
    expect(decidedStatus("Accepted")).toBe("Accepted");
    expect(decidedStatus("Modified")).toBe("Modified");
    expect(decidedStatus("Rejected")).toBe("Rejected");
  });

  // Never Executed: whether the write happened is the execution columns' to
  // say, and claiming it here would tell someone their change landed before
  // anything had run.
  it("never claims the tool ran", () => {
    expect(decidedStatus("Accepted")).not.toBe("Executed");
  });
});

describe("applyDecision", () => {
  it("takes the decided proposal out of the awaiting state", () => {
    const [first] = applyDecision([pending("prop_1")], "prop_1", "Accepted");

    expect(first.status).toBe("Accepted");
  });

  it("leaves every other proposal alone", () => {
    const after = applyDecision([pending("prop_1"), pending("prop_2")], "prop_1", "Rejected");

    expect(after[0].status).toBe("Rejected");
    expect(after[1].status).toBe("Pending");
  });

  it("does not mutate the cached list", () => {
    const before = [pending("prop_1")];
    applyDecision(before, "prop_1", "Accepted");

    expect(before[0].status).toBe("Pending");
  });

  it("is a no-op for a proposal it does not hold", () => {
    const after = applyDecision([pending("prop_1")], "prop_other", "Accepted");

    expect(after[0].status).toBe("Pending");
  });
});
