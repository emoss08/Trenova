import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError, api, clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { GraphQLRequestError, requestGraphQL } from "@trenova/shared/lib/graphql";
import { isExplainedError } from "@trenova/shared/lib/explained-errors";
import {
  planLimitFromError,
  planLimitFromGraphQLErrors,
  planLimitFromProblem,
  planRestrictedRefusal,
  planRestrictionOf,
  reportPlanLimit,
  setPlanLimitHandler,
  suspensionOf,
  type PlanLimitHandler,
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
      feature: "",
      addOn: "",
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
    expect(isExplainedError(error)).toBe(true);
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
    expect(isExplainedError(error)).toBe(false);
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
    expect(isExplainedError(error)).toBe(false);
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
                  params: {
                    feature: "platform.administration",
                    addOn: "",
                    reason: "",
                    plan: "free_demo",
                  },
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
      feature: "platform.administration",
    });
    expect(isExplainedError(error)).toBe(true);
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
                extensions: {
                  code: "PLAN_RESTRICTED",
                  params: { feature: "platform.administration" },
                },
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

describe("both transports tell the handler whether a read or a write was refused", () => {
  const addOnRefusal = {
    type: `${PROBLEM_BASE}plan-restricted`,
    title: "Plan Restricted",
    status: 403,
    code: "PLAN_RESTRICTED",
    params: { feature: "tms.factoring", addOn: "Accounting & Cash" },
  };
  let handler: ReturnType<typeof vi.fn<PlanLimitHandler>>;

  beforeEach(() => {
    clearCsrfToken();
    setCsrfToken("csrf");
    handler = vi.fn<PlanLimitHandler>(() => false);
    setPlanLimitHandler(handler);
  });

  afterEach(() => {
    setPlanLimitHandler(null);
    clearCsrfToken();
    vi.unstubAllGlobals();
  });

  it("a REST GET is a read and a POST a write", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse(addOnRefusal, 403)),
    );

    await api.get("/factoring-companies/").catch(() => undefined);
    await api.post("/factoring-companies/", {}).catch(() => undefined);

    expect(handler.mock.calls.map(([notice, request]) => [notice.kind, request])).toEqual([
      ["restricted", "read"],
      ["restricted", "write"],
    ]);
  });

  it("a GraphQL query is a read and a mutation a write", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse(
          {
            data: null,
            errors: [
              {
                message: "refused",
                extensions: { code: "PLAN_RESTRICTED", params: addOnRefusal.params },
              },
            ],
          },
          200,
        ),
      ),
    );

    await requestGraphQL({ document: "query Q { factoringCompanies { id } }" }).catch(
      () => undefined,
    );
    await requestGraphQL({ document: "mutation M { assignFactoring { id } }" }).catch(
      () => undefined,
    );

    expect(handler.mock.calls.map(([, request]) => request)).toEqual(["read", "write"]);
  });

  it("reads the add-on refusal off a thrown error, and nothing off other refusals", () => {
    expect(planRestrictionOf(new ApiRequestError(403, addOnRefusal))).toEqual({
      feature: "tms.factoring",
      addOn: "Accounting & Cash",
    });
    expect(planRestrictionOf(new ApiRequestError(403, readOnlyProblem))).toBeNull();
    expect(planRestrictionOf(new ApiRequestError(402, quotaProblem))).toBeNull();
  });
});

describe("planRestrictedRefusal", () => {
  it("carries the body the API sends for the same refusal", () => {
    const refusal = planRestrictedRefusal({
      feature: "tms.factoring",
      addOn: "Accounting & Cash",
    });

    expect(refusal.status).toBe(403);
    const body = refusal.data as Record<string, unknown>;
    expect(planLimitFromProblem(403, body)).toMatchObject({
      kind: "restricted",
      feature: "tms.factoring",
      addOn: "Accounting & Cash",
    });
  });
});

describe("reportPlanLimit", () => {
  afterEach(() => setPlanLimitHandler(null));

  it("reports nothing and explains nothing without a handler", () => {
    const error = new ApiRequestError(402, quotaProblem);
    expect(reportPlanLimit(error, "write")).toBe(false);
    expect(isExplainedError(error)).toBe(false);
  });

  it("treats a throwing handler as not having shown the notice", () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    setPlanLimitHandler(() => {
      throw new Error("boom");
    });
    const error = new ApiRequestError(402, quotaProblem);

    expect(reportPlanLimit(error, "write")).toBe(false);
    expect(isExplainedError(error)).toBe(false);
    consoleError.mockRestore();
  });

  it("recognises a thrown error structurally", () => {
    expect(planLimitFromError(new ApiRequestError(402, quotaProblem))?.kind).toBe("quota");
    expect(planLimitFromError(new Error("plain"))).toBeNull();
    expect(planLimitFromError("string")).toBeNull();
  });
});

describe("suspensionOf", () => {
  const suspended = (manageUrl: string) =>
    new ApiRequestError(403, {
      type: PROBLEM_BASE + "plan-restricted",
      code: "PLAN_RESTRICTED",
      detail: "This organization's Trenova account is suspended.",
      params: { feature: "", addOn: "", reason: "organization_suspended", plan: "", manageUrl },
    });

  it("reads the console address of a suspended organization", () => {
    expect(suspensionOf(suspended("https://console.test/instances/1/plan"))).toEqual({
      manageUrl: "https://console.test/instances/1/plan",
    });
  });

  it("drops an address that is not a plain web address", () => {
    expect(suspensionOf(suspended("javascript:alert(1)"))).toEqual({ manageUrl: "" });
    expect(suspensionOf(suspended("https://user:pass@console.test/"))).toEqual({ manageUrl: "" });
  });

  it("is null for any other refusal", () => {
    expect(
      suspensionOf(
        new ApiRequestError(403, {
          code: "PLAN_RESTRICTED",
          params: { reason: "subscription_read_only", manageUrl: "https://console.test/" },
        }),
      ),
    ).toBeNull();
    expect(suspensionOf(new Error("plain"))).toBeNull();
  });
});
