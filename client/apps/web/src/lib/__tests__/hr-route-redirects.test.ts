import { gateOutcome, hasRoute, redirectTarget } from "@/test/route-tree";
import { describe, expect, it } from "vitest";

/**
 * Links already sent to people — notification rows, bookmarks — still carry the
 * old dispatch paths, so each one keeps working and lands on its HR home.
 */
const MOVED = [
  ["/dispatch/workers", "/hr/workers"],
  ["/dispatch/configuration-files/checklist-templates", "/hr/checklist-templates"],
  ["/dispatch/configuration-files/review-templates", "/hr/review-templates"],
  ["/dispatch/configuration-files/training-courses", "/hr/training-courses"],
  ["/dispatch/configuration-files/credential-types", "/hr/credential-types"],
  ["/dispatch/configuration-files/holidays", "/hr/holidays"],
  ["/dispatch/configuration-files/pto-policies", "/hr/pto-policies"],
] as const;

async function locationFor(path: string, search = ""): Promise<string | null> {
  return redirectTarget(await gateOutcome(path, { search }));
}

describe("HR routes", () => {
  it("registers every page under /hr", () => {
    for (const [, next] of MOVED) {
      expect(hasRoute(next)).toBe(true);
    }
  });

  it("redirects the old dispatch paths to their HR home", async () => {
    for (const [legacy, next] of MOVED) {
      await expect(locationFor(legacy)).resolves.toBe(next);
    }
  });

  it("keeps the query string, so a deep link still opens the panel it named", async () => {
    await expect(
      locationFor("/dispatch/workers", "?panelType=edit&panelEntityId=wrk_1&tab=credentials"),
    ).resolves.toBe("/hr/workers?panelType=edit&panelEntityId=wrk_1&tab=credentials");
  });
});
