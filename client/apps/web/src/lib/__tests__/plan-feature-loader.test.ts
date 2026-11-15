import { PlanFeature } from "@/lib/plan-feature";
import { createPlanFeatureLoader } from "@/lib/plan-feature-loader";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest, UnavailableFeature } from "@trenova/shared/types/permission";
import type { LoaderFunctionArgs } from "react-router";
import { afterEach, describe, expect, it } from "vitest";

const dispatchIntelligence: UnavailableFeature = {
  feature: "tms.dispatch_intelligence",
  featureName: "Dispatch intelligence",
  addOn: "DSP",
  addOnName: "Dispatch Intelligence",
};

function holdManifest(unavailableFeatures: UnavailableFeature[]) {
  usePermissionStore.setState({
    manifest: {
      permissions: {},
      routeAccess: {},
      unavailableFeatures,
    } as unknown as PermissionManifest,
    lastFetched: Date.now(),
  });
}

async function run() {
  const loader = createPlanFeatureLoader(PlanFeature.DispatchIntelligence);
  try {
    return {
      value: await loader({
        request: new Request("http://localhost/dispatch/outlook"),
        params: {},
        context: undefined,
      } as unknown as LoaderFunctionArgs),
      thrown: undefined,
    };
  } catch (thrown) {
    return { value: undefined, thrown };
  }
}

describe("createPlanFeatureLoader", () => {
  afterEach(() => usePermissionStore.getState().clearPermissions());

  it("lets the page open when the plan includes the feature", async () => {
    holdManifest([]);

    expect(await run()).toEqual({ value: null, thrown: undefined });
  });

  it("throws the plan refusal naming the add-on when the plan leaves the feature out", async () => {
    holdManifest([dispatchIntelligence]);

    const { thrown } = await run();

    expect(thrown).toBeInstanceOf(Response);
    const body = (await (thrown as Response).json()) as Record<string, unknown>;
    expect(body.code).toBe("PLAN_RESTRICTED");
    expect(body.params).toEqual({
      feature: "tms.dispatch_intelligence",
      addOn: "Dispatch Intelligence",
    });

    // The router hands the error boundary the thrown response's status and parsed body; it
    // reads that the way it reads a refused read.
    const routeError = { status: 403, statusText: "Plan Restricted", internal: false, data: body };
    expect(describeError(routeError)).toMatchObject({
      kind: "plan-restricted",
      plan: { feature: "tms.dispatch_intelligence", addOn: "Dispatch Intelligence" },
    });
  });
});
