import { act, cleanup, render, screen } from "@testing-library/react";
import { api, clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { isExplainedError } from "@trenova/shared/lib/explained-errors";
import { setPlanLimitHandler } from "@trenova/shared/lib/plan-limit";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DriverAppPlanGate } from "../_components/driver-app-plan-gate";
import { installDashPlanLimitHandler, resetDriverAppPlan } from "./plan-handler";

// The API's refusal when the company's plan leaves the driver portal out
// (pkg/errortypes NewFeatureRestrictionError): PLAN_RESTRICTED naming the feature and its add-on.
function planRefusal(feature: string, addOn: string): Response {
  return new Response(
    JSON.stringify({
      type: "https://trenova.app/problems/plan-restricted",
      title: "Plan Restricted",
      status: 403,
      code: "PLAN_RESTRICTED",
      params: { feature, addOn },
    }),
    { status: 403, headers: { "Content-Type": "application/json" } },
  );
}

beforeEach(() => {
  clearCsrfToken();
  setCsrfToken("csrf");
  installDashPlanLimitHandler();
});

afterEach(() => {
  cleanup();
  setPlanLimitHandler(null);
  resetDriverAppPlan();
  clearCsrfToken();
  vi.unstubAllGlobals();
});

describe("the driver app on a plan without it", () => {
  it("shows one plain message instead of the app once sign-in is refused", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => planRefusal("tms.driver_portal", "Driver Pay")),
    );
    render(
      <DriverAppPlanGate>
        <p>Sign in</p>
      </DriverAppPlanGate>,
    );
    expect(screen.getByText("Sign in")).toBeInTheDocument();

    let error: unknown;
    await act(async () => {
      error = await api.post("/auth/login", { username: "d", password: "p" }).catch((e) => e);
    });

    expect(isExplainedError(error)).toBe(true);
    expect(
      screen.getByRole("heading", { name: "The driver app is not part of your company's plan." }),
    ).toBeInTheDocument();
    expect(screen.getByText("Ask your dispatcher.")).toBeInTheDocument();
    expect(screen.queryByText("Sign in")).toBeNull();
    expect(document.body.textContent).not.toContain("tms.driver_portal");
  });

  it("does the same when a read is refused", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => planRefusal("tms.driver_portal", "Driver Pay")),
    );
    render(
      <DriverAppPlanGate>
        <p>Loads</p>
      </DriverAppPlanGate>,
    );

    await act(async () => {
      await api.get("/users/me/").catch(() => undefined);
    });

    expect(screen.getByText("Ask your dispatcher.")).toBeInTheDocument();
  });

  it("leaves another add-on's refusal to the screen that asked", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => planRefusal("workforce.time_off", "HR Suite")),
    );
    render(
      <DriverAppPlanGate>
        <p>Loads</p>
      </DriverAppPlanGate>,
    );

    let error: unknown;
    await act(async () => {
      error = await api.get("/me/leave/").catch((e) => e);
    });

    expect(isExplainedError(error)).toBe(false);
    expect(screen.getByText("Loads")).toBeInTheDocument();
  });
});
