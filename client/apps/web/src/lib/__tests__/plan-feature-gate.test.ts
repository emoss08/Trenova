import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest, UnavailableFeature } from "@trenova/shared/types/permission";
import { RouteRefusal } from "@trenova/shared/lib/router/refusal";
import { enterContext } from "@trenova/shared/testing/gate-context";
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
  const gate = requirePlanFeature(PlanFeature.DispatchIntelligence);
  try {
    return { value: await gate(enterContext("/dispatch/outlook")), thrown: undefined };
  } catch (thrown) {
    return { value: undefined, thrown };
  }
}

describe("requirePlanFeature", () => {
  afterEach(() => usePermissionStore.getState().clearPermissions());

  it("lets the page open when the plan includes the feature", async () => {
    holdManifest([]);

    expect(await run()).toEqual({ value: undefined, thrown: undefined });
  });

  it("throws the plan refusal naming the add-on when the plan leaves the feature out", async () => {
    holdManifest([dispatchIntelligence]);

    const { thrown } = await run();

    expect(thrown).toBeInstanceOf(RouteRefusal);
    const body = (thrown as RouteRefusal).data as Record<string, unknown>;
    expect(body.code).toBe("PLAN_RESTRICTED");
    expect(body.params).toEqual({
      feature: "tms.dispatch_intelligence",
      addOn: "Dispatch Intelligence",
    });

    // The route's error component is handed the refusal itself, and reads it the way it reads
    // a refused read.
    expect(describeError(thrown)).toMatchObject({
      kind: "plan-restricted",
      plan: { feature: "tms.dispatch_intelligence", addOn: "Dispatch Intelligence" },
    });
  });
});
