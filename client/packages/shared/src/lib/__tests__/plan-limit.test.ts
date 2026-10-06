import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError, api, clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { GraphQLRequestError, requestGraphQL } from "@trenova/shared/lib/graphql";
import {
  isExplainedPlanLimitError,
  planLimitFromError,
  planLimitFromGraphQLErrors,
  planLimitFromProblem,
  reportPlanLimit,
  setPlanLimitHandler,
  type PlanLimitNotice,
} from "@trenova/shared/lib/plan-limit";

const PROBLEM_BASE = "https://trenova.app/problems/";

function jsonResponse(body: unknown, status: number): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

// Fixtures mirror the server's plan-limit errors: REST problem params are a
// map[string]string (helpers.ProblemDetail.Params), so limit and used arrive as strings;
// GraphQL extensions carry code, type and params, where numbers may arrive as numbers.
const quotaProblem = {
  type: `${PROBLEM_BASE}quota-exceeded`,
  title: "Quota Exceeded",
  status: 402,
  detail: "The free demo allows 12 shipments.",
  code: "QUOTA_EXCEEDED",
  params: { meter: "shipments.total", limit: "12", used: "12", plan: "free_demo" },
};

const readOnlyProblem = {
  type: `${PROBLEM_BASE}plan-restricted`,
  title: "Plan Restricted",
  status: 403,
  detail: "This organization is read-only.",
  code: "PLAN_RESTRICTED",
  params: { capability: "", reason: "subscription_read_only", plan: "free_demo" },
};

describe("planLimitFromProblem", () => {
  it("reads a quota refusal and coerces its string params to numbers", () => {
    expect(planLimitFromProblem(402, quotaProblem)).toEqual({
      kind: "quota",
      meter: "shipments.total",
      limit: 12,
      used: 12,
      plan: "free_demo",
      message: "The free demo allows 12 shipments.",
    });
  });

  it("recognises a quota refusal by problem type alone when the code is absent", () => {
    const { code: _code, ...withoutCode } = quotaProblem;
    expect(planLimitFromProblem(402, withoutCode)?.kind).toBe("quota");
  });

  it("recognises a bare 402 as a quota refusal even without a body", () => {
    expect(planLimitFromProblem(402, undefined)).toMatchObject({
      kind: "quota",
      meter: "",
      limit: null,
      used: null,
    });
  });

  it("reads a plan restriction with its reason", () => {
    expect(planLimitFromProblem(403, readOnlyProblem)).toMatchObject({
      kind: "restricted",
      capability: "",
      reason: "subscription_read_only",
      plan: "free_demo",
    });
  });

  it("does not mistake an ordinary 403 for a plan restriction", () => {
    expect(
      planLimitFromProblem(403, {
        type: `${PROBLEM_BASE}authorization-error`,
        title: "Authorization Required",
        status: 403,
      }),
    ).toBeNull();
  });

  it("does not treat a non-numeric limit as zero", () => {
    expect(
      planLimitFromProblem(402, { ...quotaProblem, params: { meter: "x", limit: "lots" } }),
    ).toMatchObject({ limit: null, used: null });
  });
});

describe("planLimitFromGraphQLErrors", () => {
  it("finds the refusal when it is not the first error in the response", () => {
    const notice = planLimitFromGraphQLErrors([
      { message: "Name is required", extensions: { code: "REQUIRED" } },
      {
        message: "Limit reached",
        extensions: {
          code: "QUOTA_EXCEEDED",
          type: "quota-exceeded",
          params: { meter: "customers.total", limit: 8, used: 8, plan: "free_demo" },
        },
      },
    ]);

    expect(notice).toEqual({
      kind: "quota",
      meter: "customers.total",
      limit: 8,
      used: 8,
      plan: "free_demo",
      message: "Limit reached",
    });
  });

  it("returns null for errors that are not plan limits", () => {
    expect(
      planLimitFromGraphQLErrors([{ message: "nope", extensions: { code: "FORBIDDEN" } }]),
    ).toBeNull();
  });
});

describe("transports report plan-limit refusals", () => {
  let handler: ReturnType<typeof vi.fn<(notice: PlanLimitNotice) => boolean>>;

  beforeEach(() => {
    clearCsrfToken();
    setCsrfToken("csrf");
    handler = vi.fn<(notice: PlanLimitNotice) => boolean>(() => true);
    setPlanLimitHandler(handler);
  });

  afterEach(() => {
    setPlanLimitHandler(null);
    clearCsrfToken();
    vi.unstubAllGlobals();
  });

  it("hands a REST quota refusal to the handler and marks the thrown error explained", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(quotaProblem, 402)),
    );

    const error = await api.post("/shipments/", { proNumber: "1" }).catch((caught) => caught);

    expect(error).toBeInstanceOf(ApiRequestError);
    expect(handler).toHaveBeenCalledTimes(1);
    expect(handler.mock.calls[0][0]).toMatchObject({ kind: "quota", meter: "shipments.total" });
    expect(isExplainedPlanLimitError(error)).toBe(true);
    // The schema keeps the code, so callers can read it off the error as well.
    expect((error as ApiRequestError).data.code).toBe("QUOTA_EXCEEDED");
  });

  it("does not mark the error explained when the handler declines it", async () => {
    handler.mockReturnValue(false);
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(readOnlyProblem, 403)),
    );

    const error = await api.patch("/customers/cus_1/", {}).catch((caught) => caught);

    expect(handler).toHaveBeenCalledTimes(1);
    expect(isExplainedPlanLimitError(error)).toBe(false);
  });

  it("leaves ordinary failures alone", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse(
          { type: `${PROBLEM_BASE}validation-error`, title: "Validation Failed", status: 400 },
          400,
        ),
      ),
    );

    const error = await api.post("/customers/", {}).catch((caught) => caught);

    expect(handler).not.toHaveBeenCalled();
    expect(isExplainedPlanLimitError(error)).toBe(false);
  });

  it("hands a GraphQL mutation's PLAN_RESTRICTED extension to the handler", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse(
          {
            data: null,
            errors: [
              {
                message: "Integrations are not included in the free demo",
                path: ["createIntegration"],
                extensions: {
                  code: "PLAN_RESTRICTED",
                  type: "plan-restricted",
                  params: { capability: "integrations", reason: "", plan: "free_demo" },
                },
              },
            ],
          },
          200,
        ),
      ),
    );

    const error = await requestGraphQL({
      document: "mutation CreateIntegration { createIntegration { id } }",
    }).catch((caught) => caught);

    expect(error).toBeInstanceOf(GraphQLRequestError);
    expect(handler).toHaveBeenCalledTimes(1);
    expect(handler.mock.calls[0][0]).toMatchObject({
      kind: "restricted",
      capability: "integrations",
    });
    expect(isExplainedPlanLimitError(error)).toBe(true);
  });

  it("does not drive the dialog from a query's partial errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse(
          {
            data: { organization: { id: "org_1", integrations: null } },
            errors: [
              {
                message: "restricted",
                path: ["organization", "integrations"],
                extensions: { code: "PLAN_RESTRICTED", params: { capability: "integrations" } },
              },
            ],
          },
          200,
        ),
      ),
    );

    await requestGraphQL({ document: "query Org { organization { id integrations { id } } }" });

    expect(handler).not.toHaveBeenCalled();
  });
});

describe("reportPlanLimit", () => {
  afterEach(() => setPlanLimitHandler(null));

  it("reports nothing and explains nothing without a handler", () => {
    const error = new ApiRequestError(402, quotaProblem);
    expect(reportPlanLimit(error)).toBe(false);
    expect(isExplainedPlanLimitError(error)).toBe(false);
  });

  it("treats a throwing handler as not having shown the notice", () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    setPlanLimitHandler(() => {
      throw new Error("boom");
    });
    const error = new ApiRequestError(402, quotaProblem);

    expect(reportPlanLimit(error)).toBe(false);
    expect(isExplainedPlanLimitError(error)).toBe(false);
    consoleError.mockRestore();
  });

  it("recognises a thrown error structurally", () => {
    expect(planLimitFromError(new ApiRequestError(402, quotaProblem))?.kind).toBe("quota");
    expect(planLimitFromError(new Error("plain"))).toBeNull();
    expect(planLimitFromError("string")).toBeNull();
  });
});
