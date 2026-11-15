import { renderHook } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest, UnavailableFeature } from "@trenova/shared/types/permission";
import { afterEach, describe, expect, it } from "vitest";
import { isFeatureOnPlan, planFeatureOf, usePlanFeature } from "./use-plan-feature";

// GET /me/permissions lists every feature the plan leaves out, in catalog order.
const factoring: UnavailableFeature = {
  feature: "tms.factoring",
  featureName: "Factoring",
  addOn: "ACC",
  addOnName: "Accounting & Cash",
};
const timeOff: UnavailableFeature = {
  feature: "workforce.time_off",
  featureName: "Time off",
  addOn: "HRS",
  addOnName: "HR Suite",
};

function manifestLeavingOut(unavailableFeatures: UnavailableFeature[]): PermissionManifest {
  return { permissions: {}, routeAccess: {}, unavailableFeatures } as unknown as PermissionManifest;
}

describe("planFeatureOf", () => {
  it("answers available, with no names, for a feature the plan includes", () => {
    expect(planFeatureOf(manifestLeavingOut([factoring]), "tms.billing")).toEqual({
      available: true,
      featureName: "",
      addOnName: "",
    });
  });

  it("names the feature and its add-on when the plan leaves it out", () => {
    expect(planFeatureOf(manifestLeavingOut([factoring, timeOff]), "workforce.time_off")).toEqual({
      available: false,
      featureName: "Time off",
      addOnName: "HR Suite",
    });
  });

  it("answers available before the manifest has loaded", () => {
    expect(planFeatureOf(null, "tms.factoring")).toEqual({
      available: true,
      featureName: "",
      addOnName: "",
    });
  });
});

describe("usePlanFeature", () => {
  afterEach(() => usePermissionStore.getState().clearPermissions());

  it("reads the permission store's manifest", () => {
    usePermissionStore.setState({ manifest: manifestLeavingOut([factoring]) });

    expect(renderHook(() => usePlanFeature("tms.factoring")).result.current).toEqual({
      available: false,
      featureName: "Factoring",
      addOnName: "Accounting & Cash",
    });
    expect(renderHook(() => usePlanFeature("tms.billing")).result.current.available).toBe(true);
  });

  it("answers available while no manifest is held", () => {
    expect(renderHook(() => usePlanFeature("tms.factoring")).result.current.available).toBe(true);
  });
});

describe("isFeatureOnPlan", () => {
  it("is false only for a listed feature, and true for an entry naming none", () => {
    expect(isFeatureOnPlan([factoring], "tms.factoring")).toBe(false);
    expect(isFeatureOnPlan([factoring], "tms.billing")).toBe(true);
    expect(isFeatureOnPlan([factoring], undefined)).toBe(true);
  });
});
