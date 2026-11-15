import { gateOutcome, redirectTarget, routeAt } from "@/test/route-tree";
import { describe, expect, it } from "vitest";

async function locationFor(path: string, search = ""): Promise<string | null> {
  return redirectTarget(await gateOutcome(path, { search }));
}

describe("Settings route", () => {
  it("is a page of its own, framed as a room beside the module rail", () => {
    expect(routeAt("/settings").options.staticData?.frame).toBe("room");
  });

  it("sends a link from an email already sent to the notifications section", async () => {
    await expect(locationFor("/", "?settings=notifications")).resolves.toBe(
      "/settings?section=notifications",
    );
  });

  it("opens Settings at the top for an old link naming a section that no longer exists", async () => {
    await expect(locationFor("/", "?settings=appearance")).resolves.toBe("/settings");
  });

  it("leaves Home alone when the address carries no old Settings link", async () => {
    await expect(locationFor("/")).resolves.toBeNull();
    await expect(locationFor("/", "?tab=today")).resolves.toBeNull();
  });
});
