import type { ResourceInvalidationEvent } from "@trenova/shared/hooks/realtime-patching";
import { describe, expect, it } from "vitest";
import { isPermissionChangeFor, PERMISSIONS_RESOURCE } from "../permission-realtime";

function event(resource: string, organizationId = "org_a"): ResourceInvalidationEvent {
  return { resource, organizationId, businessUnitId: "bu_a", action: "changed" };
}

describe("isPermissionChangeFor", () => {
  it("accepts a permission change in the reader's organization", () => {
    expect(isPermissionChangeFor(event(PERMISSIONS_RESOURCE), "org_a")).toBe(true);
  });

  it("ignores another organization's change", () => {
    expect(isPermissionChangeFor(event(PERMISSIONS_RESOURCE, "org_b"), "org_a")).toBe(false);
  });

  it("ignores other resources and empty events", () => {
    expect(isPermissionChangeFor(event("shipment"), "org_a")).toBe(false);
    expect(isPermissionChangeFor(null, "org_a")).toBe(false);
  });
});
