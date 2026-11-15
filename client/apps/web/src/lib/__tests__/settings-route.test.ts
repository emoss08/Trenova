import { routeAt } from "@/test/route-tree";
import { describe, expect, it } from "vitest";

describe("Settings route", () => {
  it("is a page of its own, framed as a room beside the module rail", () => {
    expect(routeAt("/settings").options.staticData?.frame).toBe("room");
  });
});
