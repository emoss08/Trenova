import { ApiRequestError } from "@trenova/shared/lib/api";
import { describe, expect, it } from "vitest";
import { turnLimitOf } from "../follow-turn";

function refusal(params: Record<string, string> | undefined) {
  return new ApiRequestError(422, {
    type: "business-logic-error",
    title: "Business Logic Error",
    status: 422,
    detail: "Refused",
    params,
  });
}

describe("turnLimitOf", () => {
  it("reads an agent's monthly budget refusal", () => {
    const limit = turnLimitOf(
      refusal({
        code: "agent_budget",
        cap: "monthly_budget",
        spent: "48.20",
        limit: "50.00",
        resetsAt: "1793577600",
      }),
    );

    expect(limit).toEqual({
      kind: "monthly_budget",
      used: "48.20",
      limit: "50.00",
      resetsAt: 1793577600,
    });
  });

  it("reads a person's allowance refusal", () => {
    const limit = turnLimitOf(
      refusal({ code: "person_allowance", used: "250", limit: "250", resetsAt: "1793577600" }),
    );

    expect(limit?.kind).toBe("person_allowance");
    expect(limit?.used).toBe("250");
  });

  it("keeps a refusal without a reset time readable", () => {
    const limit = turnLimitOf(
      refusal({
        code: "agent_budget",
        cap: "daily_runs",
        spent: "40",
        limit: "40",
        resetsAt: "soon",
      }),
    );

    expect(limit?.resetsAt).toBe(0);
  });

  it("is null for a tool cap, which never stops a whole question", () => {
    expect(
      turnLimitOf(
        refusal({ code: "agent_budget", cap: "tool_daily_limit", spent: "5", limit: "5" }),
      ),
    ).toBeNull();
  });

  it("is null for any other failure", () => {
    expect(turnLimitOf(refusal(undefined))).toBeNull();
    expect(turnLimitOf(new Error("network"))).toBeNull();
  });
});
