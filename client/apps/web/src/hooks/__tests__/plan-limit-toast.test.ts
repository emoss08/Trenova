import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { reportPlanLimit, setPlanLimitHandler } from "@trenova/shared/lib/plan-limit";
import { handleMutationError } from "@/hooks/use-api-mutation";

const toastError = vi.hoisted(() => vi.fn());
vi.mock("sonner", () => ({ toast: { error: toastError, success: vi.fn() } }));

function readOnlyError() {
  return new ApiRequestError(403, {
    type: "https://trenova.app/problems/plan-restricted",
    title: "Plan Restricted",
    status: 403,
    code: "PLAN_RESTRICTED",
    params: { capability: "", reason: "subscription_read_only", plan: "free_demo" },
  });
}

describe("handleMutationError and the plan-limit dialog", () => {
  beforeEach(() => toastError.mockReset());
  afterEach(() => setPlanLimitHandler(null));

  it("stays quiet for a refusal the dialog has already explained", () => {
    setPlanLimitHandler(() => true);
    const error = readOnlyError();
    reportPlanLimit(error);

    handleMutationError({ error, resourceName: "Customer" });

    expect(toastError).not.toHaveBeenCalled();
  });

  it("still reports a plan refusal the dialog declined, so the person is told something", () => {
    setPlanLimitHandler(() => false);
    const error = readOnlyError();
    reportPlanLimit(error);

    handleMutationError({ error, resourceName: "Customer" });

    expect(toastError).toHaveBeenCalledTimes(1);
  });
});
